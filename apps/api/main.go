// Package main is the entry point for the ContextOS HTTP API.
// It only wires routes to handlers; all logic lives in handler/, request/, response/, and middleware/.
package main

import (
	"log"
	"net/http"
	"os"

	"context-os/apps/api/handler"
	"context-os/apps/api/middleware"
)

// defaultAddr is the port the API binds to when API_ADDR is not set.
const defaultAddr = ":8080"

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	mux := http.NewServeMux()
	mux.Handle("/health", middleware.WithCORS(http.HandlerFunc(handler.Health)))
	mux.Handle("/github/ingest", middleware.WithCORS(http.HandlerFunc(handler.GithubIngest)))

	log.Printf("context-os api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err)
	}
}
