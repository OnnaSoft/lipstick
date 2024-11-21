package manager

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/server/auth"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/traffic"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type request struct {
	conn   net.Conn
	ticket string
}

type ProxyNotificationConn struct {
	Domain                   string
	AllowMultipleConnections bool
	*websocket.Conn
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

func (manager *Manager) handleTunnel(conn net.Conn, ticket string) {
	domainName, err := helper.GetDomainName(conn)
	if err != nil {
		log.Println("Unable to get domain name", err)
		return
	}

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

func (manager *Manager) HandleHTTPConn(conn net.Conn) {
	manager.HandleTCPConn(conn)
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
		fmt.Println("Domain not found")
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
