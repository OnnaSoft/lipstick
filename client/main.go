package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	fmt.Println("Connecting to", serverURL)
	retryDelay := 3 * time.Second
	headers := http.Header{}
	headers.Set("authorization", configuration.APISecret)

	env := os.Getenv("ENV")
	if env == "development" {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		fmt.Println("Warning: Allowing insecure connections in development mode")
	}

	for {
		conn, err := httpmanager.Connect(serverURL, headers)
		if err != nil {
			log.Printf("Error creating WebSocket connection: %v\n", serverURL)
			time.Sleep(retryDelay)
			continue
		}
		helper.ReadUntilHeadersEnd(conn)

		fmt.Println("Connected to server at", serverURL)
		go checkConnection(conn)
		handleTickets(conn, proxyTarget)
		fmt.Println("Disconnected from server at", serverURL)
		time.Sleep(retryDelay)
	}
}

func checkConnection(connection io.ReadWriter) {
	writeMessage := []byte("ping")
	for {
		_, err := connection.Write(writeMessage)
		if err != nil {
			return
		}
		time.Sleep(30 * time.Second)
	}
}
func readMessage(reader *bufio.Reader) (string, string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("error reading until newline: %w", err)
	}
	line = line[:len(line)-1]
	if line == "close" {
		fmt.Println("Connection closed by server")
		return "", "", fmt.Errorf("connection closed by server")
	}

	data := strings.Split(line, ":")

	return data[0], data[1], nil
}

func handleTickets(connection net.Conn, proxyTarget string) {
	defer func() {
		recover()
	}()
	defer connection.Close()

	reader := bufio.NewReader(connection)
	for {
		addr, ticket, err := readMessage(reader)
		if err != nil {
			return
		}

		if len(ticket) > 0 {
			protocol, targetAddress := helper.ParseTargetEndpoint(proxyTarget)
			go establishConnection(protocol, addr, targetAddress, string(ticket))
		}
	}
}

func establishConnection(protocol, addr, proxyTarget, uuid string) {
	defer func() {
		recover()
	}()
	url := serverURL + "/" + uuid

	connection, err := httpmanager.ConnectByAddress(addr, url, nil)
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
