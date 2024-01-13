package proxy

import (
	"log"
	"net"

	"github.com/juliotorresmoreno/kitty/manager"
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

func (proxy *Proxy) Listen(manager *manager.Manager) {
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		manager.Pipe <- conn
	}
}
