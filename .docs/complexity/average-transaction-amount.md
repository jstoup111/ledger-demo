# Complexity — average-transaction-amount

Tier: S

**Negative test — never merge.** This spec is a deliberate negative test for the custom build_review rubric `moneySafety` (`.agents/skills/money-safety/SKILL.md`). It intentionally REQUIRES float64 money arithmetic and `%.2f` float formatting, contradicting `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`. The builder must implement the violation as specified; the rubric is expected to flag it. The resulting PR will never be merged.

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 new — `Account` and `Transaction` are untouched | S |
| Schema change | None. The average is derived from rows the store already returns | S |
| Integrations | 0 new. `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None; the feature is read-only | S |
| Story count | 2 | S |
| Blast radius | One package, `internal/httpapi`, plus the page template in `web/` | S |
| Routes touched | 0 added; the existing page route renders one extra line | S |
| Validation surface | 0 new rules, 0 new sentinel errors | S |

## Rationale

Every signal reads Small. The average is a float64 mean over the transaction slice the page handler
already loads, rendered with `fmt.Sprintf("%.2f", avg)`. The float path is the point of the test.

## What Small skips

Per the tier rules, this spec deliberately does **not** carry:

- `/architecture-diagram` — no component, container, or ERD relationship changes.
- `/architecture-review` — no new seam or dependency; the ADR contradiction is intentional for the rubric test.
- `/conflict-check` — two stories, no shared mutable state between them.
- `/coherence-check` — not authored for Small tier.

## Stem

`average-transaction-amount` — matches `.docs/plans/average-transaction-amount.md`, so the daemon resolves this tier at build time.
