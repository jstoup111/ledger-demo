// Package httpapi holds handlers, routing, and JSON + HTML rendering.
package httpapi

import (
	"encoding/json"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

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
}

type pageData struct {
	Accounts        []pageAccount
	Balance         string
	ErrorMessage    string
	AccountNotFound bool
	FormAction      string
	Transactions    []pageTransaction
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
		if len(accounts) == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := page.Execute(w, pageData{}); err != nil {
				http.Error(w, "template render failed", http.StatusInternalServerError)
			}
			return
		}

		data := pageData{}
		for _, account := range accounts {
			data.Accounts = append(data.Accounts, pageAccount{
				Name: account.Name,
				Link: "/?account=" + url.QueryEscape(account.ID),
			})
		}

		selected := accounts[0]
		if requested := r.URL.Query().Get("account"); requested != "" {
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
				data.ErrorMessage = "Account not found."
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
		transactions, err := store.Transactions(selected.ID)
		if err != nil {
			http.Error(w, "list transactions failed", http.StatusInternalServerError)
			return
		}

		data.Balance = formatDollars(balance)
		data.ErrorMessage = pageErrorMessage(r.URL.Query().Get("error"))
		data.FormAction = "/api/accounts/" + url.PathEscape(selected.ID) + "/transactions"
		data.Transactions = make([]pageTransaction, 0, len(transactions))
		for _, transaction := range transactions {
			data.Transactions = append(data.Transactions, pageTransaction{
				Amount:      formatDollars(transaction.Amount),
				Description: transaction.Description,
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, data); err != nil {
			http.Error(w, "template render failed", http.StatusInternalServerError)
		}
	}
}

func pageErrorMessage(code string) string {
	if code == "" {
		return ""
	}

	messages := map[string]string{
		"account_not_found":         "Account not found.",
		"amount_zero":               "Amount must not be zero.",
		"description_empty":         "Description must not be empty.",
		"description_too_long":      "Description is too long.",
		"amount_malformed":          "Amount is malformed.",
		"balance_would_go_negative": "Balance would go negative.",
	}
	if message, ok := messages[code]; ok {
		return message
	}
	return "Unable to post transaction."
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
			http.Error(w, "list accounts failed", http.StatusInternalServerError)
			return
		}

		accounts = append([]ledger.Account(nil), accounts...)
		sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })

		response := make([]accountResponse, 0, len(accounts))
		for _, account := range accounts {
			balance, err := ledger.Balance(store, account.ID)
			if err != nil {
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
			http.Error(w, "list transactions failed", http.StatusInternalServerError)
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
		if err != nil {
			writeJSONError(w, ledger.ErrAmountMalformed)
			return
		}

		amount, err := parseAmount(request.amount)
		if err != nil {
			writeJSONError(w, err)
			return
		}

		accountID := r.PathValue("id")
		transaction, err := ledger.PostTransaction(clock, store, accountID, amount, request.description)
		if err != nil {
			if codeFor(err).status != 0 {
				writeJSONError(w, err)
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

type transactionPostRequest struct {
	amount      string
	description string
}

func postRequest(r *http.Request) (transactionPostRequest, bool, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && mediaType == "application/json" {
		var request struct {
			Amount      string `json:"amount"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return transactionPostRequest{}, true, err
		}
		return transactionPostRequest{amount: request.Amount, description: request.Description}, true, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transactionPostRequest{}, false, err
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return transactionPostRequest{}, false, err
	}
	return transactionPostRequest{amount: values.Get("amount"), description: values.Get("description")}, false, nil
}
