package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/helper"
)

var configuration, _ = config.GetConfig()
var baseServerURL = configuration.ServerUrl

func main() {
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)

	// Configuración TLS personalizada para ignorar la verificación del certificado SSL
	tlsConfiguration := &tls.Config{
		InsecureSkipVerify: true, // Ignorar la verificación del certificado
	}

	// Crear un marcador con la configuración TLS personalizada
	websocketDialer := websocket.DefaultDialer
	websocketDialer.TLSClientConfig = tlsConfiguration

	// Establecer un tiempo de espera para la conexión
	websocketDialer.HandshakeTimeout = 5 * time.Second

	fmt.Println(baseServerURL, configuration.ProxyPass)

	for _, proxyTarget := range configuration.ProxyPass {
		go func(proxyTarget string) {
			retryDelay := 3 * time.Second
			for {
				websocketURL := baseServerURL
				headers := http.Header{}
				headers.Set("authorization", configuration.Keyword)

				connection, _, err := websocketDialer.Dial(websocketURL, headers)
				if err != nil {
					fmt.Println("Error al conectar al servidor WebSocket:", err)
					time.Sleep(retryDelay)
					continue
				}
				fmt.Println("Cliente WebSocket iniciado. Presiona Ctrl+C para salir.")

				go sendPingMessages(connection)
				handleWebSocketMessages(connection, proxyTarget)
				time.Sleep(retryDelay)
				connection.Close()
			}
		}(proxyTarget)
	}

	<-interruptChannel
	fmt.Println("Desconectando...")
}

func sendPingMessages(connection *websocket.Conn) {
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

func parseTargetEndpoint(target string) (string, string) {
	var protocol, address string

	// Comprobar el esquema del target
	switch {
	case strings.HasPrefix(target, "tcp://"):
		protocol = "tcp"
		address = strings.TrimPrefix(target, "tcp://")
	case strings.HasPrefix(target, "tls://"):
		protocol = "tls"
		address = strings.TrimPrefix(target, "tls://")
	case strings.HasPrefix(target, "http://"):
		protocol = "http"
		address = strings.TrimPrefix(target, "http://")
	case strings.HasPrefix(target, "https://"):
		protocol = "https"
		address = strings.TrimPrefix(target, "https://")
	default:
		// Caso sin prefijo
		protocol = "tcp"
		address = target
	}

	return protocol, address
}

func handleWebSocketMessages(connection *websocket.Conn, proxyTarget string) {
	defer func() {
		recover()
	}()

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			break
		}
		messageContent := struct {
			UUID string `json:"uuid"`
		}{}
		json.Unmarshal(message, &messageContent)

		protocol, targetAddress := parseTargetEndpoint(proxyTarget)

		if messageContent.UUID != "" {
			go establishConnection(protocol, targetAddress, messageContent.UUID)
		}
	}
}

var httpErrorHeader = `HTTP/1.1 502 Bad Gateway
Content-Type: text/html
Content-Length: `

var httpErrorContent = `<!DOCTYPE html>
<html>
<head>
    <title>502 Bad Gateway</title>
</head>
<body>
    <h1>Bad Gateway</h1>
    <p>The server encountered a temporary error and could not complete your request.</p>
</body>
</html>`

var httpErrorResponse = httpErrorHeader + fmt.Sprint(len(httpErrorContent)) + "\n\n" + httpErrorContent

func establishConnection(protocol, proxyTarget, uuid string) {
	websocketURL := baseServerURL + "/" + uuid

	// Configuración para el cliente WebSocket
	websocketDialer := websocket.DefaultDialer
	websocketDialer.HandshakeTimeout = 5 * time.Second

	// Conectar al WebSocket
	connection, _, err := websocketDialer.Dial(websocketURL, nil)
	if err != nil {
		log.Printf("Error al conectar al servidor WebSocket: %v\n", err)
		return
	}
	defer func() {
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Cierre normal")
		connection.WriteMessage(websocket.CloseMessage, closeMessage)
		connection.Close()
	}()
	connection.SetReadLimit(1024 * 1024 * 32)
	_, message, err := connection.ReadMessage()
	if err != nil && err != io.EOF {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}

	isHTTP := helper.IsHTTPRequest(string(message))

	if isHTTP {
		handleHTTP(connection, proxyTarget, protocol, message)
		return
	}

	if protocol == "http" || protocol == "https" {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}

	handleTCP(connection, proxyTarget, protocol, message)
}

func handleTCP(connection *websocket.Conn, proxyTarget, protocol string, message []byte) {
	var err error
	var serverConnection net.Conn
	if protocol == "tcp" {
		serverConnection, err = net.Dial("tcp", proxyTarget)
	} else {
		serverConnection, err = tls.Dial("tcp", proxyTarget, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}
	defer serverConnection.Close()

	_, err = serverConnection.Write(message)
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}

	go func() {
		for {
			buffer := make([]byte, 1024)
			n, err := serverConnection.Read(buffer)
			if err != nil {
				break
			}
			err = connection.WriteMessage(websocket.BinaryMessage, buffer[:n])
			if err != nil {
				break
			}
		}
	}()

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			break
		}
		serverConnection.Write(message)
	}
}

func handleHTTP(connection *websocket.Conn, proxyTarget, protocol string, message []byte) {
	req, err := helper.ParseHTTPRequest(message, connection)
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}

	host := strings.Split(proxyTarget, ":")[0]
	if protocol != "http" && protocol != "https" {
		protocol = "http"
	}
	serverURL := protocol + "://" + proxyTarget + req.URL.String()

	requestToServer, err := http.NewRequest(req.Method, serverURL, req.Body)
	if err != nil {
		return
	}
	requestToServer.Header = req.Header
	requestToServer.Host = host
	requestToServer.Header.Add("Host", host)

	var serverConnection net.Conn
	if protocol == "http" {
		serverConnection, err = net.Dial("tcp", proxyTarget)
	} else {
		serverConnection, err = tls.Dial("tcp", proxyTarget, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}
	defer serverConnection.Close()

	buf := helper.HTTPRequestToString(requestToServer)

	_, err = serverConnection.Write([]byte(buf))
	if err != nil {
		connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
		return
	}

	go func() {
		for {
			buffer := make([]byte, 1024)
			n, err := serverConnection.Read(buffer)
			if err != nil {
				break
			}
			err = connection.WriteMessage(websocket.BinaryMessage, buffer[:n])
			if err != nil {
				break
			}
		}
	}()

	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			break
		}
		serverConnection.Write(message)
	}
}
