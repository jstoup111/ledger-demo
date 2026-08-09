package httpapi

import (
	"encoding/json"
	"errors"
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
		{
			name: "balance overflow",
			err:  fmt.Errorf("post transaction: %w", ledger.ErrBalanceOverflow),
			want: codedError{status: http.StatusBadRequest, code: "balance_overflow", message: "Balance would overflow."},
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

func TestWriteJSONErrorUsesStableGenericEnvelopeForUnmappedErrors(t *testing.T) {
	first := httptest.NewRecorder()
	second := httptest.NewRecorder()

	writeJSONError(first, errors.New("database unavailable"))
	writeJSONError(second, errors.New("different internal failure"))

	for _, recorder := range []*httptest.ResponseRecorder{first, second} {
		if got, want := recorder.Code, http.StatusInternalServerError; got != want {
			t.Errorf("status = %d, want %d", got, want)
		}
		if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		var response errorEnvelope
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Errorf("response is not JSON: %v; body = %s", err, recorder.Body.String())
		}
		if response.Error.Code == "" || response.Error.Message == "" {
			t.Errorf("generic response = %#v, want non-empty stable code and message", response)
		}
	}
	if got, want := first.Body.String(), second.Body.String(); got != want {
		t.Errorf("generic response bodies differ: %q and %q", got, want)
	}
}
