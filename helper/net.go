package helper

import "net"

type RemoteConn struct {
	Domain string
	net.Conn
}

type ConnWithBuffer struct {
	net.Conn
	buff []byte
}

func NewConnWithBuffer(conn net.Conn, buff []byte) *ConnWithBuffer {
	return &ConnWithBuffer{Conn: conn, buff: buff}
}

func (c *ConnWithBuffer) SetBuffer(buff []byte) {
	c.buff = buff
}

func (c *ConnWithBuffer) Read(b []byte) (n int, err error) {
	if len(c.buff) > 0 {
		n = copy(b, c.buff)
		c.buff = c.buff[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}
