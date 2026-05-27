// Package main is the entry point for the ContextOS HTTP API.
// It only wires routes to handlers; all logic lives in handler/, request/, response/, and middleware/.
//
// @title          ContextOS API
// @version        1.0
// @description    Local-first pipeline API for ingesting and reasoning over engineering context.
// @host           localhost:8080
// @BasePath       /
package main

import (
	"log"
	"net/http"
	"os"

	_ "context-os/apps/api/docs"
	"context-os/apps/api/handler"
	"context-os/apps/api/middleware"

	httpSwagger "github.com/swaggo/http-swagger"
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
	mux.Handle("/github/status", middleware.WithCORS(http.HandlerFunc(handler.GithubStatus)))
	mux.Handle("/slack/ingest", middleware.WithCORS(http.HandlerFunc(handler.SlackIngest)))
	mux.Handle("/slack/status", middleware.WithCORS(http.HandlerFunc(handler.SlackStatus)))
	mux.Handle("/slack/connect", middleware.WithCORS(http.HandlerFunc(handler.SlackConnect)))
	mux.Handle("/slack/callback", http.HandlerFunc(handler.SlackCallback))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	log.Printf("context-os api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err)
	}
}
