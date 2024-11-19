package helper

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
)

type messageWrapper struct {
	messageType int
	payload     []byte
}

type WebSocketIO struct {
	*websocket.Conn
	buff    []byte
	closed  bool
	writeCh chan *messageWrapper
}

// NewWebSocketIO crea una nueva instancia de WebSocketIO.
func NewWebSocketIO(conn *websocket.Conn) *WebSocketIO {
	ws := &WebSocketIO{Conn: conn, writeCh: make(chan *messageWrapper)}
	go ws.processWrite()
	return ws
}

func (w *WebSocketIO) processWrite() {
	for data := range w.writeCh {
		w.Conn.WriteMessage(data.messageType, data.payload)
	}
}

func (w *WebSocketIO) SetBuffer(buff []byte) {
	w.buff = buff
}

func (w *WebSocketIO) WriteMessage(messageType int, data []byte) error {
	w.writeCh <- &messageWrapper{messageType, data}
	return nil
}

func WriteJSON(w *WebSocketIO, v interface{}) error {
	b, err := sonic.Marshal(v)
	if err != nil {
		return err
	}
	return w.WriteMessage(websocket.TextMessage, b)
}

func (w *WebSocketIO) WriteControl(messageType int, data []byte) error {
	return w.WriteMessage(messageType, data)
}

// Close cierra la conexión WebSocket.
func (w *WebSocketIO) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	close(w.writeCh)
	return w.Conn.Close()
}

// Write escribe datos en la conexión WebSocket como un mensaje de texto.
func (w *WebSocketIO) Write(p []byte) (int, error) {
	w.writeCh <- &messageWrapper{websocket.BinaryMessage, p}
	return len(p), nil
}

// Read lee datos de la conexión WebSocket.
func (w *WebSocketIO) Read(p []byte) (int, error) {
	if len(w.buff) > 0 {
		n := copy(p, w.buff)
		w.buff = w.buff[n:]
		return n, nil
	}

	messageType, message, err := w.Conn.ReadMessage()
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

func (w *WebSocketIO) ReadMessage() (int, []byte, error) {
	if len(w.buff) > 0 {
		message := w.buff
		w.buff = nil
		return websocket.TextMessage, message, nil
	}

	messageType, message, err := w.Conn.ReadMessage()
	if err != nil {
		return 0, nil, err
	}
	return messageType, message, nil
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

func ReadHTTPRequestFromWebSocket(conn *WebSocketIO) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	for {
		pos := bytes.Index(buffer.Bytes(), []byte{13, 10, 13, 10})
		if pos != -1 {
			break
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("error reading WebSocket message: %w", err)
		}

		buffer.Write(message)
	}

	return buffer.Bytes(), nil
}

func ParseHTTPRequest(conn net.Conn) (*http.Request, error) {
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTTP request: %w", err)
	}

	return request, nil
}
