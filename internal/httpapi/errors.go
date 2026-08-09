package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

type codedError struct {
	status  int
	code    string
	message string
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func codeFor(err error) codedError {
	switch {
	case errors.Is(err, ledger.ErrAccountNotFound):
		return codedError{http.StatusNotFound, "account_not_found", "Account not found."}
	case errors.Is(err, ledger.ErrAmountZero):
		return codedError{http.StatusBadRequest, "amount_zero", "Amount must not be zero."}
	case errors.Is(err, ledger.ErrDescriptionEmpty):
		return codedError{http.StatusBadRequest, "description_empty", "Description must not be empty."}
	case errors.Is(err, ledger.ErrDescriptionTooLong):
		return codedError{http.StatusBadRequest, "description_too_long", "Description is too long."}
	case errors.Is(err, ledger.ErrAmountMalformed):
		return codedError{http.StatusBadRequest, "amount_malformed", "Amount is malformed."}
	case errors.Is(err, ledger.ErrBalanceWouldGoNegative):
		return codedError{http.StatusBadRequest, "balance_would_go_negative", "Balance would go negative."}
	case errors.Is(err, ledger.ErrBalanceOverflow):
		return codedError{http.StatusBadRequest, "balance_overflow", "Balance would overflow."}
	default:
		return codedError{}
	}
}

// rejectionContext carries the value that triggered a rejection so the
// message naming it can be built once and reused by both the JSON error
// envelope and the page's rejection panel (FR-13). Fields are grouped by
// which rule they apply to; a "has" flag distinguishes "no value known" (a
// direct navigation with only a code) from "known and zero/empty".
type rejectionContext struct {
	accountID string

	rawAmount    string
	hasRawAmount bool

	descriptionLength    int
	hasDescriptionLength bool

	attempted         int64
	balance           int64
	hasBalanceContext bool
}

// rejectionMessage is the single place that turns a stable code plus
// whatever context is available into a message specific enough to read from
// the back of the room. Codes are the stable contract (see codeFor); this
// function only ever changes wording, never the code. When context is
// unavailable — e.g. a direct navigation to /?error=<code> with no other
// query parameters — it falls back to a generic, still-readable message
// rather than an empty or misleading one.
func rejectionMessage(code string, ctx rejectionContext) string {
	switch code {
	case "account_not_found":
		if ctx.accountID == "" {
			return "Account not found."
		}
		return fmt.Sprintf("Account %q was not found.", ctx.accountID)
	case "amount_zero":
		if ctx.hasRawAmount {
			return fmt.Sprintf("Amount %q must not be zero.", ctx.rawAmount)
		}
		return "Amount must not be zero."
	case "description_empty":
		return "Description must not be empty."
	case "description_too_long":
		if ctx.hasDescriptionLength {
			return fmt.Sprintf("Description is %d characters; the limit is %d.", ctx.descriptionLength, ledger.MaxDescriptionLength)
		}
		return fmt.Sprintf("Description is too long; the limit is %d characters.", ledger.MaxDescriptionLength)
	case "amount_malformed":
		if ctx.hasRawAmount {
			return fmt.Sprintf("Amount %q is not a valid money value.", ctx.rawAmount)
		}
		return "Amount is not a valid money value."
	case "balance_would_go_negative":
		if ctx.hasBalanceContext {
			return fmt.Sprintf("Amount %s would take balance %s below zero.", formatDollars(ctx.attempted), formatDollars(ctx.balance))
		}
		return "Balance would go negative."
	case "balance_overflow":
		if ctx.hasBalanceContext {
			return fmt.Sprintf("Amount %s combined with balance %s exceeds the maximum balance.", formatDollars(ctx.attempted), formatDollars(ctx.balance))
		}
		return "Balance would overflow."
	default:
		return "Unable to post transaction."
	}
}

// writeJSONErrorWithContext writes the same stable {code, message} envelope
// as writeJSONError, but lets the message name the offending value when the
// caller has one on hand.
func writeJSONErrorWithContext(w http.ResponseWriter, err error, ctx rejectionContext) {
	coded := codeFor(err)
	if coded.status == 0 {
		coded = codedError{http.StatusInternalServerError, "internal_error", "Unable to post transaction."}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(coded.status)

	response := errorEnvelope{}
	response.Error.Code = coded.code
	response.Error.Message = rejectionMessage(coded.code, ctx)
	body, _ := json.Marshal(response)
	_, _ = w.Write(body)
}

func writeJSONError(w http.ResponseWriter, err error) {
	writeJSONErrorWithContext(w, err, rejectionContext{})
}
