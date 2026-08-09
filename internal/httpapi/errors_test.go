package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestCodeForMapsWrappedDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codedError
	}{
		{
			name: "account not found",
			err:  fmt.Errorf("lookup account: %w", ledger.ErrAccountNotFound),
			want: codedError{status: http.StatusNotFound, code: "account_not_found", message: "Account not found."},
		},
		{
			name: "amount zero",
			err:  fmt.Errorf("validate amount: %w", ledger.ErrAmountZero),
			want: codedError{status: http.StatusBadRequest, code: "amount_zero", message: "Amount must not be zero."},
		},
		{
			name: "description empty",
			err:  fmt.Errorf("validate description: %w", ledger.ErrDescriptionEmpty),
			want: codedError{status: http.StatusBadRequest, code: "description_empty", message: "Description must not be empty."},
		},
		{
			name: "description too long",
			err:  fmt.Errorf("validate description: %w", ledger.ErrDescriptionTooLong),
			want: codedError{status: http.StatusBadRequest, code: "description_too_long", message: "Description is too long."},
		},
		{
			name: "amount malformed",
			err:  fmt.Errorf("parse amount: %w", ledger.ErrAmountMalformed),
			want: codedError{status: http.StatusBadRequest, code: "amount_malformed", message: "Amount is malformed."},
		},
		{
			name: "balance would go negative",
			err:  fmt.Errorf("post transaction: %w", ledger.ErrBalanceWouldGoNegative),
			want: codedError{status: http.StatusBadRequest, code: "balance_would_go_negative", message: "Balance would go negative."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := codeFor(tt.err)
			if got != tt.want {
				t.Fatalf("codeFor(%v) = %#v, want %#v", tt.err, got, tt.want)
			}
		})
	}
}

func TestWriteJSONErrorEncodesExactErrorEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeJSONError(recorder, fmt.Errorf("validate amount: %w", ledger.ErrAmountZero))

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got, want := recorder.Body.String(), `{"error":{"code":"amount_zero","message":"Amount must not be zero."}}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
