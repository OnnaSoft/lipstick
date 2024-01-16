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

type wsChain struct {
	Conn *websocket.Conn
	err  error
}

type registerChain struct {
	Conn *websocket.Conn
	uuid string
}

type Manager struct {
	Pipe            chan net.Conn
	engine          *gin.Engine
	remoteConn      *websocket.Conn
	registerWs      chan *websocket.Conn
	registerChain   chan *registerChain
	unregisterChain chan string
	pipes           map[string]net.Conn
	channels        map[string]*websocket.Conn
	wsChain         chan wsChain
}

func SetupManager(keyword string) *Manager {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	manager := &Manager{
		engine:          r,
		pipes:           make(map[string]net.Conn),
		channels:        make(map[string]*websocket.Conn),
		wsChain:         make(chan wsChain),
		registerWs:      make(chan *websocket.Conn),
		registerChain:   make(chan *registerChain),
		unregisterChain: make(chan string),
		Pipe:            make(chan net.Conn),
	}

	r.GET("/ws", func(c *gin.Context) {
		remoteConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if keyword != c.GetHeader("authorization") {
			err = errors.New("Unauthorized")
		}

		manager.wsChain <- wsChain{remoteConn, err}
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
		manager.registerChain <- &registerChain{uuid: uuid, Conn: conn}
	})

	return manager
}

// Listening port
func (manager *Manager) Listen(addr string) {
	log.Println("Listening on", addr)
	manager.engine.Run(addr)
}

// get ws con to manage
func (manager *Manager) Accept() (*websocket.Conn, error) {
	wsChain := <-manager.wsChain

	return wsChain.Conn, wsChain.err
}

// here you can accept new websocket client
func (manager *Manager) Forward() {
	for {
		remoteConn, err := manager.Accept()
		if err != nil {
			log.Println(err)
			remoteConn.Close()
			continue
		}
		manager.registerWs <- remoteConn
	}
}

func (manager *Manager) Manage() {
	for {
		select {
		case conn := <-manager.registerWs:
			manager.remoteConn = conn
		case channel := <-manager.registerChain:
			manager.channels[channel.uuid] = channel.Conn

			dest := helper.NewWebSocketIO(channel.Conn)
			pipe := manager.pipes[channel.uuid]

			go func() {
				go helper.Copy(pipe, dest)

				defer func() {
					manager.unregisterChain <- channel.uuid
				}()

				helper.Copy(dest, pipe)
			}()
		case channel := <-manager.unregisterChain:
			manager.channels[channel].Close()
			manager.pipes[channel].Close()
			delete(manager.channels, channel)
			delete(manager.pipes, channel)
		case pipe := <-manager.Pipe:
			ticket := uuid.NewString()
			if ws := manager.remoteConn; ws != nil {
				ws.WriteJSON(map[string]string{"uuid": ticket})
				manager.pipes[ticket] = pipe
			}
		}
	}
}
