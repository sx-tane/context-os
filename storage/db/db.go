// Package db opens and manages a shared *sql.DB connection pool backed by PostgreSQL.
// Call Open() once at startup; pass the returned *sql.DB to store implementations.
//
// Migrations are passed as an fs.FS so that the caller (typically apps/api/main.go)
// embeds the SQL files next to where they live in the source tree.
package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	// Register the postgres driver via side-effect import.
	_ "github.com/lib/pq"
)

// Open creates a *sql.DB using DATABASE_URL from the environment, falling back
// to the local-dev default, then runs pending migrations from migrationsFS.
// Pass a nil migrationsFS to skip auto-migration.
func Open(migrationsFS fs.FS) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://contextos:contextos@localhost:5432/contextos?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	if migrationsFS != nil {
		if err := migrate(db, migrationsFS); err != nil {
			db.Close()
			return nil, fmt.Errorf("db: migrate: %w", err)
		}
	}
	return db, nil
}

// migrate runs every *.sql file found in the root of migrationsFS in
// lexicographic order. Files already recorded in schema_migrations are skipped.
func migrate(db *sql.DB, migrationsFS fs.FS) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations fs: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		filename := entry.Name()

		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, filename,
		).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", filename, err)
		}
		if count > 0 {
			continue
		}

		data, err := fs.ReadFile(migrationsFS, filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", filename, err)
		}
		if _, err := db.Exec(
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, filename,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", filename, err)
		}
	}
	return nil
}
