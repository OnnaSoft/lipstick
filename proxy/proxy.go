package proxy

import (
	"log"
	"net"
)

type Proxy struct {
	listener net.Listener
}

func SetupProxy(addr string) *Proxy {
	proxy := Proxy{}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	proxy.listener = listener

	return &proxy
}

func (proxy *Proxy) Listen(manager chan net.Conn) {
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		manager <- conn
	}
}

func (proxy *Proxy) Close() error {
	return proxy.listener.Close()
}
