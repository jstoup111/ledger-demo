package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestSQLiteAppendsTransactionsAndCountsAllAccounts(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	for _, account := range []ledger.Account{
		{ID: "acct-1", Name: "First"},
		{ID: "acct-2", Name: "Second"},
	} {
		if err := store.InsertAccount(account); err != nil {
			t.Fatalf("InsertAccount(%q) error = %v", account.ID, err)
		}
	}

	createdAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	for _, transaction := range []ledger.Transaction{
		{ID: "txn-0001", AccountID: "acct-1", Amount: 100, Description: "first", CreatedAt: createdAt},
		{ID: "txn-0002", AccountID: "acct-1", Amount: 200, Description: "second", CreatedAt: createdAt},
		{ID: "txn-0003", AccountID: "acct-2", Amount: 300, Description: "third", CreatedAt: createdAt},
		{ID: "txn-0004", AccountID: "acct-2", Amount: 400, Description: "fourth", CreatedAt: createdAt},
		{ID: "txn-0005", AccountID: "acct-2", Amount: 500, Description: "fifth", CreatedAt: createdAt},
	} {
		if err := store.Append(transaction); err != nil {
			t.Fatalf("Append(%q) error = %v", transaction.ID, err)
		}
	}

	beforeAppend, err := store.CountTransactions()
	if err != nil {
		t.Fatalf("CountTransactions() before append error = %v", err)
	}
	appended := ledger.Transaction{ID: "txn-0006", AccountID: "acct-1", Amount: 600, Description: "sixth", CreatedAt: createdAt}
	if err := store.Append(appended); err != nil {
		t.Fatalf("Append(%q) error = %v", appended.ID, err)
	}
	afterAppend, err := store.CountTransactions()
	if err != nil {
		t.Fatalf("CountTransactions() after append error = %v", err)
	}
	var read ledger.Transaction
	if err := store.db.QueryRow(`
		SELECT id, account_id, amount, description
		FROM transactions
		WHERE id = ?
	`, appended.ID).Scan(&read.ID, &read.AccountID, &read.Amount, &read.Description); err != nil {
		t.Fatalf("read appended transaction error = %v", err)
	}
	if got, want := struct {
		beforeAppend int
		afterAppend  int
		read         ledger.Transaction
	}{beforeAppend, afterAppend, read}, (struct {
		beforeAppend int
		afterAppend  int
		read         ledger.Transaction
	}{5, 6, ledger.Transaction{ID: "txn-0006", AccountID: "acct-1", Amount: 600, Description: "sixth"}}); got != want {
		t.Fatalf("transaction storage result = %+v, want %+v", got, want)
	}
}
