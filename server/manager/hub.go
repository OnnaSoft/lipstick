package manager

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/server/traffic"
)

var rng = helper.NewXORShift(uint32(time.Now().UnixNano()))

type NetworkHub struct {
	HubName                         string
	incomingClientConns             map[string]net.Conn
	ProxyNotificationConns          map[*ProxyNotificationConn]bool
	registerProxyNotificationConn   chan *ProxyNotificationConn
	unregisterProxyNotificationConn chan *ProxyNotificationConn
	incomingClientConn              chan *helper.RemoteConn
	serverRequests                  chan *request
	trafficManager                  *traffic.TrafficManager
	dataUsageAccumulator            int64
	threshold                       int64
	mu                              sync.Mutex
	totalDataTransferred            int64
	tickerManager                   *TickerManager
	shutdownSignal                  chan struct{}
}

func NewNetworkHub(name string, trafficManager *traffic.TrafficManager, threshold int64) *NetworkHub {
	return &NetworkHub{
		HubName:                         name,
		incomingClientConns:             make(map[string]net.Conn),
		ProxyNotificationConns:          make(map[*ProxyNotificationConn]bool),
		registerProxyNotificationConn:   make(chan *ProxyNotificationConn),
		unregisterProxyNotificationConn: make(chan *ProxyNotificationConn),
		incomingClientConn:              make(chan *helper.RemoteConn),
		serverRequests:                  make(chan *request),
		trafficManager:                  trafficManager,
		dataUsageAccumulator:            0,
		threshold:                       threshold,
		tickerManager:                   &TickerManager{},
		shutdownSignal:                  make(chan struct{}),
	}
}

func (hub *NetworkHub) syncConnections(pipe net.Conn, destination net.Conn) {
	defer pipe.Close()
	defer destination.Close()

	go func() {
		written, _ := io.Copy(destination, pipe)
		hub.addDataUsage(written)
	}()

	written, _ := io.Copy(pipe, destination)
	hub.addDataUsage(written)
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
		case conn := <-hub.registerProxyNotificationConn:
			hub.handleRegisterProxyNotificationConn(conn)
		case ws := <-hub.unregisterProxyNotificationConn:
			hub.handleUnregisterProxyNotificationConn(ws)
		case request := <-hub.serverRequests:
			hub.handleServerRequest(request)
		case remoteConn := <-hub.incomingClientConn:
			hub.handleIncomingClientConn(remoteConn)
		case <-hub.shutdownSignal:
			hub.handleShutdown(shutdownComplete)
			return
		}
	}
}

func (hub *NetworkHub) handleRegisterProxyNotificationConn(conn *ProxyNotificationConn) {
	if !conn.AllowMultipleConnections && len(hub.ProxyNotificationConns) > 0 {
		conn.Close()
		return
	}
	if hub.ProxyNotificationConns == nil {
		hub.ProxyNotificationConns = make(map[*ProxyNotificationConn]bool)
	}
	hub.ProxyNotificationConns[conn] = true
	go hub.checkConnection(conn)
}

func (hub *NetworkHub) handleUnregisterProxyNotificationConn(ws *ProxyNotificationConn) {
	ws.Close()
	connections := hub.ProxyNotificationConns
	if connections[ws] {
		delete(connections, ws)
	}
}

func (hub *NetworkHub) handleServerRequest(request *request) {
	destination := request.conn
	pipe, exists := hub.incomingClientConns[request.ticket]
	if !exists {
		fmt.Fprint(destination, helper.BadGatewayResponse)
		destination.Close()
		return
	}
	delete(hub.incomingClientConns, request.ticket)
	go hub.syncConnections(pipe, destination)
}

func (hub *NetworkHub) handleIncomingClientConn(remoteConn *helper.RemoteConn) {
	if len(hub.ProxyNotificationConns) == 0 {
		fmt.Fprint(remoteConn, helper.BadGatewayResponse)
		remoteConn.Close()
		return
	}
	conns := make([]*ProxyNotificationConn, 0, len(hub.ProxyNotificationConns))
	for key := range hub.ProxyNotificationConns {
		conns = append(conns, key)
	}
	var ws *ProxyNotificationConn
	if len(conns) == 1 {
		ws = conns[0]
	} else {
		ws = conns[int(rng.Next()%uint32(len(conns)))]
	}
	ticket := hub.tickerManager.generate()
	hub.incomingClientConns[ticket] = remoteConn
	_, err := ws.Write([]byte(ticket))
	if err != nil {
		fmt.Fprint(remoteConn, helper.BadGatewayResponse)
		remoteConn.Close()
		delete(hub.incomingClientConns, ticket)
	}
}

func (hub *NetworkHub) handleShutdown(shutdownComplete chan struct{}) {
	close(hub.registerProxyNotificationConn)
	close(hub.unregisterProxyNotificationConn)
	close(hub.incomingClientConn)
	close(hub.serverRequests)
	close(hub.shutdownSignal)
	shutdownComplete <- struct{}{}
}

func (h *NetworkHub) checkConnection(connection *ProxyNotificationConn) {
	defer func() {
		h.unregisterProxyNotificationConn <- connection
	}()
	for {
		b := make([]byte, 16)
		_, err := connection.Read(b)
		if err != nil {
			break
		}
		time.Sleep(30 * time.Second)
	}
}
