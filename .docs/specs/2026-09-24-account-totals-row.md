# PRD: Account totals row

**Date:** 2026-09-24
**Status:** Approved

## Problem / Background

The account page lists an account's transactions and shows its balance, but nothing summarises how
much went in and how much went out. A presenter cannot point at "total deposits" or "total withdrawals"
without adding up rows by hand on stage.

## Goals & Non-Goals

**Goals**

- Show total deposits, total withdrawals, and net for the listed transactions in one footer row.
- Keep every total exact: `int64` cents end to end, formatted with the existing dollar formatter.

**Non-Goals**

- Any float, decimal library, or dollar-denominated intermediate value.
- Date-range, per-day, or per-category subtotals; running balances.
- Any schema, route, JSON API, or CSV change.

## Users / Personas

- **Presenter:** operates the account page on a projector and wants a one-glance summary.

## Functional Requirements

- **FR-1:** When a selected account has at least one transaction, the transaction table ends with one footer row showing total deposits, total withdrawals, and net.
- **FR-2:** Total deposits is the sum of all positive amounts; total withdrawals is the sum of all negative amounts, shown signed exactly like a negative Amount cell; net is deposits plus withdrawals.
- **FR-3:** All three totals are computed as `int64` cents; no floating-point value participates in computing or formatting them.
- **FR-4:** Each total is rendered with the same formatter as the Amount column and the balance.
- **FR-5:** Net equals the account's balance shown on the same page.
- **FR-6:** An account with no transactions shows no footer row; the existing "No transactions." message is unchanged.
- **FR-7:** A total that would overflow `int64` is detected and fails the page render rather than wrapping.

## Non-Functional Requirements

- The public HTTP surface remains exactly five endpoints; the page carries no JavaScript.
- No new dependency. `time.Now()` still appears once.

## Acceptance Criteria / Success Metrics

See `.docs/stories/account-totals-row.md`.

## Scope

### In Scope

- A pure totals fold in `internal/ledger`, the page handler, and the page template.

### Out of Scope

- JSON listing, CSV, seed data, store, schema.

## Key Decisions & Rationale

- Totals live in `internal/ledger` beside `Balance` so they reuse `checkedAdd` and its overflow sentinel.
- Governed by `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`, unmodified.

## Dependencies

None.

## Open Questions

None.
