package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

func TestWriteAccountCSV(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		transactions []ledger.Transaction
		want         string
		wantRows     int
	}{
		{
			name:      "transactions retain store order and exact values",
			accountID: "acct-export",
			transactions: []ledger.Transaction{
				{
					ID:          "txn-positive",
					AccountID:   "acct-export",
					Amount:      158579,
					Description: "Deposit",
					CreatedAt:   time.Date(2026, time.August, 8, 10, 30, 0, 0, time.FixedZone("UTC-4", -4*60*60)),
				},
				{
					ID:          "txn-negative",
					AccountID:   "acct-export",
					Amount:      -6450,
					Description: "Lunch, \"team\"",
					CreatedAt:   time.Date(2026, time.August, 8, 15, 45, 0, 0, time.UTC),
				},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-negative,-6450,\"Lunch, \"\"team\"\"\",2026-08-08T15:45:00Z\n" +
				"txn-positive,158579,Deposit,2026-08-08T14:30:00Z\n",
			wantRows: 3,
		},
		{
			name:      "existing empty account is header only",
			accountID: "acct-empty",
			want:      "id,amount_cents,description,created_at\n",
			wantRows:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, err := store.Open(":memory:")
			if err != nil {
				t.Fatalf("Open(:memory:) error = %v", err)
			}
			t.Cleanup(func() { _ = database.Close() })
			if err := database.InsertAccount(ledger.Account{ID: test.accountID, Name: "CSV Fixture"}); err != nil {
				t.Fatalf("InsertAccount(%q) error = %v", test.accountID, err)
			}
			for _, transaction := range test.transactions {
				if err := database.Append(transaction); err != nil {
					t.Fatalf("Append(%q) error = %v", transaction.ID, err)
				}
			}

			var first bytes.Buffer
			if err := writeAccountCSV(&first, database, test.accountID); err != nil {
				t.Fatalf("writeAccountCSV(first) error = %v", err)
			}
			if got := first.String(); got != test.want {
				t.Errorf("writeAccountCSV() = %q, want exact document %q", got, test.want)
			}
			rows, err := csv.NewReader(strings.NewReader(first.String())).ReadAll()
			if err != nil {
				t.Fatalf("read rendered CSV error = %v", err)
			}
			if got := len(rows); got != test.wantRows {
				t.Errorf("rendered CSV rows = %d, want %d", got, test.wantRows)
			}

			var second bytes.Buffer
			if err := writeAccountCSV(&second, database, test.accountID); err != nil {
				t.Fatalf("writeAccountCSV(second) error = %v", err)
			}
			if !bytes.Equal(first.Bytes(), second.Bytes()) {
				t.Errorf("rendering the same store twice differs:\nfirst:  %q\nsecond: %q", first.Bytes(), second.Bytes())
			}
		})
	}
}

func TestWriteAccountCSVUnknownAccount(t *testing.T) {
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	const accountID = "acct-missing"
	var output bytes.Buffer
	err = writeAccountCSV(&output, database, accountID)
	if !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Errorf("writeAccountCSV(%q) error = %v, want error wrapping %v", accountID, err, ledger.ErrAccountNotFound)
	}
	if err == nil || !strings.Contains(err.Error(), accountID) {
		t.Errorf("writeAccountCSV(%q) error = %v, want message containing account ID", accountID, err)
	}
	if got := output.Len(); got != 0 {
		t.Errorf("writeAccountCSV(%q) wrote %d bytes, want 0", accountID, got)
	}
}
