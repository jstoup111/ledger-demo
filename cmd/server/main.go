// Command server runs the ledger-demo HTTP server and its deterministic seed
// command.
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/httpapi"
	"github.com/jstoup111/ledger-demo/internal/store"
)

var (
	listenAndServe           = http.ListenAndServe
	stdout         io.Writer = os.Stdout
)

const (
	defaultPort   = "8080"
	defaultDBPath = "./ledger.db"
)

func main() {
	log.SetFlags(0)

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if err := run(command, os.Args[2:]...); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run(command string, args ...string) error {
	switch command {
	case "serve":
		return serve()
	case "seed":
		return seed()
	case "export":
		return export(args)
	default:
		return fmt.Errorf("unknown command %q (want: serve, seed, export)", command)
	}
}

func serve() error {
	dbPath := env("LEDGER_DB_PATH", defaultDBPath)
	database, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database %q: %w", dbPath, err)
	}
	defer database.Close()

	router, err := httpapi.NewRouter(database, clock.SystemClock{})
	if err != nil {
		return fmt.Errorf("building router: %w", err)
	}

	addr := ":" + env("PORT", defaultPort)
	log.Printf("ledger-demo listening on http://localhost%s", addr)

	if err := listenAndServe(addr, router); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	return nil
}

func seed() error {
	dbPath := env("LEDGER_DB_PATH", defaultDBPath)
	database, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database %q: %w", dbPath, err)
	}
	defer database.Close()

	seedClock := clock.FixedClock{T: time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)}
	if err := loadSeedData(seedClock, database); err != nil {
		return fmt.Errorf("load seed data: %w", err)
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
