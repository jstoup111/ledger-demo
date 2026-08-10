package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

func TestWriteAccountCSV(t *testing.T) {
	newer := time.Date(2026, time.August, 8, 14, 30, 0, 0, time.FixedZone("fixture", -4*60*60))
	older := time.Date(2026, time.August, 7, 12, 15, 0, 0, time.UTC)
	tests := []struct {
		name         string
		accountID    string
		transactions []ledger.Transaction
		want         string
		wantRows     [][]string
	}{
		{
			name:      "positive and negative integer cents with UTC RFC3339 times",
			accountID: "acct-signed",
			transactions: []ledger.Transaction{
				{ID: "txn-0001", AccountID: "acct-signed", Amount: -6450, Description: "Supplies", CreatedAt: older},
				{ID: "txn-0002", AccountID: "acct-signed", Amount: 158579, Description: "Deposit", CreatedAt: newer},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-0002,158579,Deposit,2026-08-08T18:30:00Z\n" +
				"txn-0001,-6450,Supplies,2026-08-07T12:15:00Z\n",
			wantRows: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-0002", "158579", "Deposit", "2026-08-08T18:30:00Z"},
				{"txn-0001", "-6450", "Supplies", "2026-08-07T12:15:00Z"},
			},
		},
		{
			name:      "description containing comma and quote",
			accountID: "acct-escaped",
			transactions: []ledger.Transaction{
				{ID: "txn-0003", AccountID: "acct-escaped", Amount: 2500, Description: `Lunch, "team"`, CreatedAt: older},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-0003,2500,\"Lunch, \"\"team\"\"\",2026-08-07T12:15:00Z\n",
			wantRows: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-0003", "2500", `Lunch, "team"`, "2026-08-07T12:15:00Z"},
			},
		},
		{
			name:      "existing empty account is header only",
			accountID: "acct-empty",
			want:      "id,amount_cents,description,created_at\n",
			wantRows:  [][]string{{"id", "amount_cents", "description", "created_at"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, err := store.Open(":memory:")
			if err != nil {
				t.Fatalf("Open(:memory:) error = %v", err)
			}
			if err := database.InsertAccount(ledger.Account{ID: test.accountID, Name: "CSV fixture"}); err != nil {
				t.Fatalf("InsertAccount(%q) error = %v", test.accountID, err)
			}
			for _, transaction := range test.transactions {
				if err := database.Append(transaction); err != nil {
					t.Fatalf("Append(%q) error = %v", transaction.ID, err)
				}
			}

			var csvStore ledger.Store = database
			var buf bytes.Buffer
			if err := writeAccountCSV(&buf, csvStore, test.accountID); err != nil {
				t.Fatalf("writeAccountCSV(%q) error = %v", test.accountID, err)
			}
			if got := buf.String(); got != test.want {
				t.Fatalf("writeAccountCSV(%q) = %q, want %q", test.accountID, got, test.want)
			}

			rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
			if err != nil {
				t.Fatalf("parse writeAccountCSV(%q) output: %v", test.accountID, err)
			}
			if !reflect.DeepEqual(rows, test.wantRows) {
				t.Fatalf("writeAccountCSV(%q) rows = %#v, want %#v", test.accountID, rows, test.wantRows)
			}
			if got, want := len(rows)-1, len(test.transactions); got != want {
				t.Fatalf("writeAccountCSV(%q) data rows = %d, want transaction count %d", test.accountID, got, want)
			}
		})
	}
}

func TestWriteAccountCSVUnknownAccountReturnsStoreErrorWithoutWriting(t *testing.T) {
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}

	const accountID = "acct-missing"
	var csvStore ledger.Store = database
	var buf bytes.Buffer
	err = writeAccountCSV(&buf, csvStore, accountID)
	if !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Fatalf("writeAccountCSV(%q) error = %v, want errors.Is ErrAccountNotFound", accountID, err)
	}
	if !strings.Contains(err.Error(), accountID) {
		t.Fatalf("writeAccountCSV(%q) error = %q, want requested account id", accountID, err)
	}
	if got := buf.Len(); got != 0 {
		t.Fatalf("writeAccountCSV(%q) wrote %d bytes, want 0", accountID, got)
	}
}

func TestWriteAccountCSVIsByteDeterministic(t *testing.T) {
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}
	const accountID = "acct-repeat"
	if err := database.InsertAccount(ledger.Account{ID: accountID, Name: "Repeat fixture"}); err != nil {
		t.Fatalf("InsertAccount(%q) error = %v", accountID, err)
	}
	if err := database.Append(ledger.Transaction{ID: "txn-0001", AccountID: accountID, Amount: 158579, Description: "Deposit", CreatedAt: time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	var csvStore ledger.Store = database
	var first, second bytes.Buffer
	for _, writer := range []io.Writer{&first, &second} {
		if err := writeAccountCSV(writer, csvStore, accountID); err != nil {
			t.Fatalf("writeAccountCSV(%q) error = %v", accountID, err)
		}
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("repeated writeAccountCSV(%q) differs:\nfirst:  %q\nsecond: %q", accountID, first.Bytes(), second.Bytes())
	}
}
