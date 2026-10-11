# Stories — Open a New Account

**Status:** Accepted

**Feature:** open-a-new-account · **Tier:** M · **Track:** product
**Source:** `.docs/specs/2026-10-10-open-a-new-account.md` (Approved, FR-1 … FR-9)
**Constrained by:** `.docs/decisions/adr-2026-10-10-sequential-account-ids-ordered-by-sequence.md`
(APPROVED), `.docs/decisions/api-response-contract.md` (amended 2026-10-10),
`.docs/decisions/adr-2026-08-09-rejection-message-composition.md`,
`.docs/decisions/adr-2026-08-08-one-negotiated-posting-endpoint.md`

Seven stories. Stories 1–3 are the domain capability and are provable without any web surface;
Stories 4–7 are the programmatic and page surfaces built on it. "The seeded ledger" below means the
state after `make reset`: accounts `acct-1` Everyday Checking, `acct-2` Weekend Savings, `acct-3`
Project Fund.

## Negative-path categories evaluated

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — blank and over-long names; malformed or non-string JSON; tampered `detail` |
| Data integrity | **Yes** — a rejection records nothing; ids continue the sequence without gaps |
| Dependency unavailability | **Yes** — a storage failure while counting or inserting returns an error and records nothing |
| Invariant side-effect on alternate branches | **Yes** — the JSON and form branches must open the same account and reject with the same message |
| Concurrent access | No — single presenter; the read-then-write id derivation is a documented known property (ADR) |
| Auth / permission failures | No — explicit non-goal |
| Timeouts & network errors | No — no network calls |
| Resource exhaustion | **Yes, bounded** — the 40-character limit and the validated `detail` |
| Partial failure & rollback | No — one insert, nothing to roll back |
| Cascade deletion effects | No — nothing is deleted |
| Model-level immutability | No — accounts are not edited by this feature |
| Exception class hierarchy | No — sentinel errors compared with `errors.Is` |
| Dedup / idempotency key analysis | No — no uniqueness or dedup criterion exists, and adding one is a non-goal |

---

## Story 1: Open an account in the domain

**Requirement:** FR-1, FR-2

As a presenter, I want the ledger to open a named, empty account with the next id in sequence so that a new account behaves exactly like a seeded one.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when an account named "Holiday Fund" is opened, then the returned account has id `acct-4` and name "Holiday Fund", the store lists four accounts, its derived balance is 0, and it has no transactions.
- Given the seeded ledger after opening "Holiday Fund", when a second account named "Holiday Fund" is opened, then it is returned with id `acct-5`, both accounts exist, and neither opening is rejected for sharing a name.
- Given a ledger with no accounts, when an account named "First" is opened, then it is returned with id `acct-1`.

#### Negative Paths
- Given a store whose account listing fails, when an account is opened, then an error wrapping the storage failure is returned, no account is returned, and nothing is inserted.
- Given a store whose insert fails, when an account is opened, then an error wrapping the storage failure is returned and no account is returned.

### Done When
- [ ] Opening through the real SQLite store (in-memory) after seeding the three demo accounts returns `acct-4`, and `Account("acct-4")` reads back the same name.
- [ ] Two openings under the same fixed state always yield the same ids across runs (no time or randomness in the id).
- [ ] Both storage-failure cases are covered against a failing fake store and return a non-sentinel wrapped error.

## Story 2: Reject a blank account name

**Requirement:** FR-3

As a presenter, I want a blank name refused so that the account list never shows an unnamed account.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when an account named "  Rainy Day  " (surrounding spaces) is opened, then it is accepted and stored with the name exactly as submitted, spaces included.

#### Negative Paths
- Given the seeded ledger, when an account is opened with an empty name, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts.
- Given the seeded ledger, when an account is opened with a name of only spaces, tabs, and a newline, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts.

### Done When
- [ ] A table-driven test covers empty, spaces-only, and mixed-whitespace names, each asserting the sentinel and an unchanged account count.
- [ ] The blank check runs before the store is asked to count or insert (a failing fake store is never reached).

## Story 3: Limit account names to 40 characters

**Requirement:** FR-4

As a presenter, I want over-long names refused so that every account name fits the navigation bar on a projector.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when an account is opened with a name of exactly 40 ASCII characters, then it is accepted as `acct-4`.
- Given the seeded ledger, when an account is opened with a name of exactly 40 "é" characters (80 bytes), then it is accepted, because length counts characters, not bytes.

#### Negative Paths
- Given the seeded ledger, when an account is opened with a name of 41 ASCII characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and the store still lists three accounts.
- Given the seeded ledger, when an account is opened with a name of 41 "é" characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and nothing is recorded.

### Done When
- [ ] The 40/41 boundary is tested for both single-byte and multi-byte names.
- [ ] Neither sentinel is ever compared by its string value in any test.

## Story 4: Open an account with a JSON request

**Requirement:** FR-5

As a programmatic client, I want to open an account with one JSON request so that a `curl` on stage shows the new account immediately.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when `POST /api/accounts` is sent with `Content-Type: application/json` and body `{"name":"Holiday Fund"}`, then the response is `201` with `Content-Type: application/json; charset=utf-8` and body `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`.
- Given that account was just opened, when `GET /api/accounts` is requested, then its fourth element is `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}` and `GET /api/accounts/acct-4/transactions` returns `[]`.
- Given that account was just opened, when a JSON transaction of amount "25.00" is posted to `/api/accounts/acct-4/transactions`, then it is accepted with `201` and `acct-4` lists a balance of 2500 cents.

#### Negative Paths
- Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `not json`, then the response is `400` with code `account_name_empty` and `GET /api/accounts` still lists three accounts.
- Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `{"name":42}` or `{}`, then the response is `400` with code `account_name_empty` and nothing is recorded.
- Given the running server, when `PUT /api/accounts` is sent, then the response is `405` with an empty body.

### Done When
- [ ] The router test asserts the exact `201` status, content type, and JSON body for the happy path.
- [ ] The malformed, non-string, and missing-name bodies are each asserted to return `400` `account_name_empty` with an unchanged account count.

## Story 5: JSON rejections name the broken rule

**Requirement:** FR-6

As a programmatic client, I want each rejected name to return a stable code and a readable message so that the cause is clear on screen.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when `POST /api/accounts` is sent as JSON with name "   ", then the response is `400` with body `{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}`.
- Given the seeded ledger, when `POST /api/accounts` is sent as JSON with a 41-character name, then the response is `400` with body `{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}`.

#### Negative Paths
- Given either rejection above, when `GET /api/accounts` is requested afterwards, then it still lists exactly three accounts and the next successful opening is `acct-4` (no id was consumed).
- Given the existing seven error codes, when this feature ships, then each still returns its original HTTP status and code string for its original rule.

### Done When
- [ ] Both new codes are asserted by exact code string and with `errors.Is` on their sentinels at the mapping boundary.
- [ ] The existing error-code mapping test still passes unchanged for the original seven codes.

## Story 6: Open an account from the page

**Requirement:** FR-7, FR-8

As a presenter, I want an "Open account" form on the page so that I can create an account on stage without a terminal.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger, when `GET /` is requested, then the page contains an "Open account" form that posts a `name` field to `/api/accounts`, placed after the transactions section so the existing posting form is still the first form on the page, and its name input carries no `autofocus`.
- Given the seeded ledger, when the open-account form is submitted with name "Holiday Fund", then the response is `303 See Other` with `Location: /?account=acct-4`, and following it shows "Holiday Fund" in the account navigation, "Selected account: Holiday Fund", balance "$0.00", and "No transactions.".
- Given an account was opened from the page, when a transaction is then posted to it from the posting form, then the page shows the new account selected with the posted amount as its balance.
- Given an account was just opened from the page, when the resulting page is reloaded, then the ledger still holds exactly one new account.

#### Negative Paths
- Given the seeded ledger, when the open-account form is submitted with a blank name, then the response is `303` to `/?error=account_name_empty`, the page shows "Account name must not be empty." above the open-account form and not above the posting form, and three accounts remain.
- Given the seeded ledger, when the open-account form is submitted with a 41-character name, then the redirect is `/?error=account_name_too_long&detail=41` and the page shows "Account name is too long. Submitted: 41 characters; the limit is 40." above the open-account form.
- Given the page is requested with `/?error=account_name_too_long&detail=abc` or `detail=12`, when it renders, then the message is exactly "Account name is too long." with no detail sentence.
- Given `GET /?account=acct-nope` or a ledger with no accounts, when the page renders, then it contains no `<form` at all, exactly as before this feature.
- Given the open-account form is submitted with name `<script>alert(1)</script>`, when the resulting page renders, then the account is opened and its name appears escaped as text, never as markup.

### Done When
- [ ] An end-to-end acceptance flow against the running server opens an account through the form, follows the redirect, and asserts the selected account and `$0.00`.
- [ ] Router tests assert both rejection redirects exactly and the message placement relative to the two forms.

## Story 7: Accounts are listed in the order they were opened

**Requirement:** FR-9

As a presenter, I want the account list to stay in opening order however many accounts I open so that the navigation never looks shuffled.

### Acceptance Criteria

#### Happy Path
- Given the seeded ledger plus nine opened accounts (`acct-4` … `acct-12`), when `GET /api/accounts` is requested, then the ids are exactly `acct-1`, `acct-2`, … `acct-12` in numeric order.
- Given the same twelve accounts, when `GET /` is requested, then the account navigation links appear in the same numeric order, with `acct-10` after `acct-9`.

#### Negative Paths
- Given only the three seeded accounts, when `GET /api/accounts` and `GET /` are requested, then the order is `acct-1`, `acct-2`, `acct-3`, identical to today.
- Given twelve accounts, when `GET /` is requested with no `account` parameter, then the selected account is still `acct-1`, not whichever id sorts first as text.

### Done When
- [ ] A domain test of the ordering rule covers `acct-1`, `acct-2`, `acct-9`, `acct-10`, `acct-12` given in shuffled order.
- [ ] Router tests assert numeric order on both `GET /api/accounts` and the page navigation with at least ten accounts.
