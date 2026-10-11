// Covers: S6.2, S6.3, S6.4, S6.5, FR-7, FR-8
package acceptance

import (
	"html"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestAcceptanceOpenAccountFormLifecycle drives the presenter's complete
// open-account loop through the running production server.
func TestAcceptanceOpenAccountFormLifecycle(t *testing.T) {
	a := newApp(t)

	opened := a.postForm("/api/accounts", url.Values{"name": {"Holiday Fund"}})
	if opened.status != http.StatusSeeOther {
		t.Fatalf("open-account form POST status = %d, want 303; body:\n%s", opened.status, opened.body)
	}
	if opened.location != "/?account=acct-4" {
		t.Fatalf("Location = %q, want %q", opened.location, "/?account=acct-4")
	}

	page := a.get(opened.location)
	if page.status != http.StatusOK {
		t.Fatalf("following the open-account redirect gave %d, want 200", page.status)
	}
	mustContain(t, page.body, `<a href="/?account=acct-4">`+html.EscapeString("Holiday Fund")+`</a>`, "account navigation")
	mustContain(t, page.body, `class="selected-account">Selected account: Holiday Fund`, "selected account")
	mustContain(t, page.body, `class="balance">$0.00`, "new account balance")
	mustContain(t, page.body, "No transactions.", "new account transaction list")

	// Reloading the redirected GET must not repeat the account-opening POST.
	a.get(opened.location)
	if got := len(a.accounts()); got != 4 {
		t.Fatalf("account count after reloading the result = %d, want 4", got)
	}

	posted := a.postForm("/api/accounts/acct-4/transactions", url.Values{
		"amount":      {"25.00"},
		"description": {"First deposit"},
	})
	if posted.status != http.StatusSeeOther {
		t.Fatalf("transaction form POST status = %d, want 303; body:\n%s", posted.status, posted.body)
	}
	if posted.location != "/?account=acct-4" {
		t.Fatalf("transaction Location = %q, want %q", posted.location, "/?account=acct-4")
	}

	page = a.get(posted.location)
	mustContain(t, page.body, `class="selected-account">Selected account: Holiday Fund`, "selected account after posting")
	mustContain(t, page.body, `class="balance">$25.00`, "new account balance after posting")
}

// TestAcceptanceOpenAccountFormRejection proves that a rejected page flow
// carries the rule to the form that produced it without opening an account.
func TestAcceptanceOpenAccountFormRejection(t *testing.T) {
	a := newApp(t)

	rejected := a.postForm("/api/accounts", url.Values{"name": {"   "}})
	if rejected.status != http.StatusSeeOther {
		t.Fatalf("blank open-account form POST status = %d, want 303; body:\n%s", rejected.status, rejected.body)
	}
	if rejected.location != "/?error=account_name_empty" {
		t.Fatalf("Location = %q, want %q", rejected.location, "/?error=account_name_empty")
	}

	page := a.get(rejected.location)
	postingFormAt := strings.Index(page.body, `<form method="post" action="/api/accounts/acct-1/transactions">`)
	messageAt := strings.Index(page.body, "Account name must not be empty.")
	openFormAt := strings.Index(page.body, `<form method="post" action="/api/accounts">`)
	if postingFormAt < 0 || messageAt < 0 || openFormAt < 0 || !(postingFormAt < messageAt && messageAt < openFormAt) {
		t.Errorf("account-name rejection must appear after the posting form and before the open-account form; body:\n%s", page.body)
	}
	if got := len(a.accounts()); got != 3 {
		t.Errorf("account count after rejected opening = %d, want 3", got)
	}
}
