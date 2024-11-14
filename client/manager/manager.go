package manager

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/helper"
)

type WebSocketManager struct {
	mu      sync.Mutex
	conns   map[string]*websocket.Conn
	dialer  *websocket.Dialer
	timeout time.Duration
}

func NewWebSocketManager(timeout time.Duration) *WebSocketManager {
	return &WebSocketManager{
		conns:   make(map[string]*websocket.Conn),
		dialer:  &websocket.Dialer{HandshakeTimeout: timeout},
		timeout: timeout,
	}
}

// Connect creates a new WebSocket connection for a given URL and stores it.
func (wm *WebSocketManager) Connect(url string, headers http.Header) (*helper.WebSocketIO, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Create a new connection
	conn, _, err := wm.dialer.Dial(url, headers)
	if err != nil {
		log.Printf("Error creating WebSocket connection: %v\n", err)
		return nil, err
	}

	// Store the connection by its URL
	wm.conns[url] = conn
	return helper.NewWebSocketIO(conn), nil
}

// CloseConnection closes a specific WebSocket connection.
func (wm *WebSocketManager) CloseConnection(url string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if conn, exists := wm.conns[url]; exists {
		conn.Close()
		delete(wm.conns, url)
	}
}

// CloseAllConnections closes all WebSocket connections.
func (wm *WebSocketManager) CloseAllConnections() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for url, conn := range wm.conns {
		conn.Close()
		delete(wm.conns, url)
	}
}
