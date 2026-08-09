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

func TestFrozenRejectionIdentifierStatusContract(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codedError
	}{
		{
			name: "account not found",
			err:  fmt.Errorf("lookup account: %w", ledger.ErrAccountNotFound),
			want: codedError{status: http.StatusNotFound, code: "account_not_found"},
		},
		{
			name: "amount zero",
			err:  fmt.Errorf("validate amount: %w", ledger.ErrAmountZero),
			want: codedError{status: http.StatusBadRequest, code: "amount_zero"},
		},
		{
			name: "description empty",
			err:  fmt.Errorf("validate description: %w", ledger.ErrDescriptionEmpty),
			want: codedError{status: http.StatusBadRequest, code: "description_empty"},
		},
		{
			name: "description too long",
			err:  fmt.Errorf("validate description: %w", ledger.ErrDescriptionTooLong),
			want: codedError{status: http.StatusBadRequest, code: "description_too_long"},
		},
		{
			name: "amount malformed",
			err:  fmt.Errorf("parse amount: %w", ledger.ErrAmountMalformed),
			want: codedError{status: http.StatusBadRequest, code: "amount_malformed"},
		},
		{
			name: "balance would go negative",
			err:  fmt.Errorf("post transaction: %w", ledger.ErrBalanceWouldGoNegative),
			want: codedError{status: http.StatusBadRequest, code: "balance_would_go_negative"},
		},
		{
			name: "balance overflow",
			err:  fmt.Errorf("post transaction: %w", ledger.ErrBalanceOverflow),
			want: codedError{status: http.StatusBadRequest, code: "balance_overflow"},
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
	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
	}
	if got, want := response.Error.Message, messageFor("amount_zero", messageContext{}); got != want {
		t.Fatalf("message = %q, want messageFor = %q", got, want)
	}
}

func TestWriteJSONErrorScreensAmountMalformedDetails(t *testing.T) {
	for _, tt := range []struct {
		name   string
		detail string
		want   string
	}{
		{name: "plausible decimal", detail: "12.50", want: "Amount is malformed."},
		{name: "plausible whole number", detail: "500", want: "Amount is malformed."},
		{name: "plausible negative decimal", detail: "-12.50", want: "Amount is malformed."},
		{name: "plausible zero decimal", detail: "0.00", want: "Amount is malformed."},
		{name: "largest plausible decimal", detail: "92233720368547758.07", want: "Amount is malformed."},
		{name: "multiple decimal points", detail: "12.3.4", want: "Amount is malformed. Submitted: 12.3.4."},
		{name: "script-bearing text", detail: "<script>alert(1)</script>", want: "Amount is malformed. Submitted: <script>alert(1)</script>."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeJSONError(recorder, fmt.Errorf("parse amount: %w", ledger.ErrAmountMalformed), messageContext{value: tt.detail})

			var response errorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
			}
			if got := response.Error.Message; got != tt.want {
				t.Errorf("message for detail %q = %q, want %q", tt.detail, got, tt.want)
			}
		})
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
		if got, want := response.Error.Code, "internal_error"; got != want {
			t.Errorf("code = %q, want %q", got, want)
		}
		if got, want := response.Error.Message, "Unable to post transaction."; got != want {
			t.Errorf("message = %q, want %q", got, want)
		}
	}
	if got, want := first.Body.String(), second.Body.String(); got != want {
		t.Errorf("generic response bodies differ: %q and %q", got, want)
	}
}
