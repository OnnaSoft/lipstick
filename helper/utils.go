package helper

import (
	"log"
	"strconv"
)

func ParseUint(value string) uint {
	parsedValue, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		log.Printf("Error al convertir la cadena a uint: %v", err)
		return 0
	}

	return uint(parsedValue)
}
