package manager

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/server/common"
	"github.com/juliotorresmoreno/lipstick/server/traffic"
)

type NetworkHub struct {
	HubName              string
	clientConnections    map[string]net.Conn
	webSocketConnections map[*websocket.Conn]bool
	registerWebSocket    chan *websocketConn
	unregisterWebSocket  chan *websocketConn
	incomingClientConn   chan *common.RemoteConn
	serverRequests       chan *request
	trafficManager       *traffic.TrafficManager
	clientUnregister     chan string
	dataUsageAccumulator int64      // Local traffic accumulator
	threshold            int64      // Threshold for reporting traffic
	mu                   sync.Mutex // Mutex to protect dataUsageAccumulator
	totalDataTransferred int64
	shutdownSignal       chan struct{}
}

func NewNetworkHub(name string, unregister chan string, trafficManager *traffic.TrafficManager, threshold int64) *NetworkHub {
	return &NetworkHub{
		HubName: name,
		// Mapa de conexiones de clientes
		clientConnections: make(map[string]net.Conn),
		// Mapa de conexiones WebSocket
		webSocketConnections: make(map[*websocket.Conn]bool),
		registerWebSocket:    make(chan *websocketConn),
		unregisterWebSocket:  make(chan *websocketConn),
		incomingClientConn:   make(chan *common.RemoteConn),
		// Canal de peticiones al servidor
		serverRequests:       make(chan *request),
		trafficManager:       trafficManager,
		clientUnregister:     unregister,
		dataUsageAccumulator: 0,
		threshold:            threshold,
		shutdownSignal:       make(chan struct{}),
	}
}

func (hub *NetworkHub) syncConnections(pipe net.Conn, destination *websocket.Conn) {
	defer pipe.Close()
	defer destination.Close()

	go func() {
		defer pipe.Close()
		defer destination.Close()

		for {
			messageType, message, err := destination.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					return
				}
				return
			}

			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				_, writeErr := pipe.Write(message)
				if writeErr != nil {
					return
				}
				hub.addDataUsage(int64(len(message)))
			} else if messageType == websocket.CloseMessage {
				return
			}
		}
	}()

	for {
		buffer := make([]byte, 1024)
		n, err := pipe.Read(buffer)
		if err != nil {
			closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			destination.WriteMessage(websocket.CloseMessage, closeMessage)
			return
		}
		err = destination.WriteMessage(websocket.BinaryMessage, buffer[:n])
		if err != nil {
			return
		}
		hub.addDataUsage(int64(n))
	}
}

func (hub *NetworkHub) addDataUsage(bytes int64) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.dataUsageAccumulator += bytes
	hub.totalDataTransferred += bytes

	if hub.dataUsageAccumulator >= hub.threshold {
		hub.trafficManager.AddTraffic(hub.HubName, hub.dataUsageAccumulator)
		hub.dataUsageAccumulator = 0 // Reset after sending
	}
}

func (hub *NetworkHub) listen() {
	shutdownComplete := make(chan struct{})
	for {
		select {
		case conn := <-hub.registerWebSocket:
			if !conn.AllowMultipleConnections && len(hub.webSocketConnections) > 0 {
				conn.Close()
				continue
			}
			if hub.webSocketConnections == nil {
				hub.webSocketConnections = make(map[*websocket.Conn]bool)
			}
			hub.webSocketConnections[conn.Conn] = true
			if hub.webSocketConnections != nil {
				go hub.checkConnection(conn)
			}
		case conn := <-hub.unregisterWebSocket:
			conn.Close()
			connections := hub.webSocketConnections
			if connections[conn.Conn] {
				delete(connections, conn.Conn)
			}
		case request := <-hub.serverRequests:
			destination := request.conn
			pipe, exists := hub.clientConnections[request.uuid]
			if !exists {
				destination.WriteMessage(websocket.TextMessage, []byte(badGatewayResponse))
				destination.Close()
				continue
			}
			delete(hub.clientConnections, request.uuid)
			go hub.syncConnections(pipe, destination)
		case remoteConn := <-hub.incomingClientConn:
			ticket := uuid.NewString()
			if hub.webSocketConnections == nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
			}

			hub.clientConnections[ticket] = remoteConn
			conns := make([]*websocket.Conn, 0, len(hub.webSocketConnections))
			for key := range hub.webSocketConnections {
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
		case <-hub.shutdownSignal:
			close(hub.registerWebSocket)
			close(hub.incomingClientConn)
			close(hub.serverRequests)
			close(hub.shutdownSignal)
			shutdownComplete <- struct{}{}
			return
		}
	}
}

func (hub *NetworkHub) checkConnection(conn *websocketConn) {
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			hub.unregisterWebSocket <- conn
			break
		}
	}
}
