package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/itisrohit/xecre/src/api"
	"github.com/itisrohit/xecre/src/engine"
)

func main() {
	dockerEngine, err := engine.NewDockerEngine()
	if err != nil {
		log.Fatalf("Failed to initialize Docker engine: %v", err)
	}

	handler := &api.Handler{Engine: dockerEngine}
	http.HandleFunc("/", handler.Home)
	http.HandleFunc("/execute", handler.Execute)

	stopping := make(chan os.Signal, 1)
	signal.Notify(stopping, syscall.SIGINT, syscall.SIGTERM)
	server := &http.Server{Addr: ":8080", ReadHeaderTimeout: 5 * time.Second}
	done := make(chan struct{})
	go func() {
		<-stopping
		ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
		server.Shutdown(ctx)
		cancel()
		dockerEngine.Close()
		close(done)
	}()
	fmt.Println("Xecre Execution API listening on :8080...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	<-done
}
