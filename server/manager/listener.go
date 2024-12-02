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
	"github.com/OnnaSoft/lipstick/logger"
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
		logger.Default.Error("Listener closed while accepting connections")
		return nil, errors.New("listener closed")
	}
	return ca.conn, ca.err
}

func (cl *CustomListener) Close() error {
	if cl.closed {
		logger.Default.Debug("Listener already closed")
		return nil
	}
	cl.closed = true
	close(cl.conn)
	err := cl.Listener.Close()
	if err != nil {
		logger.Default.Error("Error closing listener:", err)
	}
	return err
}

func (cl *CustomListener) handle(conn net.Conn) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		logger.Default.Error("Error reading from connection:", err)
		cl.conn <- CustomerAccepter{nil, err}
		return
	}

	requestLine, url, valid := parseRequest(buffer[:n])
	if !valid {
		logger.Default.Error("Invalid request received:", requestLine)
		cl.conn <- CustomerAccepter{nil, errors.New("invalid request")}
		return
	}

	urlsToIgnore := []string{"/", "/health", "/traffic"}
	if slices.Contains(urlsToIgnore, url) {
		logger.Default.Debug("Request to ignored URL:", url)
		cl.conn <- CustomerAccepter{helper.NewConnWithBuffer(conn, buffer[:n]), nil}
		return
	}

	if strings.HasPrefix(url, "/") && len(url) > 1 && strings.Count(url, "/") == 1 {
		ticket := url[1:]
		b, err := helper.ReadUntilHeadersEnd(helper.NewConnWithBuffer(conn, buffer[:n]))
		if err != nil {
			logger.Default.Error("Error reading headers from connection:", err)
			cl.conn <- CustomerAccepter{nil, err}
			return
		}
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
		if err != nil {
			logger.Default.Error("Error reading HTTP request:", err)
			cl.conn <- CustomerAccepter{nil, err}
			return
		}
		logger.Default.Debug("Handling tunnel with ticket:", ticket)
		cl.manager.handleTunnel(conn, req, ticket)
		return
	}

	logger.Default.Error("Invalid request with line:", requestLine)
	conn.Close()
	cl.conn <- CustomerAccepter{nil, errors.New("invalid request: " + requestLine)}
}

func (cl *CustomListener) acceptConnections() {
	for {
		conn, err := cl.Listener.Accept()
		if err != nil {
			if cl.closed {
				logger.Default.Debug("Stopped accepting connections, listener is closed")
				return
			}
			logger.Default.Error("Error accepting connection:", err)
			cl.conn <- CustomerAccepter{nil, err}
			continue
		}
		logger.Default.Debug("New connection accepted")
		go cl.handle(conn)
	}
}

func parseRequest(buffer []byte) (requestLine, url string, valid bool) {
	lines := bytes.Split(buffer, []byte("\n"))
	if len(lines) == 0 {
		logger.Default.Error("Received empty request buffer")
		return "", "", false
	}
	requestLine = string(lines[0])
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		logger.Default.Error("Invalid request line format:", requestLine)
		return requestLine, "", false
	}
	url = strings.Split(parts[1], "?")[0]
	return requestLine, url, true
}
