// Package httpapi holds handlers, routing, and JSON + HTML rendering.
package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/web"
)

// NewRouter builds the HTTP handler using the stdlib ServeMux method/pattern
// routing introduced in Go 1.22. No web framework.
//
// Only the HTML page and its stylesheet are wired so far. These routes are added
// during the demo, alongside the domain they expose:
//
//	GET  /api/accounts                    → accounts with derived balances
//	GET  /api/accounts/{id}/transactions  → transactions, newest first
//	POST /api/accounts/{id}/transactions  → post a transaction
//
// JSON errors return a typed code plus a human-readable message.
func NewRouter(store ledger.Store, clock clock.Clock) (http.Handler, error) {
	page, err := template.ParseFS(web.FS, "index.html.tmpl")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/accounts", handleAccounts(store))
	mux.HandleFunc("GET /api/accounts/{id}/transactions", handleAccountTransactions(store))
	mux.HandleFunc("POST /api/accounts/{id}/transactions", handlePostTransaction(store, clock))
	mux.Handle("GET /style.css", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", handlePage(page, store))

	return emptyDefaultErrors(mux), nil
}

type pageAccount struct {
	Name string
	Link string
}

type pageTransaction struct {
	Amount      string
	Description string
	RecordedAt  string
}

type pageData struct {
	Accounts           []pageAccount
	Balance            string
	SelectedAccount    string
	HasSelectedAccount bool
	ErrorMessage       string
	AccountNotFound    bool
	RequestedAccount   string
	FormAction         string
	Transactions       []pageTransaction
}

func handlePage(page *template.Template, store ledger.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := store.Accounts()
		if err != nil {
			http.Error(w, "list accounts failed", http.StatusInternalServerError)
			return
		}
		accounts = append([]ledger.Account(nil), accounts...)
		sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
		query := r.URL.Query()
		errorCode := query.Get("error")
		detail := query.Get("detail")
		requested := query.Get("account")
		data := pageData{RequestedAccount: requested}
		if len(accounts) == 0 {
			data.ErrorMessage = messageFor(errorCode, messageContext{
				value:     detail,
				accountID: requested,
			})
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := page.Execute(w, data); err != nil {
				http.Error(w, "template render failed", http.StatusInternalServerError)
			}
			return
		}

		for _, account := range accounts {
			data.Accounts = append(data.Accounts, pageAccount{
				Name: account.Name,
				Link: "/?account=" + url.QueryEscape(account.ID),
			})
		}

		selected := accounts[0]
		if requested != "" {
			found := false
			for _, account := range accounts {
				if account.ID == requested {
					selected = account
					found = true
					break
				}
			}
			if !found {
				data.AccountNotFound = true
				if errorCode == "" {
					errorCode = "account_not_found"
				}
				data.ErrorMessage = messageFor(errorCode, messageContext{
					value:     detail,
					accountID: requested,
				})
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := page.Execute(w, data); err != nil {
					http.Error(w, "template render failed", http.StatusInternalServerError)
				}
				return
			}
		}

		balance, err := ledger.Balance(store, selected.ID)
		if err != nil {
			http.Error(w, "derive balance failed", http.StatusInternalServerError)
			return
		}
		data.ErrorMessage = messageFor(errorCode, messageContext{
			value:        detail,
			balance:      balance,
			balanceKnown: true,
		})
		transactions, err := store.Transactions(selected.ID)
		if err != nil {
			http.Error(w, "list transactions failed", http.StatusInternalServerError)
			return
		}

		data.Balance = formatDollars(balance)
		data.SelectedAccount = selected.Name
		data.HasSelectedAccount = true
		data.FormAction = "/api/accounts/" + url.PathEscape(selected.ID) + "/transactions"
		data.Transactions = make([]pageTransaction, 0, len(transactions))
		for _, transaction := range transactions {
			data.Transactions = append(data.Transactions, pageTransaction{
				Amount:      formatDollars(transaction.Amount),
				Description: transaction.Description,
				RecordedAt:  transaction.CreatedAt.UTC().Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, data); err != nil {
			http.Error(w, "template render failed", http.StatusInternalServerError)
		}
	}
}

func formatDollars(cents int64) string {
	sign := ""
	magnitude := uint64(cents)
	if cents < 0 {
		sign = "-"
		magnitude = uint64(-(cents + 1)) + 1
	}

	dollars := strconv.FormatUint(magnitude/100, 10)
	for index := len(dollars) - 3; index > 0; index -= 3 {
		dollars = dollars[:index] + "," + dollars[index:]
	}
	return sign + "$" + dollars + "." + strconv.FormatUint(magnitude%100+100, 10)[1:]
}

// emptyDefaultErrors preserves ServeMux's route selection while keeping its
// default 404 and 405 responses bodyless. Matched handlers continue to write
// their own response bodies, including coded JSON errors.
func emptyDefaultErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}

		status := newStatusResponseWriter()
		handler.ServeHTTP(status, r)
		for key, values := range status.Header() {
			w.Header()[key] = values
		}
		w.WriteHeader(status.status)
	})
}

type statusResponseWriter struct {
	header http.Header
	status int
}

func newStatusResponseWriter() *statusResponseWriter {
	return &statusResponseWriter{header: make(http.Header)}
}

func (w *statusResponseWriter) Header() http.Header { return w.header }

func (w *statusResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *statusResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return len(body), nil
}

type accountResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BalanceCents int64  `json:"balance_cents"`
}

type transactionResponse struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func handleAccounts(store ledger.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := store.Accounts()
		if err != nil {
			log.Printf("list accounts: %v", err)
			http.Error(w, "list accounts failed", http.StatusInternalServerError)
			return
		}

		accounts = append([]ledger.Account(nil), accounts...)
		sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })

		response := make([]accountResponse, 0, len(accounts))
		for _, account := range accounts {
			balance, err := ledger.Balance(store, account.ID)
			if err != nil {
				log.Printf("derive balance for account %q: %v", account.ID, err)
				http.Error(w, "derive balance failed", http.StatusInternalServerError)
				return
			}
			response = append(response, accountResponse{
				ID:           account.ID,
				Name:         account.Name,
				BalanceCents: balance,
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func handleAccountTransactions(store ledger.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transactions, err := store.Transactions(r.PathValue("id"))
		if err != nil {
			if codeFor(err).status != 0 {
				writeJSONError(w, err)
				return
			}
			log.Printf("list transactions for account %q: %v", r.PathValue("id"), err)
			http.Error(w, "list transactions failed", http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("format") == "csv" {
			body, err := renderTransactionsCSV(transactions)
			if err != nil {
				log.Printf("render transactions for account %q as CSV: %v", r.PathValue("id"), err)
				http.Error(w, "list transactions failed", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="transactions.csv"`)
			_, _ = w.Write(body)
			return
		}

		response := make([]transactionResponse, 0, len(transactions))
		for _, transaction := range transactions {
			response = append(response, transactionResponse{
				ID:          transaction.ID,
				AccountID:   transaction.AccountID,
				AmountCents: transaction.Amount,
				Description: transaction.Description,
				CreatedAt:   transaction.CreatedAt.UTC().Format(time.RFC3339),
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func handlePostTransaction(store ledger.Store, clock clock.Clock) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, jsonResponse, err := postRequest(r)
		accountID := r.PathValue("id")
		if err != nil {
			writePostError(w, r, jsonResponse, accountID, ledger.ErrAmountMalformed, messageContext{accountID: accountID})
			return
		}

		amount, err := parseAmount(request.amount)
		if err != nil {
			writePostError(w, r, jsonResponse, accountID, err, postRejectionContext(request, amount, accountID, err))
			return
		}

		transaction, err := ledger.PostTransaction(clock, store, accountID, amount, request.description)
		if err != nil {
			coded := codeFor(err)
			if coded.status != 0 {
				context := postRejectionContext(request, amount, accountID, err)
				if jsonResponse && (coded.code == "balance_would_go_negative" || coded.code == "balance_overflow") {
					if balance, balanceErr := ledger.Balance(store, accountID); balanceErr == nil {
						context.balance = balance
						context.balanceKnown = true
					}
				}
				writePostError(w, r, jsonResponse, accountID, err, context)
				return
			}
			http.Error(w, "post transaction failed", http.StatusInternalServerError)
			return
		}

		if !jsonResponse {
			http.Redirect(w, r, "/?account="+url.QueryEscape(accountID), http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(transactionResponse{
			ID:          transaction.ID,
			AccountID:   transaction.AccountID,
			AmountCents: transaction.Amount,
			Description: transaction.Description,
			CreatedAt:   transaction.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func postRejectionContext(request transactionPostRequest, amount int64, accountID string, err error) messageContext {
	context := messageContext{accountID: accountID}
	switch codeFor(err).code {
	case "amount_zero", "amount_malformed":
		context.value = request.amount
	case "description_too_long":
		context.value = strconv.Itoa(utf8.RuneCountInString(request.description))
	case "balance_would_go_negative", "balance_overflow":
		context.value = strconv.FormatInt(amount, 10)
	}
	return context
}

func writePostError(w http.ResponseWriter, r *http.Request, jsonResponse bool, accountID string, err error, context messageContext) {
	if jsonResponse {
		writeJSONError(w, err, context)
		return
	}

	log.Printf("post transaction for account %q: %v", accountID, err)
	code := codeFor(err).code
	location := "/?account=" + url.QueryEscape(accountID) + "&error=" + url.QueryEscape(code)
	if detail := postRedirectDetail(code, context.value); detail != "" {
		location += "&detail=" + url.QueryEscape(detail)
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func postRedirectDetail(code, value string) string {
	switch code {
	case "amount_zero":
		if detail, ok := zeroAmountCarriedValue(value); ok {
			return detail
		}
	case "amount_malformed":
		if detail, ok := malformedAmountCarriedValue(value); ok {
			return detail
		}
	case "description_too_long":
		if detail, ok := characterCountCarriedValue(value); ok {
			return detail
		}
	case "balance_would_go_negative", "balance_overflow":
		if detail, ok := integerCentsCarriedValue(value); ok {
			return detail
		}
	}
	return ""
}

type transactionPostRequest struct {
	amount      string
	description string
}

func postRequest(r *http.Request) (transactionPostRequest, bool, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && mediaType == "application/json" {
		var fields map[string]json.RawMessage
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&fields); err != nil {
			return transactionPostRequest{}, true, err
		}
		var extra json.RawMessage
		switch err := decoder.Decode(&extra); err {
		case io.EOF:
		case nil:
			return transactionPostRequest{}, true, fmt.Errorf("request body contains multiple JSON values")
		default:
			return transactionPostRequest{}, true, err
		}

		request := transactionPostRequest{}
		if raw, ok := fields["amount"]; ok {
			if err := json.Unmarshal(raw, &request.amount); err != nil {
				return transactionPostRequest{}, true, err
			}
		}
		if raw, ok := fields["description"]; ok {
			if err := json.Unmarshal(raw, &request.description); err != nil {
				return transactionPostRequest{}, true, err
			}
		}
		return request, true, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transactionPostRequest{}, false, err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return transactionPostRequest{}, false, err
	}
	return transactionPostRequest{amount: lastFormValue(values["amount"]), description: lastFormValue(values["description"])}, false, nil
}

func lastFormValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}
