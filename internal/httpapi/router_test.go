package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/ledger"
)

var routerClock = clock.FixedClock{T: time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)}

type routerTestStore struct {
	accounts     []ledger.Account
	transactions map[string][]ledger.Transaction
}

type failingRouterStore struct {
	ledger.Store
	accountsErr     error
	transactionsErr error
}

type balanceReadFailsAfterPostingStore struct {
	ledger.Store
	transactionsCalls int
	err               error
}

func (s *balanceReadFailsAfterPostingStore) Transactions(accountID string) ([]ledger.Transaction, error) {
	s.transactionsCalls++
	if s.transactionsCalls > 1 {
		return nil, s.err
	}
	return s.Store.Transactions(accountID)
}

type transactionCountingRouterStore struct {
	ledger.Store
	transactionsCalls int
}

func (s *transactionCountingRouterStore) Transactions(accountID string) ([]ledger.Transaction, error) {
	s.transactionsCalls++
	return s.Store.Transactions(accountID)
}

func (s failingRouterStore) Accounts() ([]ledger.Account, error) {
	if s.accountsErr != nil {
		return nil, s.accountsErr
	}
	return s.Store.Accounts()
}

func (s failingRouterStore) Transactions(accountID string) ([]ledger.Transaction, error) {
	if s.transactionsErr != nil {
		return nil, s.transactionsErr
	}
	return s.Store.Transactions(accountID)
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

func (s *routerTestStore) CountTransactions() (int, error) {
	count := 0
	for _, transactions := range s.transactions {
		count += len(transactions)
	}
	return count, nil
}

func (s *routerTestStore) Append(transaction ledger.Transaction) error {
	if s.transactions == nil {
		s.transactions = make(map[string][]ledger.Transaction)
	}
	s.transactions[transaction.AccountID] = append(s.transactions[transaction.AccountID], transaction)
	return nil
}

// Table-driven, stdlib testing only. This is the convention the whole suite
// follows: a case per behavior, including a negative case for every rule.
func TestRouter(t *testing.T) {
	router, err := NewRouter(&routerTestStore{}, routerClock)
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

func TestRouterRendersPageWithoutOutboundReferences(t *testing.T) {
	router, err := NewRouter(&routerTestStore{}, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := strings.ToLower(rec.Body.String())

	for _, forbidden := range []string{"http://", "https://", "<script"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("rendered page must not contain %q; body = %s", forbidden, body)
		}
	}
}

func TestRouterRendersAccountPageMarkup(t *testing.T) {
	createdEarlier := time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)
	store := routerTestStore{
		accounts: []ledger.Account{
			{ID: "acct-2", Name: "Savings"},
			{ID: "acct-3", Name: "Vacation"},
			{ID: "acct-1", Name: "Checking"},
		},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {
				{ID: "txn-0001", AccountID: "acct-1", Amount: 100000, Description: "Paycheck", CreatedAt: createdEarlier},
				{ID: "txn-0002", AccountID: "acct-1", Amount: 28350, Description: "Groceries", CreatedAt: createdEarlier.Add(time.Minute)},
			},
		},
	}
	router, err := NewRouter(&store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	t.Run("default selection renders the first account by ID in styleguide layout order", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		body := rec.Body.String()

		if got, want := rec.Code, http.StatusOK; got != want {
			t.Fatalf("status = %d, want %d", got, want)
		}
		if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		for _, link := range []string{
			`<a href="/?account=acct-1">Checking</a>`,
			`<a href="/?account=acct-2">Savings</a>`,
			`<a href="/?account=acct-3">Vacation</a>`,
		} {
			if !strings.Contains(body, link) {
				t.Errorf("page does not contain account link %q; body = %s", link, body)
			}
		}
		for _, markup := range []string{
			`class="balance">$1,283.50`,
			`class="selected-account">Selected account: Checking`,
			`<form method="post" action="/api/accounts/acct-1/transactions">`,
		} {
			if !strings.Contains(body, markup) {
				t.Errorf("page does not contain %q; body = %s", markup, body)
			}
		}
		if strings.Contains(strings.ToLower(body), "<script") {
			t.Errorf("page contains a script tag; body = %s", body)
		}

		positions := []int{
			strings.Index(body, "<h1>"),
			strings.Index(body, `href="/?account=acct-1"`),
			strings.Index(body, `class="balance"`),
			strings.Index(body, "<form"),
			strings.Index(body, "Groceries"),
		}
		for index, position := range positions {
			if position < 0 {
				t.Fatalf("layout part %d is missing; body = %s", index, body)
			}
			if index > 0 && positions[index-1] >= position {
				t.Errorf("layout positions = %v, want heading, selector, balance, form, transaction list", positions)
			}
		}
	})

	t.Run("selected account page matches the JSON transaction order", func(t *testing.T) {
		jsonRec := httptest.NewRecorder()
		router.ServeHTTP(jsonRec, httptest.NewRequest(http.MethodGet, "/api/accounts/acct-1/transactions", nil))
		if got, want := jsonRec.Code, http.StatusOK; got != want {
			t.Fatalf("JSON status = %d, want %d; body = %s", got, want, jsonRec.Body.String())
		}
		var jsonTransactions []transactionResponse
		if err := json.Unmarshal(jsonRec.Body.Bytes(), &jsonTransactions); err != nil {
			t.Fatalf("JSON transactions cannot be decoded: %v; body = %s", err, jsonRec.Body.String())
		}
		if got, want := len(jsonTransactions), 2; got != want {
			t.Fatalf("JSON transaction count = %d, want %d", got, want)
		}

		pageRec := httptest.NewRecorder()
		router.ServeHTTP(pageRec, httptest.NewRequest(http.MethodGet, "/?account=acct-1", nil))
		body := pageRec.Body.String()
		for _, markup := range []string{
			`class="balance">$1,283.50`,
			`<form method="post" action="/api/accounts/acct-1/transactions">`,
			`<th>Recorded</th>`,
		} {
			if !strings.Contains(body, markup) {
				t.Errorf("selected account page does not contain %q; body = %s", markup, body)
			}
		}
		expectedAmounts := map[string]string{
			"Paycheck":  "$1,000.00",
			"Groceries": "$283.50",
		}
		for index, transaction := range jsonTransactions {
			position := strings.Index(body, transaction.Description)
			if position < 0 {
				t.Errorf("selected account page does not render JSON transaction %q; body = %s", transaction.Description, body)
				continue
			}
			if index > 0 {
				previous := strings.Index(body, jsonTransactions[index-1].Description)
				if previous >= position {
					t.Errorf("page transaction order differs from JSON order %q then %q; body = %s", jsonTransactions[index-1].Description, transaction.Description, body)
				}
			}
			createdAt, err := time.Parse(time.RFC3339, transaction.CreatedAt)
			if err != nil || createdAt.Location() != time.UTC || !strings.HasSuffix(transaction.CreatedAt, "Z") {
				t.Errorf("JSON transaction created_at = %q, want UTC RFC3339", transaction.CreatedAt)
			}
			if !strings.Contains(body, "<td>"+transaction.CreatedAt+"</td>") {
				t.Errorf("selected account page does not render JSON transaction timestamp %q; body = %s", transaction.CreatedAt, body)
			}
			expectedAmount, ok := expectedAmounts[transaction.Description]
			if !ok {
				t.Fatalf("missing independently authored amount for transaction %q", transaction.Description)
			}
			row := "<tr><td>" + transaction.Description + "</td><td>" + expectedAmount + "</td><td>" + transaction.CreatedAt + "</td></tr>"
			if !strings.Contains(body, row) {
				t.Errorf("selected account page does not render formatted transaction row %q; body = %s", row, body)
			}
		}
	})

	t.Run("empty account has a zero balance and explicit transaction empty state", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-3", nil))
		body := rec.Body.String()

		if !strings.Contains(body, `class="balance">$0.00`) {
			t.Errorf("empty account balance is missing; body = %s", body)
		}
		if !strings.Contains(strings.ToLower(body), "no transactions") {
			t.Errorf("empty account does not show an explicit transaction empty state; body = %s", body)
		}
	})
}

func TestRouterRendersPageErrorStates(t *testing.T) {
	store := routerTestStore{
		accounts: []ledger.Account{
			{ID: "acct-1", Name: "Checking"},
			{ID: "acct-2", Name: "Savings"},
		},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {{ID: "txn-0001", AccountID: "acct-1", Amount: 10000, Description: "Opening balance"}},
		},
	}
	router, err := NewRouter(&store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	t.Run("known error code renders its matching message inside the panel above the form", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error=description_empty", nil))
		body := rec.Body.String()

		errorPosition, panel := pageErrorPanel(t, body)
		if !strings.Contains(panel, "Description must not be empty.") {
			t.Errorf("error panel = %q, want matching description error message; body = %s", panel, body)
		}
		balancePosition := strings.Index(body, `class="balance"`)
		panelEndOffset := strings.Index(body[errorPosition:], "</p>")
		formPosition := strings.Index(body, "<form")
		if balancePosition < 0 || balancePosition >= errorPosition || panelEndOffset < 0 || formPosition < 0 || strings.TrimSpace(body[errorPosition+panelEndOffset+len("</p>"):formPosition]) != "" {
			t.Errorf("error panel must follow the balance and immediately precede the form; body = %s", body)
		}
	})

	t.Run("specified error codes render their matching messages", func(t *testing.T) {
		for _, tt := range []struct {
			code    string
			message string
		}{
			{code: "account_not_found", message: "Account not found."},
			{code: "amount_zero", message: "Amount must not be zero."},
			{code: "description_too_long", message: "Description is too long."},
			{code: "amount_malformed", message: "Amount is malformed."},
			{code: "balance_would_go_negative", message: "Balance would go negative."},
		} {
			t.Run(tt.code, func(t *testing.T) {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error="+tt.code, nil))
				_, panel := pageErrorPanel(t, rec.Body.String())
				if !strings.Contains(panel, tt.message) {
					t.Errorf("error=%q panel = %q, want %q", tt.code, panel, tt.message)
				}
			})
		}
	})

	t.Run("unknown error codes render the same non-empty generic panel without echoing the code", func(t *testing.T) {
		const firstUnknownCode = "not_a_real_code"
		const secondUnknownCode = "another_unknown_code"

		panels := make([]string, 0, 2)
		for _, code := range []string{firstUnknownCode, secondUnknownCode} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error="+code, nil))
			body := rec.Body.String()
			_, panel := pageErrorPanel(t, body)
			if strings.TrimSpace(panel) == "" {
				t.Errorf("error=%q rendered an empty generic error panel; body = %s", code, body)
			}
			if strings.Contains(body, code) {
				t.Errorf("error=%q was echoed in the page; body = %s", code, body)
			}
			panels = append(panels, panel)
		}
		if panels[0] != panels[1] {
			t.Errorf("unknown error panels = %q and %q, want one static generic panel", panels[0], panels[1])
		}
	})

	t.Run("script-like error code is escaped", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error=%3Cscript%3Ealert%281%29%3C%2Fscript%3E", nil))
		body := rec.Body.String()

		if strings.Contains(strings.ToLower(body), "<script") {
			t.Errorf("script-like error code must not render a raw script tag; body = %s", body)
		}
	})

	t.Run("unknown account shows only selector and not-found message", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-nope", nil))
		body := rec.Body.String()

		for _, link := range []string{`href="/?account=acct-1"`, `href="/?account=acct-2"`} {
			if !strings.Contains(body, link) {
				t.Errorf("unknown account page does not contain account link %q; body = %s", link, body)
			}
		}
		if !strings.Contains(body, "Account not found.") || strings.Contains(body, `class="balance"`) || strings.Contains(body, "<form") || strings.Contains(body, `aria-label="Transactions"`) {
			t.Errorf("unknown account page must show the selector and not-found message only; body = %s", body)
		}
	})

	t.Run("script-like unknown account is rendered escaped", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil))
		body := rec.Body.String()

		if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
			t.Errorf("unknown account value is not rendered escaped; body = %s", body)
		}
		if strings.Contains(strings.ToLower(body), "<script") {
			t.Errorf("unknown account value must not render a raw script tag; body = %s", body)
		}
	})

	t.Run("unknown account preserves a specific requested error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-nope&error=amount_malformed", nil))
		body := rec.Body.String()

		_, panel := pageErrorPanel(t, body)
		if !strings.Contains(panel, "Amount is malformed.") {
			t.Errorf("error panel = %q, want requested error message", panel)
		}
	})

	t.Run("zero-account page preserves its requested error and omits the posting form", func(t *testing.T) {
		emptyRouter, err := NewRouter(&routerTestStore{}, routerClock)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}

		rec := httptest.NewRecorder()
		emptyRouter.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?error=amount_malformed", nil))
		body := rec.Body.String()

		_, panel := pageErrorPanel(t, body)
		if !strings.Contains(panel, "Amount is malformed.") {
			t.Errorf("error panel = %q, want requested error message", panel)
		}
		if strings.Contains(body, "<form") || strings.Contains(body, `action=""`) {
			t.Errorf("zero-account page must not render a posting form; body = %s", body)
		}
	})
}

func TestRouterComposesDetailedPageRejectionMessages(t *testing.T) {
	store := routerTestStore{
		accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {{ID: "txn-0001", AccountID: "acct-1", Amount: 10000, Description: "Opening balance"}},
		},
	}
	router, err := NewRouter(&store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	for _, tt := range []struct {
		name string
		path string
		want string
	}{
		{
			name: "zero amount carries a zero submitted value",
			path: "/?account=acct-1&error=amount_zero&detail=0.00",
			want: "Amount must not be zero. Submitted: 0.00.",
		},
		{
			name: "zero amount carries a negative-zero submitted value",
			path: "/?account=acct-1&error=amount_zero&detail=-0.00",
			want: "Amount must not be zero. Submitted: -0.00.",
		},
		{
			name: "malformed amount carries the submitted value",
			path: "/?account=acct-1&error=amount_malformed&detail=12.3.4",
			want: "Amount is malformed. Submitted: 12.3.4.",
		},
		{
			name: "description count carries the submitted count",
			path: "/?account=acct-1&error=description_too_long&detail=141",
			want: "Description is too long. Submitted: 141 characters; the limit is 140.",
		},
		{
			name: "negative balance rejection uses the derived balance",
			path: "/?account=acct-1&error=balance_would_go_negative&detail=-20000",
			want: "Balance would go negative. Posting -$200.00 against a balance of $100.00.",
		},
		{
			name: "overflow balance rejection uses the derived balance",
			path: "/?account=acct-1&error=balance_overflow&detail=9223372036854775807",
			want: "Balance would overflow. Posting $92,233,720,368,547,758.07 against a balance of $100.00.",
		},
		{
			name: "unknown account identifies the requested account",
			path: "/?account=acct-nope",
			want: "Account not found. Requested: acct-nope.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			body := rec.Body.String()

			panelPosition, panel := pageErrorPanel(t, body)
			if got := strings.TrimSpace(panel); got != tt.want {
				t.Errorf("error panel = %q, want %q", got, tt.want)
			}
			if strings.Contains(tt.path, "account=acct-1") {
				balancePosition := strings.Index(body, `class="balance"`)
				panelEnd := panelPosition + strings.Index(body[panelPosition:], "</p>") + len("</p>")
				formPosition := strings.Index(body, "<form")
				if balancePosition < 0 || balancePosition >= panelPosition || formPosition < 0 || panelEnd > formPosition || strings.TrimSpace(body[panelEnd:formPosition]) != "" {
					t.Errorf("error panel must follow the balance and immediately precede the form; body = %s", body)
				}
			}
		})
	}

	t.Run("tampered rejection details degrade to plain messages", func(t *testing.T) {
		for _, tt := range []struct {
			name    string
			path    string
			want    string
			omitted string
		}{
			{
				name: "absent free-text detail",
				path: "/?account=acct-1&error=amount_malformed",
				want: "Amount is malformed.",
			},
			{
				name:    "over-long free-text detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=" + strings.Repeat("a", 33),
				want:    "Amount is malformed.",
				omitted: strings.Repeat("a", 33),
			},
			{
				name:    "control free-text detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=bad%0Avalue",
				want:    "Amount is malformed.",
				omitted: "bad\nvalue",
			},
			{
				name:    "malformed amount with a plausible decimal detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=12.50",
				want:    "Amount is malformed.",
				omitted: "12.50",
			},
			{
				name:    "malformed amount with a plausible whole number detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=500",
				want:    "Amount is malformed.",
				omitted: "500",
			},
			{
				name:    "malformed amount with a plausible negative decimal detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=-12.50",
				want:    "Amount is malformed.",
				omitted: "-12.50",
			},
			{
				name:    "malformed amount with a plausible zero decimal detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=0.00",
				want:    "Amount is malformed.",
				omitted: "0.00",
			},
			{
				name:    "malformed amount with the largest plausible decimal detail",
				path:    "/?account=acct-1&error=amount_malformed&detail=92233720368547758.07",
				want:    "Amount is malformed.",
				omitted: "92233720368547758.07",
			},
			{
				name:    "zero amount with non-zero detail",
				path:    "/?account=acct-1&error=amount_zero&detail=5.00",
				want:    "Amount must not be zero.",
				omitted: "5.00",
			},
			{
				name:    "zero amount with non-numeric detail",
				path:    "/?account=acct-1&error=amount_zero&detail=not-zero-at-all",
				want:    "Amount must not be zero.",
				omitted: "not-zero-at-all",
			},
			{
				name:    "zero amount with negative non-zero detail",
				path:    "/?account=acct-1&error=amount_zero&detail=-12.34",
				want:    "Amount must not be zero.",
				omitted: "-12.34",
			},
			{
				name:    "non-numeric description count",
				path:    "/?account=acct-1&error=description_too_long&detail=abc",
				want:    "Description is too long.",
				omitted: "abc",
			},
			{
				name:    "too-small description count",
				path:    "/?account=acct-1&error=description_too_long&detail=3",
				want:    "Description is too long.",
				omitted: "3",
			},
			{
				name:    "decimal cents detail",
				path:    "/?account=acct-1&error=balance_would_go_negative&detail=12.50",
				want:    "Balance would go negative.",
				omitted: "12.50",
			},
			{
				name:    "detail for a rule with no value",
				path:    "/?account=acct-1&error=description_empty&detail=ignored-detail",
				want:    "Description must not be empty.",
				omitted: "ignored-detail",
			},
			{
				name:    "detail for an unknown identifier",
				path:    "/?account=acct-1&error=not_a_real_code&detail=unknown-detail",
				want:    "Unable to post transaction.",
				omitted: "unknown-detail",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
				body := rec.Body.String()

				_, panel := pageErrorPanel(t, body)
				if got := strings.TrimSpace(panel); got != tt.want {
					t.Errorf("error panel = %q, want %q", got, tt.want)
				}
				if tt.omitted != "" && strings.Contains(panel, tt.omitted) {
					t.Errorf("error panel rendered tampered detail %q; panel = %s", tt.omitted, panel)
				}
			})
		}
	})

	t.Run("balance detail reachability boundaries preserve only producible messages", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			path string
			want string
		}{
			{
				name: "negative balance at the current balance is plain",
				path: "/?account=acct-1&error=balance_would_go_negative&detail=-10000",
				want: "Balance would go negative.",
			},
			{
				name: "negative balance one cent beyond the current balance is enriched",
				path: "/?account=acct-1&error=balance_would_go_negative&detail=-10001",
				want: "Balance would go negative. Posting -$100.01 against a balance of $100.00.",
			},
			{
				name: "overflow at the maximum remaining capacity is plain",
				path: "/?account=acct-1&error=balance_overflow&detail=9223372036854765807",
				want: "Balance would overflow.",
			},
			{
				name: "overflow one cent beyond the maximum remaining capacity is enriched",
				path: "/?account=acct-1&error=balance_overflow&detail=9223372036854765808",
				want: "Balance would overflow. Posting $92,233,720,368,547,658.08 against a balance of $100.00.",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

				_, panel := pageErrorPanel(t, rec.Body.String())
				if got := strings.TrimSpace(panel); got != tt.want {
					t.Errorf("error panel = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("implausible requested account IDs degrade the error panel", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			path string
		}{
			{
				name: "absent requested account ID",
				path: "/?error=account_not_found",
			},
			{
				name: "over-long requested account ID",
				path: "/?account=" + strings.Repeat("a", 33),
			},
			{
				name: "control-character-bearing requested account ID",
				path: "/?account=acct%0Anope",
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

				_, panel := pageErrorPanel(t, rec.Body.String())
				if got, want := strings.TrimSpace(panel), "Account not found."; got != want {
					t.Errorf("error panel = %q, want %q", got, want)
				}
			})
		}
	})

	t.Run("resolved account ignores crafted account-not-found error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error=account_not_found", nil))

		_, panel := pageErrorPanel(t, rec.Body.String())
		if got, want := strings.TrimSpace(panel), "Account not found."; got != want {
			t.Errorf("error panel = %q, want %q", got, want)
		}
	})

	t.Run("script-like amount detail is escaped visible text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1&error=amount_malformed&detail=%3Cscript%3Ealert%281%29%3C%2Fscript%3E", nil))
		body := rec.Body.String()

		_, panel := pageErrorPanel(t, body)
		if got, want := strings.TrimSpace(panel), "Amount is malformed. Submitted: &lt;script&gt;alert(1)&lt;/script&gt;."; got != want {
			t.Errorf("error panel = %q, want %q", got, want)
		}
		if strings.Contains(strings.ToLower(body), "<script") {
			t.Errorf("page rendered a raw script element; body = %s", body)
		}
	})

	t.Run("unknown account cannot supply balance context for a valid detail", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-nope&error=balance_would_go_negative&detail=-200000", nil))

		_, panel := pageErrorPanel(t, rec.Body.String())
		if got, want := strings.TrimSpace(panel), "Balance would go negative."; got != want {
			t.Errorf("error panel = %q, want %q", got, want)
		}
		if strings.Contains(panel, "Posting") {
			t.Errorf("unknown-account balance rejection included a Posting clause; panel = %q", panel)
		}
	})

	t.Run("no error parameter renders no panel", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?account=acct-1", nil))
		if strings.Contains(rec.Body.String(), `class="error"`) {
			t.Errorf("page without an error parameter rendered an error panel; body = %s", rec.Body.String())
		}
	})
}

func pageErrorPanel(t *testing.T, body string) (int, string) {
	t.Helper()

	classPosition := strings.Index(body, `class="error"`)
	if classPosition < 0 {
		t.Fatalf("page does not contain an error-class element; body = %s", body)
	}
	elementPosition := strings.LastIndex(body[:classPosition], "<")
	openingTagEnd := strings.Index(body[classPosition:], ">")
	if elementPosition < 0 || openingTagEnd < 0 {
		t.Fatalf("error-class element has malformed markup; body = %s", body)
	}
	contentStart := classPosition + openingTagEnd + 1
	closingTagOffset := strings.Index(body[contentStart:], "</")
	if closingTagOffset < 0 {
		t.Fatalf("error-class element is not closed; body = %s", body)
	}
	return elementPosition, body[contentStart : contentStart+closingTagOffset]
}

func TestPostRedirectDetailScreensAmountMalformedDetails(t *testing.T) {
	for _, tt := range []struct {
		name   string
		detail string
		want   string
	}{
		{name: "plausible decimal", detail: "12.50", want: ""},
		{name: "plausible whole number", detail: "500", want: ""},
		{name: "plausible negative decimal", detail: "-12.50", want: ""},
		{name: "plausible zero decimal", detail: "0.00", want: ""},
		{name: "largest plausible decimal", detail: "92233720368547758.07", want: ""},
		{name: "multiple decimal points", detail: "12.3.4", want: "12.3.4"},
		{name: "script-bearing text", detail: "<script>alert(1)</script>", want: "<script>alert(1)</script>"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := postRedirectDetail("amount_malformed", tt.detail); got != tt.want {
				t.Errorf("postRedirectDetail(amount_malformed, %q) = %q, want %q", tt.detail, got, tt.want)
			}
		})
	}
}

func TestRouterPostsTransactionsForJSONAndFormRequests(t *testing.T) {
	store := &routerTestStore{
		accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}, {ID: "acct?2", Name: "Escaped"}},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {{ID: "txn-0001", AccountID: "acct-1", Amount: 20000}},
			"acct?2": {{ID: "txn-0002", AccountID: "acct?2", Amount: 10000}},
		},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	t.Run("JSON returns the created transaction", func(t *testing.T) {
		rec := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"-42.50","description":"Coffee beans"}`)
		if got, want := rec.Code, http.StatusCreated; got != want {
			t.Fatalf("status = %d, want %d; body = %s", got, want, rec.Body.String())
		}
		if got, want := rec.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		var got transactionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
		}
		if got, want := got.AmountCents, int64(-4250); got != want {
			t.Errorf("amount_cents = %d, want %d", got, want)
		}
	})

	for _, contentType := range []string{"application/x-www-form-urlencoded", "text/plain", ""} {
		t.Run("form branch redirects for "+contentType, func(t *testing.T) {
			rec := postTransaction(router, "/api/accounts/acct-1/transactions", contentType, "amount=-42.50&description=Coffee+beans")
			if got, want := rec.Code, http.StatusSeeOther; got != want {
				t.Fatalf("status = %d, want %d; body = %s", got, want, rec.Body.String())
			}
			if got, want := rec.Header().Get("Location"), "/?account=acct-1"; got != want {
				t.Errorf("Location = %q, want %q", got, want)
			}
		})
	}

	t.Run("form redirect escapes the account ID and reloads do not repost", func(t *testing.T) {
		rec := postTransaction(router, "/api/accounts/acct%3F2/transactions", "application/x-www-form-urlencoded", "amount=1.00&description=Coffee")
		if got, want := rec.Header().Get("Location"), "/?account=acct%3F2"; got != want {
			t.Fatalf("Location = %q, want %q", got, want)
		}
		countAfterPost, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		for range 2 {
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, rec.Header().Get("Location"), nil))
		}
		countAfterReloads, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		if got, want := countAfterReloads, countAfterPost; got != want {
			t.Errorf("transactions after redirect reloads = %d, want %d", got, want)
		}
	})

	t.Run("invalid input has the same wire code for both branches without appending", func(t *testing.T) {
		countBefore, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		var codes []string
		for _, request := range []struct {
			contentType string
			body        string
		}{
			{contentType: "application/json", body: `{"amount":"0","description":"Coffee"}`},
			{contentType: "application/x-www-form-urlencoded", body: "amount=0&description=Coffee"},
		} {
			rec := postTransaction(router, "/api/accounts/acct-1/transactions", request.contentType, request.body)
			if request.contentType == "application/json" {
				var response errorEnvelope
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
				}
				codes = append(codes, response.Error.Code)
			} else {
				location, err := url.Parse(rec.Header().Get("Location"))
				if err != nil {
					t.Fatalf("Location parse error = %v", err)
				}
				codes = append(codes, location.Query().Get("error"))
			}
			countAfterRequest, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}
			if got, want := countAfterRequest, countBefore; got != want {
				t.Fatalf("transactions after rejected %s post = %d, want %d", request.contentType, got, want)
			}
		}
		if got, want := codes[1], codes[0]; got != want {
			t.Errorf("form error code = %q, want JSON error code %q", got, want)
		}
		countAfter, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		if got, want := countAfter, countBefore; got != want {
			t.Errorf("transactions after rejected posts = %d, want %d", got, want)
		}
	})
}

func TestRouterNegotiatesCodedPostErrorsByContentType(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 20000}}},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	for _, tt := range []struct {
		name       string
		path       string
		body       string
		code       string
		jsonStatus int
	}{
		{name: "malformed amount at the boundary", path: "/api/accounts/acct-1/transactions", body: "amount=bad&description=Coffee", code: "amount_malformed", jsonStatus: http.StatusBadRequest},
		{name: "empty description in the domain", path: "/api/accounts/acct-1/transactions", body: "amount=1.00&description=", code: "description_empty", jsonStatus: http.StatusBadRequest},
		{name: "unknown account in the domain", path: "/api/accounts/acct-nope/transactions", body: "amount=1.00&description=Coffee", code: "account_not_found", jsonStatus: http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			countBefore, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}
			form := postTransaction(router, tt.path, "application/x-www-form-urlencoded", tt.body)
			if got, want := form.Code, http.StatusSeeOther; got != want {
				t.Fatalf("form status = %d, want %d; body = %s", got, want, form.Body.String())
			}
			wantLocation := "/?account=" + url.QueryEscape(strings.TrimSuffix(strings.TrimPrefix(tt.path, "/api/accounts/"), "/transactions")) + "&error=" + tt.code
			if tt.code == "amount_malformed" {
				wantLocation += "&detail=bad"
			}
			if got, want := form.Header().Get("Location"), wantLocation; got != want {
				t.Errorf("form Location = %q, want %q", got, want)
			}

			jsonBody := `{"amount":"1.00","description":"Coffee"}`
			if tt.code == "amount_malformed" {
				jsonBody = `{"amount":"bad","description":"Coffee"}`
			}
			if tt.code == "description_empty" {
				jsonBody = `{"amount":"1.00","description":""}`
			}
			jsonResponse := postTransaction(router, tt.path, "application/json", jsonBody)
			var response errorEnvelope
			if err := json.Unmarshal(jsonResponse.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON response is not JSON: %v; body = %s", err, jsonResponse.Body.String())
			}
			if got, want := response.Error.Code, tt.code; got != want {
				t.Errorf("JSON error code = %q, want %q", got, want)
			}
			if got, want := jsonResponse.Code, tt.jsonStatus; got != want {
				t.Errorf("JSON status = %d, want %d", got, want)
			}
			countAfter, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}
			if got, want := countAfter, countBefore; got != want {
				t.Errorf("transactions after rejected form and JSON posts = %d, want %d", got, want)
			}
		})
	}
}

func TestRouterFormPostRejectionRedirectCarriesDetail(t *testing.T) {
	longAmount := strings.Repeat("1", 33)
	longDescription := strings.Repeat("x", 141)

	for _, tt := range []struct {
		name         string
		accountID    string
		transactions map[string][]ledger.Transaction
		body         string
		code         string
		detail       string
	}{
		{
			name:      "zero amount",
			accountID: "acct-1",
			body:      "amount=0.00&description=Coffee",
			code:      "amount_zero",
			detail:    "0.00",
		},
		{
			name:      "malformed amount",
			accountID: "acct-1",
			body:      "amount=12.3.4&description=Coffee",
			code:      "amount_malformed",
			detail:    "12.3.4",
		},
		{
			name:      "long description",
			accountID: "acct-1",
			body:      "amount=1.00&description=" + longDescription,
			code:      "description_too_long",
			detail:    "141",
		},
		{
			name:         "negative resulting balance",
			accountID:    "acct-1",
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 100}}},
			body:         "amount=-2.00&description=Coffee",
			code:         "balance_would_go_negative",
			detail:       "-200",
		},
		{
			name:         "balance overflow",
			accountID:    "acct-1",
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: math.MaxInt64}}},
			body:         "amount=0.01&description=Coffee",
			code:         "balance_overflow",
			detail:       "1",
		},
		{
			name:      "unknown account",
			accountID: "acct &",
			body:      "amount=1.00&description=Coffee",
			code:      "account_not_found",
		},
		{
			name:      "empty description",
			accountID: "acct-1",
			body:      "amount=1.00&description=",
			code:      "description_empty",
		},
		{
			name:      "overlong submitted amount is omitted rather than truncated",
			accountID: "acct &",
			body:      "amount=" + longAmount + "&description=Coffee",
			code:      "amount_malformed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &routerTestStore{
				accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
				transactions: tt.transactions,
			}
			router, err := NewRouter(store, routerClock)
			if err != nil {
				t.Fatalf("NewRouter() error = %v, want nil", err)
			}

			response := postTransaction(router, "/api/accounts/"+url.PathEscape(tt.accountID)+"/transactions", "application/x-www-form-urlencoded", tt.body)
			if got, want := response.Code, http.StatusSeeOther; got != want {
				t.Fatalf("form status = %d, want %d; body = %s", got, want, response.Body.String())
			}

			wantLocation := "/?account=" + url.QueryEscape(tt.accountID) + "&error=" + tt.code
			if tt.detail != "" {
				wantLocation += "&detail=" + url.QueryEscape(tt.detail)
			}
			if got := response.Header().Get("Location"); got != wantLocation {
				t.Errorf("form Location = %q, want %q", got, wantLocation)
			}
		})
	}
}

func TestRouterFormAndJSONRejectionsRenderTheSameMessage(t *testing.T) {
	longDescription := strings.Repeat("x", 141)

	for _, tt := range []struct {
		name         string
		accountID    string
		transactions map[string][]ledger.Transaction
		amount       string
		description  string
	}{
		{
			name:        "malformed amount",
			accountID:   "acct-1",
			amount:      "12.3.4",
			description: "Coffee",
		},
		{
			name:        "zero amount",
			accountID:   "acct-1",
			amount:      "0.00",
			description: "Coffee",
		},
		{
			name:        "description too long",
			accountID:   "acct-1",
			amount:      "1.00",
			description: longDescription,
		},
		{
			name:         "balance would go negative",
			accountID:    "acct-1",
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 100}}},
			amount:       "-2.00",
			description:  "Coffee",
		},
		{
			name:         "balance overflow",
			accountID:    "acct-1",
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: math.MaxInt64}}},
			amount:       "0.01",
			description:  "Coffee",
		},
		{
			name:        "account not found",
			accountID:   "acct-nope",
			amount:      "1.00",
			description: "Coffee",
		},
		{
			name:        "empty description",
			accountID:   "acct-1",
			amount:      "1.00",
			description: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &routerTestStore{
				accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
				transactions: tt.transactions,
			}
			router, err := NewRouter(store, routerClock)
			if err != nil {
				t.Fatalf("NewRouter() error = %v, want nil", err)
			}

			path := "/api/accounts/" + url.PathEscape(tt.accountID) + "/transactions"
			form := postTransaction(router, path, "application/x-www-form-urlencoded", url.Values{
				"amount":      {tt.amount},
				"description": {tt.description},
			}.Encode())
			if got, want := form.Code, http.StatusSeeOther; got != want {
				t.Fatalf("form status = %d, want %d; body = %s", got, want, form.Body.String())
			}

			page := httptest.NewRecorder()
			router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, form.Header().Get("Location"), nil))
			_, formMessage := pageErrorPanel(t, page.Body.String())

			jsonBody, err := json.Marshal(struct {
				Amount      string `json:"amount"`
				Description string `json:"description"`
			}{Amount: tt.amount, Description: tt.description})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			jsonResponse := postTransaction(router, path, "application/json", string(jsonBody))
			if jsonResponse.Code < http.StatusBadRequest || jsonResponse.Code >= http.StatusInternalServerError {
				t.Fatalf("JSON status = %d, want a 4xx rejection; body = %s", jsonResponse.Code, jsonResponse.Body.String())
			}
			var response errorEnvelope
			if err := json.Unmarshal(jsonResponse.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON response is not JSON: %v; body = %s", err, jsonResponse.Body.String())
			}

			if got, want := strings.TrimSpace(formMessage), response.Error.Message; got != want {
				t.Errorf("form page message = %q, want JSON message %q", got, want)
			}
		})
	}
}

func TestRouterJSONPostRejectionsNameTheirContext(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	description := strings.Repeat("🙂", 187)
	for _, tt := range []struct {
		name      string
		path      string
		body      string
		status    int
		code      string
		message   string
		exactBody string
	}{
		{
			name:      "malformed amount names submitted amount",
			path:      "/api/accounts/acct-1/transactions",
			body:      `{"amount":"12.3.4","description":"Coffee"}`,
			status:    http.StatusBadRequest,
			code:      "amount_malformed",
			message:   "Amount is malformed. Submitted: 12.3.4.",
			exactBody: `{"error":{"code":"amount_malformed","message":"Amount is malformed. Submitted: 12.3.4."}}`,
		},
		{
			name:    "zero amount names submitted amount",
			path:    "/api/accounts/acct-1/transactions",
			body:    `{"amount":"0.00","description":"Coffee"}`,
			status:  http.StatusBadRequest,
			code:    "amount_zero",
			message: "Amount must not be zero. Submitted: 0.00.",
		},
		{
			name:    "long description names rune count",
			path:    "/api/accounts/acct-1/transactions",
			body:    `{"amount":"1.00","description":"` + description + `"}`,
			status:  http.StatusBadRequest,
			code:    "description_too_long",
			message: "Description is too long. Submitted: 187 characters; the limit is 140.",
		},
		{
			name:    "missing account names requested id",
			path:    "/api/accounts/acct-nope/transactions",
			body:    `{"amount":"1.00","description":"Coffee"}`,
			status:  http.StatusNotFound,
			code:    "account_not_found",
			message: "Account not found. Requested: acct-nope.",
		},
		{
			name:    "over-long missing account ID degrades to plain message",
			path:    "/api/accounts/" + strings.Repeat("a", 33) + "/transactions",
			body:    `{"amount":"1.00","description":"Coffee"}`,
			status:  http.StatusNotFound,
			code:    "account_not_found",
			message: "Account not found.",
		},
		{
			name:    "control-character-bearing missing account ID degrades to plain message",
			path:    "/api/accounts/acct%0Anope/transactions",
			body:    `{"amount":"1.00","description":"Coffee"}`,
			status:  http.StatusNotFound,
			code:    "account_not_found",
			message: "Account not found.",
		},
		{
			name:    "blank description remains plain sentence",
			path:    "/api/accounts/acct-1/transactions",
			body:    `{"amount":"1.00","description":""}`,
			status:  http.StatusBadRequest,
			code:    "description_empty",
			message: "Description must not be empty.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := postTransaction(router, tt.path, "application/json", tt.body)
			if got, want := recorder.Code, tt.status; got != want {
				t.Fatalf("status = %d, want %d; body = %s", got, want, recorder.Body.String())
			}
			if tt.exactBody != "" && recorder.Body.String() != tt.exactBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), tt.exactBody)
			}

			var response errorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
			}
			if got, want := response.Error.Code, tt.code; got != want {
				t.Errorf("error code = %q, want %q", got, want)
			}
			if got, want := response.Error.Message, tt.message; got != want {
				t.Errorf("error message = %q, want %q", got, want)
			}
		})
	}
}

func TestRouterJSONAmountZeroMessageDegradesTamperedDetails(t *testing.T) {
	for _, tt := range []struct {
		name   string
		detail string
		want   string
	}{
		{
			name:   "zero amount preserves a zero detail",
			detail: "0.00",
			want:   "Amount must not be zero. Submitted: 0.00.",
		},
		{
			name:   "zero amount preserves a negative-zero detail",
			detail: "-0.00",
			want:   "Amount must not be zero. Submitted: -0.00.",
		},
		{
			name:   "non-zero amount detail is plain",
			detail: "5.00",
			want:   "Amount must not be zero.",
		},
		{
			name:   "non-numeric detail is plain",
			detail: "not-zero-at-all",
			want:   "Amount must not be zero.",
		},
		{
			name:   "negative non-zero amount detail is plain",
			detail: "-12.34",
			want:   "Amount must not be zero.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/accounts/acct-1/transactions", nil)
			writePostError(recorder, request, true, "acct-1", ledger.ErrAmountZero, messageContext{value: tt.detail})

			if got, want := recorder.Code, http.StatusBadRequest; got != want {
				t.Fatalf("status = %d, want %d; body = %s", got, want, recorder.Body.String())
			}
			var response errorEnvelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
			}
			if got := response.Error.Message; got != tt.want {
				t.Errorf("error message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRouterJSONPostErrorWithAbsentAccountIDUsesPlainNotFoundMessage(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/accounts//transactions", nil)

	writePostError(recorder, request, true, "", ledger.ErrAccountNotFound, messageContext{})

	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
	}
	if got, want := response.Error.Message, "Account not found."; got != want {
		t.Errorf("error message = %q, want %q", got, want)
	}
}

func TestRouterJSONPostEscapesScriptBearingMalformedAmount(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	recorder := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"<script>alert(1)</script>","description":"Coffee"}`)
	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d; body = %s", got, want, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "<script") {
		t.Fatalf("response contains raw script tag: %s", recorder.Body.String())
	}

	var response map[string]map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
	}
	if got, want := response["error"], map[string]string{
		"code":    "amount_malformed",
		"message": "Amount is malformed. Submitted: <script>alert(1)</script>.",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", got, want)
	}
}

func TestRouterJSONPostRejectsUnparseableAndMultipleValueBodiesWithPlainError(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "unparseable", body: `{"amount":"1.00"`},
		{name: "two JSON values", body: `{"amount":"1.00","description":"Coffee"} {"amount":"2.00"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", tt.body)
			if got, want := recorder.Code, http.StatusBadRequest; got != want {
				t.Fatalf("status = %d, want %d; body = %s", got, want, recorder.Body.String())
			}

			var response map[string]map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, recorder.Body.String())
			}
			if got, want := response["error"], map[string]string{
				"code":    "amount_malformed",
				"message": "Amount is malformed.",
			}; !reflect.DeepEqual(got, want) {
				t.Errorf("error = %#v, want %#v", got, want)
			}
		})
	}
}

func TestRouterJSONBalanceRejectionNamesDerivedBalance(t *testing.T) {
	t.Run("accepted JSON post does not reread the balance for rejection detail", func(t *testing.T) {
		base := &routerTestStore{
			accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 128350}}},
		}
		store := &transactionCountingRouterStore{Store: base}
		router, err := NewRouter(store, routerClock)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}

		accepted := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"1.00","description":"Coffee"}`)
		if got, want := accepted.Code, http.StatusCreated; got != want {
			t.Fatalf("status = %d, want %d; body = %s", got, want, accepted.Body.String())
		}
		if got, want := store.transactionsCalls, 1; got != want {
			t.Errorf("transaction reads after accepted post = %d, want %d", got, want)
		}
	})

	t.Run("balance detail matches the account balance endpoint", func(t *testing.T) {
		store := &routerTestStore{
			accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 128350}}},
		}
		router, err := NewRouter(store, routerClock)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}

		rejected := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"-2000.00","description":"Rent"}`)
		if got, want := rejected.Code, http.StatusBadRequest; got != want {
			t.Fatalf("status = %d, want %d; body = %s", got, want, rejected.Body.String())
		}
		var rejection errorEnvelope
		if err := json.Unmarshal(rejected.Body.Bytes(), &rejection); err != nil {
			t.Fatalf("rejection is not JSON: %v; body = %s", err, rejected.Body.String())
		}
		if got, want := rejection.Error.Code, "balance_would_go_negative"; got != want {
			t.Errorf("error code = %q, want %q", got, want)
		}
		if got, want := rejection.Error.Message, "Balance would go negative. Posting -$2,000.00 against a balance of $1,283.50."; got != want {
			t.Errorf("error message = %q, want %q", got, want)
		}

		accounts := httptest.NewRecorder()
		router.ServeHTTP(accounts, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
		if got, want := accounts.Code, http.StatusOK; got != want {
			t.Fatalf("GET /api/accounts status = %d, want %d; body = %s", got, want, accounts.Body.String())
		}
		var response []accountResponse
		if err := json.Unmarshal(accounts.Body.Bytes(), &response); err != nil {
			t.Fatalf("accounts response is not JSON: %v; body = %s", err, accounts.Body.String())
		}
		if got, want := response[0].BalanceCents, int64(128350); got != want {
			t.Errorf("GET /api/accounts balance_cents = %d, want %d", got, want)
		}
	})

	t.Run("a balance read failure keeps the rejection message plain", func(t *testing.T) {
		base := &routerTestStore{
			accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
			transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 128350}}},
		}
		store := &balanceReadFailsAfterPostingStore{Store: base, err: errors.New("balance read unavailable")}
		router, err := NewRouter(store, routerClock)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}

		rejected := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"-2000.00","description":"Rent"}`)
		if got, want := rejected.Code, http.StatusBadRequest; got != want {
			t.Fatalf("status = %d, want %d; body = %s", got, want, rejected.Body.String())
		}
		var rejection errorEnvelope
		if err := json.Unmarshal(rejected.Body.Bytes(), &rejection); err != nil {
			t.Fatalf("rejection is not JSON: %v; body = %s", err, rejected.Body.String())
		}
		if got, want := rejection.Error.Message, "Balance would go negative."; got != want {
			t.Errorf("error message = %q, want %q", got, want)
		}
	})
}

func TestRouterMapsBalanceOverflowAtBothPostingBoundaries(t *testing.T) {
	newRouter := func(t *testing.T) (*routerTestStore, http.Handler) {
		t.Helper()
		store := &routerTestStore{
			accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
			transactions: map[string][]ledger.Transaction{},
		}
		router, err := NewRouter(store, routerClock)
		if err != nil {
			t.Fatalf("NewRouter() error = %v, want nil", err)
		}
		return store, router
	}

	t.Run("form redirects, logs, and renders the overflow rejection without changing the maximum balance", func(t *testing.T) {
		store, router := newRouter(t)
		accepted := postTransaction(router, "/api/accounts/acct-1/transactions", "application/x-www-form-urlencoded", "amount=92233720368547758.07&description=Maximum+deposit")
		if got, want := accepted.Code, http.StatusSeeOther; got != want {
			t.Fatalf("accepted form status = %d, want %d; body = %s", got, want, accepted.Body.String())
		}
		countBefore, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		balanceBefore, err := ledger.Balance(store, "acct-1")
		if err != nil {
			t.Fatalf("Balance() error = %v", err)
		}
		if got, want := balanceBefore, int64(math.MaxInt64); got != want {
			t.Fatalf("balance after accepted maximum deposit = %d, want %d", got, want)
		}

		var logs bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&logs)
		t.Cleanup(func() { log.SetOutput(originalOutput) })

		rejected := postTransaction(router, "/api/accounts/acct-1/transactions", "application/x-www-form-urlencoded", "amount=0.01&description=One+cent")
		if got, want := rejected.Code, http.StatusSeeOther; got != want {
			t.Fatalf("rejected form status = %d, want %d; body = %s", got, want, rejected.Body.String())
		}
		if got, want := rejected.Header().Get("Location"), "/?account=acct-1&error=balance_overflow&detail=1"; got != want {
			t.Errorf("rejected form Location = %q, want %q", got, want)
		}
		if output := logs.String(); !strings.Contains(output, ledger.ErrBalanceOverflow.Error()) {
			t.Errorf("log output = %q, want overflow rejection", output)
		}
		page := httptest.NewRecorder()
		router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, rejected.Header().Get("Location"), nil))
		errorPosition, panel := pageErrorPanel(t, page.Body.String())
		if !strings.Contains(panel, "Balance would overflow.") {
			t.Errorf("error panel = %q, want overflow message", panel)
		}
		formPosition := strings.Index(page.Body.String(), "<form")
		if formPosition < 0 || errorPosition >= formPosition {
			t.Errorf("error position = %d, form position = %d; want panel above form", errorPosition, formPosition)
		}
		countAfter, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		balanceAfter, err := ledger.Balance(store, "acct-1")
		if err != nil {
			t.Fatalf("Balance() error = %v", err)
		}
		if countAfter != countBefore || balanceAfter != balanceBefore {
			t.Errorf("rejected form changed count/balance to %d/%d, want %d/%d", countAfter, balanceAfter, countBefore, balanceBefore)
		}
	})

	t.Run("JSON returns the typed overflow envelope without changing the maximum balance", func(t *testing.T) {
		store, router := newRouter(t)
		accepted := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"92233720368547758.07","description":"Maximum deposit"}`)
		if got, want := accepted.Code, http.StatusCreated; got != want {
			t.Fatalf("accepted JSON status = %d, want %d; body = %s", got, want, accepted.Body.String())
		}
		countBefore, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		balanceBefore, err := ledger.Balance(store, "acct-1")
		if err != nil {
			t.Fatalf("Balance() error = %v", err)
		}

		rejected := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"0.01","description":"One cent"}`)
		if got, want := rejected.Code, http.StatusBadRequest; got != want {
			t.Errorf("rejected JSON status = %d, want %d; body = %s", got, want, rejected.Body.String())
		}
		if got, want := rejected.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		var response errorEnvelope
		if err := json.Unmarshal(rejected.Body.Bytes(), &response); err != nil {
			t.Fatalf("response is not JSON: %v; body = %s", err, rejected.Body.String())
		}
		if got, want := response.Error.Code, "balance_overflow"; got != want {
			t.Errorf("error code = %q, want %q", got, want)
		}
		if got, want := response.Error.Message, "Balance would overflow. Posting $0.01 against a balance of $92,233,720,368,547,758.07."; got != want {
			t.Errorf("error message = %q, want %q", got, want)
		}
		countAfter, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		balanceAfter, err := ledger.Balance(store, "acct-1")
		if err != nil {
			t.Fatalf("Balance() error = %v", err)
		}
		if countAfter != countBefore || balanceAfter != balanceBefore {
			t.Errorf("rejected JSON changed count/balance to %d/%d, want %d/%d", countAfter, balanceAfter, countBefore, balanceBefore)
		}
	})
}

func TestRouterRejectsMalformedJSONAmountsWithoutAppending(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 10000}}},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{"amount":"-42.50"`},
		{name: "omitted amount", body: `{"description":"Coffee"}`},
		{name: "numeric amount", body: `{"amount":-42.50,"description":"Coffee"}`},
		{name: "trailing non-JSON content", body: `{"amount":"-42.50","description":"Coffee"} trailing`},
		{name: "second JSON value", body: `{"amount":"-42.50","description":"Coffee"} {"amount":"1.00"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			countBefore, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}

			rec := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", tt.body)
			if got, want := rec.Code, http.StatusBadRequest; got != want {
				t.Errorf("status = %d, want %d; body = %s", got, want, rec.Body.String())
			}
			var response errorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v; body = %s", err, rec.Body.String())
			}
			if got, want := response.Error.Code, "amount_malformed"; got != want {
				t.Errorf("error code = %q, want %q", got, want)
			}
			countAfter, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}
			if got, want := countAfter, countBefore; got != want {
				t.Errorf("transactions after rejected request = %d, want %d", got, want)
			}
		})
	}

	t.Run("extra JSON field is ignored", func(t *testing.T) {
		countBefore, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}

		rec := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"-42.50","description":"Coffee","ignored":"value"}`)
		if got, want := rec.Code, http.StatusCreated; got != want {
			t.Fatalf("status = %d, want %d; body = %s", got, want, rec.Body.String())
		}
		countAfter, err := store.CountTransactions()
		if err != nil {
			t.Fatalf("CountTransactions() error = %v", err)
		}
		if got, want := countAfter, countBefore+1; got != want {
			t.Errorf("transactions after accepted request = %d, want %d", got, want)
		}
	})
}

func TestRouterRejectsInvalidAmountSignsWithoutAppending(t *testing.T) {
	store := &routerTestStore{
		accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 10000}}},
	}
	router, err := NewRouter(store, routerClock)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}

	for _, amount := range []string{"+5", "1.+5", "25.+5"} {
		t.Run(amount, func(t *testing.T) {
			countBefore, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}

			jsonResponse := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", `{"amount":"`+amount+`","description":"Coffee"}`)
			if got, want := jsonResponse.Code, http.StatusBadRequest; got != want {
				t.Errorf("JSON status = %d, want %d; body = %s", got, want, jsonResponse.Body.String())
			}
			var response errorEnvelope
			if err := json.Unmarshal(jsonResponse.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON response is not JSON: %v; body = %s", err, jsonResponse.Body.String())
			}
			if got, want := response.Error.Code, "amount_malformed"; got != want {
				t.Errorf("JSON error code = %q, want %q", got, want)
			}

			formResponse := postTransaction(router, "/api/accounts/acct-1/transactions", "application/x-www-form-urlencoded", "amount="+url.QueryEscape(amount)+"&description=Coffee")
			if got, want := formResponse.Code, http.StatusSeeOther; got != want {
				t.Errorf("form status = %d, want %d; body = %s", got, want, formResponse.Body.String())
			}
			if got, want := formResponse.Header().Get("Location"), "/?account=acct-1&error=amount_malformed&detail="+url.QueryEscape(amount); got != want {
				t.Errorf("form Location = %q, want %q", got, want)
			}

			countAfter, err := store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() error = %v", err)
			}
			if got, want := countAfter, countBefore; got != want {
				t.Errorf("transactions after rejected posts = %d, want %d", got, want)
			}
		})
	}
}

func TestRouterEnrichesMalformedAmountMessagesConsistentlyForPageAndJSON(t *testing.T) {
	for _, tt := range []struct {
		name     string
		amount   string
		wantPage string
		wantJSON string
	}{
		{
			name:     "multiple decimal points",
			amount:   "12.3.4",
			wantPage: "Amount is malformed. Submitted: 12.3.4.",
			wantJSON: "Amount is malformed. Submitted: 12.3.4.",
		},
		{
			name:     "script-bearing text",
			amount:   "<script>alert(1)</script>",
			wantPage: "Amount is malformed. Submitted: &lt;script&gt;alert(1)&lt;/script&gt;.",
			wantJSON: "Amount is malformed. Submitted: <script>alert(1)</script>.",
		},
		{
			name:     "space",
			amount:   " ",
			wantPage: "Amount is malformed.",
			wantJSON: "Amount is malformed.",
		},
		{
			name:     "non-breaking space",
			amount:   "\u00a0",
			wantPage: "Amount is malformed.",
			wantJSON: "Amount is malformed.",
		},
		{
			name:     "format character",
			amount:   "\u200d",
			wantPage: "Amount is malformed.",
			wantJSON: "Amount is malformed.",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &routerTestStore{
				accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
				transactions: map[string][]ledger.Transaction{"acct-1": {{Amount: 10000}}},
			}
			router, err := NewRouter(store, routerClock)
			if err != nil {
				t.Fatalf("NewRouter() error = %v, want nil", err)
			}

			form := postTransaction(router, "/api/accounts/acct-1/transactions", "application/x-www-form-urlencoded", url.Values{"amount": {tt.amount}, "description": {"Coffee"}}.Encode())
			if got, want := form.Code, http.StatusSeeOther; got != want {
				t.Fatalf("form status = %d, want %d; body = %s", got, want, form.Body.String())
			}
			page := httptest.NewRecorder()
			router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, form.Header().Get("Location"), nil))
			_, panel := pageErrorPanel(t, page.Body.String())
			if got := strings.TrimSpace(panel); got != tt.wantPage {
				t.Errorf("page error panel = %q, want %q", got, tt.wantPage)
			}

			body, err := json.Marshal(map[string]string{"amount": tt.amount, "description": "Coffee"})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			jsonResponse := postTransaction(router, "/api/accounts/acct-1/transactions", "application/json", string(body))
			if got, want := jsonResponse.Code, http.StatusBadRequest; got != want {
				t.Fatalf("JSON status = %d, want %d; body = %s", got, want, jsonResponse.Body.String())
			}
			var response errorEnvelope
			if err := json.Unmarshal(jsonResponse.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON response is not JSON: %v; body = %s", err, jsonResponse.Body.String())
			}
			if got := response.Error.Message; got != tt.wantJSON {
				t.Errorf("JSON error message = %q, want %q", got, tt.wantJSON)
			}
		})
	}
}

func postTransaction(router http.Handler, path, contentType, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func TestRouterRejectsWrongMethodsAndUnknownPathsWithoutABody(t *testing.T) {
	store := routerTestStore{
		accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}},
		transactions: map[string][]ledger.Transaction{
			"acct-1": {{ID: "txn-0001", AccountID: "acct-1", Amount: 100}},
		},
	}
	router, err := NewRouter(&store, routerClock)
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

func TestNewRouterDeclaresExactlyFiveRoutes(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	routerFile := strings.TrimSuffix(testFile, "router_test.go") + "router.go"
	parsed, err := parser.ParseFile(token.NewFileSet(), routerFile, nil, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v", routerFile, err)
	}

	var registrations int
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "NewRouter" {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Handle" && selector.Sel.Name != "HandleFunc") {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if ok && receiver.Name == "mux" {
				registrations++
			}
			return true
		})
	}

	if got, want := registrations, 5; got != want {
		t.Errorf("NewRouter() route registrations = %d, want %d", got, want)
	}
}

func TestRouterFrozenRejectionsDoNotRecordTransactions(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		amount       string
		description  string
		transactions map[string][]ledger.Transaction
		code         string
		jsonStatus   int
	}{
		{
			name:        "account_not_found",
			accountID:   "acct-nope",
			amount:      "1.00",
			description: "Coffee",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "account_not_found",
			jsonStatus: http.StatusNotFound,
		},
		{
			name:        "amount_zero",
			accountID:   "acct-1",
			amount:      "0",
			description: "Coffee",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "amount_zero",
			jsonStatus: http.StatusBadRequest,
		},
		{
			name:        "description_empty",
			accountID:   "acct-1",
			amount:      "1.00",
			description: "",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "description_empty",
			jsonStatus: http.StatusBadRequest,
		},
		{
			name:        "description_too_long",
			accountID:   "acct-1",
			amount:      "1.00",
			description: strings.Repeat("x", 141),
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "description_too_long",
			jsonStatus: http.StatusBadRequest,
		},
		{
			name:        "amount_malformed",
			accountID:   "acct-1",
			amount:      "12.3.4",
			description: "Coffee",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "amount_malformed",
			jsonStatus: http.StatusBadRequest,
		},
		{
			name:        "balance_would_go_negative",
			accountID:   "acct-1",
			amount:      "-200.00",
			description: "Coffee",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: 10000}},
			},
			code:       "balance_would_go_negative",
			jsonStatus: http.StatusBadRequest,
		},
		{
			name:        "balance_overflow",
			accountID:   "acct-1",
			amount:      "0.01",
			description: "Coffee",
			transactions: map[string][]ledger.Transaction{
				"acct-1": {{Amount: math.MaxInt64}},
			},
			code:       "balance_overflow",
			jsonStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, branch := range []struct {
				name        string
				contentType string
				body        func() string
				wantStatus  int
			}{
				{
					name:        "form",
					contentType: "application/x-www-form-urlencoded",
					body: func() string {
						return url.Values{"amount": {tt.amount}, "description": {tt.description}}.Encode()
					},
					wantStatus: http.StatusSeeOther,
				},
				{
					name:        "json",
					contentType: "application/json",
					body: func() string {
						body, err := json.Marshal(map[string]string{"amount": tt.amount, "description": tt.description})
						if err != nil {
							t.Fatalf("json.Marshal() error = %v", err)
						}
						return string(body)
					},
					wantStatus: tt.jsonStatus,
				},
			} {
				t.Run(branch.name, func(t *testing.T) {
					store := &routerTestStore{
						accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
						transactions: tt.transactions,
					}
					router, err := NewRouter(store, routerClock)
					if err != nil {
						t.Fatalf("NewRouter() error = %v, want nil", err)
					}

					countBefore, err := store.CountTransactions()
					if err != nil {
						t.Fatalf("CountTransactions() error = %v", err)
					}
					balanceBefore, err := ledger.Balance(store, "acct-1")
					if err != nil {
						t.Fatalf("Balance() error = %v", err)
					}

					path := "/api/accounts/" + url.PathEscape(tt.accountID) + "/transactions"
					response := postTransaction(router, path, branch.contentType, branch.body())
					if got, want := response.Code, branch.wantStatus; got != want {
						t.Fatalf("status = %d, want %d; body = %s", got, want, response.Body.String())
					}
					if branch.name == "form" {
						if got := response.Header().Get("Location"); !strings.Contains(got, "error="+tt.code) {
							t.Errorf("Location = %q, want error code %q", got, tt.code)
						}
					} else {
						var envelope errorEnvelope
						if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
							t.Fatalf("json.Unmarshal() error = %v; body = %s", err, response.Body.String())
						}
						if got, want := envelope.Error.Code, tt.code; got != want {
							t.Errorf("error code = %q, want %q", got, want)
						}
					}

					countAfter, err := store.CountTransactions()
					if err != nil {
						t.Fatalf("CountTransactions() error = %v", err)
					}
					balanceAfter, err := ledger.Balance(store, "acct-1")
					if err != nil {
						t.Fatalf("Balance() error = %v", err)
					}
					if got, want := countAfter, countBefore; got != want {
						t.Errorf("transaction count after rejection = %d, want %d", got, want)
					}
					if got, want := balanceAfter, balanceBefore; got != want {
						t.Errorf("derived balance after rejection = %d, want %d", got, want)
					}
				})
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
	router, err := NewRouter(&store, routerClock)
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
	router, err := NewRouter(&store, routerClock)
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

func TestRouterSelectsCSVForAccountTransactions(t *testing.T) {
	createdEarlier := time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)
	transactions := []ledger.Transaction{
		{ID: "txn-0002", AccountID: "acct-1", Amount: -4250, Description: "Groceries", CreatedAt: createdEarlier.Add(time.Minute)},
		{ID: "txn-0001", AccountID: "acct-1", Amount: 128350, Description: "Deposit", CreatedAt: createdEarlier},
	}
	const csvHeader = "id,amount_cents,description,created_at\n"
	const jsonBody = "[{\"id\":\"txn-0002\",\"account_id\":\"acct-1\",\"amount_cents\":-4250,\"description\":\"Groceries\",\"created_at\":\"2026-08-08T14:31:00Z\"},{\"id\":\"txn-0001\",\"account_id\":\"acct-1\",\"amount_cents\":128350,\"description\":\"Deposit\",\"created_at\":\"2026-08-08T14:30:00Z\"}]\n"

	tests := []struct {
		name                string
		path                string
		newStore            func() ledger.Store
		wantStatus          int
		wantContentType     string
		wantDisposition     string
		wantBody            string
		wantNoCSVHeaders    bool
		wantNoHeaderBody    bool
		wantRepeatUnchanged bool
	}{
		{
			name: "populated account returns the renderer's exact CSV as an attachment",
			path: "/api/accounts/acct-1/transactions?format=csv",
			newStore: func() ledger.Store {
				return &routerTestStore{
					accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
					transactions: map[string][]ledger.Transaction{"acct-1": append([]ledger.Transaction(nil), transactions...)},
				}
			},
			wantStatus:          http.StatusOK,
			wantContentType:     "text/csv; charset=utf-8",
			wantDisposition:     `attachment; filename="transactions.csv"`,
			wantBody:            string(renderTransactionsCSV(transactions)),
			wantRepeatUnchanged: true,
		},
		{
			name: "empty account returns a header-only CSV attachment",
			path: "/api/accounts/acct-empty/transactions?format=csv",
			newStore: func() ledger.Store {
				return &routerTestStore{accounts: []ledger.Account{{ID: "acct-empty", Name: "Empty"}}}
			},
			wantStatus:      http.StatusOK,
			wantContentType: "text/csv; charset=utf-8",
			wantDisposition: `attachment; filename="transactions.csv"`,
			wantBody:        csvHeader,
		},
		{
			name: "missing account keeps the JSON 404 without CSV metadata or a header-only body",
			path: "/api/accounts/acct-nope/transactions?format=csv",
			newStore: func() ledger.Store {
				return &routerTestStore{accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}}}
			},
			wantStatus:       http.StatusNotFound,
			wantContentType:  "application/json; charset=utf-8",
			wantNoCSVHeaders: true,
			wantNoHeaderBody: true,
		},
		{
			name: "unexpected store failure stays undisclosed with no partial CSV",
			path: "/api/accounts/acct-1/transactions?format=csv",
			newStore: func() ledger.Store {
				return failingRouterStore{
					Store:           &routerTestStore{accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}}},
					transactionsErr: errors.New("private database failure"),
				}
			},
			wantStatus:       http.StatusInternalServerError,
			wantContentType:  "text/plain; charset=utf-8",
			wantNoCSVHeaders: true,
			wantNoHeaderBody: true,
		},
		{
			name: "the same URL without CSV selection retains its exact JSON representation",
			path: "/api/accounts/acct-1/transactions",
			newStore: func() ledger.Store {
				return &routerTestStore{
					accounts:     []ledger.Account{{ID: "acct-1", Name: "Checking"}},
					transactions: map[string][]ledger.Transaction{"acct-1": append([]ledger.Transaction(nil), transactions...)},
				}
			},
			wantStatus:      http.StatusOK,
			wantContentType: "application/json; charset=utf-8",
			wantBody:        jsonBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.newStore()
			router, err := NewRouter(store, routerClock)
			if err != nil {
				t.Fatalf("NewRouter() error = %v, want nil", err)
			}

			var before []ledger.Transaction
			if concrete, ok := store.(*routerTestStore); ok {
				before = append([]ledger.Transaction(nil), concrete.transactions["acct-1"]...)
			}
			request := func() *httptest.ResponseRecorder {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
				return recorder
			}
			recorder := request()

			if got := recorder.Code; got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if got := recorder.Header().Get("Content-Type"); got != tt.wantContentType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantContentType)
			}
			if got := recorder.Header().Get("Content-Disposition"); got != tt.wantDisposition {
				t.Errorf("Content-Disposition = %q, want %q", got, tt.wantDisposition)
			}
			if tt.wantBody != "" && recorder.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", recorder.Body.String(), tt.wantBody)
			}
			if tt.wantNoCSVHeaders && strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/csv") {
				t.Errorf("failure response has CSV Content-Type %q", recorder.Header().Get("Content-Type"))
			}
			if tt.wantNoHeaderBody && recorder.Body.String() == csvHeader {
				t.Errorf("failure response has misleading header-only CSV body %q", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "private database failure") {
				t.Errorf("response disclosed the underlying store failure: %q", recorder.Body.String())
			}

			if tt.wantRepeatUnchanged {
				repeat := request()
				if !bytes.Equal(recorder.Body.Bytes(), repeat.Body.Bytes()) {
					t.Errorf("unchanged requests differ: first %q, second %q", recorder.Body.Bytes(), repeat.Body.Bytes())
				}
				concrete := store.(*routerTestStore)
				if !reflect.DeepEqual(concrete.transactions["acct-1"], before) {
					t.Errorf("GET mutated store transactions: got %#v, want %#v", concrete.transactions["acct-1"], before)
				}
			}
		})
	}
}

func TestRouterLogsUnexpectedStoreErrorsWithoutDisclosingThem(t *testing.T) {
	baseStore := &routerTestStore{
		accounts: []ledger.Account{{ID: "acct-1", Name: "Checking"}},
	}

	for _, tt := range []struct {
		name  string
		path  string
		store failingRouterStore
		err   error
	}{
		{
			name: "account list failure",
			path: "/api/accounts",
			err:  errors.New("database account query unavailable"),
			store: failingRouterStore{
				Store: baseStore,
			},
		},
		{
			name: "account balance failure",
			path: "/api/accounts",
			err:  errors.New("database balance query unavailable"),
			store: failingRouterStore{
				Store: baseStore,
			},
		},
		{
			name: "transaction list failure",
			path: "/api/accounts/acct-1/transactions",
			err:  errors.New("database transaction query unavailable"),
			store: failingRouterStore{
				Store: baseStore,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "account list failure" {
				tt.store.accountsErr = tt.err
			} else {
				tt.store.transactionsErr = tt.err
			}
			router, err := NewRouter(tt.store, routerClock)
			if err != nil {
				t.Fatalf("NewRouter() error = %v, want nil", err)
			}

			var logs bytes.Buffer
			originalOutput := log.Writer()
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(originalOutput) })

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if got, want := rec.Code, http.StatusInternalServerError; got != want {
				t.Errorf("status = %d, want %d", got, want)
			}
			if got, want := rec.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			if body := rec.Body.String(); strings.Contains(body, tt.err.Error()) {
				t.Errorf("500 response disclosed internal error %q in body %q", tt.err, body)
			}
			if output := logs.String(); !strings.Contains(output, tt.err.Error()) {
				t.Errorf("log output = %q, want underlying error %q", output, tt.err)
			}
		})
	}
}

var _ ledger.Store = (*routerTestStore)(nil)
var _ ledger.Store = failingRouterStore{}
