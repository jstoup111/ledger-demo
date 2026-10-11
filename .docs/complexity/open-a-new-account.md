# Complexity — open-a-new-account

Tier: M
Stacked-Delivery: approved

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | `Account` gains its first write path; no column or table added | S |
| Contracts changed | The `ledger.Store` interface gains `InsertAccount`; the Accepted API contract gains a route and two error codes | M |
| Integrations | 0 new. `modernc.org/sqlite` stays the only dependency | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None | S |
| Story count | 7 | M |
| Blast radius | Four packages: `internal/ledger`, `internal/store`, `internal/httpapi`, `web/` | M |
| Routes touched | 1 added (`POST /api/accounts`, negotiated JSON/form); the page gains a second form | M |
| Validation surface | 2 new rules, 2 new sentinel errors, 2 new frozen error codes | M |
| Design decisions | 1 new ADR (account id scheme and the ordering it forces past nine accounts) | M |

## Rationale

Six of ten signals read Medium. The data model barely moves, but the feature crosses every layer
the project has and changes two accepted contracts — the domain `Store` interface
(`adr-2026-08-08-store-interface-in-domain-package.md`) and the frozen code set in
`api-response-contract.md`. It also forces one genuine design decision: sequential `acct-N` ids
sort lexicographically wrong from the tenth account, so account ordering must be defined by id
sequence rather than id text. That is more than a Small one-package change and far short of Large
(no new persistence, no external integration, no state machine).

## Stacked delivery

The work splits cleanly at the domain boundary, so it is delivered as two stacked children:

1. **Domain account opening** — `ledger.OpenAccount`, its validation sentinels, and
   `InsertAccount` on the `Store` interface. Useful alone: seed code and tests can open accounts
   through the validated domain path, with its own unit and SQLite tests.
2. **Open accounts over HTTP and on the page** — the negotiated route, its error codes and
   messages, the page form, and sequence-ordered account listing. Builds on child 1.

Stacked delivery approved by the operator (pre-authorized for this spec).

## Tier-required artifacts

Medium: `/architecture-diagram`, lightweight `/architecture-review`, `/conflict-check`, and
`/coherence-check` are all authored.

## Stem

`open-a-new-account` — matches `.docs/plans/open-a-new-account.md`, so the daemon resolves this
tier at build time.
