# Architecture Review: Base Ledger

**Date:** 2026-08-08
**Mode:** Lightweight (Medium tier — Feasibility and Alignment only; complexity already assessed by
`/conduct`, domain integrity handled per-cycle by the TDD domain reviewer)
**Reviewed:** `.docs/specs/2026-08-08-base-ledger.md` (Approved, 16 FRs). Stories and plan do not
exist yet — this review runs before `/stories` per
`adr-2026-06-29-architecture-before-stories-convergent-kickback`.
**Verdict:** APPROVED WITH CONDITIONS

## Feasibility

| Check | Assessment |
|---|---|
| **Stack compatibility** | Clear. Nothing in the 16 FRs needs anything beyond Go's standard library plus the already-pinned `modernc.org/sqlite`. Content negotiation, form parsing, `303` redirects, and template rendering are all `net/http` + `html/template`. `go.mod`/`go.sum` are unchanged by this feature. |
| **Prerequisites** | One: the schema must exist before anything can read or write. No external account, no config beyond the two environment variables that already exist (`PORT`, `LEDGER_DB_PATH`). |
| **Integration surface** | Five packages, all internal, all in one binary. Zero external systems — `.docs/architecture/system-context.md` draws no third party, and NFR-2 forbids network calls. |
| **Data implications** | Schema creation only. No migration, no backfill, no destructive change — the tables do not exist yet. `make reset` drops the file wholesale, so there is no upgrade path to design. |
| **Performance risk** | Derived balance folds an account's whole transaction log on every read, and the JSON account listing does that per account. At seeded scale (24–36 rows across 3 accounts) this is irrelevant, and unbounded growth is not reachable in a demo. Explicitly **not** flagged: adding pagination or a cached balance would violate FR-2 and the derived-balance ADR respectively. |
| **Worktree isolation** | Already solved and unchanged. `PORT` and `LEDGER_DB_PATH` come from `.env.local` per worktree; SQLite is a file, not a service. Two worktrees collide on nothing. No new port, service, or shared resource is introduced. |

No infeasibility found.

## Complexity

Skipped per Lightweight Mode — assessed at `.docs/complexity/base-ledger.md` (Tier: M, operator-set
over an S recommendation).

## Alignment

**Domain boundaries** — Respected as drawn. `internal/store` depends on `internal/ledger`, never the
reverse; the `Store` interface is declared in the domain
(`adr-2026-08-08-store-interface-in-domain-package.md`). One boundary question the spec raises and
this review answers: **the dollars-to-cents parse belongs at the HTTP boundary, not in the domain.**
FR-12e rejects malformed *input*, which is a property of an untrusted string, whereas the domain's
five other rules are properties of a well-formed transaction. Parsing at the edge means the domain
only ever receives an `int64` and cannot be handed a malformed amount at all.

**Pattern consistency** — The two decisions taken here are new patterns for this codebase, and both
are now documented: `adr-2026-08-08-one-negotiated-posting-endpoint.md` and
`adr-2026-08-08-deterministic-transaction-ids-and-ordering.md`. No other new pattern is introduced;
everything else follows an existing Accepted ADR.

**State management** — No state machine, and deliberately none: there is no pending/posted status and
no boolean status flags (`.docs/architecture/erd.md` records their absence as intentional). A
transaction is a immutable row in an append-only log. No invalid state is representable because
there are no states.

**Diagram accuracy** — `.docs/architecture/` was refreshed in this same DECIDE pass and
operator-validated; `conduct render-diagrams --check` passes on all seven Mermaid blocks. The ERD is
byte-identical to bootstrap: this feature adds no column and no constraint.

**Security boundaries** — Authentication and authorization are explicit non-goals, so "are new
endpoints authenticated" does not apply. What *does* apply is input validation at the boundary, and
there are three untrusted inputs worth naming: the amount string, the description, and — newly, as a
consequence of the redirect-based error flow — the `error` and `account` query parameters on `GET /`.
`html/template` escapes on output, so injection is not the risk; a **blank error panel from an
unrecognized code** is, and it is Condition 2 below.

**Production DI defaults** — Checks out. The in-memory SQLite implementation is confined to tests;
the server path is file-backed via `LEDGER_DB_PATH`. This is the intended split
(`CLAUDE.md`, Tech Stack) and not an in-memory production default.

## Domain Integrity

Skipped per Lightweight Mode — handled per-cycle by the TDD domain reviewer. One item noted for it
rather than resolved here: money is already a domain concern via `int64` cents
(`adr-2026-08-08-money-as-int64-cents.md`), so the parse boundary above is the place primitive
obsession would show up if it shows up at all.

## Wiring Surface

Design-time commitment: where each new production surface gets called from. No `file:line` yet — the
code does not exist.

| New surface | Wired into |
|---|---|
| `clock.Clock`, `clock.SystemClock` | Constructed in `cmd/server` for both the `serve` and `seed` commands, injected into the domain. `FixedClock` is test-only and is **not** a production surface. |
| `ledger.Store` (interface) | Declared in `internal/ledger`; satisfied by the SQLite implementation; the concrete value is constructed in `cmd/server` and passed inward. |
| Schema creation | Invoked from `cmd/server` on open, for both `serve` and `seed`. |
| Domain posting operation (validation + derive + append) | Called from the single posting handler in `internal/httpapi`, per `adr-2026-08-08-one-negotiated-posting-endpoint.md`. It is the **only** caller. |
| Domain read operations (accounts with balances, an account's transactions) | Called from the JSON list handlers and from the page handler in `internal/httpapi`. |
| Sentinel → error-code mapping | One mapping function at the `internal/httpapi` boundary, called by the posting handler and by the page renderer; codes are fixed by `.docs/decisions/api-response-contract.md`. |
| `httpapi.NewRouter(...)` (gaining dependencies) | Already called from `cmd/server` `serve()`; the call site gains the store and clock arguments. |
| Seed data loader | Invoked from the `seed` command in `cmd/server`, which `make seed`/`make reset` already run. |
| Page template + uncommented CSS rules | Rendered by `GET /`; assets already served from the embedded FS in `web`. |

`conduct-ts overlap-scan` over these paths: **no overlap detected, no open blockers** (advisory).

## Risks

| Risk | Type | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| Per-account rather than global id numbering collides on the `TEXT PRIMARY KEY` | Data | Medium | High | Condition 1. Pinned explicitly in the ids ADR: the sequence is global across the table. |
| An unrecognized `error` code in the URL renders an empty error panel, so a rejection looks like success | Technical | Medium | Medium | Condition 2. Unknown code must fall back to a generic message. |
| Zero-padding width bounds the ordering guarantee; ids of differing width sort wrongly | Data | Very low | Medium | Condition 3, plus the Negative consequence recorded in the ids ADR. Unreachable at demo scale (24–36 seeded rows). |
| Dollars-to-cents parsing done with floats, silently losing precision | Data | Low | High | Forbidden by `adr-2026-08-08-money-as-int64-cents.md`; FR-12e requires a negative test for malformed input. String parse only. |
| Balance derived in the handler rather than the domain, so the page and the JSON route disagree | Technical | Low | Medium | Single domain operation is the only source of a balance; the PRD's acceptance criteria require the two to be equal always. |

The read-then-write balance check without a lock is **not** entered here. It is an accepted, recorded
characteristic (`.docs/architecture/sequences.md`, and "Known Characteristics" in the PRD), not a
risk this feature should mitigate.

## ADRs Created

| ADR | Status | Decides |
|---|---|---|
| `adr-2026-08-08-one-negotiated-posting-endpoint.md` | Accepted | Open Question 1 — one endpoint, response mode by request content type. FR-9 becomes structural rather than conventional. |
| `adr-2026-08-08-deterministic-transaction-ids-and-ordering.md` | Accepted | Open Question 2 — `txn-%04d` global sequence; `ORDER BY created_at DESC, id DESC`; no schema change. |

Both ratify the operator-confirmed direction. Neither reopens it. Both recorded the rejected
alternative and, more usefully, the conditions under which the decision would stop holding.

Status is written as `Accepted` to match the four existing ADRs in this repository. The engineer
`land` gate rejects an unapproved ADR fail-closed, so an unapproved intermediate state was never
valid here.

## Conditions

Three, all narrow and all mechanically checkable at code review.

1. **Transaction ids are numbered globally across the `transactions` table, not per account.** `id`
   is the table's primary key; per-account numbering produces a duplicate `txn-0001` on the second
   account seeded and fails the insert. Derive from the count of all rows.
2. **An unrecognized `error` query value renders a generic message, never an empty panel.** The code
   arrives from the URL and is therefore client-supplied. FR-13 requires the rejection be visible; a
   blank panel satisfies the letter and defeats the purpose.
3. **All transaction ids share one width** (`txn-%04d`). The lexicographic tiebreak that makes
   ordering total depends on constant width; a differently-padded id silently breaks FR-3's total
   order.

Conditions are tracked into `/plan` and checked by the evaluator at code review. Unmet at `/finish`,
they are blocking.
