package manager

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/server/common"
	"github.com/juliotorresmoreno/lipstick/server/traffic"
)

var rng = common.NewXORShift(uint32(time.Now().UnixNano()))

type NetworkHub struct {
	HubName              string
	clientConnections    map[string]net.Conn
	webSocketConnections map[*WebSocketWrapper]bool
	registerWebSocket    chan *websocketConn
	unregisterWebSocket  chan *WebSocketWrapper
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
		HubName:              name,
		clientConnections:    make(map[string]net.Conn),
		webSocketConnections: make(map[*WebSocketWrapper]bool),
		registerWebSocket:    make(chan *websocketConn),
		unregisterWebSocket:  make(chan *WebSocketWrapper),
		incomingClientConn:   make(chan *common.RemoteConn),
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
			if writeErr := destination.WriteMessage(websocket.CloseMessage, closeMessage); writeErr != nil {
				return
			}
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
		hub.dataUsageAccumulator = 0
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
				hub.webSocketConnections = make(map[*WebSocketWrapper]bool)
			}
			ws := NewWebSocketWrapper(conn.Conn, 50)
			hub.webSocketConnections[ws] = true
			if hub.webSocketConnections != nil {
				go hub.checkConnection(ws)
			}
		case ws := <-hub.unregisterWebSocket:
			ws.Close()
			connections := hub.webSocketConnections
			if connections[ws] {
				delete(connections, ws)
			}
		case request := <-hub.serverRequests:
			destination := request.conn
			pipe, exists := hub.clientConnections[request.uuid]
			if !exists {
				func() {
					destination.WriteMessage(websocket.TextMessage, []byte(badGatewayResponse))
					destination.Close()
				}()
				continue
			}
			delete(hub.clientConnections, request.uuid)
			go hub.syncConnections(pipe, destination)
		case remoteConn := <-hub.incomingClientConn:
			if len(hub.webSocketConnections) == 0 {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
			}

			conns := make([]*WebSocketWrapper, 0, len(hub.webSocketConnections))
			for key := range hub.webSocketConnections {
				conns = append(conns, key)
			}

			var ws *WebSocketWrapper
			if len(conns) == 1 {
				ws = conns[0]
			} else {
				ws = conns[int(rng.Next()%uint32(len(conns)))]
			}
			ticket := uuid.NewString()
			hub.clientConnections[ticket] = remoteConn
			err := ws.WriteMessage(websocket.TextMessage, []byte(ticket))
			if err != nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				delete(hub.clientConnections, ticket)
			}
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

func (hub *NetworkHub) checkConnection(ws *WebSocketWrapper) {
	defer ws.Close()
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			hub.unregisterWebSocket <- ws
			break
		}
	}
}
