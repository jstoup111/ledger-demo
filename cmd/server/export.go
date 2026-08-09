package main

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func writeAccountCSV(w io.Writer, store ledger.Store, accountID string) error {
	transactions, err := store.Transactions(accountID)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "amount_cents", "description", "created_at"}); err != nil {
		return err
	}
	for _, transaction := range transactions {
		if err := writer.Write([]string{
			transaction.ID,
			strconv.FormatInt(transaction.Amount, 10),
			transaction.Description,
			transaction.CreatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
