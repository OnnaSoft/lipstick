package manager

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/logger"
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
	n, err := p.ReadWriter.Write(b)
	if err != nil {
		logger.Default.Error("Error writing to ReadWriter:", err)
		return n, err
	}
	err = p.Flush()
	if err != nil {
		logger.Default.Error("Error flushing ReadWriter:", err)
		return n, err
	}
	return n, nil
}

func (p *ProxyNotificationConn) Close() error {
	_, err := p.conn.Write([]byte("close"))
	if err != nil {
		logger.Default.Debug("Error writing 'close' to connection:", err)
	}
	closeErr := p.conn.Close()
	if closeErr != nil {
		logger.Default.Error("Error closing connection:", closeErr)
	}
	if err != nil {
		return err
	}
	return closeErr
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

	logger.Default.Info("Manager setup completed")

	return manager
}

func (m *Manager) AddHub(domain string, hub *NetworkHub) {
	logger.Default.Debug("Adding hub for domain:", domain)
	m.hubs.Store(domain, hub)
}

func (m *Manager) GetHub(domain string) (*NetworkHub, bool) {
	value, ok := m.hubs.Load(domain)
	if !ok {
		logger.Default.Debug("hub not found for domain:", domain)
		return nil, false
	}
	return value.(*NetworkHub), true
}

func (m *Manager) RemoveHub(domain string) {
	logger.Default.Debug("Removing hub for domain:", domain)
	m.hubs.Delete(domain)
}

func (manager *Manager) handleTunnel(conn net.Conn, req *http.Request, ticket string) {
	host := req.Host
	domainName := strings.Split(host, ":")[0]

	domain, ok := manager.GetHub(domainName)
	if !ok {
		logger.Default.Error("hub not found for domain:", domainName)
		return
	}

	logger.Default.Debug("Handling tunnel for domain:", domainName)
	domain.serverRequests <- &request{ticket: ticket, conn: conn}
}

func (manager *Manager) Listen() {
	var err error
	config, configErr := config.GetConfig()
	if configErr != nil {
		logger.Default.Error("Error getting config:", configErr)
		return
	}

	logger.Default.Info("Listening manager on ", config.Manager.Address)

	var listener net.Listener
	if manager.tlsConfig != nil {
		listener, err = tls.Listen("tcp", config.Manager.Address, manager.tlsConfig)
	} else {
		listener, err = net.Listen("tcp", config.Manager.Address)
	}
	if err != nil {
		logger.Default.Error("Error on listen:", err)
		return
	}

	l := NewCustomListener(listener, manager)
	engineErr := manager.engine.RunListener(l)
	if engineErr != nil {
		logger.Default.Error("Error running listener:", engineErr)
	}
}

func (manager *Manager) HandleHTTPConn(conn net.Conn, req *http.Request) {
	host := req.Host
	domain := strings.Split(host, ":")[0]

	hub, ok := manager.GetHub(domain)
	if !ok {
		logger.Default.Error("Hub not found for domain:", domain)
		_, err := fmt.Fprint(conn, helper.BadGatewayResponse)
		if err != nil {
			logger.Default.Error("Error writing BadGatewayResponse to HTTP connection:", err)
		}
		conn.Close()
		return
	}

	logger.Default.Debug("Handling HTTP connection for domain:", domain)
	if remoteConn, ok := conn.(*helper.RemoteConn); ok {
		hub.incomingClientConn <- remoteConn
		return
	}

	remoteConn := &helper.RemoteConn{Conn: conn, Domain: domain}
	remoteConn.MonitorIdle(30 * time.Second)
	hub.incomingClientConn <- remoteConn
}

func (manager *Manager) HandleTCPConn(conn net.Conn) {
	domain, err := helper.GetDomainName(conn)
	if err != nil {
		logger.Default.Error("Error getting domain name from TCP connection:", err)
		_, writeErr := fmt.Fprint(conn, helper.BadGatewayResponse)
		if writeErr != nil {
			logger.Default.Error("Error writing BadGatewayResponse to TCP connection:", writeErr)
		}
		conn.Close()
		return
	}

	hub, ok := manager.GetHub(domain)
	if !ok {
		logger.Default.Error("Hub not found for domain:", domain)
		_, err := fmt.Fprint(conn, helper.BadGatewayResponse)
		if err != nil {
			logger.Default.Error("Error writing BadGatewayResponse to TCP connection:", err)
		}
		conn.Close()
		return
	}

	logger.Default.Debug("Handling TCP connection for domain:", domain)
	if remoteConn, ok := conn.(*helper.RemoteConn); ok {
		hub.incomingClientConn <- remoteConn
		return
	}

	remoteConn := &helper.RemoteConn{Conn: conn, Domain: domain}
	remoteConn.MonitorIdle(30 * time.Second)
	hub.incomingClientConn <- remoteConn
}
