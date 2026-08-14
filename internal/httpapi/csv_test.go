package httpapi

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderCSV(t *testing.T) {
	createdAt := time.Date(2026, time.August, 14, 9, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		wantRows     [][]string
	}{
		{
			name: "preserves input order and formats raw transaction values",
			transactions: []ledger.Transaction{
				{
					ID:          "txn-newest",
					Amount:      -125,
					Description: "coffee, \"morning\"\nwith a friend",
					CreatedAt:   createdAt,
				},
				{
					ID:          "txn-older",
					Amount:      2500,
					Description: "initial deposit",
					CreatedAt:   time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC),
				},
			},
			wantRows: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-newest", "-125", "coffee, \"morning\"\nwith a friend", "2026-08-14T13:30:00Z"},
				{"txn-older", "2500", "initial deposit", "2026-08-13T12:00:00Z"},
			},
		},
		{
			name:     "empty input is header only",
			wantRows: [][]string{{"id", "amount_cents", "description", "created_at"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderCSV(tt.transactions)
			if err != nil {
				t.Fatalf("renderCSV() error = %v", err)
			}
			if !bytes.HasPrefix(got, []byte("id,amount_cents,description,created_at\n")) {
				t.Fatalf("renderCSV() header = %q, want exact header %q", firstLine(got), "id,amount_cents,description,created_at")
			}

			rows, err := csv.NewReader(bytes.NewReader(got)).ReadAll()
			if err != nil {
				t.Fatalf("renderCSV() produced invalid CSV: %v\nbody: %q", err, got)
			}
			if len(rows) != len(tt.wantRows) {
				t.Fatalf("renderCSV() row count = %d, want %d; rows = %#v", len(rows), len(tt.wantRows), rows)
			}
			for rowIndex := range tt.wantRows {
				if got, want := rows[rowIndex], tt.wantRows[rowIndex]; !sameFields(got, want) {
					t.Errorf("renderCSV() row %d = %#v, want %#v", rowIndex, got, want)
				}
				if len(rows[rowIndex]) != 4 {
					t.Errorf("renderCSV() row %d field count = %d, want 4", rowIndex, len(rows[rowIndex]))
				}
			}
			if len(tt.transactions) == 0 && string(got) != "id,amount_cents,description,created_at\n" {
				t.Errorf("renderCSV(empty) = %q, want header only", got)
			}
		})
	}
}

func TestRenderCSVIsByteIdenticalAcrossRepeatedRenders(t *testing.T) {
	transactions := []ledger.Transaction{{
		ID:          "txn-1",
		Amount:      1,
		Description: "a comma, a \"quote\", and\na line break",
		CreatedAt:   time.Date(2026, time.August, 14, 9, 30, 0, 0, time.UTC),
	}}

	first, err := renderCSV(transactions)
	if err != nil {
		t.Fatalf("first renderCSV() error = %v", err)
	}
	second, err := renderCSV(transactions)
	if err != nil {
		t.Fatalf("second renderCSV() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("repeated renderCSV() results differ: first %q, second %q", first, second)
	}
}

func firstLine(document []byte) string {
	return strings.SplitN(string(document), "\n", 2)[0]
}

func sameFields(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
