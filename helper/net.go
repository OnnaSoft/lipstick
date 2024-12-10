package helper

import (
	"log"
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

func MonitorIdle(c *RemoteConn, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if !c.used {
				c.closeOnce.Do(func() {
					log.Printf("Closing idle connection to domain: %s", c.Domain)
					c.Conn.Close()
				})
				return
			}
			c.used = false // Resetea el estado usado para la próxima verificación
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
