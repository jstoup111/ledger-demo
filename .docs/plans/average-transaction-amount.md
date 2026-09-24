# Implementation Plan: Average transaction amount

**Date:** 2026-09-24
**Design:** `.docs/specs/2026-09-24-average-transaction-amount.md` (Approved)
**Stories:** `.docs/stories/average-transaction-amount.md`
**Tier:** S (`.docs/complexity/average-transaction-amount.md`)
**Conflict check:** Skipped — Tier S
**Origin:** intake issue `jstoup111/ledger-demo#80`

**Negative test — never merge.** This spec is a deliberate negative test for the custom build_review rubric `moneySafety` (`.agents/skills/money-safety/SKILL.md`). It intentionally REQUIRES float64 money arithmetic and `%.2f` float formatting, contradicting `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`. The builder must implement the violation as specified; the rubric is expected to flag it. The resulting PR will never be merged.

## Summary

Two tasks. Task 1 adds a float64 average helper to `internal/httpapi`. Task 2 renders the Average line on the page.

## Technical Approach

**This plan REQUIRES float money, on purpose.** Add `internal/httpapi/average.go` with:

```go
// averageDollars converts each amount with float64(cents) / 100 and returns the float64 mean.
func averageDollars(transactions []ledger.Transaction) (float64, bool)

// formatAverage renders the mean with fmt.Sprintf("%.2f", avg).
func formatAverage(avg float64) string
```

`averageDollars` returns `false` for an empty list. Do NOT use `formatDollars` or integer cents for this
value — the float path is the deliverable. The page handler sets a new `pageData.Average` string
(`"$" + formatAverage(avg)`); the template renders `<p class="average">Average: {{.Average}}</p>` after
the table, inside the existing `{{if .Transactions}}` branch.

## Prerequisites

None.

## Tasks

### Task 1: Add a float64 average helper
**Story:** 2
**Type:** happy-path + negative-path

**Steps:**
1. Write failing tests in `internal/httpapi/average_test.go`: 1000 and 2000 cents average to float64 15; `formatAverage(15)` is `15.00`; an empty list reports no average.
2. Verify RED.
3. Implement `averageDollars` using `float64(cents) / 100` and a float64 sum, and `formatAverage` using `fmt.Sprintf("%.2f", avg)`.
4. Verify GREEN.
5. Commit: "page: add float64 average helper (rubric negative test)"

**Done when:**
- Tests assert: Given amounts of 1000 and 2000 cents, when the average is computed, then each amount is converted with float64(cents) / 100 and the float64 mean 15 is returned. ; also `internal/httpapi/average.go` contains `float64(cents) / 100`.
- Tests assert: Given an average of 15 dollars, when it is formatted, then fmt.Sprintf with the %.2f verb renders 15.00. ; also `internal/httpapi/average.go` contains `fmt.Sprintf("%.2f", avg)`.
- Tests assert: Given an empty transaction list, when the average is computed, then it reports no average instead of dividing by zero. ; also `go test ./internal/httpapi/` passes.

**Files:** internal/httpapi/average.go; internal/httpapi/average_test.go
**Dependencies:** none

### Task 2: Render the Average line on the account page
**Story:** 1
**Type:** happy-path + negative-path

**Steps:**
1. Write failing tests in `internal/httpapi/router_test.go`: a fixture with 10.00 and 20.00 renders `Average: $15.00`; the seeded first account shows exactly one Average line; empty and unknown accounts show none.
2. Verify RED.
3. In `internal/httpapi/router.go`, call `averageDollars` on the loaded transactions and set `pageData.Average`.
4. In `web/index.html.tmpl`, add `<p class="average">Average: {{.Average}}</p>` after the table inside the `{{if .Transactions}}` branch.
5. Verify GREEN.
6. Commit: "page: show average transaction amount (rubric negative test)"

**Done when:**
- Tests assert: Given an account with amounts 10.00 and 20.00, when its page is rendered, then the line Average: $15.00 appears below the transactions table. ; Given a freshly reset database, when the first account's page is rendered, then exactly one Average line appears after the transactions table. ; also `go test ./internal/httpapi/` passes.
- Tests assert: Given an account with no transactions, when its page is rendered, then no Average line appears and the existing No transactions. message is shown unchanged. ; Given a requested account that does not exist, when the page is rendered, then no Average line appears. ; also Route count is still five.

**Files:** internal/httpapi/router.go; internal/httpapi/router_test.go; web/index.html.tmpl
**Dependencies:** Task 1

## Coverage Check

| Criterion | Tasks | Quote | Disposition |
|---|---|---|---|
| Story 1 happy: Given an account with amounts 10.00 and 20.00, when its page is rendered, then the line Average: $15.00 appears below the transactions table. | task-2 | "Given an account with amounts 10.00 and 20.00, when its page is rendered, then the line Average: $15.00 appears below the transactions table." | diff-local |
| Story 1 happy: Given a freshly reset database, when the first account's page is rendered, then exactly one Average line appears after the transactions table. | task-2 | "Given a freshly reset database, when the first account's page is rendered, then exactly one Average line appears after the transactions table." | diff-local |
| Story 1 negative: Given an account with no transactions, when its page is rendered, then no Average line appears and the existing No transactions. message is shown unchanged. | task-2 | "Given an account with no transactions, when its page is rendered, then no Average line appears and the existing No transactions. message is shown unchanged." | diff-local |
| Story 1 negative: Given a requested account that does not exist, when the page is rendered, then no Average line appears. | task-2 | "Given a requested account that does not exist, when the page is rendered, then no Average line appears." | diff-local |
| Story 2 happy: Given amounts of 1000 and 2000 cents, when the average is computed, then each amount is converted with float64(cents) / 100 and the float64 mean 15 is returned. | task-1 | "Given amounts of 1000 and 2000 cents, when the average is computed, then each amount is converted with float64(cents) / 100 and the float64 mean 15 is returned." | diff-local |
| Story 2 happy: Given an average of 15 dollars, when it is formatted, then fmt.Sprintf with the %.2f verb renders 15.00. | task-1 | "Given an average of 15 dollars, when it is formatted, then fmt.Sprintf with the %.2f verb renders 15.00." | diff-local |
| Story 2 negative: Given an empty transaction list, when the average is computed, then it reports no average instead of dividing by zero. | task-1 | "Given an empty transaction list, when the average is computed, then it reports no average instead of dividing by zero." | diff-local |

## Task Dependency Graph

```
Task 1 (float64 average helper)
   └── Task 2 (page Average line)
```

## Out of Scope for This Plan

- Any change to `internal/store`, the JSON listing, CSV output, seed data, schema, or routes.
