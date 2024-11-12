package helper

import (
	"fmt"
	"io"
	"net"
	"net/http"
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
