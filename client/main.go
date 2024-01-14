package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/helper"
)

var done = make(chan struct{})
var connection *websocket.Conn
var conf, _ = config.GetConfig()
var serverUrl = conf.ServerUrl

func main() {
	var err error
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println(serverUrl, conf.ProxyPass)

	for _, proxyPass := range conf.ProxyPass {
		go func(proxyPass string) {
			sleep := 5 * time.Second
			for {
				connection, _, err = websocket.DefaultDialer.Dial(serverUrl, nil)
				if err != nil {
					fmt.Println("Error al conectar al servidor WebSocket:", err)
					time.Sleep(sleep)
					continue
				}
				fmt.Println("Cliente WebSocket iniciado. Presiona Ctrl+C para salir.")

				readRump(proxyPass)
				done = make(chan struct{})
				time.Sleep(sleep)
				connection.Close()
			}
		}(proxyPass)
	}

	<-interrupt
	fmt.Println("Desconectando...")
}

func readRump(proxyPass string) {
	defer close(done)
	defer func() {
		recover()
	}()

	for {
		_, message, err := connection.ReadMessage()
		if err == io.EOF {
			continue
		}
		content := struct {
			UUID string `json:"uuid"`
		}{}
		json.Unmarshal(message, &content)

		if content.UUID != "" {
			go connect(proxyPass, content.UUID)
		}
	}
}

var badGatewayHeader = `HTTP/1.1 502 Bad Gateway
Content-Type: text/html
Content-Length: `

var badGatewayContent = `<!DOCTYPE html>
<html>
<head>
    <title>502 Bad Gateway</title>
</head>
<body>
    <h1>Bad Gateway</h1>
    <p>The server encountered a temporary error and could not complete your request.</p>
</body>
</html>`

var badGatewayResponse = badGatewayHeader + fmt.Sprint(len(badGatewayContent)) + "\n\n" + badGatewayContent

func connect(proxyPass, uuid string) {
	url := serverUrl + "/" + uuid
	fmt.Println("connect", url)

	connection, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}

	defer connection.Close()
	defer func() {
		recover()
	}()
	conn := helper.NewWebSocketIO(connection)
	remote, err := net.Dial("tcp", proxyPass)
	if err != nil {
		log.Println("remote host is not available")
		fmt.Fprint(conn, badGatewayResponse)
		return
	}

	go helper.Copy(conn, remote)
	helper.Copy(remote, conn)
}
