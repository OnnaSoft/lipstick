package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/OnnaSoft/lipstick/client/config"
	"github.com/OnnaSoft/lipstick/client/handlers"
	"github.com/OnnaSoft/lipstick/client/manager"
	"github.com/OnnaSoft/lipstick/helper"
	"github.com/gorilla/websocket"
)

var httpmanager = manager.NewHTTPManager()
var configuration, _ = config.GetConfig()
var serverURL = configuration.ServerURL

func main() {
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)

	fmt.Println(serverURL, configuration.ProxyPass)

	for _, proxyTarget := range configuration.ProxyPass {
		go startClient(proxyTarget)
	}
	fmt.Println("Presiona Ctrl+C para salir.")

	<-interruptChannel
	fmt.Println("Desconectando...")
}

func startClient(proxyTarget string) {
	retryDelay := 3 * time.Second
	headers := http.Header{}
	headers.Set("authorization", configuration.APISecret)

	for {
		conn, _, err := websocket.DefaultDialer.Dial(serverURL, headers)
		if err != nil {
			log.Printf("Error creating WebSocket connection: %v\n", serverURL)
			time.Sleep(retryDelay)
			continue
		}

		fmt.Println("Connected to server at", serverURL)
		handleTickets(conn, proxyTarget)
		time.Sleep(retryDelay)
	}
}

func checkConnection(connection *websocket.Conn) {
	for {
		err := connection.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			break
		}
		time.Sleep(30 * time.Second)
	}
}

func handleTickets(connection *websocket.Conn, proxyTarget string) {
	defer func() {
		recover()
	}()
	defer connection.Close()

	for {
		_, ticket, err := connection.ReadMessage()
		if err != nil {
			break
		}

		if len(ticket) > 0 {
			protocol, targetAddress := helper.ParseTargetEndpoint(proxyTarget)
			go establishConnection(protocol, targetAddress, string(ticket))
		}
	}
}

func establishConnection(protocol, proxyTarget, uuid string) {
	url := serverURL + "/" + uuid

	connection, err := httpmanager.Connect(url, nil)
	if err != nil {
		fmt.Fprintf(connection, helper.BadGatewayResponse)
		return
	}
	defer connection.Close()

	b := make([]byte, 1024)
	n, err := connection.Read(b)
	if err != nil {
		fmt.Fprintf(connection, helper.BadGatewayResponse)
		return
	}
	buff := b[:n]
	conn := helper.NewConnWithBuffer(connection, buff)

	if helper.IsHTTPRequest(string(buff)) {
		handlers.HandleHTTP(conn, proxyTarget, protocol)
		return
	}

	handlers.HandleTCP(conn, proxyTarget, protocol)
}
