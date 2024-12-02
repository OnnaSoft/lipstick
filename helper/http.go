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

func GetHijack(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	sw := GetResponseWriter(w)
	hijacker, ok := sw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%v", "Hijacking not supported")
	}

	return hijacker.Hijack()
}

func GetDomainName(conn net.Conn) (string, error) {
	var rawConn net.Conn = conn
	connWithBuffer, ok := conn.(*ConnWithBuffer)
	if ok {
		rawConn = connWithBuffer.Conn
	}

	tlsConn, ok := rawConn.(*tls.Conn)
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
	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		return false
	}

	requestLine := strings.TrimSpace(lines[0])
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return false
	}

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

func ReadUntilHeadersEnd(conn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(conn)
	result := []byte{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return []byte{}, fmt.Errorf("error reading headers: %w", err)
		}
		result = append(result, []byte(line)...)
		if line == "\r\n" {
			break
		}
	}
	return result, nil
}
