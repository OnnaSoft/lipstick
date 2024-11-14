package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	// Definir los parámetros de entrada
	var url string
	var targetRate int
	var totalRequests int
	flag.StringVar(&url, "url", "http://localhost:8080", "URL objetivo para las peticiones")
	flag.IntVar(&targetRate, "rate", 500, "Número de peticiones por segundo")
	flag.IntVar(&totalRequests, "requests", 10000, "Número total de peticiones")
	flag.Parse()

	concurrencyLimit := 50 // Número máximo de goroutines concurrentes

	var wg sync.WaitGroup
	concurrencyCh := make(chan struct{}, concurrencyLimit)

	// Map para contar los códigos de estado
	statusCounts := make(map[int]int)
	var statusMu sync.Mutex

	// Contadores para estadísticas en tiempo real
	var completedRequests int
	var completedRequestsMu sync.Mutex

	// Timer para mantener la tasa de solicitudes
	ticker := time.NewTicker(time.Second / time.Duration(targetRate))
	defer ticker.Stop()

	startTime := time.Now()

	// Goroutine para imprimir estadísticas cada segundo
	go func() {
		for range time.Tick(1 * time.Second) {
			elapsed := time.Since(startTime).Seconds()
			completedRequestsMu.Lock()
			reqs := completedRequests
			completedRequestsMu.Unlock()

			statusMu.Lock()
			// Sobrescribir la línea anterior con \r
			fmt.Printf("\rTiempo: %.2fs, Completadas: %d, RPS: %.2f", elapsed, reqs, float64(reqs)/elapsed)
			statusMu.Unlock()

			// Salir del loop si todas las solicitudes están completadas
			if reqs >= totalRequests {
				break
			}
		}
	}()

	for i := 0; i < totalRequests; i++ {
		<-ticker.C // Espacia las solicitudes para mantener la tasa configurada

		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			// Adquiere un slot en el limitador de concurrencia
			concurrencyCh <- struct{}{}
			defer func() { <-concurrencyCh }()

			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("\nError en la solicitud #%d: %v\n", requestID, err)
				os.Exit(1) // Salir del programa si hay un error
				return
			}
			defer resp.Body.Close()

			// Contar los códigos de estado
			statusMu.Lock()
			statusCounts[resp.StatusCode]++
			statusMu.Unlock()

			// Incrementar el contador de solicitudes completadas
			completedRequestsMu.Lock()
			completedRequests++
			completedRequestsMu.Unlock()
		}(i)
	}

	wg.Wait()

	// Estadísticas finales
	elapsedTime := time.Since(startTime)
	actualRate := float64(totalRequests) / elapsedTime.Seconds()

	fmt.Printf("\nTodas las solicitudes han terminado.\n")
	fmt.Printf("Tiempo total: %s\n", elapsedTime)
	fmt.Printf("Peticiones por segundo: %.2f\n", actualRate)

	// Imprimir estadísticas de códigos de estado
	fmt.Println("Códigos de estado:")
	for statusCode, count := range statusCounts {
		fmt.Printf("  %d: %d\n", statusCode, count)
	}
}
