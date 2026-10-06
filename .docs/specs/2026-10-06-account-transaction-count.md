# PRD: Account transaction count

**Date:** 2026-10-06
**Status:** Approved

## Problem / Background

The account page lists an account's transactions but never states how many there are. A presenter
walking an audience through a long list has to count rows by hand.

## Goals & Non-Goals

**Goals**

- Show one line `Transactions: N` below the transactions table for the selected account.

**Non-Goals**

- Any schema, route, JSON API, CSV, or seed-data change.
- Pluralization or localization of the label.

## Users / Personas

- **Presenter:** operates the account page on a projector.

## Functional Requirements

- **FR-1:** When a selected account has at least one transaction, the page shows exactly one line `Transactions: N` below the transactions table, where N is the number of transactions listed in that table.
- **FR-2:** An account with no transactions shows no count line; the "No transactions." message is unchanged. An unknown account shows no count line.

## Non-Functional Requirements

- The public HTTP surface remains exactly five endpoints; the page carries no JavaScript.

## Acceptance Criteria / Success Metrics

See `.docs/stories/account-transaction-count.md`.

## Scope

### In Scope

- The page handler in `internal/httpapi` and the page template in `web/`.

### Out of Scope

- JSON listing, CSV, seed data, store, schema.

## Key Decisions & Rationale

- The count is derived from the transactions the page already loads, so it can never disagree with the rows shown.

## Dependencies

None.

## Open Questions

None.
