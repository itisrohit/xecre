package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/itisrohit/xecre/src/api"
	"github.com/itisrohit/xecre/src/engine"
)

func main() {
	dockerEngine, err := engine.NewDockerEngine()
	if err != nil {
		log.Fatalf("Failed to initialize Docker engine: %v", err)
	}

	handler := &api.Handler{Engine: dockerEngine}
	http.HandleFunc("/execute", handler.Execute)

	fmt.Println("Xecre Execution API listening on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
