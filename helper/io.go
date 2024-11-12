package helper

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// CopyData transfiere datos desde la conexión de origen (src) hacia la conexión de destino (dst).
func CopyData(destination, source net.Conn) (int64, error) {
	var totalBytesWritten int64
	buffer := make([]byte, 32*1024) // Tamaño del buffer: 32 KB

	for {
		// Leer datos del origen
		bytesRead, readErr := source.Read(buffer)
		if bytesRead > 0 {
			// Escribir datos en el destino
			bytesWritten, writeErr := destination.Write(buffer[:bytesRead])
			totalBytesWritten += int64(bytesWritten)
			//os.Stdout.Write(buffer[:bytesRead])

			// Manejar errores de escritura
			if writeErr != nil {
				fmt.Println("Error: conexión de escritura cerrada durante la escritura.")
				return totalBytesWritten, fmt.Errorf("error writing to destination: %w", writeErr)
			}

			// Verificar si se escribió menos de lo leído
			if bytesWritten < bytesRead {
				fmt.Println("Error: escritura parcial detectada.")
				return totalBytesWritten, fmt.Errorf("partial write: wrote %d of %d bytes", bytesWritten, bytesRead)
			}
		}

		// Manejar errores de lectura
		if readErr != nil {
			if readErr == io.EOF {
				fmt.Println("EOF alcanzado: conexión de lectura cerrada.")
				break
			}
			fmt.Println("Error: conexión de lectura cerrada inesperadamente.")
			return totalBytesWritten, fmt.Errorf("error reading from source: %w", readErr)
		}
	}

	fmt.Println("Copia completa. Total de bytes escritos:", totalBytesWritten)
	return totalBytesWritten, nil
}

// HandleTcpBypass maneja solicitudes TCP y las redirige al servidor de destino.
func HandleTcpBypass(clientConnection net.Conn, proxyTarget string, protocol string) error {
	defer func() {
		fmt.Println("Cerrando conexión TCP")
	}()
	host := strings.Split(proxyTarget, ":")[0]
	reader := bufio.NewReader(clientConnection)
	clientRequest, err := http.ReadRequest(reader)
	if err != nil {
		return fmt.Errorf("error reading HTTP request: %w", err)
	}

	if protocol != "http" && protocol != "https" {
		protocol = "http"
	}
	serverURL := protocol + "://" + proxyTarget + clientRequest.URL.String()
	clientRequest.Host = host

	if clientRequest.Header.Get("Upgrade") == "websocket" {
		serverConnection, err := net.Dial("tcp", proxyTarget)
		if err != nil {
			return fmt.Errorf("error connecting to WebSocket server: %w", err)
		}
		//defer serverConnection.Close()

		_, err = serverConnection.Write([]byte(HTTPRequestToString(clientRequest)))
		if err != nil {
			return fmt.Errorf("error sending HTTP upgrade request to server: %w", err)
		}

		responseReader := bufio.NewReader(serverConnection)
		serverResponse, err := http.ReadResponse(responseReader, clientRequest)
		if err != nil {
			return fmt.Errorf("error reading HTTP upgrade response from server: %w", err)
		}
		defer serverResponse.Body.Close()

		responseBytes := HTTPResponseToBytes(serverResponse)
		_, err = clientConnection.Write(responseBytes)
		if err != nil {
			return fmt.Errorf("error sending HTTP upgrade response to client: %w", err)
		}

		go func() {
			for {
				time.Sleep(5 * time.Second)
				fmt.Println("Enviando ping a serverConnection..")
				_, err := serverConnection.Write([]byte("hola mundo"))
				if err != nil {
					fmt.Println("Error al enviar ping a serverConnection:", err)
					serverConnection.Close()
					break
				}
				time.Sleep(10 * time.Second)
			}
		}()
		go func() {
			for {
				time.Sleep(5 * time.Second)
				fmt.Println("Enviando ping a clientConnection...")
				_, err := clientConnection.Write([]byte("hola mundo"))
				if err != nil {
					fmt.Println("Error al enviar ping a clientConnection:", err)
					clientConnection.Close()
					break
				}
				time.Sleep(10 * time.Second)
			}
		}()

		/*go func() {
			CopyData(serverConnection, clientConnection)
		}()
		CopyData(clientConnection, serverConnection)*/
		time.Sleep(1 * time.Hour)

		return nil
	}

	requestToServer, err := http.NewRequest(clientRequest.Method, serverURL, clientRequest.Body)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}
	serverResponse, err := http.DefaultClient.Do(requestToServer)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %w", err)
	}
	return serverResponse.Write(clientConnection)
}

// HTTPRequestToString convierte una solicitud HTTP en una cadena para ser enviada como texto plano.
func HTTPRequestToString(request *http.Request) string {
	var requestString string
	requestString += fmt.Sprintf("%s %s %s\r\n", request.Method, request.URL.RequestURI(), request.Proto)
	for key, values := range request.Header {
		for _, value := range values {
			requestString += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}
	requestString += "\r\n"
	return requestString
}

// HTTPResponseToBytes convierte una respuesta HTTP en un arreglo de bytes para ser enviada al cliente.
func HTTPResponseToBytes(response *http.Response) []byte {
	var buffer []byte
	buffer = append(buffer, fmt.Sprintf("%s %d %s\r\n", response.Proto, response.StatusCode, response.Status)...)
	for key, values := range response.Header {
		for _, value := range values {
			buffer = append(buffer, fmt.Sprintf("%s: %s\r\n", key, value)...)
		}
	}
	buffer = append(buffer, "\r\n"...)
	body, _ := io.ReadAll(response.Body)
	buffer = append(buffer, body...)
	return buffer
}
