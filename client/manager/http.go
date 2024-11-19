package manager

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type HTTPManager struct {
}

func NewHTTPManager() *HTTPManager {
	return &HTTPManager{}
}

type CustomConn struct {
	net.Conn
	buffer []byte
}

func (c *CustomConn) SetBuffer(buffer []byte) {
	c.buffer = buffer
}

func (c *CustomConn) Read(b []byte) (n int, err error) {
	if len(c.buffer) == 0 {
		return c.Conn.Read(b)
	}
	n = copy(b, c.buffer)
	c.buffer = c.buffer[n:]
	return n, nil
}

func (m *HTTPManager) Connect(url string, headers http.Header) (*CustomConn, error) {
	// Crear la solicitud HTTP
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header = headers

	// Resolver el host y puerto del URL
	host := req.URL.Host
	if host == "" {
		host = req.URL.Hostname()
	}
	if !strings.Contains(host, ":") {
		if req.URL.Port() == "" {
			host += ":443"
		} else {
			host += ":" + req.URL.Port()
		}
	}

	var conn net.Conn
	if req.URL.Scheme == "http" || req.URL.Scheme == "ws" {
		conn, err = net.Dial("tcp", host)
	} else {
		conn, err = tls.Dial("tcp", host, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("error connecting to host: %w", err)
	}

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error writing request to connection: %w", err)
	}

	// Retorna la conexión para leer la respuesta o continuar con el uso
	return &CustomConn{Conn: conn}, nil
}
