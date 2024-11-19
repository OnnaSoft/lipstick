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

func NewCustomListener(l net.Listener) *CustomListener {
	cl := &CustomListener{
		Listener: l,
		conn:     make(chan CustomerAccepter),
	}
	go cl.accept()
	return cl
}

func (cl *CustomListener) Accept() (net.Conn, error) {
	ca := <-cl.conn
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

func (cl *CustomListener) accept() {
	for {
		conn, err := cl.Listener.Accept()
		if err != nil {
			cl.conn <- CustomerAccepter{nil, err}
			continue
		}
		b := make([]byte, 1024)
		n, err := conn.Read(b)
		if err != nil {
			cl.conn <- CustomerAccepter{nil, err}
			continue
		}
		line := strings.Split(string(b[:n]), "\n")[0]
		url := strings.Split(line, " ")[1]

		if url == "/" {
			cl.conn <- CustomerAccepter{&CustomConn{Conn: conn, buff: b[:n]}, nil}
			continue
		}

		if strings.HasPrefix(url, "/") && len(url) > 1 && strings.Count(url, "/") == 1 {
			ticket := url[1:]
			err := readUntilHeadersEnd(&CustomConn{Conn: conn, buff: b[:n]})
			if err != nil {
				cl.conn <- CustomerAccepter{nil, err}
				continue
			}
			go cl.manager.handleTunnel(conn, ticket)
			continue
		}

		conn.Close()
		cl.conn <- CustomerAccepter{nil, errors.New("invalid request")}
	}
}

func readUntilHeadersEnd(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	var buffer bytes.Buffer
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading from connection: %w", err)
		}
		buffer.WriteString(line)
		if buffer.String() == "\r\n" || bytes.HasSuffix(buffer.Bytes(), []byte("\r\n\r\n")) {
			break
		}
	}

	return nil
}

type CustomConn struct {
	net.Conn
	buff []byte
}

func (cc *CustomConn) Read(b []byte) (n int, err error) {
	if len(cc.buff) == 0 {
		return cc.Conn.Read(b)
	}
	n = copy(b, cc.buff)
	cc.buff = cc.buff[n:]
	return n, nil
}
