# Complexity — base-ledger

Tier: M

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 2 — `Account`, `Transaction` | S |
| Integrations | 1 — SQLite via `modernc.org/sqlite`, already pinned in `go.mod` | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None — no pending/posted status, no holds (explicit non-goals) | S |
| Story count | ~6 | S |
| Blast radius | Whole-app build-out across 5 packages: `ledger`, `store`, `clock`, `httpapi`, `cmd/server`, plus `web/` markup and seed data | M |
| Validation surface | 6 distinct rules, each needing its own sentinel error and negative test | M |

## Rationale

Every per-feature signal reads Small — two models, one already-pinned dependency, no auth, no state
machine, roughly six stories. What lifts this to **Medium** is that it is not a change to an
existing app but the construction of the entire application: five packages that currently hold only
doc comments, a schema that does not exist yet, six validation rules each carrying a distinct
sentinel error, and a page that must go from placeholder to full markup. Correctness here is also
unusually load-bearing — the artifact is a stage prop that has to behave deterministically in front
of an audience, and a defect found on stage is not recoverable.

Tier chosen by the operator over a Small recommendation, accepting the extra rigor of
conflict-check and the committed coherence mapping.

## What Medium requires

- `/architecture-diagram` — refresh `.docs/architecture/` so the diagrams describe the built shape
  rather than an intended one (they currently carry `Status: Skeleton` headers).
- `/architecture-review` — lightweight for Medium. The four existing ADRs are already
  `Status: Accepted`; any new ADR must be Accepted before `land`.
- `/conflict-check` — `.docs/conflicts/`.
- `/coherence-check` — `.docs/coherence/base-ledger.md`, mapping outcomes → FRs → stories → tasks.

## Stem

`base-ledger` — matches `.docs/plans/base-ledger.md` and `.docs/specs/base-ledger.md` so the daemon
resolves the tier at build time.
