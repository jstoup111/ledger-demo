# Complexity — account-transaction-count

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 new — `Account` and `Transaction` are untouched | S |
| Schema change | None. The count is the length of the slice the page handler already loads | S |
| Integrations | 0 new. `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None; the feature is read-only | S |
| Story count | 1 | S |
| Blast radius | One package, `internal/httpapi`, plus the page template in `web/` | S |
| Routes touched | 0 added; the existing page route renders one extra line | S |
| Validation surface | 0 new rules, 0 new sentinel errors | S |

## Rationale

Every signal reads Small. The count is `len(transactions)` over the slice the page handler already
loads, rendered as one line in the existing template. No money arithmetic is involved, so
`.docs/decisions/adr-2026-08-08-money-as-int64-cents.md` is not engaged.

## What Small skips

Per the tier rules, this spec deliberately does **not** carry:

- `/architecture-diagram` — no component, container, or ERD relationship changes.
- `/architecture-review` — no new seam, dependency, or decision.
- `/conflict-check` — one story.
- `/coherence-check` — not authored for Small tier.

## Stem

`account-transaction-count` — matches `.docs/plans/account-transaction-count.md`, so the daemon resolves this tier at build time.
