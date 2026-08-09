package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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
		formPosition := strings.Index(body, "<form")
		if formPosition < 0 || errorPosition >= formPosition {
			t.Errorf("error panel position = %d, form position = %d, want panel before form; body = %s", errorPosition, formPosition, body)
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
			if got, want := form.Header().Get("Location"), "/?account="+url.QueryEscape(strings.TrimSuffix(strings.TrimPrefix(tt.path, "/api/accounts/"), "/transactions"))+"&error="+tt.code; got != want {
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

var _ ledger.Store = (*routerTestStore)(nil)
