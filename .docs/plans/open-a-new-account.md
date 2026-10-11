# Implementation Plan: Open a New Account

**Date:** 2026-10-10
**Design:** [2026-10-10-open-a-new-account.md](../specs/2026-10-10-open-a-new-account.md)
**Stories:** [open-a-new-account.md](../stories/open-a-new-account.md)
**Conflict check:** Clean as of 2026-10-10 (`.docs/conflicts/2026-10-10-open-a-new-account.md` — 2 blocking resolved in DECIDE, 1 degrading accepted)
**Architecture review:** APPROVED (`.docs/decisions/architecture-review-2026-10-10-open-a-new-account.md`)
**Tier:** M (`.docs/complexity/open-a-new-account.md`, `Stacked-Delivery: approved`)

## Summary

Lets a presenter open a named, empty account — first as a domain capability with its own tests
(slice 1, Tasks 1–5), then through `POST /api/accounts` and an "Open account" form on the page, with
account listings ordered by id sequence (slice 2, Tasks 6–12). Twelve TDD tasks.

## Slices

| Slice | Title | Tasks |
| --- | --- | --- |
| 1 | Domain account opening | 1, 2, 3, 4, 5 |
| 2 | Open accounts over HTTP and on the page | 6, 7, 8, 9, 10, 11, 12 |

Slice 1 owns Stories 1–3; slice 2 owns Stories 4–7. No story has a task in both slices, and no slice 1
task depends on a slice 2 task.

## Technical Approach

- **Domain first (slice 1).** `internal/ledger/open.go` adds `OpenAccount(store Store, name string) (Account, error)`.
  It validates the name (blank check with `strings.TrimSpace`, length with `utf8.RuneCountInString`,
  limit 40), then lists accounts, derives the id as `fmt.Sprintf("acct-%d", len(accounts)+1)`, and
  records through a new `Store.InsertAccount(Account) error` interface method. `store.SQLite` already
  implements `InsertAccount` (used by seeding), so the store package changes only in its tests. The two
  test fakes that implement `ledger.Store` (`fakeStore` in `internal/ledger/store_test.go` and
  `routerTestStore` in `internal/httpapi/router_test.go`) gain the method in Task 1 so every package
  still compiles. Sentinels `ErrAccountNameEmpty` and `ErrAccountNameTooLong` join
  `internal/ledger/errors.go`; every error is wrapped `fmt.Errorf("opening account: %w", err)`, exactly
  like `PostTransaction`'s `"posting transaction: %w"`. Name is stored exactly as submitted.
- **HTTP second (slice 2).** `POST /api/accounts` is registered in `httpapi.NewRouter` and follows the
  posting route's pattern (`adr-2026-08-08-one-negotiated-posting-endpoint.md`): negotiate on request
  `Content-Type` only (`application/json` → JSON branch, anything else → form branch); JSON success is
  `201` with the `accountResponse` shape and `balance_cents: 0`; form success is `303` to
  `/?account=«id»`. A JSON body that does not decode, or whose `name` is absent or not a string, is
  rejected as `account_name_empty` (ADR D4). Rejections go through `codeFor`/`messageFor`/`writeJSONError`
  on the JSON branch and an `error`/`detail` redirect on the form branch, reusing
  `characterCountCarriedValue`-style validation with threshold 40 for the too-long detail. Search hints:
  `handlePostTransaction`, `postRequest`, `writePostError`, `postRedirectDetail` in
  `internal/httpapi/router.go`; `messageFor` and `characterCountCarriedValue` in
  `internal/httpapi/message.go`.
- **Page.** `web/index.html.tmpl` renders the open-account form only in the selected-account branch,
  after the Transactions section, so the posting form stays the first `<form` (several existing tests
  locate it that way) and the unknown- and zero-account branches keep rendering no form. `pageData`
  gains a field for the open-account message; `account_name_*` codes populate it instead of the existing
  `ErrorMessage`, so the message renders directly above the open-account form and never above the
  posting form. The name input has `required` and no `autofocus`.
- **Ordering.** `ledger.SortAccounts([]Account)` sorts by id length, then id text — numeric order for the
  shared `acct-` prefix (ADR D2). `handleAccounts` and `handlePage` call it in place of their
  `sort.Slice` string comparisons.
- **Existing tests that change with the route (Task 6).** `TestNewRouterDeclaresExactlyFiveRoutes`
  becomes a six-route assertion, and the acceptance subtest "POST /api/accounts is not allowed" becomes a
  `PUT`/`DELETE` → `405` check, matching the resolved base-ledger story.

## Prerequisites

- None. No schema change, no dependency, no seed change.

## Tasks

### Task 1: Open an account in the domain with the next sequential id
**Story:** Story 1
**Type:** happy-path

**Steps:**
1. Write failing table-driven tests in `internal/ledger/open_test.go` against an in-memory fake store that records inserted accounts: seeded three accounts → `OpenAccount(store, "Holiday Fund")` returns `{acct-4, Holiday Fund}`; opening "Holiday Fund" again returns `acct-5` with both stored; an empty store returns `acct-1`; `Balance(store, "acct-4")` is 0 and its transactions are empty.
2. Verify RED (`OpenAccount` and `Store.InsertAccount` do not exist).
3. Implement: add `InsertAccount(Account) error` to the `Store` interface in `internal/ledger/store.go`; add `OpenAccount` in `internal/ledger/open.go` (list accounts, id `acct-%d` from count+1, insert, return); add `InsertAccount` to `fakeStore` in `internal/ledger/store_test.go` and to `routerTestStore` in `internal/httpapi/router_test.go`.
4. Verify GREEN with `go test ./...`, then commit.

**Done when:**
- [test] `OpenAccount` over a fake store seeded with acct-1..acct-3 returns id `acct-4` and name "Holiday Fund", the fake then lists four accounts, `Balance` for acct-4 returns 0, and its transaction list is empty.
- [test] A second `OpenAccount` with the same name "Holiday Fund" returns `acct-5` with no error, and the fake lists both acct-4 and acct-5.
- [test] `OpenAccount` over an empty fake store returns id `acct-1`.
- `go build ./...` and `go vet ./...` succeed with `InsertAccount` declared on `ledger.Store` and implemented by `fakeStore`, `routerTestStore`, and `store.SQLite`.

**Files likely touched:**
- `internal/ledger/store.go` — `InsertAccount` on the interface
- `internal/ledger/open.go` — `OpenAccount`
- `internal/ledger/open_test.go` — happy-path tests
- `internal/ledger/store_test.go` — `fakeStore.InsertAccount`
- `internal/httpapi/router_test.go` — `routerTestStore.InsertAccount`

**Dependencies:** none

### Task 2: Storage failures while opening return a wrapped error and record nothing
**Story:** Story 1
**Type:** negative-path

**Steps:**
1. Write failing tests in `internal/ledger/open_test.go` with a fake whose `Accounts` returns an error, and one whose `InsertAccount` returns an error.
2. Verify RED, then make `OpenAccount` wrap both failures as `opening account: %w` and return a zero `Account`.
3. Verify GREEN, then commit.

**Done when:**
- [test] When the fake's `Accounts` fails, `OpenAccount` returns a zero `Account` and an error for which `errors.Is(err, storageErr)` holds and neither name sentinel matches, and the fake records zero `InsertAccount` calls.
- [test] When the fake's `InsertAccount` fails, `OpenAccount` returns a zero `Account` and an error for which `errors.Is(err, insertErr)` holds.

**Files likely touched:**
- `internal/ledger/open.go` — error wrapping
- `internal/ledger/open_test.go` — failing-store tests

**Dependencies:** Task 1

### Task 3: Opening through the real SQLite store
**Story:** Story 1
**Type:** infrastructure

**Steps:**
1. Write a failing test in `internal/store/sqlite_test.go`: open `:memory:`, insert the three demo accounts with `InsertAccount`, call `ledger.OpenAccount(db, "Holiday Fund")`, then read it back with `db.Account("acct-4")`; repeat the whole sequence on a second fresh database.
2. Verify RED (fails until Task 1's `OpenAccount` exists in this build), GREEN, commit.

**Done when:**
- [test] On an in-memory `store.SQLite` holding acct-1..acct-3, `ledger.OpenAccount` returns `acct-4` and `db.Account("acct-4")` returns name "Holiday Fund".
- [test] Two fresh in-memory databases given the same three accounts and the same opening produce the identical id `acct-4`, with no clock or random input to `OpenAccount`.

**Files likely touched:**
- `internal/store/sqlite_test.go` — integration test

**Dependencies:** Task 1

### Task 4: Reject a blank account name
**Story:** Story 2
**Type:** negative-path

**Steps:**
1. Write failing table-driven tests in `internal/ledger/open_test.go`: `""`, `"   "`, and `" \t\n"` against a seeded fake; `"  Rainy Day  "` accepted.
2. Verify RED; add `ErrAccountNameEmpty` to `internal/ledger/errors.go`; in `OpenAccount`, reject `strings.TrimSpace(name) == ""` before calling the store.
3. Verify GREEN, then commit.

**Done when:**
- [test] For names `""`, `"   "`, and `" \t\n"`, `OpenAccount` returns an error satisfying `errors.Is(err, ledger.ErrAccountNameEmpty)`, the fake still lists three accounts, and the fake records zero `Accounts` and zero `InsertAccount` calls.
- [test] `OpenAccount(store, "  Rainy Day  ")` succeeds and the stored account's name equals `"  Rainy Day  "` byte for byte.

**Files likely touched:**
- `internal/ledger/errors.go` — `ErrAccountNameEmpty`
- `internal/ledger/open.go` — blank check
- `internal/ledger/open_test.go` — blank-name tests

**Dependencies:** Task 2

### Task 5: Limit account names to 40 characters
**Story:** Story 3
**Type:** negative-path

**Steps:**
1. Write failing table-driven tests in `internal/ledger/open_test.go` for 40 and 41 ASCII characters and 40 and 41 `é` characters.
2. Verify RED; add `ErrAccountNameTooLong`; reject `utf8.RuneCountInString(name) > 40` before calling the store.
3. Verify GREEN, then commit.

**Done when:**
- [test] A 40-ASCII-character name is accepted as `acct-4`, and a name of 40 `é` characters (80 bytes) is accepted.
- [test] A 41-ASCII-character name and a 41-`é` name each return an error satisfying `errors.Is(err, ledger.ErrAccountNameTooLong)`, and after each the fake still lists three accounts with zero `InsertAccount` calls.
- No test in `internal/ledger` compares either name sentinel by its `Error()` string; all comparisons use `errors.Is`.

**Files likely touched:**
- `internal/ledger/errors.go` — `ErrAccountNameTooLong`
- `internal/ledger/open.go` — length check
- `internal/ledger/open_test.go` — boundary tests

**Dependencies:** Task 4

### Task 6: `POST /api/accounts` opens an account for a JSON client
**Story:** Story 4
**Type:** happy-path

**Steps:**
1. Write failing router tests in `internal/httpapi/router_test.go`: JSON `POST /api/accounts` with `{"name":"Holiday Fund"}` on the seeded test store; then `GET /api/accounts`, `GET /api/accounts/acct-4/transactions`, and a JSON post of amount "25.00" to acct-4.
2. Verify RED; register `mux.HandleFunc("POST /api/accounts", handleOpenAccount(store))` in `NewRouter`; implement the JSON branch (decode `{"name"}`, call `ledger.OpenAccount`, `201` with `accountResponse{ID, Name, BalanceCents: 0}`), following the posting handler's negotiation on `Content-Type` only.
3. Change `TestNewRouterDeclaresExactlyFiveRoutes` to expect six registrations, and change the acceptance subtest "POST /api/accounts is not allowed" in `test/acceptance/ledger_acceptance_test.go` to assert `PUT` and `DELETE` on `/api/accounts` return `405` with an empty body.
4. Verify GREEN with `go test ./...`, then commit.

**Done when:**
- [test] Through `NewRouter`, a JSON `POST /api/accounts` with `{"name":"Holiday Fund"}` returns `201`, `Content-Type: application/json; charset=utf-8`, and body exactly `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`.
- [test] After that request, `GET /api/accounts` returns four elements whose fourth is `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`, and `GET /api/accounts/acct-4/transactions` returns `[]`.
- [test] After that request, a JSON post of amount "25.00" to `/api/accounts/acct-4/transactions` returns `201` and `GET /api/accounts` then reports acct-4 with `balance_cents` 2500.
- [test] The route-count test asserts exactly six `mux` registrations in `NewRouter`, and the acceptance suite asserts `PUT` and `DELETE` on `/api/accounts` return `405` with an empty body.

**Files likely touched:**
- `internal/httpapi/router.go` — route and JSON handler
- `internal/httpapi/router_test.go` — JSON happy-path tests, route count
- `test/acceptance/ledger_acceptance_test.go` — replace the `POST /api/accounts` 405 subtest

**Dependencies:** Task 1

### Task 7: Malformed JSON bodies and unsupported methods
**Story:** Story 4
**Type:** negative-path

**Steps:**
1. Write failing router tests for JSON bodies `not json`, `{"name":42}`, and `{}`, and for `PUT /api/accounts`.
2. Verify RED; in the JSON branch, treat a decode failure or a missing/non-string `name` as `ledger.ErrAccountNameEmpty` and write it with `writeJSONError`.
3. Verify GREEN, then commit.

**Done when:**
- [test] JSON `POST /api/accounts` with body `not json`, with `{"name":42}`, and with `{}` each returns `400` with error code `account_name_empty`, and `GET /api/accounts` afterwards still returns exactly three accounts.
- [test] `PUT /api/accounts` through `NewRouter` returns `405` with an empty body.

**Files likely touched:**
- `internal/httpapi/router.go` — body decoding rules
- `internal/httpapi/router_test.go` — malformed-body tests

**Dependencies:** Task 6, Task 8

> **Amended 2026-10-11 by operator review — build halt, cyclic Task 7/8 dependency:** Task 7 asserts the `account_name_empty` code that only Task 8 maps, so Task 7 now depends on Task 8 and Task 8 no longer depends on Task 7.

### Task 8: Codes and messages for the two name rules
**Story:** Story 5
**Type:** negative-path

**Steps:**
1. Write failing tests in `internal/httpapi/errors_test.go` and `internal/httpapi/router_test.go`: JSON name `"   "` and a 41-character name; the existing seven-code mapping table unchanged; a successful opening after both rejections.
2. Verify RED; map `ErrAccountNameEmpty` → `{400, "account_name_empty"}` and `ErrAccountNameTooLong` → `{400, "account_name_too_long"}` in `codeFor`; add both identifiers to `messageFor` ("Account name must not be empty."; "Account name is too long." enriched with "Submitted: «n» characters; the limit is 40." when the carried count validates as 1–6 ASCII digits greater than 40); pass the rune count as context on the JSON branch.
3. Verify GREEN, then commit.

**Done when:**
- [test] JSON `POST /api/accounts` with name `"   "` returns `400` with body exactly `{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}`.
- [test] JSON `POST /api/accounts` with a 41-character name returns `400` with body exactly `{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}`.
- [test] After both rejections `GET /api/accounts` returns exactly three accounts, and the next successful JSON opening returns id `acct-4`.
- [test] `codeFor` maps both new sentinels (matched with `errors.Is` on wrapped errors) to their codes with status 400, and the existing table for the original seven sentinels passes with every status and code string unchanged.

**Files likely touched:**
- `internal/httpapi/errors.go` — two mappings
- `internal/httpapi/message.go` — two messages and the 40-limit count validator
- `internal/httpapi/errors_test.go` — mapping tests
- `internal/httpapi/router.go` — rejection context on the JSON branch
- `internal/httpapi/router_test.go` — JSON rejection body tests

**Dependencies:** Task 5, Task 6

> **Amended 2026-10-11 by operator review — build halt, cyclic Task 7/8 dependency:** Task 7 asserts the `account_name_empty` code that only Task 8 maps, so Task 7 now depends on Task 8 and Task 8 no longer depends on Task 7.

### Task 9: The page's open-account form opens and selects the account
**Story:** Story 6
**Type:** happy-path

**Steps:**
1. Write failing router tests in `internal/httpapi/router_test.go` for `GET /` markup, a form-encoded `POST /api/accounts` with `name=Holiday+Fund`, following its redirect, re-requesting the redirected page, and posting a transaction to the new account from the posting form.
2. Verify RED; implement the form branch of the handler (`303 See Other` to `/?account=«id»`) and render the open-account form in `web/index.html.tmpl` inside the selected-account branch, after the Transactions section: `<form method="post" action="/api/accounts">` with a required `name` input and no `autofocus`.
3. Verify GREEN, then commit.

**Done when:**
- [test] `GET /` on the seeded store contains a form with `action="/api/accounts"` and an input named `name`, positioned after the Transactions section so that the first `<form` in the body is still the posting form, and the name input carries no `autofocus` attribute.
- [test] A form-encoded `POST /api/accounts` with `name=Holiday+Fund` returns `303` with `Location: /?account=acct-4`, and `GET /?account=acct-4` shows "Holiday Fund" among the navigation links, "Selected account: Holiday Fund", `class="balance">$0.00`, and "No transactions.".
- [test] Requesting `GET /?account=acct-4` a second time leaves `GET /api/accounts` at exactly four accounts.
- [test] A form-encoded post of amount "25.00" to `/api/accounts/acct-4/transactions` redirects to `/?account=acct-4`, which shows "Selected account: Holiday Fund" and `class="balance">$25.00`.

**Files likely touched:**
- `internal/httpapi/router.go` — form branch
- `internal/httpapi/router_test.go` — page form tests
- `web/index.html.tmpl` — open-account form

**Dependencies:** Task 6

### Task 10: Page rejections render above the open-account form
**Story:** Story 6
**Type:** negative-path

**Steps:**
1. Write failing router tests for form-encoded blank and 41-character names, `GET /?error=account_name_too_long&detail=abc` and `detail=12`, `GET /?account=acct-nope`, the zero-account page, and a form opening with name `<script>alert(1)</script>`.
2. Verify RED; on form rejection redirect to `/?error=«code»` (plus `&detail=«rune count»` for too-long, using the Task 8 validator); add an open-account message field to `pageData` filled only for `account_name_*` codes, rendered directly above the open-account form, leaving `ErrorMessage` for every other code.
3. Verify GREEN, then commit.

**Done when:**
- [test] A form-encoded blank name returns `303` to `/?error=account_name_empty`; that page contains "Account name must not be empty." after the posting form's closing `</form>` and before the open-account `<form`, contains no error panel before the posting form, and `GET /api/accounts` still returns three accounts.
- [test] A form-encoded 41-character name returns `303` to `/?error=account_name_too_long&detail=41`, and that page shows "Account name is too long. Submitted: 41 characters; the limit is 40." before the open-account `<form` and after the posting form.
- [test] `GET /?error=account_name_too_long&detail=abc` and `GET /?error=account_name_too_long&detail=12` each render the message exactly "Account name is too long." with no "Submitted:" sentence.
- [test] `GET /?account=acct-nope` and the zero-account page each contain no `<form` substring.
- [test] A form opening with name `<script>alert(1)</script>` returns `303` to `/?account=acct-4`, and that page contains `&lt;script&gt;alert(1)&lt;/script&gt;` and no raw `<script>` tag.

**Files likely touched:**
- `internal/httpapi/router.go` — form rejection redirect, page message routing
- `internal/httpapi/router_test.go` — page rejection tests
- `web/index.html.tmpl` — open-account message panel

**Dependencies:** Task 8, Task 9

### Task 11: A domain ordering rule for accounts
**Story:** Story 7
**Type:** infrastructure

**Steps:**
1. Write a failing test in `internal/ledger/order_test.go`: `SortAccounts` on accounts `acct-10`, `acct-2`, `acct-12`, `acct-1`, `acct-9` (shuffled) yields `acct-1`, `acct-2`, `acct-9`, `acct-10`, `acct-12`.
2. Verify RED; implement `SortAccounts(accounts []Account)` in `internal/ledger/order.go` with `sort.SliceStable` comparing id length, then id text.
3. Verify GREEN, then commit.

**Done when:**
- [test] `ledger.SortAccounts` given `acct-10`, `acct-2`, `acct-12`, `acct-1`, `acct-9` orders them `acct-1`, `acct-2`, `acct-9`, `acct-10`, `acct-12`.
- [test] `ledger.SortAccounts` given `acct-3`, `acct-1`, `acct-2` orders them `acct-1`, `acct-2`, `acct-3`.

**Files likely touched:**
- `internal/ledger/order.go` — `SortAccounts`
- `internal/ledger/order_test.go` — ordering tests

**Dependencies:** none

### Task 12: Both account listings use sequence order
**Story:** Story 7
**Type:** happy-path

**Steps:**
1. Write failing router tests with the seeded store plus nine accounts opened through JSON `POST /api/accounts` (acct-4 … acct-12).
2. Verify RED; replace the `sort.Slice` string comparisons in `handleAccounts` and `handlePage` with `ledger.SortAccounts`.
3. Verify GREEN, then commit.

**Done when:**
- [test] With twelve accounts, `GET /api/accounts` returns ids exactly `acct-1`, `acct-2`, … `acct-12` in numeric order.
- [test] With twelve accounts, `GET /` renders the navigation links in that same numeric order, with the `acct-10` link after the `acct-9` link, and with no `account` parameter the selected account is acct-1 ("Selected account: Everyday Checking").
- [test] With only the three seeded accounts, `GET /api/accounts` and the `GET /` navigation both list `acct-1`, `acct-2`, `acct-3` in that order.

**Files likely touched:**
- `internal/httpapi/router.go` — use `ledger.SortAccounts`
- `internal/httpapi/router_test.go` — ordering tests

**Dependencies:** Task 6, Task 11

## Task Dependency Graph

```
Slice 1 (domain):            Slice 2 (HTTP and page):
T1 -> T2 -> T4 -> T5         T1 -> T6 -> T8 (also needs T5) -> T7
T1 -> T3                     T8 -> T10
                             T6 -> T9 -> T10
                             T11 -> T12 (also needs T6)
```

## Integration Points

- After Task 3: opening an account works end to end through the real SQLite store (slice 1 boundary).
- After Task 6: a `curl` can open an account on a running server.
- After Task 9: the presenter can open an account from the page.

## Coverage Check

| Criterion | Task id(s) | Done when quote | Disposition |
| --- | --- | --- | --- |
| Story 1 happy: Given the seeded ledger, when an account named "Holiday Fund" is opened, then the returned account has id `acct-4` and name "Holiday Fund", the store lists four accounts, its derived balance is 0, and it has no transactions. | 1 | "returns id `acct-4` and name "Holiday Fund", the fake then lists four accounts" | diff-local |
| Story 1 happy: Given the seeded ledger after opening "Holiday Fund", when a second account named "Holiday Fund" is opened, then it is returned with id `acct-5`, both accounts exist, and neither opening is rejected for sharing a name. | 1 | "returns `acct-5` with no error, and the fake lists both acct-4 and acct-5" | diff-local |
| Story 1 happy: Given a ledger with no accounts, when an account named "First" is opened, then it is returned with id `acct-1`. | 1 | "over an empty fake store returns id `acct-1`" | diff-local |
| Story 1 negative: Given a store whose account listing fails, when an account is opened, then an error wrapping the storage failure is returned, no account is returned, and nothing is inserted. | 2 | "When the fake's `Accounts` fails, `OpenAccount` returns a zero `Account`" | diff-local |
| Story 1 negative: Given a store whose insert fails, when an account is opened, then an error wrapping the storage failure is returned and no account is returned. | 2 | "When the fake's `InsertAccount` fails, `OpenAccount` returns a zero `Account`" | diff-local |
| Story 2 happy: Given the seeded ledger, when an account named "  Rainy Day  " (surrounding spaces) is opened, then it is accepted and stored with the name exactly as submitted, spaces included. | 4 | "the stored account's name equals `"  Rainy Day  "` byte for byte" | diff-local |
| Story 2 negative: Given the seeded ledger, when an account is opened with an empty name, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts. | 4 | "returns an error satisfying `errors.Is(err, ledger.ErrAccountNameEmpty)`, the fake still lists three accounts" | diff-local |
| Story 2 negative: Given the seeded ledger, when an account is opened with a name of only spaces, tabs, and a newline, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts. | 4 | "For names `""`, `"   "`, and `" \t\n"`" | diff-local |
| Story 3 happy: Given the seeded ledger, when an account is opened with a name of exactly 40 ASCII characters, then it is accepted as `acct-4`. | 5 | "A 40-ASCII-character name is accepted as `acct-4`" | diff-local |
| Story 3 happy: Given the seeded ledger, when an account is opened with a name of exactly 40 "é" characters (80 bytes), then it is accepted, because length counts characters, not bytes. | 5 | "a name of 40 `é` characters (80 bytes) is accepted" | diff-local |
| Story 3 negative: Given the seeded ledger, when an account is opened with a name of 41 ASCII characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and the store still lists three accounts. | 5 | "A 41-ASCII-character name and a 41-`é` name each return an error satisfying `errors.Is(err, ledger.ErrAccountNameTooLong)`" | diff-local |
| Story 3 negative: Given the seeded ledger, when an account is opened with a name of 41 "é" characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and nothing is recorded. | 5 | "after each the fake still lists three accounts with zero `InsertAccount` calls" | diff-local |
| Story 4 happy: Given the seeded ledger, when `POST /api/accounts` is sent with `Content-Type: application/json` and body `{"name":"Holiday Fund"}`, then the response is `201` with `Content-Type: application/json; charset=utf-8` and body `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`. | 6 | "returns `201`, `Content-Type: application/json; charset=utf-8`, and body exactly `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`" | diff-local |
| Story 4 happy: Given that account was just opened, when `GET /api/accounts` is requested, then its fourth element is `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}` and `GET /api/accounts/acct-4/transactions` returns `[]`. | 6 | "`GET /api/accounts/acct-4/transactions` returns `[]`" | diff-local |
| Story 4 happy: Given that account was just opened, when a JSON transaction of amount "25.00" is posted to `/api/accounts/acct-4/transactions`, then it is accepted with `201` and `acct-4` lists a balance of 2500 cents. | 6 | "returns `201` and `GET /api/accounts` then reports acct-4 with `balance_cents` 2500" | diff-local |
| Story 4 negative: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `not json`, then the response is `400` with code `account_name_empty` and `GET /api/accounts` still lists three accounts. | 7 | "with body `not json`, with `{"name":42}`, and with `{}` each returns `400` with error code `account_name_empty`" | diff-local |
| Story 4 negative: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `{"name":42}` or `{}`, then the response is `400` with code `account_name_empty` and nothing is recorded. | 7 | "`GET /api/accounts` afterwards still returns exactly three accounts" | diff-local |
| Story 4 negative: Given the running server, when `PUT /api/accounts` is sent, then the response is `405` with an empty body. | 7 | "`PUT /api/accounts` through `NewRouter` returns `405` with an empty body." | diff-local |
| Story 5 happy: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with name "   ", then the response is `400` with body `{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}`. | 8 | "{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}" | diff-local |
| Story 5 happy: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with a 41-character name, then the response is `400` with body `{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}`. | 8 | "{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}" | diff-local |
| Story 5 negative: Given either rejection above, when `GET /api/accounts` is requested afterwards, then it still lists exactly three accounts and the next successful opening is `acct-4` (no id was consumed). | 8 | "After both rejections `GET /api/accounts` returns exactly three accounts, and the next successful JSON opening returns id `acct-4`." | diff-local |
| Story 5 negative: Given the existing seven error codes, when this feature ships, then each still returns its original HTTP status and code string for its original rule. | 8 | "the existing table for the original seven sentinels passes with every status and code string unchanged" | diff-local |
| Story 6 happy: Given the seeded ledger, when `GET /` is requested, then the page contains an "Open account" form that posts a `name` field to `/api/accounts`, placed after the transactions section so the existing posting form is still the first form on the page, and its name input carries no `autofocus`. | 9 | "positioned after the Transactions section so that the first `<form` in the body is still the posting form, and the name input carries no `autofocus` attribute" | diff-local |
| Story 6 happy: Given the seeded ledger, when the open-account form is submitted with name "Holiday Fund", then the response is `303 See Other` with `Location: /?account=acct-4`, and following it shows "Holiday Fund" in the account navigation, "Selected account: Holiday Fund", balance "$0.00", and "No transactions.". | 9 | "returns `303` with `Location: /?account=acct-4`" | diff-local |
| Story 6 happy: Given an account was opened from the page, when a transaction is then posted to it from the posting form, then the page shows the new account selected with the posted amount as its balance. | 9 | "which shows "Selected account: Holiday Fund" and `class="balance">$25.00`" | diff-local |
| Story 6 happy: Given an account was just opened from the page, when the resulting page is reloaded, then the ledger still holds exactly one new account. | 9 | "Requesting `GET /?account=acct-4` a second time leaves `GET /api/accounts` at exactly four accounts." | diff-local |
| Story 6 negative: Given the seeded ledger, when the open-account form is submitted with a blank name, then the response is `303` to `/?error=account_name_empty`, the page shows "Account name must not be empty." above the open-account form and not above the posting form, and three accounts remain. | 10 | "A form-encoded blank name returns `303` to `/?error=account_name_empty`" | diff-local |
| Story 6 negative: Given the seeded ledger, when the open-account form is submitted with a 41-character name, then the redirect is `/?error=account_name_too_long&detail=41` and the page shows "Account name is too long. Submitted: 41 characters; the limit is 40." above the open-account form. | 10 | "A form-encoded 41-character name returns `303` to `/?error=account_name_too_long&detail=41`" | diff-local |
| Story 6 negative: Given the page is requested with `/?error=account_name_too_long&detail=abc` or `detail=12`, when it renders, then the message is exactly "Account name is too long." with no detail sentence. | 10 | "each render the message exactly "Account name is too long." with no "Submitted:" sentence" | diff-local |
| Story 6 negative: Given `GET /?account=acct-nope` or a ledger with no accounts, when the page renders, then it contains no `<form` at all, exactly as before this feature. | 10 | "`GET /?account=acct-nope` and the zero-account page each contain no `<form` substring." | diff-local |
| Story 6 negative: Given the open-account form is submitted with name `<script>alert(1)</script>`, when the resulting page renders, then the account is opened and its name appears escaped as text, never as markup. | 10 | "contains `&lt;script&gt;alert(1)&lt;/script&gt;` and no raw `<script>` tag" | diff-local |
| Story 7 happy: Given the seeded ledger plus nine opened accounts (`acct-4` … `acct-12`), when `GET /api/accounts` is requested, then the ids are exactly `acct-1`, `acct-2`, … `acct-12` in numeric order. | 12 | "returns ids exactly `acct-1`, `acct-2`, … `acct-12` in numeric order" | diff-local |
| Story 7 happy: Given the same twelve accounts, when `GET /` is requested, then the account navigation links appear in the same numeric order, with `acct-10` after `acct-9`. | 12 | "with the `acct-10` link after the `acct-9` link" | diff-local |
| Story 7 negative: Given only the three seeded accounts, when `GET /api/accounts` and `GET /` are requested, then the order is `acct-1`, `acct-2`, `acct-3`, identical to today. | 12 | "With only the three seeded accounts, `GET /api/accounts` and the `GET /` navigation both list `acct-1`, `acct-2`, `acct-3` in that order." | diff-local |
| Story 7 negative: Given twelve accounts, when `GET /` is requested with no `account` parameter, then the selected account is still `acct-1`, not whichever id sorts first as text. | 12 | "with no `account` parameter the selected account is acct-1" | diff-local |

## Architecture Obligation Coverage

| Decision | Disposition | Task(s) | Evidence |
| --- | --- | --- | --- |
| adr-2026-10-10-sequential-account-ids-ordered-by-sequence#D1 | task | task-1 | `OpenAccount` over a fake store seeded with acct-1..acct-3 returns id `acct-4` |
| adr-2026-10-10-sequential-account-ids-ordered-by-sequence#D2 | task | task-11, task-12 | With twelve accounts, `GET /api/accounts` returns ids exactly `acct-1`, `acct-2`, … `acct-12` in numeric order. |
| adr-2026-10-10-sequential-account-ids-ordered-by-sequence#D3 | task | task-4, task-5, task-8 | `codeFor` maps both new sentinels (matched with `errors.Is` on wrapped errors) to their codes with status 400 |
| adr-2026-10-10-sequential-account-ids-ordered-by-sequence#D4 | task | task-6, task-7, task-9, task-10 | A form-encoded `POST /api/accounts` with `name=Holiday+Fund` returns `303` with `Location: /?account=acct-4` |

## Verification

- [ ] All happy path criteria covered by at least one task
- [ ] All negative path criteria covered by at least one task
- [ ] No task exceeds 5 minutes of work
- [ ] Every task has a `Done when:` block of falsifiable checks
- [ ] Dependencies are explicit and acyclic; no slice 1 task depends on a slice 2 task
- [ ] Every story's tasks sit in exactly one slice
