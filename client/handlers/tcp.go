package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
)

func HandleTCP(connection net.Conn, proxyTarget, protocol string) {
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

	go func() {
		for {
			buffer := make([]byte, 1024)
			n, err := serverConnection.Read(buffer)
			if err != nil {
				break
			}
			_, err = connection.Write(buffer[:n])
			if err != nil {
				break
			}
		}
	}()

	io.Copy(serverConnection, connection)
}
