package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/helper"
)

var conf, _ = config.GetConfig()
var serverUrl = conf.ServerUrl

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Crear configuración TLS personalizada para ignorar la verificación del certificado SSL
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Ignorar la verificación del certificado
	}

	// Crear un marcador con la configuración TLS personalizada
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = tlsConfig

	// Establecer un tiempo de espera para la conexión
	dialer.HandshakeTimeout = 5 * time.Second

	fmt.Println(serverUrl, conf.ProxyPass)

	for _, proxyPass := range conf.ProxyPass {
		go func(proxyPass string) {
			sleep := 3 * time.Second
			for {
				url := serverUrl
				headers := http.Header{}
				headers.Set("authorization", conf.Keyword)

				connection, _, err := dialer.Dial(url, headers)
				if err != nil {
					fmt.Println("Error al conectar al servidor WebSocket:", err)
					time.Sleep(sleep)
					continue
				}
				fmt.Println("Cliente WebSocket iniciado. Presiona Ctrl+C para salir.")

				go ping(connection)
				serve(connection, proxyPass)
				time.Sleep(sleep)
				connection.Close()
			}
		}(proxyPass)
	}

	<-interrupt
	fmt.Println("Desconectando...")
}

func ping(connection *websocket.Conn) {
	defer func() {
		recover()
	}()
	for {
		time.Sleep(30 * time.Second)
		err := connection.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			break
		}
	}
}

func parseEndpoint(endpoint string) (string, string) {
	var scheme, address string

	// Comprobar el esquema del endpoint
	switch {
	case strings.HasPrefix(endpoint, "tcp://"):
		scheme = "tcp"
		address = strings.TrimPrefix(endpoint, "tcp://")
	case strings.HasPrefix(endpoint, "tls://"):
		scheme = "tls"
		address = strings.TrimPrefix(endpoint, "tls://")
	case strings.HasPrefix(endpoint, "http://"):
		scheme = "http"
		address = strings.TrimPrefix(endpoint, "http://")
	case strings.HasPrefix(endpoint, "https://"):
		scheme = "https"
		address = strings.TrimPrefix(endpoint, "https://")
	default:
		// Caso sin prefijo
		scheme = "tcp"
		address = endpoint
	}

	return scheme, address
}

func serve(connection *websocket.Conn, proxyPass string) {
	defer func() {
		recover()
	}()

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			break
		}
		content := struct {
			UUID string `json:"uuid"`
		}{}
		json.Unmarshal(message, &content)

		protocol, proxyPass := parseEndpoint(proxyPass)

		if content.UUID != "" {
			go connect(protocol, proxyPass, content.UUID)
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

func connect(protocol, proxyPass, uuid string) {
	url := serverUrl + "/" + uuid

	// Configuración para el cliente WebSocket
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second

	// Conectar al WebSocket
	connection, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Printf("Error connecting to WebSocket server: %v\n", err)
		return
	}
	defer connection.Close()

	conn := helper.NewWebSocketIO(connection)
	fmt.Println("Connected to server:", proxyPass)

	err = helper.TcpBypass(conn, proxyPass, protocol)
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(badGatewayResponse))
		log.Printf("Error preparing TCP connection: %v\n", err)
		return
	}

	fmt.Println("Connection closed:", proxyPass)

}
