package manager

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"

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

var badGatewayHeader = `HTTP/1.1 502 Bad Gateway
Content-Type: text/html
Content-Length: `

var badGatewayContent = `<!DOCTYPE html>
<html>
<head>
    <title>502 Bad Gateway</title>
</head>
<body>
    <h1>Bad Gateway</h1>
    <p>The server encountered a temporary error and could not complete your request.</p>
</body>
</html>`

var badGatewayResponse = badGatewayHeader + fmt.Sprint(len(badGatewayContent)) + "\n\n" + badGatewayContent

type ProxyNotificationConn struct {
	Domain                   string
	AllowMultipleConnections bool
	*HttpReadWriter
}

type Manager struct {
	engine                          *gin.Engine
	hubs                            map[string]*NetworkHub
	incomingClientConn              chan *helper.RemoteConn
	registerProxyNotificationConn   chan *ProxyNotificationConn
	unregisterProxyNotificationConn chan string
	trafficManager                  *traffic.TrafficManager
	authManager                     auth.AuthManager
	tlsConfig                       *tls.Config
}

func SetupManager(tlsConfig *tls.Config) *Manager {
	gin.SetMode(gin.ReleaseMode)

	manager := &Manager{
		hubs:                            make(map[string]*NetworkHub),
		incomingClientConn:              make(chan *helper.RemoteConn),
		registerProxyNotificationConn:   make(chan *ProxyNotificationConn),
		unregisterProxyNotificationConn: make(chan string),
		authManager:                     auth.MakeAuthManager(),
		trafficManager:                  traffic.NewTrafficManager(64 * 1024),
		tlsConfig:                       tlsConfig,
	}

	configureRouter(manager)

	return manager
}

func (manager *Manager) handleTunnel(conn net.Conn, ticket string) {
	domainName, err := helper.GetDomainName(conn)
	if err != nil {
		log.Println("Unable to get domain name", err)
		return
	}

	domain, ok := manager.hubs[domainName]
	if !ok {
		return
	}

	domain.serverRequests <- &request{ticket: ticket, conn: conn}
}

func (manager *Manager) Listen() {
	var err error
	config, _ := config.GetConfig()
	go manager.processRequest()

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
		fmt.Fprint(conn, badGatewayResponse)
		conn.Close()
		return
	}
	if _, ok := conn.(helper.RemoteConn); ok {
		manager.incomingClientConn <- conn.(*helper.RemoteConn)
		return
	}
	manager.incomingClientConn <- &helper.RemoteConn{Conn: conn, Domain: domain}
}

func (manager *Manager) processRequest() {
	defer fmt.Println("Manager closed")
	for {
		select {
		case conn := <-manager.registerProxyNotificationConn:
			if manager.hubs[conn.Domain] == nil {
				manager.hubs[conn.Domain] = NewNetworkHub(
					conn.Domain,
					manager.unregisterProxyNotificationConn,
					manager.trafficManager,
					64*1024,
				)
				go manager.hubs[conn.Domain].listen()
			}
			manager.hubs[conn.Domain].registerProxyNotificationConn <- conn
		case domain := <-manager.unregisterProxyNotificationConn:
			if manager.hubs[domain] != nil {
				manager.hubs[domain].shutdownSignal <- struct{}{}
				delete(manager.hubs, domain)
			}
		case incomingClientConn := <-manager.incomingClientConn:
			if manager.hubs[incomingClientConn.Domain] == nil {
				fmt.Fprint(incomingClientConn, badGatewayResponse)
				incomingClientConn.Close()
				continue
			}
			domain := manager.hubs[incomingClientConn.Domain]
			domain.incomingClientConn <- incomingClientConn
		}
	}
}
