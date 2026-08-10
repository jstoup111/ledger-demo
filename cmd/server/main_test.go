package main

import (
	"bytes"
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
	const want = `unknown command "frobnicate" (want: serve, seed, export)`
	if err == nil || err.Error() != want {
		t.Errorf("run(frobnicate) error = %q, want %q", err, want)
	}
}

func TestRunExportWritesSeededAccountCSVToStdout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)
	if err := run("seed"); err != nil {
		t.Fatalf("run(seed) error = %v", err)
	}

	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	var output bytes.Buffer
	stdout = &output

	if err := run("export", "acct-1"); err != nil {
		t.Fatalf("run(export, acct-1) error = %v", err)
	}
	want := "id,amount_cents,description,created_at\n" +
		"txn-0012,-5500,Concert tickets,2026-08-08T14:30:00Z\n"
	if got := output.String(); !strings.HasPrefix(got, want) {
		t.Errorf("run(export, acct-1) output = %q, want first two lines %q", got, want)
	}
}

func TestRunExportRequiresExactlyOneAccountID(t *testing.T) {
	for _, testCase := range []struct {
		name string
		args []string
	}{
		{name: "zero arguments", args: []string{"export"}},
		{name: "two arguments", args: []string{"export", "acct-1", "acct-2"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			originalStdout := stdout
			t.Cleanup(func() { stdout = originalStdout })
			var output bytes.Buffer
			stdout = &output

			err := run(testCase.args[0], testCase.args[1:]...)
			if err == nil || !strings.Contains(err.Error(), "exactly one account ID") {
				t.Errorf("run(%q) error = %q, want error saying exactly one account ID is expected", testCase.args, err)
			}
			if got := output.String(); got != "" {
				t.Errorf("run(%q) stdout = %q, want empty", testCase.args, got)
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

	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	var output bytes.Buffer
	stdout = &output

	const accountID = "acct-missing"
	err := run("export", accountID)
	if err == nil || !strings.Contains(err.Error(), accountID) {
		t.Errorf("run(export, %q) error = %q, want error naming account ID", accountID, err)
	}
	if got := output.String(); got != "" {
		t.Errorf("run(export, %q) stdout = %q, want empty", accountID, got)
	}
}

func TestRunExportReportsMissingParentDatabaseDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "ledger.db")
	t.Setenv("LEDGER_DB_PATH", dbPath)

	originalStdout := stdout
	t.Cleanup(func() { stdout = originalStdout })
	var output bytes.Buffer
	stdout = &output

	err := run("export", "acct-1")
	if err == nil || !strings.Contains(err.Error(), dbPath) {
		t.Errorf("run(export, acct-1) error = %q, want database path %q", err, dbPath)
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Errorf("run(export, acct-1) created database %q; stat error = %v", dbPath, statErr)
	}
	if got := output.String(); got != "" {
		t.Errorf("run(export, acct-1) stdout = %q, want empty", got)
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
