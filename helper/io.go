package helper

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func Copy(dst, src net.Conn) (int64, error) {
	var totalWritten int64 = 0
	buff := make([]byte, 32*1024) // Tamaño del buffer de 32 KB

	for {
		// Leer datos del origen
		n, readErr := src.Read(buff)
		if n > 0 {
			// Escribir datos en el destino
			w, writeErr := dst.Write(buff[:n])
			totalWritten += int64(w)
			os.Stdout.Write(buff[:n])

			// Manejar errores de escritura
			if writeErr != nil {
				// Mensaje de cierre claro
				fmt.Println("Error: conexión de escritura cerrada durante la escritura.")
				return totalWritten, fmt.Errorf("error writing to destination: %w", writeErr)
			}

			// Verificar si se escribió menos de lo leído
			if w < n {
				fmt.Println("Error: escritura parcial detectada.")
				return totalWritten, fmt.Errorf("partial write: wrote %d of %d bytes", w, n)
			}
		}

		// Manejar errores de lectura
		if readErr != nil {
			if readErr == io.EOF {
				// EOF indica que no hay más datos para leer
				fmt.Println("EOF alcanzado: conexión de lectura cerrada.")
				break
			}
			// Otros errores se devuelven directamente
			fmt.Println("Error: conexión de lectura cerrada inesperadamente.")
			return totalWritten, fmt.Errorf("error reading from source: %w", readErr)
		}
	}

	// Retorna el total de bytes escritos y el error (si aplica)
	fmt.Println("Copia completa. Total de bytes escritos:", totalWritten)
	return totalWritten, nil
}
func TcpBypass(conn net.Conn, proxyPass string, protocol string) error {
	host := strings.Split(proxyPass, ":")[0]
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		return fmt.Errorf("error reading HTTP request: %w", err)
	}

	if protocol != "http" && protocol != "https" {
		protocol = "http"
	}
	url := protocol + "://" + proxyPass + request.URL.String()
	request.Host = host

	if request.Header.Get("Upgrade") == "websocket" {
		serverConn, err := net.Dial("tcp", proxyPass)
		if err != nil {
			return fmt.Errorf("error connecting to WebSocket server: %w", err)
		}
		defer serverConn.Close()

		_, err = serverConn.Write([]byte(requestToString(request)))
		if err != nil {
			return fmt.Errorf("error sending HTTP upgrade request to server: %w", err)
		}

		responseReader := bufio.NewReader(serverConn)
		response, err := http.ReadResponse(responseReader, request)
		if err != nil {
			return fmt.Errorf("error reading HTTP upgrade response from server: %w", err)
		}
		defer response.Body.Close()

		responseBytes := responseToBytes(response)
		_, err = conn.Write(responseBytes)
		if err != nil {
			return fmt.Errorf("error sending HTTP upgrade response to client: %w", err)
		}

		go func() {
			io.Copy(serverConn, conn)
		}()
		io.Copy(conn, serverConn)

		return nil
	}

	req, err := http.NewRequest(request.Method, url, request.Body)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %w", err)
	}
	return response.Write(conn)
}

func requestToString(req *http.Request) string {
	var result string
	result += fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto)
	for k, v := range req.Header {
		for _, value := range v {
			result += fmt.Sprintf("%s: %s\r\n", k, value)
		}
	}
	result += "\r\n"
	return result
}

func responseToBytes(res *http.Response) []byte {
	var buf []byte
	buf = append(buf, fmt.Sprintf("%s %d %s\r\n", res.Proto, res.StatusCode, res.Status)...)
	for k, v := range res.Header {
		for _, value := range v {
			buf = append(buf, fmt.Sprintf("%s: %s\r\n", k, value)...)
		}
	}
	buf = append(buf, "\r\n"...)
	body, _ := io.ReadAll(res.Body)
	buf = append(buf, body...)
	return buf
}
