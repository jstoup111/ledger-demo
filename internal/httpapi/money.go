package httpapi

import (
	"math"
	"strconv"
	"strings"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func parseAmount(amount string) (int64, error) {
	negative := strings.HasPrefix(amount, "-")
	if negative {
		amount = amount[1:]
	}
	if amount == "" {
		return 0, ledger.ErrAmountMalformed
	}

	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, ledger.ErrAmountMalformed
	}
	if !decimalDigits(parts[0]) {
		return 0, ledger.ErrAmountMalformed
	}

	dollars, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, ledger.ErrAmountMalformed
	}

	cents := uint64(0)
	if len(parts) == 2 {
		if len(parts[1]) != 2 || !decimalDigits(parts[1]) {
			return 0, ledger.ErrAmountMalformed
		}
		cents, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, ledger.ErrAmountMalformed
		}
	}
	if dollars > math.MaxUint64/100 {
		return 0, ledger.ErrAmountMalformed
	}
	magnitude := dollars * 100
	if magnitude > math.MaxUint64-cents {
		return 0, ledger.ErrAmountMalformed
	}
	magnitude += cents

	if negative {
		if magnitude > uint64(math.MaxInt64)+1 {
			return 0, ledger.ErrAmountMalformed
		}
		if magnitude == uint64(math.MaxInt64)+1 {
			return math.MinInt64, nil
		}
		return -int64(magnitude), nil
	}
	if magnitude > math.MaxInt64 {
		return 0, ledger.ErrAmountMalformed
	}
	return int64(magnitude), nil
}

func decimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
