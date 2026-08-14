package httpapi

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func renderCSV(transactions []ledger.Transaction) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "amount_cents", "description", "created_at"}); err != nil {
		return nil, err
	}
	for _, transaction := range transactions {
		if err := writer.Write([]string{transaction.ID, strconv.FormatInt(transaction.Amount, 10), transaction.Description, transaction.CreatedAt.UTC().Format(time.RFC3339)}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return append([]byte(nil), buffer.Bytes()...), nil
}
