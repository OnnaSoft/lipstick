package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/OnnaSoft/lipstick/helper"
)

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 100,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		WriteBufferSize: 1024,
		ReadBufferSize:  1024,
	},
}

func HandleHTTP(connection net.Conn, proxyTarget, protocol string) {
	req, err := helper.ParseHTTPRequest(connection)
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

// handleHTTPRequest forwards HTTP requests using Go's http.Client.
func handleHTTPRequest(connection net.Conn, serverURL string, req *http.Request, host string) {

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

func sendHTTPResponse(connection net.Conn, resp *http.Response) error {
	headers := ""
	for key, values := range resp.Header {
		for _, value := range values {
			headers += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}
	responseHeader := fmt.Sprintf("HTTP/1.1 %d %s\r\n%s\r\n", resp.StatusCode, resp.Status, headers)
	if _, err := fmt.Fprint(connection, responseHeader); err != nil {
		fmt.Println("Error sending HTTP response header:", err)
		return err
	}

	_, err := io.Copy(connection, resp.Body)
	return err
}

func handleWebSocket(connection net.Conn, proxyTarget, protocol string, requestToServer *http.Request) {
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

func sendErrorResponse(connection net.Conn) {
	connection.Write([]byte(helper.HttpErrorResponse))
}
