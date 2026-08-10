# Complexity — csv-export-single-account

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 new — `Account` and `Transaction` are untouched | S |
| Schema change | None. No migration, no index, no constraint | S |
| Integrations | 0 new. `encoding/csv` is standard library, so `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None. The command reads and writes nothing to the database | S |
| Story count | 2 | S |
| Blast radius | One package: `cmd/server` (one new file, one dispatcher change, their tests, plus one acceptance spec). No domain change, no store change, no HTTP change, no template change, no stylesheet change | S |
| Routes touched | 0 of 5 | S |
| Validation surface | 1 new rule (the command requires exactly one account id), 0 new sentinel errors — the unknown-account failure reuses the existing `ErrAccountNotFound` | S |
| New interfaces / seams | 0. Reads go through the existing `ledger.Store` interface; the only new seam is a package-level writer variable in `cmd/server`, mirroring the `listenAndServe` variable already there | S |

## Rationale

Every signal reads Small and none reads otherwise. The work is a rendering function over data an
existing interface method already returns in the required order, plus three lines of dispatch. There
is no new domain concept: the export decides nothing, derives nothing, and validates nothing about
money. The one genuinely careful part is that the failure path must produce *no* output rather than a
headers-only document, and that falls out of resolving the account before the first byte is written —
which the existing store method already does.

Tier chosen by the operator (assumed decision, operator unavailable — see the PRD's assumption
ledger), matching the `size: S` label already on the originating intake issue.

## What Small skips

Per the tier rules, this spec deliberately does **not** carry:

- `/architecture-diagram` — no component, container, or ERD relationship changes. The command is a
  second consumer of an interface the diagrams already show; the committed diagrams in
  `.docs/architecture/` remain accurate as written.
- `/architecture-review` — no new seam, no new dependency direction, no schema decision, no new
  package. The governing decisions already cover every constraint this feature touches; see the
  PRD's Governing Decisions table for the conformance check. No ADR is authored.
- `/conflict-check` — two stories over one package, with no shared mutable state and no contended
  resource between them.
- `/coherence-check` — not authored for Small tier.

## Stem

`csv-export-single-account` — matches `.docs/plans/csv-export-single-account.md`, so the daemon
resolves this tier at build time.
