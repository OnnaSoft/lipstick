package common

import "net"

type Event struct {
	UUID string
}

type RemoteConn struct {
	Domain string
	net.Conn
}
