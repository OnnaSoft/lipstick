package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	url := "http://localhost:8080"
	requests := 100000

	// Limita la concurrencia a 10 goroutines
	concurrencyLimit := make(chan struct{}, 512)

	startTime := time.Now()

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Adquiere un semáforo para limitar la concurrencia
			concurrencyLimit <- struct{}{}
			defer func() { <-concurrencyLimit }()

			resp, err := http.Get(url)
			if err != nil {
				fmt.Println("Error en la solicitud:", err)
				return
			}
			defer resp.Body.Close()

			// Simplemente imprimir el código de estado, podrías agregar más lógica según tus necesidades.
			// fmt.Println("Código de estado:", resp.Status)
		}()
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	requestsPerSecond := float64(requests) / elapsedTime.Seconds()

	fmt.Printf("Todas las solicitudes han terminado.\n")
	fmt.Printf("Tiempo total: %s\n", elapsedTime)
	fmt.Printf("Peticiones por segundo: %.2f\n", requestsPerSecond)
}
