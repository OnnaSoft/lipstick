package helper

import (
	"net"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/logger"
)

type RemoteConn struct {
	Domain string
	net.Conn
	used      bool
	closeOnce sync.Once
}

func (c *RemoteConn) Read(b []byte) (n int, err error) {
	c.markAsUsed()
	return c.Conn.Read(b)
}

func (c *RemoteConn) Write(b []byte) (n int, err error) {
	c.markAsUsed()
	return c.Conn.Write(b)
}

func (c *RemoteConn) MonitorIdle(timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(timeout)
		defer ticker.Stop()

		<-ticker.C
		if !c.used {
			logger.Default.Debug("Closing connection to ", c.Domain)
			c.Close()
		}
	}()
}

// Marca la conexión como usada
func (c *RemoteConn) markAsUsed() {
	c.used = true
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

func (c *ConnWithBuffer) Close() error {
	return c.Conn.Close()
}

var publicIP string

func GetPublicIP() string {
	if publicIP != "" {
		return publicIP
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Default.Info("Error al obtener las interfaces de red: ", err)
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				publicIP = ipNet.IP.String()
				return publicIP
			}
		}
	}

	logger.Default.Info("No se encontró una dirección IP pública válida")
	return ""
}
