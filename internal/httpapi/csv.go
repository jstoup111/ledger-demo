package httpapi

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func renderTransactionsCSV(transactions []ledger.Transaction) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write([]string{"id", "amount_cents", "description", "created_at"}); err != nil {
		return nil
	}

	for _, transaction := range transactions {
		if err := writer.Write([]string{
			transaction.ID,
			strconv.FormatInt(transaction.Amount, 10),
			transaction.Description,
			transaction.CreatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return nil
		}
	}

	writer.Flush()
	if writer.Error() != nil {
		return nil
	}

	return append([]byte(nil), buffer.Bytes()...)
}
