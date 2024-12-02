package manager

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/server/auth"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/traffic"
	"github.com/gin-gonic/gin"
)

type request struct {
	conn   net.Conn
	ticket string
}

type ProxyNotificationConn struct {
	Domain                   string
	AllowMultipleConnections bool
	*bufio.ReadWriter
	conn net.Conn
}

func (p *ProxyNotificationConn) Write(b []byte) (int, error) {
	p.ReadWriter.Write(b)
	p.Flush()
	return len(b), nil
}

func (p *ProxyNotificationConn) Close() error {
	p.conn.Write([]byte("close"))
	return p.conn.Close()
}

type Manager struct {
	engine         *gin.Engine
	hubs           sync.Map
	trafficManager *traffic.TrafficManager
	authManager    auth.AuthManager
	tlsConfig      *tls.Config
}

func SetupManager(tlsConfig *tls.Config) *Manager {
	gin.SetMode(gin.ReleaseMode)

	manager := &Manager{
		hubs:           sync.Map{},
		authManager:    auth.MakeAuthManager(),
		trafficManager: traffic.NewTrafficManager(64 * 1024),
		tlsConfig:      tlsConfig,
	}

	configureRouter(manager)

	return manager
}

func (m *Manager) AddHub(domain string, hub *NetworkHub) {
	m.hubs.Store(domain, hub)
}

func (m *Manager) GetHub(domain string) (*NetworkHub, bool) {
	value, ok := m.hubs.Load(domain)
	if !ok {
		return nil, false
	}
	return value.(*NetworkHub), true
}

func (m *Manager) RemoveHub(domain string) {
	m.hubs.Delete(domain)
}

func (manager *Manager) handleTunnel(conn net.Conn, req *http.Request, ticket string) {
	host := req.Host
	domainName := strings.Split(host, ":")[0]

	domain, ok := manager.GetHub(domainName)
	if !ok {
		return
	}

	domain.serverRequests <- &request{ticket: ticket, conn: conn}
}

func (manager *Manager) Listen() {
	var err error
	config, _ := config.GetConfig()

	log.Println("Listening manager on", config.Manager.Address)

	var listener net.Listener
	if manager.tlsConfig != nil {
		listener, err = tls.Listen("tcp", config.Manager.Address, manager.tlsConfig)
	} else {
		listener, err = net.Listen("tcp", config.Manager.Address)
	}
	if err != nil {
		log.Println("Error on listen", err)
	} else {
		l := NewCustomListener(listener, manager)
		manager.engine.RunListener(l)
	}
}

func (manager *Manager) HandleHTTPConn(conn net.Conn, req *http.Request) {
	host := req.Host
	domain := strings.Split(host, ":")[0]

	hub, ok := manager.GetHub(domain)
	if !ok {
		fmt.Println("Domain ", domain, " not found")
		fmt.Fprint(conn, helper.BadGatewayResponse)
		conn.Close()
		return
	}

	if _, ok := conn.(helper.RemoteConn); ok {
		hub.incomingClientConn <- conn.(*helper.RemoteConn)
		return
	}

	hub.incomingClientConn <- &helper.RemoteConn{Conn: conn, Domain: domain}
}

func (manager *Manager) HandleTCPConn(conn net.Conn) {
	domain, err := helper.GetDomainName(conn)
	if err != nil {
		fmt.Fprint(conn, helper.BadGatewayResponse)
		conn.Close()
		return
	}
	hub, ok := manager.GetHub(domain)
	if !ok {
		fmt.Println("Domain ", domain, " not found")
		fmt.Fprint(conn, helper.BadGatewayResponse)
		conn.Close()
		return
	}

	if _, ok := conn.(helper.RemoteConn); ok {
		hub.incomingClientConn <- conn.(*helper.RemoteConn)
		return
	}

	hub.incomingClientConn <- &helper.RemoteConn{Conn: conn, Domain: domain}
}
