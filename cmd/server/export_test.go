package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

func TestWriteAccountCSV(t *testing.T) {
	firstRecordedAt := time.Date(2026, time.August, 9, 14, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	secondRecordedAt := time.Date(2026, time.August, 9, 14, 31, 0, 0, time.UTC)

	for _, testCase := range []struct {
		name         string
		account      ledger.Account
		transactions []ledger.Transaction
		wantCSV      string
	}{
		{
			name:    "positive and negative cents",
			account: ledger.Account{ID: "acct-cents", Name: "Cents"},
			transactions: []ledger.Transaction{
				{ID: "txn-positive", AccountID: "acct-cents", Amount: 158579, Description: "deposit", CreatedAt: firstRecordedAt},
				{ID: "txn-negative", AccountID: "acct-cents", Amount: -6450, Description: "withdrawal", CreatedAt: secondRecordedAt},
			},
			wantCSV: "id,amount_cents,description,created_at\n" +
				"txn-positive,158579,deposit,2026-08-09T18:30:00Z\n" +
				"txn-negative,-6450,withdrawal,2026-08-09T14:31:00Z\n",
		},
		{
			name:    "description with comma and quotation mark",
			account: ledger.Account{ID: "acct-quoted", Name: "Quoted"},
			transactions: []ledger.Transaction{
				{ID: "txn-quoted", AccountID: "acct-quoted", Amount: 158579, Description: "said, \"hello\"", CreatedAt: firstRecordedAt},
			},
			wantCSV: "id,amount_cents,description,created_at\n" +
				"txn-quoted,158579,\"said, \"\"hello\"\"\",2026-08-09T18:30:00Z\n",
		},
		{
			name:    "existing account without transactions",
			account: ledger.Account{ID: "acct-empty", Name: "Empty"},
			wantCSV: "id,amount_cents,description,created_at\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			database := newExportTestStore(t)
			if err := database.InsertAccount(testCase.account); err != nil {
				t.Fatalf("InsertAccount(%q) error = %v", testCase.account.ID, err)
			}
			for _, transaction := range testCase.transactions {
				if err := database.Append(transaction); err != nil {
					t.Fatalf("Append(%q) error = %v", transaction.ID, err)
				}
			}

			var buffer bytes.Buffer
			if err := writeAccountCSV(&buffer, database, testCase.account.ID); err != nil {
				t.Fatalf("writeAccountCSV(%q) error = %v", testCase.account.ID, err)
			}
			if got := buffer.String(); got != testCase.wantCSV {
				t.Errorf("writeAccountCSV(%q) = %q, want %q", testCase.account.ID, got, testCase.wantCSV)
			}
		})
	}
}

func TestWriteAccountCSVUnknownAccountLeavesBufferEmpty(t *testing.T) {
	database := newExportTestStore(t)
	var buffer bytes.Buffer

	err := writeAccountCSV(&buffer, database, "acct-nope")
	if !errors.Is(err, ledger.ErrAccountNotFound) {
		t.Fatalf("writeAccountCSV() error = %v, want error wrapping %v", err, ledger.ErrAccountNotFound)
	}
	if !strings.Contains(err.Error(), "acct-nope") {
		t.Errorf("writeAccountCSV() error = %q, want account ID %q", err, "acct-nope")
	}
	if got := buffer.Len(); got != 0 {
		t.Errorf("buffer length after unknown account = %d, want 0", got)
	}
}

func TestWriteAccountCSVReturnsFlushWriteError(t *testing.T) {
	database := newExportTestStore(t)
	account := ledger.Account{ID: "acct-write-error", Name: "Write Error"}
	if err := database.InsertAccount(account); err != nil {
		t.Fatalf("InsertAccount(%q) error = %v", account.ID, err)
	}

	want := errors.New("write failed")
	err := writeAccountCSV(failingWriter{err: want}, database, account.ID)
	if err != want {
		t.Errorf("writeAccountCSV() error = %v, want exact error %v", err, want)
	}
}

func TestWriteAccountCSVIsByteIdenticalOnRepeatedCalls(t *testing.T) {
	database := newExportTestStore(t)
	account := ledger.Account{ID: "acct-repeat", Name: "Repeat"}
	if err := database.InsertAccount(account); err != nil {
		t.Fatalf("InsertAccount(%q) error = %v", account.ID, err)
	}
	if err := database.Append(ledger.Transaction{
		ID:          "txn-repeat",
		AccountID:   account.ID,
		Amount:      -6450,
		Description: "repeatable",
		CreatedAt:   time.Date(2026, time.August, 9, 14, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Append(txn-repeat) error = %v", err)
	}

	var first, second bytes.Buffer
	if err := writeAccountCSV(&first, database, account.ID); err != nil {
		t.Fatalf("first writeAccountCSV() error = %v", err)
	}
	if err := writeAccountCSV(&second, database, account.ID); err != nil {
		t.Fatalf("second writeAccountCSV() error = %v", err)
	}
	if got, want := second.Bytes(), first.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("repeated writeAccountCSV() output = %q, want byte-identical %q", got, want)
	}
}

func newExportTestStore(t *testing.T) *store.SQLite {
	t.Helper()
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return database
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
