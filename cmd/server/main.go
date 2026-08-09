// Command server runs the ledger-demo HTTP server and its deterministic seed
// command.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/teamupstart/ledger-demo/internal/httpapi"
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

	var err error
	switch command {
	case "serve":
		err = serve()
	case "seed":
		err = seed()
	default:
		err = fmt.Errorf("unknown command %q (want: serve, seed)", command)
	}

	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func serve() error {
	router, err := httpapi.NewRouter()
	if err != nil {
		return fmt.Errorf("building router: %w", err)
	}

	addr := ":" + env("PORT", defaultPort)
	log.Printf("ledger-demo listening on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, router); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listening on %s: %w", addr, err)
	}
	return nil
}

// seed loads deterministic demo data: 3 accounts with 8-12 plausible
// transactions each, fixed timestamps, no randomness. It is not implemented yet
// because it depends on the ledger domain, which is built during the demo.
func seed() error {
	log.Printf("seed: nothing to load yet — the ledger domain is built during the demo")
	log.Printf("seed: target database would be %s", env("LEDGER_DB_PATH", defaultDBPath))
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
