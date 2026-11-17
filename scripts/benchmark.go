package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	requests := flag.Int("requests", 10, "total requests")
	concurrency := flag.Int("concurrency", 2, "maximum simultaneous requests")
	flag.Parse()
	if *requests < 1 || *concurrency < 1 {
		panic("requests and concurrency must be positive")
	}
	totalRequests := *requests
	semaphore := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup
	latencies := make([]time.Duration, totalRequests)
	var failed int
	var failedMu sync.Mutex
	start := time.Now()
	httpClient := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("Starting concurrency test with %d requests...\n", totalRequests)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			requestStart := time.Now()

			payload := map[string]string{
				"language": "python",
				"code":     fmt.Sprintf("print('Hello from request %d')", id),
			}

			jsonData, _ := json.Marshal(payload)
			resp, err := httpClient.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonData))

			if err != nil {
				fmt.Printf("Request %d failed: %v\n", id, err)
				failedMu.Lock()
				failed++
				failedMu.Unlock()
				return
			}
			defer resp.Body.Close()
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				failedMu.Lock()
				failed++
				failedMu.Unlock()
				fmt.Printf("Request %d response failed: %v\n", id, err)
				return
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				failedMu.Lock()
				failed++
				failedMu.Unlock()
				fmt.Printf("Request %d failed with status %v\n", id, resp.StatusCode)
				return
			}
			latencies[id] = time.Since(requestStart)
			fmt.Printf("Request %d finished with status %v in %v\n", id, resp.StatusCode, latencies[id])
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	valid := make([]time.Duration, 0, len(latencies))
	for _, latency := range latencies {
		if latency > 0 {
			valid = append(valid, latency)
		}
	}
	var total time.Duration
	for _, latency := range valid {
		total += latency
	}
	percentile := func(p float64) time.Duration {
		if len(valid) == 0 {
			return 0
		}
		index := int(math.Ceil(float64(len(valid))*p)) - 1
		return valid[index]
	}
	fmt.Println(strings.Repeat("-", 36))
	fmt.Printf("Requests: %d · concurrency: %d\n", totalRequests, *concurrency)
	fmt.Printf("Completed: %d · Failed: %d\n", len(valid), failed)
	fmt.Printf("Wall time: %v\n", duration)
	if len(valid) > 0 {
		fmt.Printf("Mean request time: %v\n", total/time.Duration(len(valid)))
		fmt.Printf("Median request time: %v\n", percentile(0.50))
		fmt.Printf("P95 request time: %v\n", percentile(0.95))
	}
}
