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
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type registerChain struct {
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
	Pipe                 chan net.Conn
	engine               *gin.Engine
	remoteConn           *websocket.Conn
	registerRemoteConn   chan *websocket.Conn
	unregisterRemoteConn chan struct{}
	registerRequest      chan *registerChain
	pipes                map[string]net.Conn
}

func SetupManager(keyword string) *Manager {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	manager := &Manager{
		engine:               r,
		pipes:                make(map[string]net.Conn),
		registerRemoteConn:   make(chan *websocket.Conn),
		unregisterRemoteConn: make(chan struct{}),
		registerRequest:      make(chan *registerChain),
		Pipe:                 make(chan net.Conn),
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
		manager.registerRequest <- &registerChain{uuid: uuid, conn: conn}
	})

	return manager
}

func (manager *Manager) Listen(addr string) {
	log.Println("Listening on", addr)

	done := make(chan struct{})
	go manager.manage(done)
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
			manager.remoteConn = conn
			if manager.remoteConn == nil {
				manager.unregisterRemoteConn <- struct{}{}
			}
			go manager.alive(conn)
		case <-manager.unregisterRemoteConn:
			manager.remoteConn.Close()
			manager.remoteConn = nil
		case request := <-manager.registerRequest:
			dest := helper.NewWebSocketIO(request.conn)
			pipe := manager.pipes[request.uuid]
			delete(manager.pipes, request.uuid)
			go manager.handle(request.uuid, dest, pipe)
		case pipe := <-manager.Pipe:
			ticket := uuid.NewString()
			remoteConn := manager.remoteConn
			if remoteConn == nil {
				fmt.Fprint(pipe, badGatewayResponse)
				pipe.Close()
				continue
			}
			remoteConn.WriteJSON(map[string]string{"uuid": ticket})
			manager.pipes[ticket] = pipe
		case <-done:
			return
		}
	}
}
