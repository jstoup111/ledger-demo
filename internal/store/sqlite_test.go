package store

import (
	"encoding/json"
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

func TestSQLiteTransactionsOrdersSameTimestampByIDDescending(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	if err := store.InsertAccount(ledger.Account{ID: "acct-1", Name: "First"}); err != nil {
		t.Fatalf("InsertAccount() error = %v", err)
	}

	createdAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	for _, transaction := range []ledger.Transaction{
		{ID: "txn-0001", AccountID: "acct-1", Amount: 100, Description: "first", CreatedAt: createdAt},
		{ID: "txn-0002", AccountID: "acct-1", Amount: 200, Description: "second", CreatedAt: createdAt},
		{ID: "txn-0003", AccountID: "acct-1", Amount: 300, Description: "third", CreatedAt: createdAt},
	} {
		if err := store.Append(transaction); err != nil {
			t.Fatalf("Append(%q) error = %v", transaction.ID, err)
		}
	}

	firstRead, err := store.Transactions("acct-1")
	if err != nil {
		t.Fatalf("Transactions() first read error = %v", err)
	}
	secondRead, err := store.Transactions("acct-1")
	if err != nil {
		t.Fatalf("Transactions() second read error = %v", err)
	}

	if got, want := [][]string{transactionIDs(firstRead), transactionIDs(secondRead)}, [][]string{{"txn-0003", "txn-0002", "txn-0001"}, {"txn-0003", "txn-0002", "txn-0001"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Transactions() IDs = %#v, want %#v", got, want)
	}
}

func TestSQLiteTransactionsOrdersByChronologicalCreatedAt(t *testing.T) {
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		wantIDs      []string
	}{
		{
			name: "fractional second precision",
			transactions: []ledger.Transaction{
				{ID: "txn-earlier", AccountID: "acct-1", Amount: 100, Description: "earlier", CreatedAt: time.Date(2026, time.August, 8, 10, 0, 0, 100_000_000, time.UTC)},
				{ID: "txn-later", AccountID: "acct-1", Amount: 200, Description: "later", CreatedAt: time.Date(2026, time.August, 8, 10, 0, 0, 900_000_000, time.UTC)},
			},
			wantIDs: []string{"txn-later", "txn-earlier"},
		},
		{
			name: "source UTC offsets",
			transactions: []ledger.Transaction{
				{ID: "txn-earlier", AccountID: "acct-1", Amount: 100, Description: "earlier", CreatedAt: time.Date(2026, time.August, 8, 10, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))},
				{ID: "txn-later", AccountID: "acct-1", Amount: 200, Description: "later", CreatedAt: time.Date(2026, time.August, 8, 9, 45, 0, 0, time.UTC)},
			},
			wantIDs: []string{"txn-later", "txn-earlier"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := Open(":memory:")
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}
			t.Cleanup(func() { _ = store.db.Close() })

			if err := store.InsertAccount(ledger.Account{ID: "acct-1", Name: "First"}); err != nil {
				t.Fatalf("InsertAccount() error = %v", err)
			}
			for _, transaction := range test.transactions {
				if err := store.Append(transaction); err != nil {
					t.Fatalf("Append(%q) error = %v", transaction.ID, err)
				}
			}

			transactions, err := store.Transactions("acct-1")
			if err != nil {
				t.Fatalf("Transactions() error = %v", err)
			}
			if got := transactionIDs(transactions); !reflect.DeepEqual(got, test.wantIDs) {
				t.Fatalf("Transactions() IDs = %#v, want %#v", got, test.wantIDs)
			}
		})
	}
}

func TestSQLiteTransactionsForExistingAccountWithoutRowsReturnsEmptySlice(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	if err := store.InsertAccount(ledger.Account{ID: "acct-empty", Name: "Empty"}); err != nil {
		t.Fatalf("InsertAccount() error = %v", err)
	}

	transactions, err := store.Transactions("acct-empty")
	if err != nil {
		t.Fatalf("Transactions() error = %v", err)
	}
	encoded, err := json.Marshal(transactions)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if got, want := string(encoded), "[]"; got != want {
		t.Fatalf("Transactions() JSON = %q, want %q", got, want)
	}
}

func TestSQLiteTransactionsForUnknownAccountWrapsAccountNotFound(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.db.Close() })

	if _, err := store.Transactions("acct-unknown"); !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Fatalf("Transactions() error = %v, want error wrapping %v", err, ledger.ErrAccountNotFound)
	}
}

func transactionIDs(transactions []ledger.Transaction) []string {
	ids := make([]string, len(transactions))
	for i, transaction := range transactions {
		ids[i] = transaction.ID
	}
	return ids
}
