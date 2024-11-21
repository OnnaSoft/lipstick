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
	clientUnregister                chan string
	dataUsageAccumulator            int64
	threshold                       int64
	mu                              sync.Mutex
	totalDataTransferred            int64
	tickerManager                   *TickerManager
	shutdownSignal                  chan struct{}
}

func NewNetworkHub(name string, unregister chan string, trafficManager *traffic.TrafficManager, threshold int64) *NetworkHub {
	return &NetworkHub{
		HubName:                         name,
		incomingClientConns:             make(map[string]net.Conn),
		ProxyNotificationConns:          make(map[*ProxyNotificationConn]bool),
		registerProxyNotificationConn:   make(chan *ProxyNotificationConn),
		unregisterProxyNotificationConn: make(chan *ProxyNotificationConn),
		incomingClientConn:              make(chan *helper.RemoteConn),
		serverRequests:                  make(chan *request),
		trafficManager:                  trafficManager,
		clientUnregister:                unregister,
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
			if !conn.AllowMultipleConnections && len(hub.ProxyNotificationConns) > 0 {
				conn.Close()
				continue
			}
			if hub.ProxyNotificationConns == nil {
				hub.ProxyNotificationConns = make(map[*ProxyNotificationConn]bool)
			}
			hub.ProxyNotificationConns[conn] = true
		case ws := <-hub.unregisterProxyNotificationConn:
			ws.Close()
			connections := hub.ProxyNotificationConns
			if connections[ws] {
				delete(connections, ws)
			}
		case request := <-hub.serverRequests:
			destination := request.conn
			pipe, exists := hub.incomingClientConns[request.ticket]
			if !exists {
				func() {
					fmt.Fprint(destination, badGatewayResponse)
					destination.Close()
				}()
				continue
			}
			delete(hub.incomingClientConns, request.ticket)
			go hub.syncConnections(pipe, destination)
		case remoteConn := <-hub.incomingClientConn:
			if len(hub.ProxyNotificationConns) == 0 {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				continue
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
			_, err := ws.WriteTicket(ticket)
			if err != nil {
				fmt.Fprint(remoteConn, badGatewayResponse)
				remoteConn.Close()
				delete(hub.incomingClientConns, ticket)
			}
		case <-hub.shutdownSignal:
			close(hub.registerProxyNotificationConn)
			close(hub.unregisterProxyNotificationConn)
			close(hub.incomingClientConn)
			close(hub.serverRequests)
			close(hub.shutdownSignal)
			shutdownComplete <- struct{}{}
			return
		}
	}
}
