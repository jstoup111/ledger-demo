package ledger

import (
	"fmt"
	"math"
)

// Balance derives an account's balance by summing its transactions.
func Balance(store Store, accountID string) (int64, error) {
	transactions, err := store.Transactions(accountID)
	if err != nil {
		return 0, err
	}

	var balance int64
	for _, transaction := range transactions {
		balance, err = checkedAdd(balance, transaction.Amount)
		if err != nil {
			return 0, fmt.Errorf("derive balance: %w", err)
		}
	}

	return balance, nil
}

func checkedAdd(left, right int64) (int64, error) {
	if (right > 0 && left > math.MaxInt64-right) || (right < 0 && left < math.MinInt64-right) {
		return 0, ErrBalanceOverflow
	}
	return left + right, nil
}
