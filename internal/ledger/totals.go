package ledger

import "fmt"

// Totals summarizes a transaction log in int64 cents.
type Totals struct {
	Deposits    int64 // sum of positive amounts, cents
	Withdrawals int64 // sum of negative amounts, cents (zero or negative)
	Net         int64 // Deposits + Withdrawals, cents
}

// TotalsOf folds transactions into deposit, withdrawal, and net totals.
func TotalsOf(transactions []Transaction) (Totals, error) {
	var totals Totals
	var err error
	for _, transaction := range transactions {
		switch {
		case transaction.Amount > 0:
			totals.Deposits, err = checkedAdd(totals.Deposits, transaction.Amount)
		case transaction.Amount < 0:
			totals.Withdrawals, err = checkedAdd(totals.Withdrawals, transaction.Amount)
		}
		if err != nil {
			return Totals{}, fmt.Errorf("derive totals: %w", err)
		}
	}

	totals.Net, err = checkedAdd(totals.Deposits, totals.Withdrawals)
	if err != nil {
		return Totals{}, fmt.Errorf("derive totals: %w", err)
	}
	return totals, nil
}
