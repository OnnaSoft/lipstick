package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/lipstick/client/config"
	"github.com/juliotorresmoreno/lipstick/client/manager"
	"github.com/juliotorresmoreno/lipstick/helper"
)

var configuration, _ = config.GetConfig()
var websocketURL = configuration.ServerURL
var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,             // Máximo de conexiones ociosas
		IdleConnTimeout:     90 * time.Second, // Tiempo máximo de inactividad
		DisableKeepAlives:   false,            // Keep-Alive habilitado
		MaxIdleConnsPerHost: 100,              // Conexiones ociosas permitidas por host
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second, // Tiempo de espera para establecer la conexión
			KeepAlive: 30 * time.Second, // Tiempo de espera para mantener la conexión
		}).Dial,
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	},
}
var wsmanager = manager.NewWebSocketManager(5 * time.Second)

func main() {
	defer wsmanager.CloseAllConnections()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)

	fmt.Println(websocketURL, configuration.ProxyPass)

	for _, proxyTarget := range configuration.ProxyPass {
		go startWebSocketClient(proxyTarget)
	}

	<-interruptChannel
	fmt.Println("Desconectando...")
}

func startWebSocketClient(proxyTarget string) {
	retryDelay := 3 * time.Second
	headers := http.Header{}
	headers.Set("authorization", configuration.APISecret)

	for {
		connection, err := wsmanager.Connect(websocketURL, headers)
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
}

func sendPingMessages(connection *helper.WebSocketIO) {
	defer func() {
		recover()
	}()
	for {
		time.Sleep(30 * time.Second)
		err := connection.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			fmt.Println("Error al enviar mensaje de ping:", err)
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

func handleWebSocketMessages(connection *helper.WebSocketIO, proxyTarget string) {
	defer func() {
		recover()
	}()

	for {
		_, ticket, err := connection.ReadMessage()
		if err != nil {
			fmt.Println("Error al leer mensaje del servidor WebSocket:", err)
			break
		}

		if len(ticket) > 0 {
			protocol, targetAddress := parseTargetEndpoint(proxyTarget)
			go establishConnection(protocol, targetAddress, string(ticket))
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
	url := websocketURL + "/" + uuid

	connection, err := wsmanager.Connect(url, nil)
	if err != nil {
		log.Printf("Error al conectar al servidor WebSocket: %v\n", err)
		return
	}
	defer func() {
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Cierre normal")
		connection.WriteMessage(websocket.CloseMessage, closeMessage)
		connection.Close()
	}()
	_, message, err := connection.ReadMessage()
	if err != nil && err != io.EOF {
		log.Printf("Error al leer mensaje del servidor WebSocket: %v\n", err)
		sendErrorResponse(connection)
		return
	}

	//isHTTP := helper.IsHTTPRequest(string(message))

	/*
		if isHTTP {
			handleHTTP(connection, proxyTarget, protocol, message)
			return
		}

		if protocol == "http" || protocol == "https" {
			fmt.Println("Protocolo HTTP no compatible con el mensaje WebSocket")
			sendErrorResponse(connection)
			return
		}*/

	handleTCP(connection, proxyTarget, protocol, message)
}

func handleTCP(connection *helper.WebSocketIO, proxyTarget, protocol string, message []byte) {
	var err error
	var serverConnection net.Conn
	if protocol == "tcp" || protocol == "http" {
		serverConnection, err = net.Dial("tcp", proxyTarget)
	} else {
		serverConnection, err = tls.Dial("tcp", proxyTarget, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if err != nil {
		fmt.Println("Error al conectar al servidor TCP:", err)
		sendErrorResponse(connection)
		return
	}
	defer serverConnection.Close()

	_, err = serverConnection.Write(message)
	if err != nil {
		fmt.Println("Error al enviar mensaje al servidor TCP:", err)
		sendErrorResponse(connection)
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

	io.Copy(serverConnection, connection)
}

// sendHTTPResponse sends an HTTP response back over the WebSocket connection.
func sendHTTPResponse(connection *helper.WebSocketIO, resp *http.Response) error {
	headers := ""
	for key, values := range resp.Header {
		for _, value := range values {
			headers += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}
	responseHeader := fmt.Sprintf("HTTP/1.1 %d %s\r\n%s\r\n", resp.StatusCode, resp.Status, headers)
	if err := connection.WriteMessage(websocket.TextMessage, []byte(responseHeader)); err != nil {
		fmt.Println("Error sending HTTP response header:", err)
		return err
	}

	_, err := io.Copy(connection, resp.Body)
	return err
}

func sendErrorResponse(connection *helper.WebSocketIO) {
	connection.WriteMessage(websocket.TextMessage, []byte(httpErrorResponse))
}

// handleHTTPRequest forwards HTTP requests using Go's http.Client.
func handleHTTPRequest(connection *helper.WebSocketIO, serverURL string, req *http.Request, host string) {

	// Prepare the request for the target server
	req.URL, _ = url.Parse(serverURL)
	req.RequestURI = "" // Clear RequestURI since http.Client uses URL instead
	req.Host = host

	// Forward the request to the target server
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error forwarding request to server:", err)
		sendErrorResponse(connection)
		return
	}
	defer resp.Body.Close()

	// Send the response back to the WebSocket client
	err = sendHTTPResponse(connection, resp)
	if err != nil {
		log.Println("Error sending HTTP response:", err)
	}
}

func handleHTTP(connection *helper.WebSocketIO, proxyTarget, protocol string, message []byte) {
	req, err := helper.ParseHTTPRequest(message, connection)
	if err != nil {
		fmt.Println("Error parsing HTTP request:", err)
		sendErrorResponse(connection)
		return
	}

	host := strings.Split(proxyTarget, ":")[0]
	if protocol != "http" && protocol != "https" {
		protocol = "http"
	}
	serverURL := protocol + "://" + proxyTarget + req.URL.String()

	requestToServer, err := http.NewRequest(req.Method, serverURL, req.Body)
	if err != nil {
		fmt.Println("Error creating request to server:", err)
		return
	}
	requestToServer.Header = req.Header
	requestToServer.Host = host
	requestToServer.Header.Add("Host", host)

	// Check if the request is an Upgrade (WebSocket) request
	hconn := strings.ToLower(req.Header.Get("Connection"))
	hupgrade := strings.ToLower(req.Header.Get("Upgrade"))
	validUpgrade := []string{"websocket"}

	isWebSocket := strings.Contains(hconn, "upgrade") && slices.Contains(validUpgrade, hupgrade)

	if !isWebSocket {
		handleHTTPRequest(connection, serverURL, requestToServer, host)
		return
	}
	handleWebSocket(connection, proxyTarget, protocol, requestToServer)
}

func handleWebSocket(connection *helper.WebSocketIO, proxyTarget, protocol string, requestToServer *http.Request) {
	var serverConnection net.Conn
	var err error

	if protocol == "http" {
		serverConnection, err = net.Dial("tcp", proxyTarget)
	} else {
		serverConnection, err = tls.Dial("tcp", proxyTarget, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		sendErrorResponse(connection)
		return
	}
	defer serverConnection.Close()

	buf := helper.HTTPRequestToString(requestToServer)
	_, err = serverConnection.Write([]byte(buf))
	if err != nil {
		fmt.Println("Error sending HTTP request to server:", err)
		sendErrorResponse(connection)
		return
	}

	go io.Copy(connection, serverConnection)
	io.Copy(serverConnection, connection)

	fmt.Println("Fin de la conexión WebSocket")
}
