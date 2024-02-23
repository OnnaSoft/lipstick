package manager

import (
	"errors"
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

type Manager struct {
	engine                  *gin.Engine
	remoteConn              chan *common.RemoteConn
	remoteConnections       map[string]net.Conn
	websocketConn           map[string]*websocket.Conn
	registerWebsocketConn   chan *websocketConn
	unregisterWebsocketConn chan string
	request                 chan *request
	proxy                   *proxy.Proxy
}

func SetupManager(keyword string, proxy *proxy.Proxy) *Manager {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	manager := &Manager{
		engine:                  r,
		remoteConnections:       make(map[string]net.Conn),
		websocketConn:           make(map[string]*websocket.Conn),
		registerWebsocketConn:   make(chan *websocketConn),
		unregisterWebsocketConn: make(chan string),
		request:                 make(chan *request),
		remoteConn:              make(chan *common.RemoteConn),
		proxy:                   proxy,
	}

	r.GET("/ws", func(c *gin.Context) {
		wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if keyword != c.GetHeader("authorization") {
			err = errors.New("Unauthorized")
		}

		if err != nil {
			log.Println(err)
			wsConn.Close()
			return
		}

		domain, err := helper.GetDomainName(wsConn.NetConn())
		if err != nil {
			log.Println(err)
			wsConn.Close()
			return
		}

		manager.registerWebsocketConn <- &websocketConn{Domain: domain, Conn: wsConn}
	})

	r.GET("/ws/:uuid", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		uuid, ok := c.Params.Get("uuid")
		if !ok {
			return
		}
		manager.request <- &request{uuid: uuid, conn: conn}
	})

	return manager
}

func (manager *Manager) Listen(addr string, cert string, key string) {
	log.Println("Listening on", addr)

	defer manager.proxy.Close()

	done := make(chan struct{})
	go manager.manage(done)
	go manager.proxy.Listen(manager.remoteConn)

	if cert != "" && key != "" {
		manager.engine.RunTLS(addr, cert, key)
	} else {
		manager.engine.Run(addr)
	}
	done <- struct{}{}
}

func (manager *Manager) alive(conn *websocketConn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			manager.unregisterWebsocketConn <- conn.Domain
			break
		}
	}
}

func (manager *Manager) handle(uuid string, dest *helper.WebSocketIO, pipe net.Conn) {
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
		case domain := <-manager.unregisterWebsocketConn:
			if manager.websocketConn[domain] == nil {
				continue
			}
			manager.websocketConn[domain].Close()
			manager.websocketConn[domain] = nil
		case request := <-manager.request:
			dest := helper.NewWebSocketIO(request.conn)
			pipe := manager.remoteConnections[request.uuid]
			delete(manager.remoteConnections, request.uuid)
			go manager.handle(request.uuid, dest, pipe)
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
