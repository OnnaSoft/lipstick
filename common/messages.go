package common

import "net/http"

type Event struct {
	UUID   string
	URL    string
	Method string
	Header http.Header
	Body   []byte
}

type Response struct {
	UUID       string
	StatusCode int
	Header     http.Header
	Body       []byte
	Done       bool
}
