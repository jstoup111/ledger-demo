# Complexity — seed-recorded-times

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 new — `Account` and `Transaction` are untouched | S |
| Schema change | None. `transactions.created_at` already exists and already stores a full instant | S |
| Integrations | 0 new. `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None | S |
| Story count | 2 | S |
| Blast radius | One package: `cmd/server` (the seed dataset and its test). No domain change, no store change, no HTTP change, no template change, no stylesheet change | S |
| Routes touched | 0 of 5 | S |
| Validation surface | 0 new rules, 0 new sentinel errors | S |

## Rationale

Every signal reads Small and none reads otherwise. The seeded rows already flow through the existing
`PostTransaction` path with an injected clock; the change is to the *value* each seeded row is stamped
with, declared alongside the row it belongs to. Nothing crosses a package boundary that is not already
crossed, no interface changes shape, and the displayed column already renders whatever instant the row
carries — so no rendering work exists either.

The one genuinely careful part is ordering: the newest-first listing is `created_at DESC, id DESC`, so
varying the timestamps changes which of the two clauses does the sorting. That is a constraint to
respect (recorded times must run in the same direction as the insertion sequence), not additional
scope — it is one property, stated once in the PRD and asserted once in a test.

Tier chosen by the operator (assumed decision, operator unavailable — see the PRD's assumption
ledger), matching the `size: S` label already on the originating intake issue.

## What Small skips

Per the tier rules, this spec deliberately does **not** carry:

- `/architecture-diagram` — no component, container, or ERD relationship changes. The committed
  diagrams in `.docs/architecture/` remain accurate as written.
- `/architecture-review` — no new seam, no new dependency direction, no schema decision. The governing
  decisions (the injected-clock ADR and the transaction-id/ordering ADR) already cover this change and
  are unmodified by it; see the PRD's Governing Decisions section for the conformance check.
- `/conflict-check` — two stories, one file of production code, no shared mutable state between them.
- `/coherence-check` — not authored for Small tier.

## Stem

`seed-recorded-times` — matches `.docs/plans/seed-recorded-times.md`, so the daemon resolves this tier
at build time.
