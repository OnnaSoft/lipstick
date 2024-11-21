package manager

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
)

var ErrNotFlusher = errors.New("http.ResponseWriter is not a http.Flusher")

type HttpReadWriter struct {
	conn net.Conn
	*bufio.ReadWriter
}

func NewHttpReadWriter(w http.ResponseWriter) (*HttpReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("http.ResponseWriter is not a http.Hijacker")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	writter := &HttpReadWriter{
		conn:       conn,
		ReadWriter: rw,
	}
	return writter, nil
}

func (w *HttpReadWriter) WriteTicket(s string) (int, error) {
	message := s + "\n"
	n, err := w.WriteString(fmt.Sprintf("%x\r\n%s\r\n", len(message), message))
	if err != nil {
		return n, err
	}
	return n, w.Flush()
}

func (w *HttpReadWriter) Close() error {
	return w.conn.Close()
}

func (w *HttpReadWriter) Write(b []byte) (int, error) {
	return w.conn.Write(b)
}

func (w *HttpReadWriter) Read(b []byte) (int, error) {
	return w.conn.Read(b)
}
