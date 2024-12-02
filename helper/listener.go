package helper

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

func NewListenerManagerTCP(addr string, tlsConfig *tls.Config) *ListenerManager {
	var l net.Listener
	var err error
	if tlsConfig != nil {
		l, err = tls.Listen("tcp", addr, tlsConfig)
		if err != nil {
			return nil
		}
		return NewListenerManager(l)
	}
	l, err = net.Listen("tcp", addr)
	if err != nil {
		return nil
	}
	return NewListenerManager(l)
}

type ListenerConn struct {
	net.Conn
}

func NewListenerConn(conn net.Conn) *ListenerConn {
	return &ListenerConn{Conn: conn}
}

func (l *ListenerConn) Close() error {
	return nil
}

func (l *ListenerConn) Accept() (net.Conn, error) {
	if l.Conn == nil {
		return nil, errors.New("listener closed")
	}
	conn := l.Conn
	l.Conn = nil
	return conn, nil
}

func (l *ListenerConn) Addr() net.Addr {
	return l.LocalAddr()
}

type ListenerManager struct {
	net.Listener
	onClose    func()
	onHTTPConn func(net.Conn, *http.Request)
	onTCPConn  func(net.Conn)
	onListen   func()
}

func NewListenerManager(l net.Listener) *ListenerManager {
	return &ListenerManager{Listener: l}
}

func (l *ListenerManager) OnListen(fn func()) {
	l.onListen = fn
}

func (l *ListenerManager) OnClose(fn func()) {
	l.onClose = fn
}

func (l *ListenerManager) OnHTTPConn(fn func(net.Conn, *http.Request)) {
	l.onHTTPConn = fn
}

func (l *ListenerManager) OnTCPConn(fn func(net.Conn)) {
	l.onTCPConn = fn
}

func (l *ListenerManager) ListenAndServe() error {
	if l.onListen != nil {
		l.onListen()
	}
	handleConnection := func(conn net.Conn) {
		if err := l.handleConnection(conn); err != nil {
			conn.Close()
		}
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return l.handleAcceptError(err)
		}
		go handleConnection(conn)
	}
}

func (l *ListenerManager) handleAcceptError(err error) error {
	if l.onClose != nil {
		l.onClose()
	}
	return err
}

func (l *ListenerManager) handleConnection(conn net.Conn) error {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return err
	}
	wconn := NewConnWithBuffer(conn, buffer[:n])
	if IsHTTPRequest(string(buffer[:n])) {
		return l.handleHTTPConn(wconn)
	}
	return l.handleTCPConn(wconn)
}

func (l *ListenerManager) handleHTTPConn(conn *ConnWithBuffer) error {
	if l.onHTTPConn != nil {
		b, err := ReadUntilHeadersEnd(conn)
		if err != nil {
			return err
		}
		conn.SetBuffer(b)
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(b)))
		if err != nil {
			return err
		}
		l.onHTTPConn(conn, req)
	}
	return nil
}

func (l *ListenerManager) handleTCPConn(conn net.Conn) error {
	if l.onTCPConn != nil {
		l.onTCPConn(conn)
	}
	return nil
}
