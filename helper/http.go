package helper

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketIO struct {
	conn *websocket.Conn
	buff []byte
}

// NewWebSocketIO crea una nueva instancia de WebSocketIO.
func NewWebSocketIO(conn *websocket.Conn) *WebSocketIO {
	return &WebSocketIO{conn: conn}
}

func (w *WebSocketIO) SetBuffer(buff []byte) {
	w.buff = buff
}

// LocalAddr devuelve la dirección local de la conexión.
func (w *WebSocketIO) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

// RemoteAddr devuelve la dirección remota de la conexión.
func (w *WebSocketIO) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

// SetDeadline establece el tiempo de espera para lectura y escritura.
func (w *WebSocketIO) SetDeadline(t time.Time) error {
	if err := w.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.conn.SetWriteDeadline(t)
}

// SetReadDeadline establece el tiempo de espera para lectura.
func (w *WebSocketIO) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

// SetWriteDeadline establece el tiempo de espera para escritura.
func (w *WebSocketIO) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

// Close cierra la conexión WebSocket.
func (w *WebSocketIO) Close() error {
	return w.conn.Close()
}

// Write escribe datos en la conexión WebSocket como un mensaje de texto.
func (w *WebSocketIO) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read lee datos de la conexión WebSocket.
func (w *WebSocketIO) Read(p []byte) (int, error) {
	if len(w.buff) > 0 {
		n := copy(p, w.buff)
		w.buff = w.buff[n:]
		return n, nil
	}

	messageType, message, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if messageType != websocket.TextMessage {
		return 0, websocket.ErrCloseSent // Puedes manejar otros tipos de mensajes si es necesario
	}

	// Verificar el tamaño del búfer antes de copiar
	if len(message) > len(p) {
		return 0, io.ErrShortBuffer
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

func GetDomainName(conn net.Conn) (string, error) {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return "localhost", nil
	}

	err := tlsConn.Handshake()
	if err != nil {
		return "", err
	}

	state := tlsConn.ConnectionState()
	domain := state.ServerName

	return domain, nil
}

func IsHTTPRequest(data string) bool {
	// Dividir el string en líneas
	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		return false
	}

	// Tomar la primera línea (request line)
	requestLine := strings.TrimSpace(lines[0])

	// Dividir la línea en partes
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return false
	}

	// Validar el método HTTP
	method := parts[0]
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"OPTIONS": true,
		"HEAD":    true,
	}
	if !validMethods[method] {
		return false
	}

	// Validar que la versión comience con "HTTP/"
	version := parts[2]
	return strings.HasPrefix(version, "HTTP/")
}

func ParseHTTPRequest(data []byte, conn *websocket.Conn) (*http.Request, error) {
	ws := NewWebSocketIO(conn)
	ws.SetBuffer(data)
	reader := io.Reader(ws)

	request, err := http.ReadRequest(bufio.NewReader(reader))
	if err != nil {
		return nil, err
	}

	return request, nil
}
