package httpapi

import (
	"encoding/json"
	"errors"
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

func writeJSONError(w http.ResponseWriter, err error) {
	coded := codeFor(err)
	if coded.status == 0 {
		coded = codedError{http.StatusInternalServerError, "internal_error", "Unable to post transaction."}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(coded.status)

	response := errorEnvelope{}
	response.Error.Code = coded.code
	response.Error.Message = coded.message
	body, _ := json.Marshal(response)
	_, _ = w.Write(body)
}
