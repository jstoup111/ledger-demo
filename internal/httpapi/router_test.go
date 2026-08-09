package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

type routerTestStore struct {
	accounts     []ledger.Account
	transactions map[string][]ledger.Transaction
}

func (s routerTestStore) Accounts() ([]ledger.Account, error) {
	return s.accounts, nil
}

func (s routerTestStore) Account(id string) (ledger.Account, error) {
	for _, account := range s.accounts {
		if account.ID == id {
			return account, nil
		}
	}
	return ledger.Account{}, ledger.ErrAccountNotFound
}

func (s routerTestStore) Transactions(accountID string) ([]ledger.Transaction, error) {
	if _, err := s.Account(accountID); err != nil {
		return nil, err
	}
	transactions := s.transactions[accountID]
	if transactions == nil {
		return []ledger.Transaction{}, nil
	}
	return transactions, nil
}

func (routerTestStore) CountTransactions() (int, error) { return 0, nil }

func (routerTestStore) Append(ledger.Transaction) error { return nil }

// Table-driven, stdlib testing only. This is the convention the whole suite
// follows: a case per behavior, including a negative case for every rule.
func TestRouter(t *testing.T) {
	router, err := NewRouter(routerTestStore{})
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantBodyHas string
	}{
		{
			name:        "GET / renders the page",
			method:      http.MethodGet,
			path:        "/",
			wantStatus:  http.StatusOK,
			wantBodyHas: "ledger-demo",
		},
		{
			name:        "GET /style.css serves the stylesheet",
			method:      http.MethodGet,
			path:        "/style.css",
			wantStatus:  http.StatusOK,
			wantBodyHas: "body",
		},
		{
			name:       "POST / is rejected — the page is read-only",
			method:     http.MethodPost,
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown path is not found",
			method:     http.MethodGet,
			path:       "/nope",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBodyHas != "" && !strings.Contains(rec.Body.String(), tt.wantBodyHas) {
				t.Errorf("body does not contain %q; got:\n%s", tt.wantBodyHas, rec.Body.String())
			}
		})
	}
}

func TestRouterRejectsWrongMethodsAndUnknownPathsWithoutABody(t *testing.T) {
	store := routerTestStore{
		accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {{ID: "txn-0001", AccountID: "acct-1", Amount: 100}},
		},
	}
	router, err := NewRouter(store)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	transactionsBefore := len(store.transactions["acct-1"])
	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodPost, path: "/api/accounts", status: http.StatusMethodNotAllowed},
		{method: http.MethodPut, path: "/api/accounts/acct-1/transactions", status: http.StatusMethodNotAllowed},
		{method: http.MethodPatch, path: "/api/accounts/acct-1/transactions", status: http.StatusMethodNotAllowed},
		{method: http.MethodDelete, path: "/api/accounts/acct-1/transactions", status: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/nope", status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if got, want := rec.Code, tt.status; got != want {
				t.Errorf("status = %d, want %d", got, want)
			}
			if got := rec.Body.String(); got != "" {
				t.Errorf("body = %q, want empty", got)
			}
		})
	}

	if got := len(store.transactions["acct-1"]); got != transactionsBefore {
		t.Errorf("stored transaction count = %d, want unchanged %d", got, transactionsBefore)
	}
}

func TestRouterServesAccountsWithDerivedBalances(t *testing.T) {
	store := routerTestStore{
		accounts: []ledger.Account{
			{ID: "acct-2", Name: "Savings"},
			{ID: "acct-1", Name: "Checking"},
		},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {
				{Amount: 128350},
				{Amount: -4250},
			},
			"acct-2": {
				{Amount: 250000},
				{Amount: -50000},
			},
		},
	}
	router, err := NewRouter(store)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	var got []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		BalanceCents int64  `json:"balance_cents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
	}
	want := []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		BalanceCents int64  `json:"balance_cents"`
	}{
		{ID: "acct-1", Name: "Checking", BalanceCents: 124100},
		{ID: "acct-2", Name: "Savings", BalanceCents: 200000},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("accounts = %#v, want %#v", got, want)
	}
}

func TestRouterServesAccountTransactions(t *testing.T) {
	createdEarlier := time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)
	createdLater := createdEarlier.Add(time.Minute)
	store := routerTestStore{
		accounts: []ledger.Account{
			{ID: "acct-1", Name: "Checking"},
			{ID: "acct-empty", Name: "Empty"},
		},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {
				{ID: "txn-0002", AccountID: "acct-1", Amount: -4250, Description: "Groceries", CreatedAt: createdLater},
				{ID: "txn-0001", AccountID: "acct-1", Amount: 128350, Description: "Deposit", CreatedAt: createdEarlier},
			},
		},
	}
	router, err := NewRouter(store)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	t.Run("existing account returns newest-first transactions as JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts/acct-1/transactions", nil))

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status = %d, want %d", got, want)
		}
		if got, want := rec.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}

		var got []struct {
			ID          string `json:"id"`
			AccountID   string `json:"account_id"`
			AmountCents int    `json:"amount_cents"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
		}
		want := []struct {
			ID          string `json:"id"`
			AccountID   string `json:"account_id"`
			AmountCents int    `json:"amount_cents"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		}{
			{ID: "txn-0002", AccountID: "acct-1", AmountCents: -4250, Description: "Groceries", CreatedAt: createdLater.Format(time.RFC3339)},
			{ID: "txn-0001", AccountID: "acct-1", AmountCents: 128350, Description: "Deposit", CreatedAt: createdEarlier.Format(time.RFC3339)},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("transactions = %#v, want %#v", got, want)
		}
		for _, transaction := range got {
			createdAt, err := time.Parse(time.RFC3339, transaction.CreatedAt)
			if err != nil {
				t.Errorf("created_at = %q, want RFC3339 UTC: %v", transaction.CreatedAt, err)
				continue
			}
			if got, want := createdAt.Location(), time.UTC; got != want {
				t.Errorf("created_at location = %v, want %v", got, want)
			}
		}
	})

	t.Run("existing account without transactions returns literal empty array", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts/acct-empty/transactions", nil))

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status = %d, want %d", got, want)
		}
		if got, want := rec.Body.String(), "[]\n"; got != want {
			t.Errorf("body = %q, want %q", got, want)
		}
	})

	t.Run("unknown account returns coded JSON not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts/acct-nope/transactions", nil))

		if got, want := rec.Code, http.StatusNotFound; got != want {
			t.Fatalf("status = %d, want %d", got, want)
		}
		var response errorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
		}
		if got, want := response.Error.Code, "account_not_found"; got != want {
			t.Errorf("error code = %q, want %q", got, want)
		}
	})
}

var _ ledger.Store = routerTestStore{}
