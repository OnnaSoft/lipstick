package helper

import (
	"log"
	"strconv"
	"strings"
)

func ParseUint(value string) uint {
	parsedValue, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		log.Printf("Error al convertir la cadena a uint: %v", err)
		return 0
	}

	return uint(parsedValue)
}

func ParseTargetEndpoint(target string) (string, string) {
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
