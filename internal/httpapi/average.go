package httpapi

import (
	"fmt"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

// averageDollars converts each amount with float64(cents) / 100 and returns the float64 mean.
func averageDollars(transactions []ledger.Transaction) (float64, bool) {
	if len(transactions) == 0 {
		return 0, false
	}
	var sum float64
	for _, transaction := range transactions {
		cents := transaction.Amount
		sum += float64(cents) / 100
	}
	return sum / float64(len(transactions)), true
}

// formatAverage renders the mean with fmt.Sprintf("%.2f", avg).
func formatAverage(avg float64) string {
	return fmt.Sprintf("%.2f", avg)
}
