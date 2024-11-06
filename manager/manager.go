package manager

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/common"
	"github.com/juliotorresmoreno/lipstick/helper"
	"github.com/juliotorresmoreno/lipstick/proxy"
	"github.com/juliotorresmoreno/lipstick/server/auth"
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
	Domain string
	*websocket.Conn
}

type UserDomain struct {
	UserID uint
	Domain string
}

type Manager struct {
	engine                  *gin.Engine
	remoteConn              chan *common.RemoteConn
	remoteConnections       map[string]net.Conn
	websocketConn           map[string]*websocket.Conn
	userConnections         map[uint]uint
	registerWebsocketConn   chan *websocketConn
	unregisterWebsocketConn chan *UserDomain
	request                 chan *request
	proxy                   *proxy.Proxy
	authManager             auth.AuthManager
	addr                    string
	cert                    string
	key                     string
}

func SetupManager(proxy *proxy.Proxy, addr string, cert string, key string) *Manager {
	gin.SetMode(gin.ReleaseMode)

	manager := &Manager{
		remoteConnections:       make(map[string]net.Conn),
		websocketConn:           make(map[string]*websocket.Conn),
		registerWebsocketConn:   make(chan *websocketConn),
		unregisterWebsocketConn: make(chan *UserDomain),
		request:                 make(chan *request),
		remoteConn:              make(chan *common.RemoteConn),
		proxy:                   proxy,
		authManager:             auth.MakeAuthManager(),
		userConnections:         make(map[uint]uint),
		addr:                    addr,
		cert:                    cert,
		key:                     key,
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

func (manager *Manager) alive(conn *websocketConn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			manager.unregisterWebsocketConn <- &UserDomain{
				Domain: conn.Domain,
			}
			break
		}
	}
}

func (manager *Manager) handle(dest *helper.WebSocketIO, pipe net.Conn) {
	defer func() {
		dest.Close()
		pipe.Close()
	}()

	go helper.Copy(pipe, dest)
	helper.Copy(dest, pipe)
}

func (manager *Manager) manage(done chan struct{}) {
	for {
		select {
		case conn := <-manager.registerWebsocketConn:
			if manager.websocketConn[conn.Domain] != nil {
				conn.Close()
				continue
			}
			manager.websocketConn[conn.Domain] = conn.Conn
			if manager.websocketConn != nil {
				go manager.alive(conn)
			}

			fmt.Println("Connected", conn.Domain)
		case userDomain := <-manager.unregisterWebsocketConn:
			if manager.websocketConn[userDomain.Domain] == nil {
				continue
			}
			manager.websocketConn[userDomain.Domain].Close()
			manager.websocketConn[userDomain.Domain] = nil

			manager.userConnections[userDomain.UserID]--

			fmt.Println("Disconnected", userDomain.Domain)
		case request := <-manager.request:
			dest := helper.NewWebSocketIO(request.conn)
			pipe := manager.remoteConnections[request.uuid]
			delete(manager.remoteConnections, request.uuid)
			go manager.handle(dest, pipe)
		case remoteConn := <-manager.remoteConn:
			ticket := uuid.NewString()
			if manager.websocketConn[remoteConn.Domain] == nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
			} else {
				manager.websocketConn[remoteConn.Domain].WriteJSON(map[string]string{"uuid": ticket})
				manager.remoteConnections[ticket] = remoteConn
			}
		case <-done:
			return
		}
	}
}
