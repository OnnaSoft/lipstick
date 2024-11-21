package helper

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
)

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

func ParseHTTPRequest(conn net.Conn) (*http.Request, error) {
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTTP request: %w", err)
	}

	return request, nil
}
