package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

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
	return s.transactions[accountID], nil
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

var _ ledger.Store = routerTestStore{}
