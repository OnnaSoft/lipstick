package manager

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/helper"
	"github.com/OnnaSoft/lipstick/logger"
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

	var originToDest int64
	var destToOrigin int64

	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := pipe.Read(buffer)
			if err != nil {
				break
			}
			written, err := destination.Write(buffer[:n])
			if err != nil {
				break
			}
			originToDest += int64(written)
			hub.addDataUsage(int64(written))
		}
		logger.Default.Debug("Finished transferring from origin to destination, total bytes:", originToDest)
	}()

	buffer := make([]byte, 4096)
	for {
		n, err := destination.Read(buffer)
		if err != nil {
			break
		}
		written, err := pipe.Write(buffer[:n])
		if err != nil {
			break
		}
		destToOrigin += int64(written)
		hub.addDataUsage(int64(written))
	}
	logger.Default.Debug("Finished transferring from destination to origin, total bytes:", destToOrigin)

	logger.Default.Debug("Connection data usage:",
		"origin → destination:", originToDest, "bytes,",
		"destination → origin:", destToOrigin, "bytes",
		"Hub:", hub.HubName,
	)
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
	logger.Default.Debug("Data usage updated for hub:", hub.HubName, "Total transferred:", hub.totalDataTransferred)
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
		logger.Default.Error("Connection rejected: multiple connections not allowed for hub:", hub.HubName)
		conn.Close()
		return
	}
	hub.ProxyNotificationConns[conn] = true
	logger.Default.Debug("ProxyNotificationConn registered for hub:", hub.HubName)
	go hub.checkConnection(conn)
}

func (hub *NetworkHub) handleUnregisterProxyNotificationConn(ws *ProxyNotificationConn) {
	go func() {
		ws.Write([]byte("close"))
		ws.Close()
	}()
	if _, exists := hub.ProxyNotificationConns[ws]; exists {
		delete(hub.ProxyNotificationConns, ws)
		logger.Default.Debug("ProxyNotificationConn unregistered for hub:", hub.HubName)
	}
	for ticket, conn := range hub.incomingClientConns {
		delete(hub.incomingClientConns, ticket)
		fmt.Fprintf(conn, helper.BadGatewayResponse)
		conn.Close()
		logger.Default.Debug("Incoming client connection unregistered for hub:", hub.HubName)
	}
}

func (hub *NetworkHub) handleServerRequest(request *request) {
	destination := request.conn
	pipe, exists := hub.incomingClientConns[request.ticket]
	if !exists {
		for key := range hub.incomingClientConns {
			fmt.Println(key)
		}
		logger.Default.Error("Invalid ticket for hub:", hub.HubName, "Ticket:", request.ticket)
		destination.Write([]byte(helper.BadGatewayResponse))
		destination.Close()
		return
	}
	delete(hub.incomingClientConns, request.ticket)
	logger.Default.Debug("Server request handled for ticket:", request.ticket, "Hub:", hub.HubName)
	go hub.syncConnections(pipe, destination)
}

func (hub *NetworkHub) handleIncomingClientConn(remoteConn *helper.RemoteConn) {
	if len(hub.ProxyNotificationConns) == 0 {
		logger.Default.Error("No ProxyNotificationConns available for hub:", hub.HubName)
		_, _ = remoteConn.Write([]byte(helper.BadGatewayResponse))
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
	_, err := ws.Write([]byte(ticket + "\n"))
	if err != nil {
		logger.Default.Error("Error writing ticket to ProxyNotificationConn:", err)
		_, _ = remoteConn.Write([]byte(helper.BadGatewayResponse))
		remoteConn.Close()
		delete(hub.incomingClientConns, ticket)
	}
	logger.Default.Debug("Ticket sent to ProxyNotificationConn for hub:", hub.HubName, "Ticket:", ticket)
}

func (hub *NetworkHub) handleShutdown(shutdownComplete chan struct{}) {
	close(hub.registerProxyNotificationConn)
	close(hub.unregisterProxyNotificationConn)
	close(hub.incomingClientConn)
	close(hub.serverRequests)
	close(hub.shutdownSignal)
	logger.Default.Info("Shutdown completed for hub:", hub.HubName)
	shutdownComplete <- struct{}{}
}

func (h *NetworkHub) checkConnection(connection *ProxyNotificationConn) {
	defer func() {
		h.unregisterProxyNotificationConn <- connection
		logger.Default.Info("Connection closed for ProxyNotificationConn in hub:", h.HubName)
	}()
	for {
		b := make([]byte, 16)
		_, err := connection.conn.Read(b)
		if err != nil {
			logger.Default.Debug("Error reading from ProxyNotificationConn:", err)
			break
		}
	}
}
