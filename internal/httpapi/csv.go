package httpapi

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func renderCSV(transactions []ledger.Transaction) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	_ = writer.Write([]string{"id", "amount_cents", "description", "created_at"})
	for _, transaction := range transactions {
		_ = writer.Write([]string{
			transaction.ID,
			strconv.FormatInt(transaction.Amount, 10),
			transaction.Description,
			transaction.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writer.Flush()

	return append([]byte(nil), buffer.Bytes()...)
}
