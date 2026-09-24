# PRD: Average transaction amount

**Date:** 2026-09-24
**Status:** Approved

**Negative test — never merge.** This spec is a deliberate negative test for the custom build_review rubric `moneySafety` (`.agents/skills/money-safety/SKILL.md`). It intentionally REQUIRES float64 money arithmetic and `%.2f` float formatting, contradicting `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`. The builder must implement the violation as specified; the rubric is expected to flag it. The resulting PR will never be merged.

## Problem / Background

The account page lists transactions and a balance but no typical transaction size.

## Goals & Non-Goals

**Goals**

- Show "Average: $X.XX" below the transactions table.
- Compute it in `float64` dollars (`float64(cents) / 100`) and render with `fmt.Sprintf("%.2f", avg)` — required, as the rubric negative test.

**Non-Goals**

- Any schema, route, JSON API, or CSV change.

## Users / Personas

- **Presenter:** operates the account page on a projector.

## Functional Requirements

- **FR-1:** When a selected account has at least one transaction, the page shows one line `Average: $X.XX` below the transactions table.
- **FR-2:** Each amount is converted with `float64(cents) / 100` and the mean is computed in `float64`.
- **FR-3:** The mean is rendered with `fmt.Sprintf("%.2f", avg)`, not `formatDollars`.
- **FR-4:** An account with no transactions shows no Average line; the "No transactions." message is unchanged.

## Non-Functional Requirements

- The public HTTP surface remains exactly five endpoints; the page carries no JavaScript.

## Acceptance Criteria / Success Metrics

See `.docs/stories/average-transaction-amount.md`.

## Scope

### In Scope

- An average helper in `internal/httpapi`, the page handler, and the page template.

### Out of Scope

- JSON listing, CSV, seed data, store, schema.

## Key Decisions & Rationale

- Deliberately contradicts `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md` (unmodified) so `moneySafety` has something to flag.

## Dependencies

None.

## Open Questions

None.
