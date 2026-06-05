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
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context-os/apps/api/bootstrap"
	_ "context-os/apps/api/docs"
	sqlmigrations "context-os/migrations"
	"context-os/storage/db"
)

// defaultAddr is the loopback address the API binds to when API_ADDR is not set.
const defaultAddr = "127.0.0.1:8080"

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Open DB and run migrations. Failure is non-fatal so the API starts even
	// if Postgres is not yet available; DB-backed routes are omitted in that case.
	sqlDB, dbErr := openDB()
	if dbErr == nil {
		defer sqlDB.Close()
	}
	mux := bootstrap.NewMux(ctx, sqlDB)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("context-os api listening on %s", addr)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Fatalf("api server error: %v", err)
	case <-ctx.Done():
		stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("api shutdown error: %v", err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api server error during shutdown: %v", err)
		}
	}
}

// openDB opens the Postgres connection pool and runs pending migrations.
func openDB() (*sql.DB, error) {
	conn, err := db.Open(sqlmigrations.Files)
	if err != nil {
		log.Printf("db: unavailable, workspace endpoints disabled: %v", err)
		return nil, err
	}
	log.Printf("db: connected and migrations applied")
	return conn, nil
}
