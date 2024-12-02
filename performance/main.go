package main

import (
	"crypto/tls"
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

	concurrencyLimit := 50

	var wg sync.WaitGroup
	concurrencyCh := make(chan struct{}, concurrencyLimit)

	http.DefaultClient.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	statusCounts := make(map[int]int)
	var statusMu sync.Mutex

	var completedRequests int
	var completedRequestsMu sync.Mutex

	ticker := time.NewTicker(time.Second / time.Duration(targetRate))
	defer ticker.Stop()

	startTime := time.Now()

	go func() {
		for range time.Tick(1 * time.Second) {
			elapsed := time.Since(startTime).Seconds()

			completedRequestsMu.Lock()
			reqs := completedRequests
			completedRequestsMu.Unlock()

			statusMu.Lock()
			stats := fmt.Sprintf("Tiempo: %.2fs, Completadas: %d, RPS: %.2f\nCódigos de estado:\n", elapsed, reqs, float64(reqs)/elapsed)
			for statusCode, count := range statusCounts {
				stats += fmt.Sprintf("  %d: %d\n", statusCode, count)
			}
			statusMu.Unlock()

			fmt.Printf("\033[H\033[J%s", stats)

			if reqs >= totalRequests {
				break
			}
		}
	}()

	for i := 0; i < totalRequests; i++ {
		<-ticker.C

		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			concurrencyCh <- struct{}{}
			defer func() { <-concurrencyCh }()

			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("\nError en la solicitud #%d: %v\n", requestID, err)
				os.Exit(1)
				return
			}
			defer resp.Body.Close()

			statusMu.Lock()
			statusCounts[resp.StatusCode]++
			statusMu.Unlock()

			completedRequestsMu.Lock()
			completedRequests++
			completedRequestsMu.Unlock()
		}(i)
	}

	wg.Wait()

	elapsedTime := time.Since(startTime)
	actualRate := float64(totalRequests) / elapsedTime.Seconds()

	fmt.Printf("\nTodas las solicitudes han terminado.\n")
	fmt.Printf("Tiempo total: %s\n", elapsedTime)
	fmt.Printf("Peticiones por segundo: %.2f\n", actualRate)

	fmt.Println("Códigos de estado:")
	for statusCode, count := range statusCounts {
		fmt.Printf("  %d: %d\n", statusCode, count)
	}
}
