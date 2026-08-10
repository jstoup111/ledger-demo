package acceptance

// Acceptance specs for the base ledger, generated from .docs/stories/base-ledger.md.
//
// Every spec here crosses two or more operations, per the acceptance-test layer's
// remit. Single-operation behavior (one endpoint's status codes and error shapes)
// and per-rule sentinel semantics are owned by the lower layers built during the
// implementation tasks — plan Tasks 18 (rule semantics), 20 (sentinel-to-code),
// 21-23 (read routes), 26-27 (page markup) — and are deliberately NOT restated
// here. See .pipeline/fr-coverage.md for the per-FR disposition.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var txnID = regexp.MustCompile(`^txn-\d{4}$`)

// TestAcceptanceCSVExport covers the command-line export boundary: it must
// expose the same ordered transaction log as the JSON API and remain stable
// across repeated invocations.
func TestAcceptanceCSVExport(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "acceptance.db")
	seedDB(t, dbPath)
	base, _ := startServer(t, dbPath)
	a := newAppAt(t, dbPath, base)

	accountID := a.accounts()[0].ID
	txs := a.transactions(accountID)
	stdout, stderr, err := exportCSV(t, dbPath, accountID)
	if err != nil {
		t.Fatalf("export %s: %v; stderr:\n%s", accountID, err, stderr)
	}
	if len(stderr) != 0 {
		t.Errorf("export %s stderr = %q, want empty", accountID, stderr)
	}
	records, err := csv.NewReader(bytes.NewReader(stdout)).ReadAll()
	if err != nil {
		t.Fatalf("parse export for %s: %v; stdout:\n%s", accountID, err, stdout)
	}
	if got, want := records[0], []string{"id", "amount_cents", "description", "created_at"}; !equalStrings(got, want) {
		t.Fatalf("CSV header = %q, want %q", got, want)
	}
	if got, want := len(records)-1, len(txs); got != want {
		t.Fatalf("CSV transaction count = %d, want %d", got, want)
	}
	for i, tx := range txs {
		want := []string{tx.ID, strconv.FormatInt(tx.AmountCents, 10), tx.Description, tx.CreatedAt}
		if got := records[i+1]; !equalStrings(got, want) {
			t.Errorf("CSV transaction %d = %q, want %q", i, got, want)
		}
	}

	stdoutAgain, stderrAgain, err := exportCSV(t, dbPath, accountID)
	if err != nil {
		t.Fatalf("second export %s: %v; stderr:\n%s", accountID, err, stderrAgain)
	}
	if !bytes.Equal(stdoutAgain, stdout) {
		t.Errorf("repeated export differs:\nfirst:  %q\nsecond: %q", stdout, stdoutAgain)
	}

	empty, emptyStderr, err := exportCSV(t, dbPath, "acct-3")
	if err != nil {
		t.Fatalf("export acct-3: %v; stderr:\n%s", err, emptyStderr)
	}
	if got, want := string(empty), "id,amount_cents,description,created_at\n"; got != want {
		t.Errorf("empty-account export = %q, want %q", got, want)
	}

	unknownID := "acct-nope"
	unknown, unknownStderr, err := exportCSV(t, dbPath, unknownID)
	if err == nil {
		t.Errorf("export %s exited zero", unknownID)
	}
	if len(unknown) != 0 {
		t.Errorf("export %s stdout = %q, want zero bytes", unknownID, unknown)
	}
	if !strings.Contains(string(unknownStderr), unknownID) {
		t.Errorf("export %s stderr = %q, want requested id", unknownID, unknownStderr)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestAcceptanceAccountSelectionAndBalance covers Story 1.
//
// Covers: FR-1, FR-2, FR-10
func TestAcceptanceAccountSelectionAndBalance(t *testing.T) {
	a := newApp(t)

	accounts := a.accounts()

	t.Run("the API lists every account with a derived balance, ordered by id", func(t *testing.T) {
		res := a.get("/api/accounts")
		if got, want := res.header.Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}
		if len(accounts) != 3 {
			t.Fatalf("GET /api/accounts returned %d accounts, want 3", len(accounts))
		}
		for i := 1; i < len(accounts); i++ {
			if accounts[i-1].ID >= accounts[i].ID {
				t.Errorf("accounts are not ordered by id ascending: %q then %q",
					accounts[i-1].ID, accounts[i].ID)
			}
		}
		for _, acct := range accounts {
			if acct.ID == "" || acct.Name == "" {
				t.Errorf("account %+v is missing id or name", acct)
			}
			// The balance is a fold over that account's own log, so the two
			// endpoints must agree exactly.
			var sum int64
			for _, tx := range a.transactions(acct.ID) {
				sum += tx.AmountCents
			}
			if acct.BalanceCents != sum {
				t.Errorf("account %s balance_cents = %d, want %d (the sum of its transactions)",
					acct.ID, acct.BalanceCents, sum)
			}
		}
	})

	t.Run("the page shows exactly the selected account's balance", func(t *testing.T) {
		selected := accounts[1]
		res := a.get("/?account=" + url.QueryEscape(selected.ID))

		if res.status != http.StatusOK {
			t.Fatalf("GET /?account=%s status = %d, want 200", selected.ID, res.status)
		}
		mustContain(t, res.body, `class="balance"`, "selected account page")
		mustContain(t, res.body, `class="selected-account">Selected account: `+html.EscapeString(selected.Name), "selected account identity")
		mustContain(t, res.body, formatDollars(selected.BalanceCents), "selected account page")

		// FR-1: exactly one account is displayed at a time.
		for _, other := range accounts {
			if other.ID == selected.ID {
				continue
			}
			formatted := formatDollars(other.BalanceCents)
			if formatted == formatDollars(selected.BalanceCents) {
				continue // indistinguishable values cannot prove the point either way
			}
			mustNotContain(t, res.body, formatted,
				fmt.Sprintf("page for %s must not show %s's balance", selected.ID, other.ID))
		}
	})

	t.Run("the page lists all three accounts as links", func(t *testing.T) {
		res := a.get("/")
		if res.status != http.StatusOK {
			t.Fatalf("GET / status = %d, want 200", res.status)
		}
		for _, acct := range accounts {
			mustContain(t, res.body, html.EscapeString(acct.Name), "account selector")
			mustContain(t, res.body, "account="+url.QueryEscape(acct.ID), "account selector")
		}
		// With no account parameter the first account by id ascending is shown.
		mustContain(t, res.body, formatDollars(accounts[0].BalanceCents), "default account balance")
	})

	t.Run("an unknown account renders a not-found page with no balance and no form", func(t *testing.T) {
		res := a.get("/?account=acct-nope")

		if res.status != http.StatusOK {
			t.Errorf("GET /?account=acct-nope status = %d, want 200", res.status)
		}
		if !strings.Contains(strings.ToLower(res.body), "not found") {
			t.Errorf("unknown-account page does not state the account was not found; body:\n%s", res.body)
		}
		mustNotContain(t, res.body, `class="balance"`, "unknown-account page")
		mustNotContain(t, res.body, "<form", "unknown-account page")
		mustNotContain(t, res.body, "<table", "unknown-account page")
		// The account list is still rendered so the presenter can recover.
		mustContain(t, res.body, html.EscapeString(accounts[0].Name), "unknown-account page")
	})

	t.Run("a script-bearing account parameter is escaped", func(t *testing.T) {
		res := a.get("/?account=" + url.QueryEscape("<script>alert(1)</script>"))

		if res.status != http.StatusOK {
			t.Errorf("status = %d, want 200", res.status)
		}
		mustNotContain(t, res.body, "<script>", "escaped account parameter")
	})
}

// TestAcceptanceTransactionLogOrder covers Story 2: the API's order and the page's
// order are the same order, and the log is append-only.
//
// Covers: FR-3, FR-11
func TestAcceptanceTransactionLogOrder(t *testing.T) {
	a := newApp(t)

	accountID := a.accounts()[0].ID

	t.Run("the page renders the log in the API's newest-first order", func(t *testing.T) {
		txs := a.transactions(accountID)
		if len(txs) < 2 {
			t.Fatalf("account %s has %d transactions, want at least 2 to prove an ordering", accountID, len(txs))
		}

		for _, tx := range txs {
			if !txnID.MatchString(tx.ID) {
				t.Errorf("transaction id %q does not match ^txn-\\d{4}$", tx.ID)
			}
			if tx.AccountID != accountID {
				t.Errorf("transaction %s account_id = %q, want %q", tx.ID, tx.AccountID, accountID)
			}
			if !strings.HasSuffix(tx.CreatedAt, "Z") {
				t.Errorf("transaction %s created_at = %q, want RFC 3339 UTC", tx.ID, tx.CreatedAt)
			}
		}

		// Newest first, and the tie between equal timestamps is broken by the id.
		for i := 1; i < len(txs); i++ {
			prev, cur := txs[i-1], txs[i]
			if prev.CreatedAt < cur.CreatedAt {
				t.Errorf("transactions are not newest first: %s (%s) precedes %s (%s)",
					prev.ID, prev.CreatedAt, cur.ID, cur.CreatedAt)
			}
			if prev.CreatedAt == cur.CreatedAt && prev.ID <= cur.ID {
				t.Errorf("equal timestamps are not tie-broken by descending id: %s then %s", prev.ID, cur.ID)
			}
		}

		// The page walks the same sequence: each description appears after the
		// previous one in the rendered body.
		body := a.get("/?account=" + url.QueryEscape(accountID)).body
		cursor := 0
		for _, tx := range txs {
			needle := html.EscapeString(tx.Description)
			offset := strings.Index(body[cursor:], needle)
			if offset < 0 {
				t.Fatalf("transaction %s (%q) does not appear on the page in the API's order; body:\n%s",
					tx.ID, tx.Description, body)
			}
			cursor += offset + len(needle)
		}
	})

	t.Run("a newly recorded transaction leads both the API and the page", func(t *testing.T) {
		res := a.postJSON("/api/accounts/"+url.PathEscape(accountID)+"/transactions",
			`{"amount":"1.00","description":"Acceptance newest-first probe"}`)
		if res.status != http.StatusCreated {
			t.Fatalf("POST status = %d, want 201; body:\n%s", res.status, res.body)
		}

		txs := a.transactions(accountID)
		if txs[0].Description != "Acceptance newest-first probe" {
			t.Errorf("newest transaction is %q, want the one just recorded", txs[0].Description)
		}

		body := a.get("/?account=" + url.QueryEscape(accountID)).body
		newest := strings.Index(body, html.EscapeString(txs[0].Description))
		second := strings.Index(body, html.EscapeString(txs[1].Description))
		if newest < 0 || second < 0 || newest > second {
			t.Errorf("the page does not show the new transaction at the top of the list; body:\n%s", body)
		}
	})

	t.Run("an unknown account's log is a 404 with account_not_found", func(t *testing.T) {
		res := a.get("/api/accounts/acct-nope/transactions")
		if res.status != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body:\n%s", res.status, res.body)
		}
		if code := decodeError(t, res).Error.Code; code != "account_not_found" {
			t.Errorf("code = %q, want %q", code, "account_not_found")
		}
	})

	t.Run("the log is append-only", func(t *testing.T) {
		before := a.transactions(accountID)
		path := "/api/accounts/" + url.PathEscape(accountID) + "/transactions"

		for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
			res := a.request(method, path, "application/json", `{"amount":"1.00","description":"mutate"}`)
			if res.status != http.StatusMethodNotAllowed {
				t.Errorf("%s %s status = %d, want 405", method, path, res.status)
			}
			if res.body != "" {
				t.Errorf("%s %s body = %q, want empty", method, path, res.body)
			}
		}

		after := a.transactions(accountID)
		if len(after) != len(before) {
			t.Fatalf("transaction count changed from %d to %d", len(before), len(after))
		}
		for i := range before {
			if before[i] != after[i] {
				t.Errorf("transaction %s changed: %+v became %+v", before[i].ID, before[i], after[i])
			}
		}
	})

	t.Run("POST /api/accounts is not allowed", func(t *testing.T) {
		res := a.postJSON("/api/accounts", `{}`)
		if res.status != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", res.status)
		}
		if res.body != "" {
			t.Errorf("body = %q, want empty", res.body)
		}
	})
}

// TestAcceptancePageFormRoundTrip covers Story 3: the presenter's full loop —
// open the page, submit the form, land back on the page with the balance moved —
// and the rejection loop that carries a code through the URL onto the page.
//
// Covers: FR-5, FR-6, FR-7, FR-13
func TestAcceptancePageFormRoundTrip(t *testing.T) {
	a := newApp(t)

	accountID := a.accounts()[0].ID
	pagePath := "/?account=" + url.QueryEscape(accountID)
	postPath := "/api/accounts/" + url.PathEscape(accountID) + "/transactions"

	t.Run("the page offers a form posting to the account's transactions", func(t *testing.T) {
		body := a.get(pagePath).body
		mustContain(t, body, `action="`+postPath+`"`, "post form")
		mustContain(t, body, `method="post"`, "post form")
		mustContain(t, body, `name="amount"`, "post form")
		mustContain(t, body, `name="description"`, "post form")
		// NFR-2 / Story 3: no JavaScript on the page at all.
		mustNotContain(t, body, "<script", "rendered page")
	})

	t.Run("submitting the form moves the balance and reloading does not double it", func(t *testing.T) {
		before := a.balanceCents(accountID)
		countBefore := a.transactionCount(accountID)

		res := a.postForm(postPath, url.Values{
			"amount":      {"25"},
			"description": {"Deposit"},
		})
		if res.status != http.StatusSeeOther {
			t.Fatalf("form POST status = %d, want 303; body:\n%s", res.status, res.body)
		}
		if want := "/?account=" + url.QueryEscape(accountID); res.location != want {
			t.Fatalf("Location = %q, want %q", res.location, want)
		}

		// Following the redirect shows the movement.
		page := a.get(res.location)
		if page.status != http.StatusOK {
			t.Fatalf("following the redirect gave %d, want 200", page.status)
		}
		mustContain(t, page.body, formatDollars(before+2500), "page after deposit")
		mustContain(t, page.body, "Deposit", "page after deposit")

		if got := a.balanceCents(accountID); got != before+2500 {
			t.Errorf("balance = %d, want %d", got, before+2500)
		}

		// FR-7: the result of the POST is a GET, so a reload replays only the GET.
		a.get(res.location)
		a.get(res.location)
		if got := a.transactionCount(accountID); got != countBefore+1 {
			t.Errorf("transaction count = %d after two reloads, want %d", got, countBefore+1)
		}
		if got := a.balanceCents(accountID); got != before+2500 {
			t.Errorf("balance = %d after two reloads, want %d", got, before+2500)
		}
	})

	t.Run("a signed amount subtracts", func(t *testing.T) {
		before := a.balanceCents(accountID)

		res := a.postForm(postPath, url.Values{
			"amount":      {"-42.50"},
			"description": {"Coffee beans"},
		})
		if res.status != http.StatusSeeOther {
			t.Fatalf("form POST status = %d, want 303; body:\n%s", res.status, res.body)
		}
		if got := a.balanceCents(accountID); got != before-4250 {
			t.Errorf("balance = %d, want %d", got, before-4250)
		}
		if got := a.transactions(accountID)[0].AmountCents; got != -4250 {
			t.Errorf("recorded amount_cents = %d, want -4250", got)
		}
	})

	t.Run("a rejected submission redirects with its code and renders it visibly", func(t *testing.T) {
		countBefore := a.transactionCount(accountID)

		res := a.postForm(postPath, url.Values{
			"amount":      {"25"},
			"description": {""},
		})
		if res.status != http.StatusSeeOther {
			t.Fatalf("rejected form POST status = %d, want 303; body:\n%s", res.status, res.body)
		}
		if !strings.Contains(res.location, "error=description_empty") {
			t.Fatalf("Location = %q, want it to carry error=description_empty", res.location)
		}
		if got := a.transactionCount(accountID); got != countBefore {
			t.Errorf("transaction count = %d after a rejection, want %d", got, countBefore)
		}

		page := a.get(res.location)
		mustContain(t, page.body, `class="error"`, "rejection page")

		// FR-13: the message sits directly above the form that produced it.
		panelAt := strings.Index(page.body, `class="error"`)
		panelEndOffset := strings.Index(page.body[panelAt:], "</p>")
		formAt := strings.Index(page.body, "<form")
		if panelAt < 0 || panelEndOffset < 0 || formAt < 0 || strings.TrimSpace(page.body[panelAt+panelEndOffset+len("</p>"):formAt]) != "" {
			t.Errorf("the error panel must be immediately before the form; body:\n%s", page.body)
		}
	})

	t.Run("an unrecognized error code renders a generic message, never an empty panel", func(t *testing.T) {
		page := a.get(pagePath + "&error=not_a_real_code")

		mustContain(t, page.body, `class="error"`, "generic rejection page")
		panel := between(page.body, `class="error"`, "</")
		if strings.TrimSpace(stripTags(panel)) == "" {
			t.Errorf("the error panel is empty for an unrecognized code; body:\n%s", page.body)
		}
		mustNotContain(t, page.body, "not_a_real_code", "generic rejection page")
	})

	t.Run("a script-bearing error code is escaped", func(t *testing.T) {
		page := a.get(pagePath + "&error=" + url.QueryEscape("<script>alert(1)</script>"))
		mustNotContain(t, page.body, "<script", "escaped error code")
	})
}

// TestAcceptanceProgrammaticPostingMatchesTheForm covers Story 4: the JSON client's
// round trip, and the FR-9 guarantee that neither content type admits input the
// other refuses. The wire codes are consumed from the contract, not re-derived.
//
// Covers: FR-8, FR-9, FR-14
func TestAcceptanceProgrammaticPostingMatchesTheForm(t *testing.T) {
	a := newApp(t)

	accountID := a.accounts()[0].ID
	postPath := "/api/accounts/" + url.PathEscape(accountID) + "/transactions"

	t.Run("a JSON post returns the created transaction and it appears in the log", func(t *testing.T) {
		res := a.postJSON(postPath, `{"amount":"-42.50","description":"Coffee beans"}`)
		if res.status != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body:\n%s", res.status, res.body)
		}
		if got, want := res.header.Get("Content-Type"), "application/json; charset=utf-8"; got != want {
			t.Errorf("Content-Type = %q, want %q", got, want)
		}

		var created apiTransaction
		decodeJSON(t, res.body, &created)
		if created.AmountCents != -4250 {
			t.Errorf("amount_cents = %d, want -4250", created.AmountCents)
		}
		if !txnID.MatchString(created.ID) {
			t.Errorf("id = %q, does not match ^txn-\\d{4}$", created.ID)
		}
		if created.AccountID != accountID || created.Description != "Coffee beans" || created.CreatedAt == "" {
			t.Errorf("created transaction = %+v, want the submitted values with a timestamp", created)
		}

		found := false
		for _, tx := range a.transactions(accountID) {
			if tx.ID == created.ID {
				found = true
				if tx != created {
					t.Errorf("listed transaction %+v differs from the created one %+v", tx, created)
				}
			}
		}
		if !found {
			t.Errorf("created transaction %s is not retrievable from the log", created.ID)
		}
	})

	t.Run("an unrecognized request field is ignored", func(t *testing.T) {
		res := a.postJSON(postPath, `{"amount":"1.00","description":"Extra field","surprise":true}`)
		if res.status != http.StatusCreated {
			t.Errorf("status = %d, want 201; body:\n%s", res.status, res.body)
		}
	})

	t.Run("a missing Content-Type takes the documented form branch", func(t *testing.T) {
		res := a.request(http.MethodPost, postPath, "",
			url.Values{"amount": {"1.00"}, "description": {"No content type"}}.Encode())
		if res.status != http.StatusSeeOther {
			t.Errorf("status = %d, want 303 (the form branch is the documented default); body:\n%s",
				res.status, res.body)
		}
	})

	t.Run("malformed JSON bodies are rejected as amount_malformed and record nothing", func(t *testing.T) {
		bodies := map[string]string{
			"not JSON at all":      `{"amount":`,
			"amount omitted":       `{"description":"No amount"}`,
			"amount as a number":   `{"amount":-42.50,"description":"Numeric amount"}`,
			"amount as null":       `{"amount":null,"description":"Null amount"}`,
			"body is a JSON array": `[]`,
		}

		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				countBefore := a.transactionCount(accountID)

				res := a.postJSON(postPath, body)
				if res.status != http.StatusBadRequest {
					t.Errorf("status = %d, want 400; body:\n%s", res.status, res.body)
				}
				if code := decodeError(t, res).Error.Code; code != "amount_malformed" {
					t.Errorf("code = %q, want %q", code, "amount_malformed")
				}
				if got := a.transactionCount(accountID); got != countBefore {
					t.Errorf("transaction count = %d after a rejection, want %d", got, countBefore)
				}
			})
		}
	})

	// FR-9, in executable form: the same invalid input, submitted both ways, is
	// rejected for the same rule. The codes come from the API response contract;
	// the sentinel behind each is asserted by the domain's own table (plan Task 18).
	//
	// Covers: FR-12, FR-12a, FR-12b, FR-12c, FR-12d, FR-12e
	t.Run("both content types reject the same input for the same rule", func(t *testing.T) {
		tooLong := strings.Repeat("d", 141)
		hugeDebit := "-99999999"

		cases := []struct {
			name        string
			account     string
			amount      string
			description string
			wantCode    string
			wantStatus  int
		}{
			{"unknown account", "acct-nope", "25", "Valid", "account_not_found", http.StatusNotFound},
			{"zero amount", accountID, "0", "Valid", "amount_zero", http.StatusBadRequest},
			{"whitespace description", accountID, "25", "   ", "description_empty", http.StatusBadRequest},
			{"description too long", accountID, "25", tooLong, "description_too_long", http.StatusBadRequest},
			{"malformed amount", accountID, "abc", "Valid", "amount_malformed", http.StatusBadRequest},
			{"balance would go negative", accountID, hugeDebit, "Valid", "balance_would_go_negative", http.StatusBadRequest},
		}

		seen := map[string]string{}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				path := "/api/accounts/" + url.PathEscape(tc.account) + "/transactions"
				countBefore := a.transactionCount(accountID)

				// JSON branch: typed code and documented status.
				jsonRes := a.postJSON(path, fmt.Sprintf(`{"amount":%q,"description":%q}`, tc.amount, tc.description))
				if jsonRes.status != tc.wantStatus {
					t.Errorf("JSON status = %d, want %d; body:\n%s", jsonRes.status, tc.wantStatus, jsonRes.body)
				}
				payload := decodeError(t, jsonRes)
				if payload.Error.Code != tc.wantCode {
					t.Errorf("JSON code = %q, want %q", payload.Error.Code, tc.wantCode)
				}
				if strings.TrimSpace(payload.Error.Message) == "" {
					t.Errorf("JSON message is empty; body:\n%s", jsonRes.body)
				}

				// Form branch: the same rule, carried as a code in the redirect.
				formRes := a.postForm(path, url.Values{
					"amount":      {tc.amount},
					"description": {tc.description},
				})
				if formRes.status != http.StatusSeeOther {
					t.Errorf("form status = %d, want 303; body:\n%s", formRes.status, formRes.body)
				}
				if !strings.Contains(formRes.location, "error="+tc.wantCode) {
					t.Errorf("form Location = %q, want it to carry error=%s", formRes.location, tc.wantCode)
				}

				if got := a.transactionCount(accountID); got != countBefore {
					t.Errorf("transaction count = %d after rejections on both branches, want %d", got, countBefore)
				}

				if prior, ok := seen[tc.wantCode]; ok {
					t.Errorf("code %q already reported by %q — the six codes must be pairwise distinct",
						tc.wantCode, prior)
				}
				seen[tc.wantCode] = tc.name
			})
		}
	})

	t.Run("an accepted submission is identical through both content types", func(t *testing.T) {
		jsonRes := a.postJSON(postPath, `{"amount":"3.50","description":"Parity probe"}`)
		if jsonRes.status != http.StatusCreated {
			t.Fatalf("JSON status = %d, want 201; body:\n%s", jsonRes.status, jsonRes.body)
		}
		var viaJSON apiTransaction
		decodeJSON(t, jsonRes.body, &viaJSON)

		formRes := a.postForm(postPath, url.Values{"amount": {"3.50"}, "description": {"Parity probe"}})
		if formRes.status != http.StatusSeeOther {
			t.Fatalf("form status = %d, want 303; body:\n%s", formRes.status, formRes.body)
		}
		viaForm := a.transactions(accountID)[0]

		if viaForm.AmountCents != viaJSON.AmountCents ||
			viaForm.Description != viaJSON.Description ||
			viaForm.AccountID != viaJSON.AccountID {
			t.Errorf("form-recorded %+v differs from JSON-recorded %+v in a persisted field", viaForm, viaJSON)
		}
		if viaJSON.AmountCents != 350 {
			t.Errorf("amount_cents = %d, want 350 (3.50 dollars parsed as cents)", viaJSON.AmountCents)
		}
	})

	t.Run("duplicate fields use last-value precedence consistently", func(t *testing.T) {
		cases := []struct {
			name            string
			jsonBody        string
			formBody        string
			wantCode        string
			wantAmount      int64
			wantDescription string
		}{
			{
				name:            "duplicate amount accepts the last value",
				jsonBody:        `{"amount":"0","amount":"4.25","description":"Last amount"}`,
				formBody:        "amount=0&amount=4.25&description=Last+amount",
				wantAmount:      425,
				wantDescription: "Last amount",
			},
			{
				name:     "duplicate description rejects the last empty value",
				jsonBody: `{"amount":"4.25","description":"First description","description":""}`,
				formBody: "amount=4.25&description=First+description&description=",
				wantCode: "description_empty",
			},
			{
				name:     "mixed-case amount does not bypass malformed amount validation",
				jsonBody: `{"Amount":"4.25","description":"Case variant"}`,
				formBody: "Amount=4.25&description=Case+variant",
				wantCode: "amount_malformed",
			},
			{
				name:     "exact amount key wins over case variant",
				jsonBody: `{"amount":"0","Amount":"4.25","description":"Exact key"}`,
				formBody: "amount=0&Amount=4.25&description=Exact+key",
				wantCode: "amount_zero",
			},
			{
				name:     "exact description key wins over case variant",
				jsonBody: `{"amount":"4.25","description":"   ","Description":"Valid description"}`,
				formBody: "amount=4.25&description=+++&Description=Valid+description",
				wantCode: "description_empty",
			},
			{
				name:     "exact duplicate amount still uses its last value",
				jsonBody: `{"amount":"0","Amount":"4.25","amount":"-99999999","description":"Exact duplicate"}`,
				formBody: "amount=0&Amount=4.25&amount=-99999999&description=Exact+duplicate",
				wantCode: "balance_would_go_negative",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				countBefore := a.transactionCount(accountID)
				jsonRes := a.postJSON(postPath, tc.jsonBody)
				formRes := a.request(http.MethodPost, postPath, "application/x-www-form-urlencoded", tc.formBody)

				if tc.wantCode != "" {
					if got := jsonRes.status; got != http.StatusBadRequest {
						t.Errorf("JSON status = %d, want 400; body:\n%s", got, jsonRes.body)
					}
					if got := decodeError(t, jsonRes).Error.Code; got != tc.wantCode {
						t.Errorf("JSON error code = %q, want %q", got, tc.wantCode)
					}
					if got := formRes.status; got != http.StatusSeeOther {
						t.Errorf("form status = %d, want 303; body:\n%s", got, formRes.body)
					}
					if !strings.Contains(formRes.location, "error="+tc.wantCode) {
						t.Errorf("form Location = %q, want error=%s", formRes.location, tc.wantCode)
					}
					if got := a.transactionCount(accountID); got != countBefore {
						t.Errorf("transaction count = %d after duplicate-key rejection, want %d", got, countBefore)
					}
					return
				}

				if got := jsonRes.status; got != http.StatusCreated {
					t.Fatalf("JSON status = %d, want 201; body:\n%s", got, jsonRes.body)
				}
				var viaJSON apiTransaction
				decodeJSON(t, jsonRes.body, &viaJSON)
				if got := formRes.status; got != http.StatusSeeOther {
					t.Fatalf("form status = %d, want 303; body:\n%s", got, formRes.body)
				}
				viaForm := a.transactions(accountID)[0]
				if viaJSON.AmountCents != tc.wantAmount || viaJSON.Description != tc.wantDescription ||
					viaForm.AmountCents != viaJSON.AmountCents || viaForm.Description != viaJSON.Description {
					t.Errorf("duplicate-key persistence differs: JSON=%+v form=%+v", viaJSON, viaForm)
				}
				if got := a.transactionCount(accountID); got != countBefore+2 {
					t.Errorf("transaction count = %d after accepted duplicate-key posts, want %d", got, countBefore+2)
				}
			})
		}
	})
}

// TestAcceptanceBalanceFloorHoldsAcrossPosts covers Story 5's cross-operation
// rule: a debit landing exactly on zero is accepted, and the next cent is not.
// Per-rule sentinel semantics belong to the domain table (plan Task 18).
//
// Covers: FR-12f
func TestAcceptanceBalanceFloorHoldsAcrossPosts(t *testing.T) {
	a := newApp(t)

	accountID := a.accounts()[0].ID
	postPath := "/api/accounts/" + url.PathEscape(accountID) + "/transactions"

	balance := a.balanceCents(accountID)
	if balance <= 0 {
		t.Fatalf("seeded balance for %s is %d, want a positive balance to draw down", accountID, balance)
	}

	// Landing exactly on zero is permitted.
	res := a.postJSON(postPath, fmt.Sprintf(`{"amount":%q,"description":"Draw down to zero"}`,
		dollarsFromCents(-balance)))
	if res.status != http.StatusCreated {
		t.Fatalf("drawing the balance to exactly zero gave %d, want 201; body:\n%s", res.status, res.body)
	}
	if got := a.balanceCents(accountID); got != 0 {
		t.Fatalf("balance = %d after drawing down, want exactly 0", got)
	}

	// One more cent is not.
	countBefore := a.transactionCount(accountID)
	res = a.postJSON(postPath, `{"amount":"-0.01","description":"One cent too far"}`)
	if res.status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body:\n%s", res.status, res.body)
	}
	if code := decodeError(t, res).Error.Code; code != "balance_would_go_negative" {
		t.Errorf("code = %q, want %q", code, "balance_would_go_negative")
	}
	if got := a.balanceCents(accountID); got != 0 {
		t.Errorf("balance = %d after the rejection, want 0", got)
	}
	if got := a.transactionCount(accountID); got != countBefore {
		t.Errorf("transaction count = %d after the rejection, want %d", got, countBefore)
	}

	// A one-cent credit is the smallest accepted amount, so zero is a floor and
	// not a freeze.
	if res := a.postJSON(postPath, `{"amount":"0.01","description":"Smallest credit"}`); res.status != http.StatusCreated {
		t.Errorf("one-cent credit status = %d, want 201; body:\n%s", res.status, res.body)
	}
	if got := a.balanceCents(accountID); got != 1 {
		t.Errorf("balance = %d, want 1", got)
	}
}

// TestAcceptanceResetAndRun covers Story 6: the two commands the presenter runs,
// and the guarantee that a redone take looks exactly like the first attempt.
//
// Covers: FR-15, FR-16
func TestAcceptanceResetAndRun(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reset.db")

	first := func() string {
		seedDB(t, dbPath)
		base, stop := startServer(t, dbPath)
		defer stop()
		return dumpState(newAppAt(t, dbPath, base))
	}()

	t.Run("a pristine reset holds the deterministic three-account fixture", func(t *testing.T) {
		base, stop := startServer(t, dbPath)
		defer stop()
		a := newAppAt(t, dbPath, base)

		accounts := a.accounts()
		if len(accounts) != 3 {
			t.Fatalf("seeded account count = %d, want 3", len(accounts))
		}

		var ids []string
		for i, acct := range accounts {
			txs := a.transactions(acct.ID)
			// FR-15, amended 2026-08-09: the first two accounts carry 8-12
			// transactions and the third is seeded EMPTY, so FR-4's empty state is
			// demonstrable from seed data with no setup. Accounts are ordered by id
			// ascending, asserted above, so index 2 is the empty one.
			if i < 2 {
				if len(txs) < 8 || len(txs) > 12 {
					t.Errorf("account %s has %d transactions, want 8-12", acct.ID, len(txs))
				}
			} else if len(txs) != 0 {
				t.Errorf("account %s has %d transactions, want 0 — the third account is seeded empty",
					acct.ID, len(txs))
			}
			for _, tx := range txs {
				ids = append(ids, tx.ID)
			}
		}
		if len(ids) < 16 || len(ids) > 24 {
			t.Errorf("seeded transaction count = %d, want 16-24", len(ids))
		}
		// Story 1 and .docs/decisions/api-response-contract.md both use 128350 cents
		// for the first account; pin it so those worked examples hold against seed
		// data rather than only against a fixture.
		if got := a.balanceCents(accounts[0].ID); got != 128350 {
			t.Errorf("seeded balance for %s = %d, want exactly 128350", accounts[0].ID, got)
		}

		emptyPage := a.get("/?account=acct-3")
		if got, want := emptyPage.status, http.StatusOK; got != want {
			t.Errorf("GET /?account=acct-3 status = %d, want %d", got, want)
		}
		mustContain(t, emptyPage.body, `class="balance">$0.00`, "seeded empty account page")
		mustContain(t, emptyPage.body, "No transactions.", "seeded empty account page")
		mustContain(t, emptyPage.body, `<form method="post" action="/api/accounts/acct-3/transactions">`, "seeded empty account page")

		// One unbroken global sequence, not a per-account restart.
		seen := map[string]bool{}
		for _, id := range ids {
			if !txnID.MatchString(id) {
				t.Errorf("seeded id %q does not match ^txn-\\d{4}$", id)
			}
			if seen[id] {
				t.Errorf("seeded id %q appears more than once — ids must be globally unique", id)
			}
			seen[id] = true
		}
		for n := 1; n <= len(ids); n++ {
			want := fmt.Sprintf("txn-%04d", n)
			if !seen[want] {
				t.Errorf("seeded ids are not one unbroken sequence: %s is missing from %d ids", want, len(ids))
			}
		}
	})

	t.Run("resetting twice reproduces byte-identical state", func(t *testing.T) {
		for _, suffix := range []string{"", "-shm", "-wal"} {
			if err := os.Remove(dbPath + suffix); err != nil && !os.IsNotExist(err) {
				t.Fatalf("removing %s: %v", dbPath+suffix, err)
			}
		}

		seedDB(t, dbPath)
		base, stop := startServer(t, dbPath)
		defer stop()

		if second := dumpState(newAppAt(t, dbPath, base)); second != first {
			t.Errorf("a second reset produced different state.\nfirst:\n%s\nsecond:\n%s", first, second)
		}
	})

	t.Run("the served page is fully offline", func(t *testing.T) {
		base, stop := startServer(t, dbPath)
		defer stop()
		a := newAppAt(t, dbPath, base)

		body := a.get("/").body
		for _, forbidden := range []string{"http://", "https://", "<script"} {
			mustNotContain(t, body, forbidden, "rendered page")
		}

		css := a.get("/style.css")
		if css.status != http.StatusOK {
			t.Fatalf("GET /style.css status = %d, want 200", css.status)
		}
		for _, forbidden := range []string{"@import", "@font-face", "@media", "prefers-color-scheme", "@keyframes"} {
			mustNotContain(t, css.body, forbidden, "stylesheet")
		}
		mustContain(t, css.body, ".balance", "stylesheet")
		mustContain(t, css.body, ".error", "stylesheet")
	})
}

// TestAcceptanceCommandLineFailures covers Story 6's negative paths at the level a
// presenter meets them: the command exits non-zero and says why. Kept separate
// from the reset flow so a missing route never masks a CLI regression.
//
// Covers: FR-15, FR-16
func TestAcceptanceCommandLineFailures(t *testing.T) {
	t.Run("seeding into a directory that does not exist fails loudly", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-such-dir", "ledger.db")

		cmd := exec.Command(serverBin, "seed")
		cmd.Env = append(os.Environ(), "LEDGER_DB_PATH="+missing)
		out, err := cmd.CombinedOutput()

		if err == nil {
			t.Errorf("seed exited 0 for an unwritable path; output:\n%s", out)
		}
		if !strings.Contains(string(out), missing) {
			t.Errorf("seed failure does not name the path %q; output:\n%s", missing, out)
		}
		if _, statErr := os.Stat(missing); statErr == nil {
			t.Errorf("seed created a stray file at %s", missing)
		}
	})

	t.Run("an unknown subcommand exits non-zero naming the valid commands", func(t *testing.T) {
		cmd := exec.Command(serverBin, "frobnicate")
		cmd.Env = append(os.Environ(), "LEDGER_DB_PATH="+filepath.Join(t.TempDir(), "unused.db"))
		out, err := cmd.CombinedOutput()

		if err == nil {
			t.Errorf("an unknown subcommand exited 0; output:\n%s", out)
		}
		for _, want := range []string{"serve", "seed"} {
			mustContain(t, string(out), want, "unknown-subcommand message")
		}
	})
}

// --- local helpers ---

// dumpState renders every account and every transaction as a canonical string, so
// two resets can be compared including ids and timestamps.
func dumpState(a *app) string {
	a.t.Helper()

	var b strings.Builder
	for _, acct := range a.accounts() {
		fmt.Fprintf(&b, "account %s %q %d\n", acct.ID, acct.Name, acct.BalanceCents)
		for _, tx := range a.transactions(acct.ID) {
			fmt.Fprintf(&b, "  txn %s %s %d %q %s\n",
				tx.ID, tx.AccountID, tx.AmountCents, tx.Description, tx.CreatedAt)
		}
	}
	return b.String()
}

// dollarsFromCents renders cents as the dollars-and-cents string a person types,
// with no float arithmetic.
func dollarsFromCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func between(body, after, before string) string {
	start := strings.Index(body, after)
	if start < 0 {
		return ""
	}
	start += len(after)
	end := strings.Index(body[start:], before)
	if end < 0 {
		return body[start:]
	}
	return body[start : start+end]
}

// stripTags removes markup so a panel's visible text can be checked for emptiness.
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(b.String())
}
