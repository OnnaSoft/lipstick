package manager

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/OnnaSoft/lipstick/helper"
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

	urlsToIgnore := []string{"/", "/health", "/traffic"}
	if slices.Contains(urlsToIgnore, url) {
		cl.conn <- CustomerAccepter{helper.NewConnWithBuffer(conn, buffer[:n]), nil}
		return
	}

	if strings.HasPrefix(url, "/") && len(url) > 1 && strings.Count(url, "/") == 1 {
		ticket := url[1:]
		b, err := helper.ReadUntilHeadersEnd(helper.NewConnWithBuffer(conn, buffer[:n]))
		if err != nil {
			cl.conn <- CustomerAccepter{nil, err}
			return
		}
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
		if err != nil {
			cl.conn <- CustomerAccepter{nil, err}
			return
		}
		cl.manager.handleTunnel(conn, req, ticket)
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
	return requestLine, strings.Split(parts[1], "?")[0], true
}
