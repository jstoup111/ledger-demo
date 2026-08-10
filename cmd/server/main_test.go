package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

func TestRunRejectsUnknownCommandWithoutStartingServer(t *testing.T) {
	err := run("frobnicate")
	if err == nil {
		t.Fatal("run(frobnicate) error = nil, want error")
	}
	for _, command := range []string{"serve", "seed", "export"} {
		if !strings.Contains(err.Error(), command) {
			t.Errorf("run(frobnicate) error = %q, want valid command %q", err, command)
		}
	}
}

func TestRunExportWritesSeededAccountCSVToStandardOutput(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)
	if err := run("seed"); err != nil {
		t.Fatalf("run(seed) error = %v", err)
	}

	var output bytes.Buffer
	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	stdout = &output

	if err := run("export", "acct-1"); err != nil {
		t.Fatalf("run(export, acct-1) error = %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if got, want := lines[0], "id,amount_cents,description,created_at"; got != want {
		t.Errorf("CSV header = %q, want %q", got, want)
	}
	if got, want := lines[1], "txn-0012,-5500,Concert tickets,2026-08-08T14:30:00Z"; got != want {
		t.Errorf("newest CSV transaction = %q, want %q", got, want)
	}
}

func TestRunExportRejectsWrongNumberOfArgumentsWithoutOutput(t *testing.T) {
	for _, args := range [][]string{nil, {"acct-1", "acct-2"}} {
		t.Run(strings.Join(args, ","), func(t *testing.T) {
			var output bytes.Buffer
			originalStdout := stdout
			t.Cleanup(func() { stdout = originalStdout })
			stdout = &output

			err := run("export", args...)
			if err == nil || !strings.Contains(err.Error(), "exactly one account ID is expected") {
				t.Fatalf("run(export, %v) error = %v, want exactly-one-account-ID error", args, err)
			}
			if got := output.Len(); got != 0 {
				t.Errorf("output length = %d, want 0", got)
			}
		})
	}
}

func TestRunExportRejectsUnknownAccountWithoutOutput(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)
	if err := run("seed"); err != nil {
		t.Fatalf("run(seed) error = %v", err)
	}

	var output bytes.Buffer
	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	stdout = &output

	err := run("export", "acct-nope")
	if !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Fatalf("run(export, acct-nope) error = %v, want error wrapping %v", err, ledger.ErrAccountNotFound)
	}
	if !strings.Contains(err.Error(), "acct-nope") {
		t.Errorf("run(export, acct-nope) error = %q, want account ID", err)
	}
	if got := output.Len(); got != 0 {
		t.Errorf("output length = %d, want 0", got)
	}
}

func TestRunExportReportsMissingParentDatabaseDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)

	var output bytes.Buffer
	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	stdout = &output

	err := run("export", "acct-1")
	if err == nil {
		t.Fatal("run(export) error = nil, want error")
	}
	if !strings.Contains(err.Error(), dbPath) {
		t.Errorf("run(export) error = %q, want database path %q", err, dbPath)
	}
	if got := output.Len(); got != 0 {
		t.Errorf("output length = %d, want 0", got)
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Errorf("export() created database %q; stat error = %v", dbPath, statErr)
	}
}

func TestSeedLoadsDemoDataIntoLedgerDBPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)

	if err := run("seed"); err != nil {
		t.Fatalf("run(seed) error = %v", err)
	}

	database, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	accounts, err := database.Accounts()
	if err != nil {
		t.Fatalf("Accounts() error = %v", err)
	}
	if len(accounts) != 3 {
		t.Errorf("seeded accounts = %d, want 3", len(accounts))
	}
}

func TestRunSeedProducesDeterministicTransactions(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), "first.db")
	t.Setenv("LEDGER_DB_PATH", firstPath)
	if err := run("seed"); err != nil {
		t.Fatalf("first run(seed) error = %v", err)
	}
	first := seededDatabaseState(t, firstPath)

	secondPath := filepath.Join(t.TempDir(), "second.db")
	t.Setenv("LEDGER_DB_PATH", secondPath)
	if err := run("seed"); err != nil {
		t.Fatalf("second run(seed) error = %v", err)
	}
	second := seededDatabaseState(t, secondPath)

	if !reflect.DeepEqual(first, second) {
		t.Errorf("command-level seed data differs between fresh databases:\nfirst:  %#v\nsecond: %#v", first, second)
	}
}

func TestRunServeUsesConfiguredDatabaseAndPortWithoutListening(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	database, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	if err := database.InsertAccount(ledger.Account{ID: "acct-serve", Name: "Serve Fixture"}); err != nil {
		t.Fatalf("InsertAccount() error = %v", err)
	}
	t.Setenv("LEDGER_DB_PATH", dbPath)
	t.Setenv("PORT", "4312")

	originalListenAndServe := listenAndServe
	t.Cleanup(func() { listenAndServe = originalListenAndServe })
	listenAndServe = func(addr string, handler http.Handler) error {
		if addr != ":4312" {
			t.Errorf("listen address = %q, want %q", addr, ":4312")
		}
		request := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET /api/accounts status = %d, want %d", response.Code, http.StatusOK)
		}
		if !strings.Contains(response.Body.String(), "Serve Fixture") {
			t.Errorf("GET /api/accounts body = %q, want account from LEDGER_DB_PATH", response.Body.String())
		}
		return http.ErrServerClosed
	}

	if err := run("serve"); err != nil {
		t.Fatalf("run(serve) error = %v", err)
	}
}

func TestRunServeUsesDefaultPortWhenPortIsUnsetOrBlank(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		setEnv bool
	}{
		{name: "unset"},
		{name: "blank", setEnv: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "ledger.db")
			t.Setenv("LEDGER_DB_PATH", dbPath)
			if testCase.setEnv {
				t.Setenv("PORT", "")
			} else {
				originalPort, wasSet := os.LookupEnv("PORT")
				if err := os.Unsetenv("PORT"); err != nil {
					t.Fatalf("Unsetenv(PORT) error = %v", err)
				}
				t.Cleanup(func() {
					if wasSet {
						_ = os.Setenv("PORT", originalPort)
						return
					}
					_ = os.Unsetenv("PORT")
				})
			}

			originalListenAndServe := listenAndServe
			t.Cleanup(func() { listenAndServe = originalListenAndServe })
			listenAndServe = func(addr string, _ http.Handler) error {
				if addr != ":"+defaultPort {
					t.Errorf("listen address = %q, want default %q", addr, ":"+defaultPort)
				}
				return http.ErrServerClosed
			}

			if err := run("serve"); err != nil {
				t.Fatalf("run(serve) error = %v", err)
			}
		})
	}
}

func TestSeedReportsMissingParentDatabaseDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)

	err := run("seed")
	if err == nil {
		t.Fatal("run(seed) error = nil, want error")
	}
	if !strings.Contains(err.Error(), dbPath) {
		t.Errorf("run(seed) error = %q, want database path %q", err, dbPath)
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Errorf("seed() created database %q; stat error = %v", dbPath, statErr)
	}
}

func seededDatabaseState(t *testing.T, dbPath string) seedState {
	t.Helper()
	database, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dbPath, err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close(%q) error = %v", dbPath, err)
		}
	}()
	accounts, err := database.Accounts()
	if err != nil {
		t.Fatalf("Accounts() error = %v", err)
	}
	transactions := make([]ledger.Transaction, 0)
	for _, account := range accounts {
		rows, err := database.Transactions(account.ID)
		if err != nil {
			t.Fatalf("Transactions(%q) error = %v", account.ID, err)
		}
		transactions = append(transactions, rows...)
	}
	return seedState{accounts: accounts, transactions: transactions}
}
