# Implementation Plan: Account transaction count

**Date:** 2026-10-06
**Design:** `.docs/specs/2026-10-06-account-transaction-count.md` (Approved)
**Stories:** `.docs/stories/account-transaction-count.md`
**Tier:** S (`.docs/complexity/account-transaction-count.md`)
**Conflict check:** Skipped — Tier S

## Summary

One task. The page handler records how many transactions it loaded, and the template renders a
`Transactions: N` line below the table.

## Technical Approach

Add `TransactionCount int` to `pageData` in `internal/httpapi/router.go` and set it to
`len(transactions)` in `handlePage`, next to where `data.Transactions` is filled. In
`web/index.html.tmpl`, add `<p class="transaction-count">Transactions: {{.TransactionCount}}</p>`
after the `</table>` inside the existing `{{if .Transactions}}` branch, so empty and unknown accounts
render nothing new.

## Prerequisites

None.

## Tasks

### Task 1: Render the transaction count on the account page
**Story:** 1
**Type:** happy-path + negative-path

**Steps:**
1. Write failing tests in `internal/httpapi/router_test.go`: a fixture account with three transactions renders `Transactions: 3` after the table; the seeded first account shows exactly one count line whose number equals its table row count; empty and unknown accounts show none.
2. Verify RED.
3. In `internal/httpapi/router.go`, add `TransactionCount` to `pageData` and set it to `len(transactions)` in `handlePage`.
4. In `web/index.html.tmpl`, add `<p class="transaction-count">Transactions: {{.TransactionCount}}</p>` after the table inside the `{{if .Transactions}}` branch.
5. Verify GREEN.
6. Commit: "page: show the account's transaction count"

**Done when:**
- Tests assert: Given an account with three transactions, when its page is rendered, then the line Transactions: 3 appears below the transactions table. ; Given a freshly reset database, when the first account's page is rendered, then exactly one Transactions: line appears after the transactions table and its number equals the number of table rows. ; also `go test ./internal/httpapi/` passes.
- Tests assert: Given an account with no transactions, when its page is rendered, then no Transactions: line appears and the existing No transactions. message is shown unchanged. ; Given a requested account that does not exist, when the page is rendered, then no Transactions: line appears. ; also Route count is still five.

**Files:** internal/httpapi/router.go; internal/httpapi/router_test.go; web/index.html.tmpl
**Dependencies:** none

## Coverage Check

| Criterion | Tasks | Quote | Disposition |
|---|---|---|---|
| Story 1 happy: Given an account with three transactions, when its page is rendered, then the line Transactions: 3 appears below the transactions table. | task-1 | "Given an account with three transactions, when its page is rendered, then the line Transactions: 3 appears below the transactions table." | diff-local |
| Story 1 happy: Given a freshly reset database, when the first account's page is rendered, then exactly one Transactions: line appears after the transactions table and its number equals the number of table rows. | task-1 | "Given a freshly reset database, when the first account's page is rendered, then exactly one Transactions: line appears after the transactions table and its number equals the number of table rows." | diff-local |
| Story 1 negative: Given an account with no transactions, when its page is rendered, then no Transactions: line appears and the existing No transactions. message is shown unchanged. | task-1 | "Given an account with no transactions, when its page is rendered, then no Transactions: line appears and the existing No transactions. message is shown unchanged." | diff-local |
| Story 1 negative: Given a requested account that does not exist, when the page is rendered, then no Transactions: line appears. | task-1 | "Given a requested account that does not exist, when the page is rendered, then no Transactions: line appears." | diff-local |

## Task Dependency Graph

```
Task 1 (page transaction count)
```

## Out of Scope for This Plan

- Any change to `internal/store`, the JSON listing, CSV output, seed data, schema, or routes.
