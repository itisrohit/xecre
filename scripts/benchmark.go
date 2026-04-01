package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	totalRequests := 10
	var wg sync.WaitGroup
	start := time.Now()

	fmt.Printf("Starting concurrency test with %d requests...\n", totalRequests)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			payload := map[string]string{
				"language": "python",
				"code":     fmt.Sprintf("print('Hello from request %d')", id),
			}
			
			jsonData, _ := json.Marshal(payload)
			resp, err := http.Post("http://localhost:8080/execute", "application/json", bytes.NewBuffer(jsonData))
			
			if err != nil {
				fmt.Printf("Request %d failed: %v\n", id, err)
				return
			}
			defer resp.Body.Close()
			
			fmt.Printf("Request %d finished with status %v\n", id, resp.StatusCode)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("-----------------------------------")
	fmt.Printf("Total Time: %v\n", duration)
	fmt.Printf("Average Time per Request: %v\n", duration/time.Duration(totalRequests))
}
