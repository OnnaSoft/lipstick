package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/turn/common"
)

var done = make(chan struct{})
var response = make(chan *common.Response)
var connection *websocket.Conn

func main() {
	var err error
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	url := "ws://localhost:8081/ws"

	go func() {
		sleep := 5 * time.Second
		for {
			connection, _, err = websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				fmt.Println("Error al conectar al servidor WebSocket:", err)
				time.Sleep(sleep)
				continue
			}
			fmt.Println("Cliente WebSocket iniciado. Presiona Ctrl+C para salir.")

			go writeDump()
			readRump()
			done = make(chan struct{})
			time.Sleep(sleep)
			connection.Close()
		}
	}()

	<-interrupt
	fmt.Println("Desconectando...")
}

func readRump() {
	defer close(done)

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			log.Println("[00001]:", err)
			continue
		}
		event := &common.Event{}
		err = json.Unmarshal(message, event)
		if err != nil {
			log.Println("[00002]:", err)
			continue
		}

		url := "http://localhost:8082" + event.URL

		req, err := http.NewRequest(event.Method, url, bytes.NewReader(event.Body))
		if err != nil {
			log.Println("[00003]:", err)
			continue
		}
		for h := range event.Header {
			req.Header.Add(h, event.Header.Get(h))
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("[00004]:", err)
			response <- &common.Response{
				UUID:       event.UUID,
				StatusCode: 502,
				Header:     http.Header{},
				Body:       []byte("Bad Gateway"),
			}
			continue
		}

		buff := make([]byte, 4096)
		n, err := res.Body.Read(buff)
		if err != nil {
			log.Println("[00005]:", err)
			response <- &common.Response{
				UUID:       event.UUID,
				StatusCode: 502,
				Header:     res.Header,
				Body:       []byte("Bad Gateway"),
			}
			continue
		}

		response <- &common.Response{
			UUID:       event.UUID,
			StatusCode: res.StatusCode,
			Header:     res.Header,
			Body:       buff[0:n],
		}

		for {
			fmt.Println("here")
			buff = make([]byte, 4096)
			n, err = res.Body.Read(buff)
			if err == io.EOF {
				break
			}
			response <- &common.Response{
				UUID: event.UUID,
				Body: buff[0:n],
			}
		}

		response <- &common.Response{
			UUID: event.UUID,
			Done: true,
		}
	}
}

func writeDump() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := connection.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				fmt.Println("Error al enviar mensaje al servidor:", err)
				return
			}
		case response := <-response:
			buff, err := json.Marshal(response)
			if err != nil {
				continue
			}
			os.Stdout.Write(buff)
			os.Stdout.Write([]byte("\n"))
			err = connection.WriteMessage(websocket.TextMessage, buff)
			if err != nil {
				fmt.Println("Error al enviar mensaje al servidor:", err)
				return
			}
		}
	}
}
