package helper

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
)

func Copy(dst io.Writer, src io.Reader) (int64, error) {

	reader := bufio.NewReader(src)

	request, err := io.ReadAll(reader)
	if err != nil {
		if err == io.EOF {
			return 0, errors.New("EOF while reading HTTP request")
		}
		return 0, errors.New("error parsing HTTP request")
	}

	var requestBuffer bytes.Buffer
	requestBuffer.Write(request)

	written, err := dst.Write(requestBuffer.Bytes())
	if err != nil {
		return 0, errors.New("error writing HTTP request")
	}

	return int64(written), nil

	/*
		var err error
		var written int64 = 0
		buff := make([]byte, 32*1024)

		for {
			n, err := src.Read(buff)
			if n > 0 {
				w, writeErr := dst.Write(buff[:n])
				written += int64(w)
				if writeErr != nil {
					return written, writeErr
				}
			}

			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
		}
		return written, err
	*/
}

func HTTPBypass(conn, remote io.ReadWriter, proxyPass string, protocol string) (err error) {
	// Extraer el host desde proxyPass para la validación
	host := strings.Split(proxyPass, ":")[0]

	reader := bufio.NewReader(conn)

	request, err := http.ReadRequest(reader)
	if err != nil {
		if err == io.EOF {
			return errors.New("EOF while reading HTTP request")
		}
		return errors.New("error parsing HTTP request")
	}

	url := protocol + "://" + proxyPass + request.URL.String()

	request.Host = host
	if protocol == "https" || protocol == "tls" {
		request.Header.Add("Protocol", "https")
	} else if protocol == "http" || protocol == "tcp" {
		request.Header.Add("Protocol", "http")
	}
	request.Header.Add("Host", proxyPass)
	request.Header.Del("Referer")

	var requestBuffer bytes.Buffer
	err = request.Write(&requestBuffer)
	if err != nil {
		return errors.New("error writing HTTP request")
	}

	req, err := http.NewRequest(request.Method, url, request.Body)
	if err != nil {
		return errors.New("error creating HTTP request")
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("error sending HTTP request")
	}

	return response.Write(conn)
}
