package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestRejectionMessageNamesTheOffendingValue is the RED-phase lock for the
// single place that builds a specific, readable-from-the-back-of-the-room
// message per code. Both the JSON envelope and the page panel call this same
// function, so wording cannot drift between the two surfaces.
func TestRejectionMessageNamesTheOffendingValue(t *testing.T) {
	tests := []struct {
		name string
		code string
		ctx  rejectionContext
		want string
	}{
		{
			name: "account not found names the requested id",
			code: "account_not_found",
			ctx:  rejectionContext{accountID: "acct-nope"},
			want: `Account "acct-nope" was not found.`,
		},
		{
			name: "account not found without context falls back to a generic message",
			code: "account_not_found",
			ctx:  rejectionContext{},
			want: "Account not found.",
		},
		{
			name: "amount zero names the submitted value",
			code: "amount_zero",
			ctx:  rejectionContext{rawAmount: "0.00", hasRawAmount: true},
			want: `Amount "0.00" must not be zero.`,
		},
		{
			name: "amount zero without context falls back to a generic message",
			code: "amount_zero",
			ctx:  rejectionContext{},
			want: "Amount must not be zero.",
		},
		{
			name: "description empty has nothing to name",
			code: "description_empty",
			ctx:  rejectionContext{},
			want: "Description must not be empty.",
		},
		{
			name: "description too long states the length and the limit",
			code: "description_too_long",
			ctx:  rejectionContext{descriptionLength: 156, hasDescriptionLength: true},
			want: "Description is 156 characters; the limit is 140.",
		},
		{
			name: "description too long without a length still states the limit",
			code: "description_too_long",
			ctx:  rejectionContext{},
			want: "Description is too long; the limit is 140 characters.",
		},
		{
			name: "amount malformed names the value received",
			code: "amount_malformed",
			ctx:  rejectionContext{rawAmount: "abc", hasRawAmount: true},
			want: `Amount "abc" is not a valid money value.`,
		},
		{
			name: "amount malformed without context falls back to a generic message",
			code: "amount_malformed",
			ctx:  rejectionContext{},
			want: "Amount is not a valid money value.",
		},
		{
			name: "balance would go negative states the attempted amount and the current balance",
			code: "balance_would_go_negative",
			ctx:  rejectionContext{attempted: -5000, balance: 4000, hasBalanceContext: true},
			want: "Amount -$50.00 would take balance $40.00 below zero.",
		},
		{
			name: "balance would go negative without context falls back to a generic message",
			code: "balance_would_go_negative",
			ctx:  rejectionContext{},
			want: "Balance would go negative.",
		},
		{
			name: "balance overflow states the attempted amount and the current balance",
			code: "balance_overflow",
			ctx:  rejectionContext{attempted: 1, balance: 9223372036854775807, hasBalanceContext: true},
			want: "Amount $0.01 combined with balance $92,233,720,368,547,758.07 exceeds the maximum balance.",
		},
		{
			name: "balance overflow without context falls back to a generic message",
			code: "balance_overflow",
			ctx:  rejectionContext{},
			want: "Balance would overflow.",
		},
		{
			name: "unmapped code renders a stable generic message and never echoes the code",
			code: "not_a_real_code",
			ctx:  rejectionContext{accountID: "acct-1", rawAmount: "abc", hasRawAmount: true},
			want: "Unable to post transaction.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rejectionMessage(tt.code, tt.ctx); got != tt.want {
				t.Errorf("rejectionMessage(%q, %#v) = %q, want %q", tt.code, tt.ctx, got, tt.want)
			}
		})
	}
}

// TestWriteJSONErrorWithContextEchoesOffendingValues locks the JSON envelope's
// enrichment: the same rejectionMessage function, fed real context, so the
// API response names what was rejected.
func TestWriteJSONErrorWithContextEchoesOffendingValues(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		ctx     rejectionContext
		wantMsg string
	}{
		{
			name:    "malformed amount",
			err:     fmt.Errorf("parse amount: %w", ledger.ErrAmountMalformed),
			ctx:     rejectionContext{rawAmount: "abc", hasRawAmount: true},
			wantMsg: `Amount "abc" is not a valid money value.`,
		},
		{
			name:    "account not found",
			err:     fmt.Errorf("lookup account: %w", ledger.ErrAccountNotFound),
			ctx:     rejectionContext{accountID: "acct-nope"},
			wantMsg: `Account "acct-nope" was not found.`,
		},
		{
			name:    "description too long",
			err:     fmt.Errorf("validate description: %w", ledger.ErrDescriptionTooLong),
			ctx:     rejectionContext{descriptionLength: 200, hasDescriptionLength: true},
			wantMsg: "Description is 200 characters; the limit is 140.",
		},
		{
			name:    "balance would go negative",
			err:     fmt.Errorf("post transaction: %w", ledger.ErrBalanceWouldGoNegative),
			ctx:     rejectionContext{attempted: -1001, balance: 1000, hasBalanceContext: true},
			wantMsg: "Amount -$10.01 would take balance $10.00 below zero.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writeJSONErrorWithContext(recorder, tt.err, tt.ctx)

			var response errorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
			}
			if response.Error.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", response.Error.Message, tt.wantMsg)
			}
		})
	}
}

// TestWriteJSONErrorWithContextEscapesScriptBearingValues proves a
// script-bearing echoed value is safely JSON-encoded (never a raw '<' or
// '>' byte in the wire body — encoding/json's default HTML-safe escaping).
func TestWriteJSONErrorWithContextEscapesScriptBearingValues(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSONErrorWithContext(recorder, fmt.Errorf("parse amount: %w", ledger.ErrAmountMalformed), rejectionContext{
		rawAmount:    "<script>alert(1)</script>",
		hasRawAmount: true,
	})

	body := recorder.Body.String()
	if strings.Contains(body, "<script") {
		t.Errorf("JSON body must not contain a raw script tag; body = %s", body)
	}

	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, body)
	}
	if !strings.Contains(response.Error.Message, "<script>alert(1)</script>") {
		t.Errorf("decoded message = %q, want the raw value present once decoded", response.Error.Message)
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
