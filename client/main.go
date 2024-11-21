package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/client/handlers"
	"github.com/juliotorresmoreno/lipstick/client/manager"
	"github.com/juliotorresmoreno/lipstick/helper"
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
	fmt.Println("Cliente iniciado. Presiona Ctrl+C para salir.")

	<-interruptChannel
	fmt.Println("Desconectando...")
}

func startClient(proxyTarget string) {
	retryDelay := 3 * time.Second
	headers := http.Header{}
	headers.Set("authorization", configuration.APISecret)

	for {
		req, err := http.NewRequest("GET", serverURL, nil)
		if err != nil {
			fmt.Println("Error al conectar al servidor WebSocket:", err)
			time.Sleep(retryDelay)
			continue
		}
		req.Header = headers
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("Error al conectar al servidor WebSocket:", err)
			time.Sleep(retryDelay)
			continue
		}

		handleTickets(resp.Body, proxyTarget)
		time.Sleep(retryDelay)
	}
}

func handleTickets(connection io.ReadCloser, proxyTarget string) {
	defer func() {
		recover()
	}()
	defer connection.Close()

	reader := bufio.NewReader(connection)
	for {
		ticket, _, err := reader.ReadLine()
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
		fmt.Fprintf(connection, helper.HttpErrorResponse)
		return
	}
	defer connection.Close()

	b := make([]byte, 1024)
	n, err := connection.Read(b)
	if err != nil {
		fmt.Fprintf(connection, helper.HttpErrorResponse)
		return
	}
	buff := b[:n]
	if helper.IsHTTPRequest(string(buff)) {
		handlers.HandleHTTP(helper.NewConnWithBuffer(connection, buff), proxyTarget, protocol)
		return
	}

	handlers.HandleTCP(connection, proxyTarget, protocol)
}
