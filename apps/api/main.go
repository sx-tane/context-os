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
	"database/sql"
	"log"
	"net/http"
	"os"

	"context-os/apps/api/bootstrap"
	_ "context-os/apps/api/docs"
	sqlmigrations "context-os/migrations"
	"context-os/storage/db"
)

// defaultAddr is the port the API binds to when API_ADDR is not set.
const defaultAddr = ":8080"

func main() {
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	// Open DB and run migrations. Failure is non-fatal so the API starts even
	// if Postgres is not yet available; DB-backed routes are omitted in that case.
	sqlDB, dbErr := openDB()
	if dbErr == nil {
		defer sqlDB.Close()
	}
	mux := bootstrap.NewMux(sqlDB)

	log.Printf("context-os api listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("api server error: %v", err)
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
