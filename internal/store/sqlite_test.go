package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestOpenCreatesOnlyLedgerSchemaWithoutExtraConstraints(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	rows, err := store.db.Query(`
		SELECT name, sql
		FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	t.Cleanup(func() { _ = rows.Close() })

	var names []string
	var transactionDDL string
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			t.Fatalf("scan sqlite_master row: %v", err)
		}
		names = append(names, name)
		if name == "transactions" {
			transactionDDL = strings.ToUpper(ddl)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite_master: %v", err)
	}

	if got, want := strings.Join(names, ","), "accounts,transactions"; got != want {
		t.Fatalf("tables = %q, want %q", got, want)
	}
	if strings.Contains(transactionDDL, "UNIQUE") {
		t.Fatalf("transactions DDL contains UNIQUE constraint: %s", transactionDDL)
	}
	if strings.Contains(transactionDDL, "BALANCE") {
		t.Fatalf("transactions DDL contains balance column: %s", transactionDDL)
	}
}

func TestSQLiteInsertsAndReadsAccounts(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	for _, account := range []ledger.Account{
		{ID: "acct-3", Name: "Third"},
		{ID: "acct-1", Name: "First"},
		{ID: "acct-2", Name: "Second"},
	} {
		if err := store.InsertAccount(account); err != nil {
			t.Fatalf("InsertAccount(%q) error = %v", account.ID, err)
		}
	}

	accounts, err := store.Accounts()
	if err != nil {
		t.Fatalf("Accounts() error = %v", err)
	}
	wantAccounts := []ledger.Account{
		{ID: "acct-1", Name: "First"},
		{ID: "acct-2", Name: "Second"},
		{ID: "acct-3", Name: "Third"},
	}
	if !reflect.DeepEqual(accounts, wantAccounts) {
		t.Errorf("Accounts() = %#v, want %#v", accounts, wantAccounts)
	}

	account, err := store.Account("acct-2")
	if err != nil {
		t.Fatalf("Account(\"acct-2\") error = %v", err)
	}
	if want := (ledger.Account{ID: "acct-2", Name: "Second"}); account != want {
		t.Errorf("Account(\"acct-2\") = %#v, want %#v", account, want)
	}

	if _, err := store.Account("nope"); !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Errorf("Account(\"nope\") error = %v, want error wrapping %v", err, ledger.ErrAccountNotFound)
	}
}
