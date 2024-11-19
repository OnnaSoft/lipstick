package manager

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
)

type CustomerAccepter struct {
	conn net.Conn
	err  error
}

type CustomListener struct {
	net.Listener
	conn    chan CustomerAccepter
	closed  bool
	manager *Manager
}

func NewCustomListener(l net.Listener, manager *Manager) *CustomListener {
	cl := &CustomListener{
		Listener: l,
		conn:     make(chan CustomerAccepter),
		manager:  manager,
	}
	go cl.acceptConnections()
	return cl
}

func (cl *CustomListener) Accept() (net.Conn, error) {
	ca, ok := <-cl.conn
	if !ok {
		return nil, errors.New("listener closed")
	}
	return ca.conn, ca.err
}

func (cl *CustomListener) Close() error {
	if cl.closed {
		return nil
	}
	cl.closed = true
	close(cl.conn)
	return cl.Listener.Close()
}

func (cl *CustomListener) handle(conn net.Conn) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		cl.conn <- CustomerAccepter{nil, err}
		return
	}

	requestLine, url, valid := parseRequest(buffer[:n])
	if !valid {
		cl.conn <- CustomerAccepter{nil, errors.New("invalid request")}
		return
	}

	if url == "/" {
		cl.conn <- CustomerAccepter{&CustomConn{Conn: conn, buff: buffer[:n]}, nil}
		return
	}

	if strings.HasPrefix(url, "/") && len(url) > 1 && strings.Count(url, "/") == 1 {
		ticket := url[1:]
		err := readUntilHeadersEnd(&CustomConn{Conn: conn, buff: buffer[:n]})
		if err != nil {
			cl.conn <- CustomerAccepter{nil, err}
			return
		}
		cl.manager.handleTunnel(conn, ticket)
		return
	}

	conn.Close()
	cl.conn <- CustomerAccepter{nil, errors.New("invalid request: " + requestLine)}
}

func (cl *CustomListener) acceptConnections() {
	for {
		conn, err := cl.Listener.Accept()
		if err != nil {
			if cl.closed {
				return
			}
			cl.conn <- CustomerAccepter{nil, err}
			continue
		}
		go cl.handle(conn)
	}
}

func readUntilHeadersEnd(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading headers: %w", err)
		}
		if line == "\r\n" {
			break
		}
	}
	return nil
}

func parseRequest(buffer []byte) (requestLine, url string, valid bool) {
	lines := bytes.Split(buffer, []byte("\n"))
	if len(lines) == 0 {
		return "", "", false
	}
	requestLine = string(lines[0])
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return requestLine, "", false
	}
	return requestLine, parts[1], true
}

type CustomConn struct {
	net.Conn
	buff []byte
}

func (cc *CustomConn) Read(b []byte) (n int, err error) {
	if len(cc.buff) > 0 {
		n = copy(b, cc.buff)
		cc.buff = cc.buff[n:]
		return n, nil
	}
	return cc.Conn.Read(b)
}
