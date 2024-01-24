package helper

import (
	"fmt"
	"net"
	"net/http"
	"reflect"

	"github.com/gorilla/websocket"
)

type WebSocketIO struct {
	conn *websocket.Conn
}

func NewWebSocketIO(conn *websocket.Conn) *WebSocketIO {
	return &WebSocketIO{conn: conn}
}

func (w *WebSocketIO) Close() (err error) {
	return w.conn.Close()
}

func (w *WebSocketIO) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (r *WebSocketIO) Read(p []byte) (n int, err error) {
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(p, message)
	return len(message), nil
}

func GetResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	metaValue := reflect.ValueOf(w).Elem()
	field := metaValue.FieldByName("ResponseWriter")
	if field.IsValid() && field.Interface() != nil {
		value := field.Interface()
		return value.(http.ResponseWriter)
	}
	return w
}

func GetHijack(w http.ResponseWriter) (net.Conn, error) {
	sw := GetResponseWriter(w)
	hijacker, ok := sw.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("%v", "Hijacking not supported")
	}

	c, _, err := hijacker.Hijack()

	return c, err
}
