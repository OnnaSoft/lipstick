package manager

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/server/auth"
	"github.com/juliotorresmoreno/lipstick/server/common"
	"github.com/juliotorresmoreno/lipstick/server/proxy"
	"github.com/juliotorresmoreno/lipstick/server/traffic"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type request struct {
	conn *websocket.Conn
	uuid string
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

type websocketConn struct {
	Domain                   string
	AllowMultipleConnections bool
	*websocket.Conn
}

type Manager struct {
	engine           *gin.Engine
	hubs             map[string]*NetworkHub
	remoteConn       chan *common.RemoteConn
	registerDomain   chan *websocketConn
	unregisterDomain chan string
	proxy            *proxy.Proxy
	trafficManager   *traffic.TrafficManager
	authManager      auth.AuthManager
	addr             string
	cert             string
	key              string
}

func SetupManager(proxy *proxy.Proxy, addr string, cert string, key string) *Manager {
	gin.SetMode(gin.ReleaseMode)

	manager := &Manager{
		hubs:             make(map[string]*NetworkHub),
		remoteConn:       make(chan *common.RemoteConn),
		registerDomain:   make(chan *websocketConn),
		unregisterDomain: make(chan string),
		proxy:            proxy,
		authManager:      auth.MakeAuthManager(),
		trafficManager:   traffic.NewTrafficManager(64 * 1024),
		addr:             addr,
		cert:             cert,
		key:              key,
	}

	configureRouter(manager)

	return manager
}

func (manager *Manager) Listen() {
	log.Println("Listening manager on", manager.addr)

	defer manager.proxy.Close()

	var err error
	done := make(chan struct{})
	go manager.manage(done)
	go manager.proxy.Listen(manager.remoteConn)

	if manager.cert != "" && manager.key != "" {
		err = manager.engine.RunTLS(manager.addr, manager.cert, manager.key)
	} else {
		err = manager.engine.Run(manager.addr)
	}

	if err != nil {
		log.Println("Error on listen", err)
	}

	done <- struct{}{}
}

func (manager *Manager) manage(done chan struct{}) {
	defer fmt.Println("Manager closed")
	for {
		select {
		case conn := <-manager.registerDomain:
			if manager.hubs[conn.Domain] == nil {
				manager.hubs[conn.Domain] = NewNetworkHub(
					conn.Domain,
					manager.unregisterDomain,
					manager.trafficManager,
					64*1024,
				)
				go manager.hubs[conn.Domain].listen()
			}
			manager.hubs[conn.Domain].registerWebSocket <- conn
			fmt.Println("Registered", conn.Domain)
		case domain := <-manager.unregisterDomain:
			if manager.hubs[domain] != nil {
				manager.hubs[domain].shutdownSignal <- struct{}{}
				delete(manager.hubs, domain)
			}
			fmt.Println("Unregistered", domain)
		case remoteConn := <-manager.remoteConn:
			if manager.hubs[remoteConn.Domain] == nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
			}
			domain := manager.hubs[remoteConn.Domain]
			domain.incomingClientConn <- remoteConn
		case <-done:
			return
		}
	}
}
