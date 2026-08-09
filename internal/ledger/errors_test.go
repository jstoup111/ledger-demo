package ledger

import (
	"errors"
	"fmt"
	"testing"
)

func TestDomainErrorSentinelsAreDistinctAndSurviveWrapping(t *testing.T) {
	sentinels := []error{
		ErrAccountNotFound,
		ErrAmountZero,
		ErrDescriptionEmpty,
		ErrDescriptionTooLong,
		ErrAmountMalformed,
		ErrBalanceWouldGoNegative,
		ErrBalanceOverflow,
	}

	for i, sentinel := range sentinels {
		if sentinel == nil {
			t.Fatalf("sentinel %d is nil", i)
		}
		if !errors.Is(fmt.Errorf("posting transaction: %w", sentinel), sentinel) {
			t.Fatalf("sentinel %d does not survive wrapping", i)
		}
		for j, other := range sentinels {
			if i != j && errors.Is(sentinel, other) {
				t.Fatalf("sentinels %d and %d are not distinct", i, j)
			}
		}
	}
}
