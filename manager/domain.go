package manager

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/common"
	"github.com/juliotorresmoreno/lipstick/helper"
)

type Domain struct {
	Name                    string `json:"name"`
	remoteConnections       map[string]net.Conn
	websocketConn           map[*websocket.Conn]bool
	registerWebsocketConn   chan *websocketConn
	unregisterWebsocketConn chan *websocketConn
	remoteConn              chan *common.RemoteConn
	request                 chan *request
	unregister              chan string
	consumption             chan int64
	total                   int64
	done                    chan struct{}
}

func NewDomain(name string, unregister chan string) *Domain {
	return &Domain{
		Name: name,
		// Host del cliente
		remoteConnections: make(map[string]net.Conn),
		// Respuesta del cliente
		websocketConn:           make(map[*websocket.Conn]bool),
		registerWebsocketConn:   make(chan *websocketConn),
		unregisterWebsocketConn: make(chan *websocketConn),
		remoteConn:              make(chan *common.RemoteConn),
		// peticion al servidor
		request:     make(chan *request),
		unregister:  unregister,
		consumption: make(chan int64),
		total:       0,
		done:        make(chan struct{}),
	}
}

func (domain *Domain) sync(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()

	go func() {
		defer dst.Close()
		defer src.Close()

		written, _ := helper.Copy(dst, src)
		domain.consumption <- written
	}()

	written, _ := helper.Copy(src, dst)
	domain.consumption <- written
}

func (domain *Domain) listen() {
	done := make(chan struct{})
	for {
		select {
		case conn := <-domain.registerWebsocketConn:
			if !conn.AllowMultiple && len(domain.websocketConn) > 0 {
				conn.Close()
				continue
			}
			if domain.websocketConn == nil {
				domain.websocketConn = make(map[*websocket.Conn]bool)
			}
			domain.websocketConn[conn.Conn] = true
			if domain.websocketConn != nil {
				go domain.alive(conn)
			}
		case conn := <-domain.unregisterWebsocketConn:
			conn.Close()
			connections := domain.websocketConn
			if connections[conn.Conn] {
				delete(connections, conn.Conn)
			}
		case request := <-domain.request:
			dest := helper.NewWebSocketIO(request.conn)
			pipe := domain.remoteConnections[request.uuid]
			delete(domain.remoteConnections, request.uuid)
			go domain.sync(pipe, dest)
		case remoteConn := <-domain.remoteConn:
			ticket := uuid.NewString()
			if domain.websocketConn == nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
			}

			domain.remoteConnections[ticket] = remoteConn
			conns := make([]*websocket.Conn, 0, len(domain.websocketConn))
			for key := range domain.websocketConn {
				conns = append(conns, key)
			}

			if len(conns) == 0 {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
			}

			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
			conn := conns[rnd.Intn(len(conns))]
			conn.WriteJSON(map[string]string{"uuid": ticket})
		case consumption := <-domain.consumption:
			fmt.Println("Consumption", consumption)
			domain.total += consumption
		case <-domain.done:
			close(domain.registerWebsocketConn)
			close(domain.consumption)
			close(domain.remoteConn)
			close(domain.request)
			close(domain.done)
			done <- struct{}{}
			return
		}
	}
}

func (domain *Domain) alive(conn *websocketConn) {
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			domain.unregisterWebsocketConn <- conn
			break
		}
	}
}
