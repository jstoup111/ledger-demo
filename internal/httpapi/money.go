package httpapi

import (
	"strconv"
	"strings"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func parseAmount(amount string) (int64, error) {
	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, ledger.ErrAmountMalformed
	}

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ledger.ErrAmountMalformed
	}
	if len(parts) == 1 {
		return dollars * 100, nil
	}

	if len(parts[1]) != 2 {
		return 0, ledger.ErrAmountMalformed
	}
	cents, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || cents < 0 {
		return 0, ledger.ErrAmountMalformed
	}

	amountInCents := dollars * 100
	if strings.HasPrefix(parts[0], "-") {
		return amountInCents - cents, nil
	}
	return amountInCents + cents, nil
}
