# Complexity — account-totals-row

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 new — `Account` and `Transaction` are untouched; one small value type for the three totals | S |
| Schema change | None. Totals are derived from rows the store already returns | S |
| Integrations | 0 new. `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None; the feature is read-only | S |
| Story count | 2 | S |
| Blast radius | Two packages: `internal/ledger` (a pure totals fold) and `internal/httpapi` plus the page template in `web/` | S |
| Routes touched | 0 added; the existing page route renders one extra row | S |
| Validation surface | 0 new rules, 0 new sentinel errors — overflow reuses `ErrBalanceOverflow` | S |

## Rationale

Every signal reads Small. The totals are a fold over the transaction slice the page handler already
loads, using the same checked `int64` addition the balance fold already uses. Formatting reuses
`formatDollars`. No interface changes shape and no route is added.

The one careful part is money exactness: the totals must stay `int64` cents end to end, per
`.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`, with overflow detected rather than wrapped.
That is a constraint, not additional scope.

## What Small skips

Per the tier rules, this spec deliberately does **not** carry:

- `/architecture-diagram` — no component, container, or ERD relationship changes.
- `/architecture-review` — no new seam, dependency direction, or schema decision; the money ADR already
  governs the arithmetic and is unmodified by this change.
- `/conflict-check` — two stories, no shared mutable state between them.
- `/coherence-check` — not authored for Small tier.

## Stem

`account-totals-row` — matches `.docs/plans/account-totals-row.md`, so the daemon resolves this tier
at build time.
