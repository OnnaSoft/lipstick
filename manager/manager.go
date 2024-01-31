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

type Manager struct {
	pipe                 chan net.Conn
	engine               *gin.Engine
	remoteConn           *websocket.Conn
	pipes                map[string]net.Conn
	registerRemoteConn   chan *websocket.Conn
	unregisterRemoteConn chan struct{}
	request              chan *request
	proxy                *proxy.Proxy
}

func SetupManager(keyword string, proxy *proxy.Proxy) *Manager {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	manager := &Manager{
		engine:               r,
		pipes:                make(map[string]net.Conn),
		registerRemoteConn:   make(chan *websocket.Conn),
		unregisterRemoteConn: make(chan struct{}),
		request:              make(chan *request),
		pipe:                 make(chan net.Conn),
		proxy:                proxy,
		remoteConn:           nil,
	}

	r.GET("/ws", func(c *gin.Context) {
		remoteConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if keyword != c.GetHeader("authorization") {
			err = errors.New("Unauthorized")
		}

		if err != nil {
			log.Println(err)
			remoteConn.Close()
			return
		}
		manager.registerRemoteConn <- remoteConn
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

func (manager *Manager) Listen(addr string) {
	log.Println("Listening on", addr)

	defer manager.proxy.Close()

	done := make(chan struct{})
	go manager.manage(done)
	go manager.proxy.Listen(manager.pipe)

	manager.engine.Run(addr)
	done <- struct{}{}
}

func (manager *Manager) alive(conn *websocket.Conn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			manager.unregisterRemoteConn <- struct{}{}
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
		case conn := <-manager.registerRemoteConn:
			if manager.remoteConn != nil {
				conn.Close()
				continue
			}
			manager.remoteConn = conn
			if manager.remoteConn != nil {
				go manager.alive(manager.remoteConn)
			}
		case <-manager.unregisterRemoteConn:
			manager.remoteConn.Close()
			manager.remoteConn = nil
		case request := <-manager.request:
			dest := helper.NewWebSocketIO(request.conn)
			pipe := manager.pipes[request.uuid]
			delete(manager.pipes, request.uuid)
			go manager.handle(request.uuid, dest, pipe)
		case pipe := <-manager.pipe:
			ticket := uuid.NewString()
			if manager.remoteConn == nil {
				fmt.Fprint(pipe, badGatewayResponse)
				pipe.Close()
			} else {
				manager.remoteConn.WriteJSON(map[string]string{"uuid": ticket})
				manager.pipes[ticket] = pipe
			}
		case <-done:
			return
		}
	}
}
