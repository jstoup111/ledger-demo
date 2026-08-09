// Package httpapi holds handlers, routing, and JSON + HTML rendering.
package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"time"

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
func NewRouter(store ledger.Store) (http.Handler, error) {
	page, err := template.ParseFS(web.FS, "index.html.tmpl")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/accounts", handleAccounts(store))
	mux.HandleFunc("GET /api/accounts/{id}/transactions", handleAccountTransactions(store))
	mux.Handle("GET /style.css", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, nil); err != nil {
			http.Error(w, "template render failed", http.StatusInternalServerError)
		}
	})

	return mux, nil
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
