package proxy

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/juliotorresmoreno/lipstick/helper"
	"github.com/juliotorresmoreno/lipstick/server/common"
)

type Proxy struct {
	listener net.Listener
	addr     string
	cert     string
	key      string
}

func SetupProxy(addr string, certFile, keyFile string) *Proxy {
	var listener net.Listener
	var err error
	proxy := Proxy{
		addr: addr,
		cert: certFile,
		key:  keyFile,
	}
	if certFile != "" && keyFile != "" {
		listener, err = proxy.serveTLS(addr, certFile, keyFile)
	} else {
		listener, err = proxy.serve(addr)
	}
	if err != nil {
		log.Fatal(err)
	}
	proxy.listener = listener

	return &proxy
}

func (p *Proxy) serve(addr string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	return listener, err
}

func (p *Proxy) serveTLS(addr string, certFile, keyFile string) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	tlsListener, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return tlsListener, nil
}

func (proxy *Proxy) Listen(manager chan *common.RemoteConn) {
	log.Println("Listening proxy on", proxy.addr)
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		domain, err := helper.GetDomainName(conn)

		if err != nil {
			log.Println(err)
			conn.Close()
			continue
		}

		manager <- &common.RemoteConn{Conn: conn, Domain: domain}
	}
}

func (proxy *Proxy) Close() error {
	return proxy.listener.Close()
}
