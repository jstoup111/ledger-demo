package store

import (
	"strings"
	"testing"
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
