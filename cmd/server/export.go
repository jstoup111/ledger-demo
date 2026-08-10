package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

func export(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("exactly one account id is expected")
	}

	dbPath := env("LEDGER_DB_PATH", defaultDBPath)
	database, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database %q: %w", dbPath, err)
	}
	defer database.Close()

	return writeAccountCSV(stdout, database, args[0])
}

func writeAccountCSV(w io.Writer, store ledger.Store, accountID string) error {
	transactions, err := store.Transactions(accountID)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	if err := csvWriter.Write([]string{"id", "amount_cents", "description", "created_at"}); err != nil {
		return err
	}
	for _, transaction := range transactions {
		if err := csvWriter.Write([]string{
			transaction.ID,
			strconv.FormatInt(transaction.Amount, 10),
			transaction.Description,
			transaction.CreatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
