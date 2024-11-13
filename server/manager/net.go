package manager

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

// WebSocketWrapper wraps a *websocket.Conn and provides safe concurrent writing.
type WebSocketWrapper struct {
	*websocket.Conn
	writeChan chan messageWrapper
	closeChan chan struct{}
	closed    bool
}

// messageWrapper encapsulates the message type and payload for WebSocket messages.
type messageWrapper struct {
	messageType int
	payload     []byte
}

// NewWebSocketWrapper creates a new WebSocketWrapper.
func NewWebSocketWrapper(conn *websocket.Conn, bufferSize int) *WebSocketWrapper {
	w := &WebSocketWrapper{
		conn,
		make(chan messageWrapper, bufferSize),
		make(chan struct{}),
		false,
	}
	go w.writeLoop()
	return w
}

func (w *WebSocketWrapper) WriteMessage(messageType int, data []byte) error {
	w.writeChan <- messageWrapper{messageType, data}
	return nil
}

func (w *WebSocketWrapper) WriteJSON(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.WriteMessage(websocket.TextMessage, b)
}

func (w *WebSocketWrapper) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	close(w.closeChan)
	return w.Conn.Close()
}

func (w *WebSocketWrapper) writeLoop() {
	for {
		select {
		case msg := <-w.writeChan:
			err := w.Conn.WriteMessage(msg.messageType, msg.payload)
			if err != nil {
				return
			}
		case <-w.closeChan:
			close(w.writeChan)
			return
		}
	}
}
