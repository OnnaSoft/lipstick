package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/helper"
)

var done = make(chan struct{})
var conf, _ = config.GetConfig()
var serverUrl = conf.ServerUrl

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	fmt.Println(serverUrl, conf.ProxyPass)

	for _, proxyPass := range conf.ProxyPass {
		go func(proxyPass string) {
			sleep := 3 * time.Second
			for {
				url := serverUrl
				headers := http.Header{}
				headers.Set("authorization", conf.Keyword)
				connection, _, err := websocket.DefaultDialer.Dial(url, headers)
				if err != nil {
					fmt.Println("Error al conectar al servidor WebSocket:", err)
					time.Sleep(sleep)
					continue
				}
				fmt.Println("Cliente WebSocket iniciado. Presiona Ctrl+C para salir.")

				go ping(connection)
				readRump(connection, proxyPass)
				done = make(chan struct{})
				time.Sleep(sleep)
				connection.Close()
			}
		}(proxyPass)
	}

	<-interrupt
	fmt.Println("Desconectando...")
}

func ping(connection *websocket.Conn) {
	for {
		time.Sleep(30 * time.Second)
		err := connection.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			break
		}
	}
}

func readRump(connection *websocket.Conn, proxyPass string) {
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

	connection, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	conn := helper.NewWebSocketIO(connection)
	remote, err := net.Dial("tcp", proxyPass)
	if err != nil {
		log.Println("remote host is not available")
		fmt.Fprint(conn, badGatewayResponse)
		return
	}
	defer remote.Close()

	go helper.Copy(conn, remote)
	helper.Copy(remote, conn)
}
