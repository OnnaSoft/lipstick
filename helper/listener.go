package helper

import (
	"crypto/tls"
	"errors"
	"net"
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
	onHTTPConn func(net.Conn)
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

func (l *ListenerManager) OnHTTPConn(fn func(net.Conn)) {
	l.onHTTPConn = fn
}

func (l *ListenerManager) OnTCPConn(fn func(net.Conn)) {
	l.onTCPConn = fn
}

func (l *ListenerManager) ListenAndServe() error {
	if l.onListen != nil {
		l.onListen()
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			if l.onClose != nil {
				l.onClose()
			}
			return err
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			continue
		}
		wconn := NewConnWithBuffer(conn, buffer[:n])
		if IsHTTPRequest(string(buffer[:n])) {
			if l.onHTTPConn != nil {
				l.onHTTPConn(wconn)
			}
		} else {
			if l.onTCPConn != nil {
				l.onTCPConn(wconn)
			}
		}
	}
}
