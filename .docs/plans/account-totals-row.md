# Implementation Plan: Account totals row

**Date:** 2026-09-24
**Design:** `.docs/specs/2026-09-24-account-totals-row.md` (Approved)
**Stories:** `.docs/stories/account-totals-row.md`
**Tier:** S (`.docs/complexity/account-totals-row.md`)
**Conflict check:** Skipped — Tier S
**Origin:** intake issue `jstoup111/ledger-demo#74`

## Summary

Three tasks. Task 1 adds a pure, checked `int64` totals fold to `internal/ledger`. Task 2 wires it into
the page handler and renders a footer row in the template. Task 3 notes the row in the README.

## Technical Approach

**Money stays `int64` cents end to end** (`.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`).
Add `internal/ledger/totals.go` with:

```go
type Totals struct {
	Deposits    int64 // sum of positive amounts, cents
	Withdrawals int64 // sum of negative amounts, cents (zero or negative)
	Net         int64 // Deposits + Withdrawals, cents
}

func TotalsOf(transactions []Transaction) (Totals, error)
```

The fold uses the existing `checkedAdd` for every addition, so overflow in either direction returns an
error wrapping `ErrBalanceOverflow` instead of wrapping silently. Withdrawals are kept signed (negative)
so no negation of `math.MinInt64` is ever needed, and the footer shows them exactly like a negative
Amount cell. No `float32`, `float64`, `math/big`, division, or dollar-denominated intermediate is
introduced; the only cents-to-dollars conversion remains the existing `formatDollars`.

The page handler already loads the account's transactions; it calls `ledger.TotalsOf` on that same
slice, and on error responds 500 with `derive totals failed`, mirroring the balance path. It sets three
new `pageData` string fields via `formatDollars`. The template renders a `<tfoot>` row labelled `Totals`
only inside the existing `{{if .Transactions}}` branch.

## Prerequisites

None.

## Tasks

### Task 1: Add a checked int64 totals fold to the ledger package
**Story:** 2
**Type:** happy-path + negative-path

**Steps:**
1. Write failing table-driven tests in `internal/ledger/totals_test.go`: ten 10-cent deposits total exactly 100; a mixed list sums deposits and withdrawals separately; an empty list yields three zeros; positive overflow and negative overflow each return an error satisfying `errors.Is(err, ErrBalanceOverflow)`.
2. Verify RED.
3. Implement `Totals` and `TotalsOf` in `internal/ledger/totals.go` using `checkedAdd` for every addition, including the net.
4. Verify GREEN.
5. Commit: "ledger: add checked int64 totals fold"

**Done when:**
- Tests assert: Given transactions of 10 cents repeated ten times, when totals are computed, then deposits is exactly 100 cents and net is exactly 100 cents. ; Given deposits whose sum exceeds the largest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow instead of a wrapped value. ; also `go test ./internal/ledger/` passes with the overflow cases asserting `ErrBalanceOverflow` in both directions.
- Tests assert: Given a mix of positive and negative amounts, when totals are computed, then deposits is the int64 sum of the positive amounts and withdrawals is the int64 sum of the negative amounts. ; Given withdrawals whose sum is below the smallest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow. ; also `internal/ledger/totals.go` contains no `float32`, `float64`, or `math/big`.
- Tests assert: Given an empty transaction list, when totals are computed, then all three totals are zero and no error is returned. ; Given the totals code, when it is searched for float32, float64, or the math/big package, then no hit is returned. ; also An empty transaction list returns three zero totals and a nil error.

**Files:** internal/ledger/totals.go; internal/ledger/totals_test.go
**Dependencies:** none

### Task 2: Render the totals footer row on the account page
**Story:** 1
**Type:** happy-path + negative-path

**Steps:**
1. Write failing tests in `internal/httpapi/router_test.go`: the seeded first account's page contains one `Totals` footer row whose net text equals the balance text; a fixture with 100.00, 25.50, and -40.25 renders $125.50, -$40.25, and $85.25; a deposits-only account renders withdrawals $0.00; an empty account and an unknown account render no footer; a store fixture whose amounts overflow the deposits sum yields HTTP 500 with no footer.
2. Verify RED.
3. In `internal/httpapi/router.go`, compute `ledger.TotalsOf(transactions)`, return 500 `derive totals failed` on error, and set `TotalDeposits`, `TotalWithdrawals`, and `TotalNet` on `pageData` via `formatDollars`.
4. In `web/index.html.tmpl`, add a `<tfoot>` row labelled `Totals` inside the existing transactions table, which already renders only when there are transactions.
5. Verify GREEN.
6. Commit: "page: show deposits, withdrawals, and net totals row"

**Done when:**
- Tests assert: Given a freshly reset database, when the first account's page is rendered, then the transaction table ends with exactly one footer row labelled Totals showing deposits, withdrawals, and net. ; Given an account with only deposits, when its page is rendered, then the footer shows withdrawals as $0.00. ; Given a requested account that does not exist, when the page is rendered, then no footer row appears. ; also `go test ./internal/httpapi/` passes including the footer, empty-account, and overflow cases.
- Tests assert: Given an account with deposits of 100.00 and 25.50 and a withdrawal of -40.25, when its page is rendered, then the footer shows deposits $125.50, withdrawals -$40.25, and net $85.25. ; Given the same unchanged ledger, when the page is rendered twice, then both footers are byte-identical. ; Given a totals computation that fails, when the page is rendered, then the response is an internal server error and no partial footer is rendered. ; also The rendered net equals the rendered balance for every seeded account.
- Tests assert: Given any account with transactions, when its page is rendered, then the footer's net value is character-for-character identical to the balance shown above the table. ; Given an account with no transactions, when its page is rendered, then no footer row appears and the existing No transactions. message is shown unchanged. ; also Route count is still five and the page has no JavaScript.

**Files:** internal/httpapi/router.go; internal/httpapi/router_test.go; web/index.html.tmpl
**Dependencies:** Task 1

### Task 3: Mention the totals row in the README
**Story:** 1
**Type:** documentation

**Steps:**
1. Add one or two lines to `README.md` stating that the account page ends with a totals row of deposits, withdrawals, and net, computed in integer cents.
2. Commit: "docs: describe the account totals row"

**Done when:**
- `README.md` mentions the totals row.
- `git diff --stat` for this task shows only `README.md`.

**Files:** README.md
**Dependencies:** Task 2

## Coverage Check

| Criterion | Tasks | Quote | Disposition |
|---|---|---|---|
| Story 1 happy: Given a freshly reset database, when the first account's page is rendered, then the transaction table ends with exactly one footer row labelled Totals showing deposits, withdrawals, and net. | task-2 | "Given a freshly reset database, when the first account's page is rendered, then the transaction table ends with exactly one footer row labelled Totals showing deposits, withdrawals, and net." | diff-local |
| Story 1 happy: Given an account with deposits of 100.00 and 25.50 and a withdrawal of -40.25, when its page is rendered, then the footer shows deposits $125.50, withdrawals -$40.25, and net $85.25. | task-2 | "Given an account with deposits of 100.00 and 25.50 and a withdrawal of -40.25, when its page is rendered, then the footer shows deposits $125.50, withdrawals -$40.25, and net $85.25." | diff-local |
| Story 1 happy: Given any account with transactions, when its page is rendered, then the footer's net value is character-for-character identical to the balance shown above the table. | task-2 | "Given any account with transactions, when its page is rendered, then the footer's net value is character-for-character identical to the balance shown above the table." | diff-local |
| Story 1 happy: Given an account with only deposits, when its page is rendered, then the footer shows withdrawals as $0.00. | task-2 | "Given an account with only deposits, when its page is rendered, then the footer shows withdrawals as $0.00." | diff-local |
| Story 1 happy: Given the same unchanged ledger, when the page is rendered twice, then both footers are byte-identical. | task-2 | "Given the same unchanged ledger, when the page is rendered twice, then both footers are byte-identical." | diff-local |
| Story 1 negative: Given an account with no transactions, when its page is rendered, then no footer row appears and the existing No transactions. message is shown unchanged. | task-2 | "Given an account with no transactions, when its page is rendered, then no footer row appears and the existing No transactions. message is shown unchanged." | diff-local |
| Story 1 negative: Given a requested account that does not exist, when the page is rendered, then no footer row appears. | task-2 | "Given a requested account that does not exist, when the page is rendered, then no footer row appears." | diff-local |
| Story 2 happy: Given transactions of 10 cents repeated ten times, when totals are computed, then deposits is exactly 100 cents and net is exactly 100 cents. | task-1 | "Given transactions of 10 cents repeated ten times, when totals are computed, then deposits is exactly 100 cents and net is exactly 100 cents." | diff-local |
| Story 2 happy: Given a mix of positive and negative amounts, when totals are computed, then deposits is the int64 sum of the positive amounts and withdrawals is the int64 sum of the negative amounts. | task-1 | "Given a mix of positive and negative amounts, when totals are computed, then deposits is the int64 sum of the positive amounts and withdrawals is the int64 sum of the negative amounts." | diff-local |
| Story 2 happy: Given an empty transaction list, when totals are computed, then all three totals are zero and no error is returned. | task-1 | "Given an empty transaction list, when totals are computed, then all three totals are zero and no error is returned." | diff-local |
| Story 2 happy: Given deposits whose sum exceeds the largest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow instead of a wrapped value. | task-1 | "Given deposits whose sum exceeds the largest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow instead of a wrapped value." | diff-local |
| Story 2 happy: Given withdrawals whose sum is below the smallest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow. | task-1 | "Given withdrawals whose sum is below the smallest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow." | diff-local |
| Story 2 negative: Given a totals computation that fails, when the page is rendered, then the response is an internal server error and no partial footer is rendered. | task-2 | "Given a totals computation that fails, when the page is rendered, then the response is an internal server error and no partial footer is rendered." | diff-local |
| Story 2 negative: Given the totals code, when it is searched for float32, float64, or the math/big package, then no hit is returned. | task-1 | "Given the totals code, when it is searched for float32, float64, or the math/big package, then no hit is returned." | diff-local |

## Task Dependency Graph

```
Task 1 (ledger totals fold)
   └── Task 2 (page footer row)
          └── Task 3 (README)
```

## Out of Scope for This Plan

- Any change to `internal/store`, the JSON listing, CSV output, seed data, schema, or routes.
- Any float, decimal library, or alternative money formatter.
