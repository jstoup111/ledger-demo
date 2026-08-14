# ADR: The Store interface is declared in the domain package

**Date:** 2026-08-08
**Status:** APPROVED
**Deciders:** james.stoup

## Context

The ledger domain needs persistence: load accounts, load an account's transactions, append a
transaction. Persistence is SQLite via `modernc.org/sqlite`, file-backed for the server and
in-memory for tests.

The domain also carries the validation rules that matter most — including the below-zero check,
which reads a derived balance before appending. Those rules are the subject of the live demo, so
their tests must be fast, exact, and free of incidental setup.

The question is which package owns the persistence *interface*, and therefore which way the
dependency arrow points.

## Options Considered

### Option A: Interface declared in `internal/ledger`, implemented in `internal/store`
- **Pros:** The domain depends on nothing. `internal/ledger` unit-tests against a tiny in-memory
  fake with no SQLite at all. The interface is shaped by what the domain needs, not by what
  SQLite offers. Idiomatic Go — the consumer declares the interface.
- **Cons:** The interface and its implementation live apart, so a reader follows one hop to see
  the SQL.

### Option B: Interface declared in `internal/store`, imported by `internal/ledger`
- **Pros:** Interface and implementation sit together; one place to look.
- **Cons:** The dependency arrow points from domain to infrastructure. The domain imports the
  persistence package, so its tests drag SQLite in, and the interface tends to drift toward
  exposing storage concerns rather than domain needs.

### Option C: No interface — the domain uses a concrete SQLite store
- **Pros:** Least code; no indirection for an audience to follow.
- **Cons:** Every domain test needs a real database. Slower, and it couples the validation rules
  under demonstration to storage setup. The "in-memory for tests" requirement would still need
  some seam.

## Decision

**Option A.** `internal/ledger` declares the `Store` interface; `internal/store` implements it
against SQLite. The domain package has no knowledge of SQLite — the dependency points inward.

Chosen because the validation rules in `internal/ledger` are what the demo actually exercises,
and they deserve tests that are exact and instant. Declaring the interface in the consumer keeps
it minimal — only the operations the domain genuinely needs — and lets those tests run against a
trivial fake. The one-hop indirection Option A costs is a small price, and it is a pattern most
Go readers recognize on sight.

Option C is tempting for a project this small, but it would put a database in the path of the
tests the audience is watching turn green.

## Consequences

### Positive
- `internal/ledger` tests need no database, so they are fast and deterministic.
- The interface stays small and domain-shaped; SQLite specifics cannot leak into the domain.
- Swapping persistence would touch only `internal/store`.

### Negative
- Two places to look to understand persistence end to end.
- Two implementations of the interface exist (SQLite plus a test fake), so both must be kept in
  step with the interface.

### Follow-up Actions
- [ ] Declare `Store` in `internal/ledger`, sized to the domain's needs only.
- [ ] Implement it in `internal/store` over `database/sql` with the `sqlite` driver.
- [ ] Note: the schema must carry **no uniqueness constraint beyond the primary key** — see the
      non-goals list in `CLAUDE.md`. This is deliberate, not an omission.
