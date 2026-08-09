# ADR: Sequential zero-padded transaction ids, with newest-first ordering tie-broken by id

**Date:** 2026-08-08
**Status:** Accepted
**Deciders:** james.stoup

## Context

Two requirements meet here and neither is satisfied by the obvious implementation.

**FR-3** requires the transaction list to be newest first in a *stable total order*: two
transactions recorded at the same instant must still have a defined, repeatable position relative to
each other. **NFR-3** forbids randomness anywhere, including in identifiers, so that a seeded
database is identical on every `make reset`.

`ORDER BY created_at DESC` alone does not deliver FR-3. Time is injected
(`adr-2026-08-08-injected-clock.md`), and every test uses `FixedClock`, so **every transaction
written during a single test shares one timestamp**. SQLite is free to return equal-key rows in any
order, so a test asserting "newest first" over three transactions stamped identically is asserting
something undefined. That is the flaky-test-in-front-of-an-audience failure the injected clock exists
to prevent, reintroduced through the ordering clause.

Random identifiers (UUID, ULID) would resolve nothing and violate NFR-3 outright.

## Options Considered

### Option A: Zero-padded sequential text ids, ordering tie-broken by id
`id` is `txn-0001`, `txn-0002`, … assigned in insertion order. Ordering is
`ORDER BY created_at DESC, id DESC`.
- **Pros:** No schema change — `.docs/architecture/erd.md` stays exactly as committed. Zero-padding
  makes the text sort lexicographically in numeric order, so the id doubles as the insertion
  sequence without a column that exists only to be a sequence. Tests can assert exact ids, which
  makes ordering genuinely covered rather than incidentally passing.
- **Cons:** The ordering guarantee rests on a formatting property (constant width), not on a typed
  integer. Two facts are now carried by one column.

### Option B: A monotonic `seq INTEGER` column, ordered by `seq DESC`
- **Pros:** The sequence is explicit and typed; ordering is obviously total; no dependence on how an
  id is formatted.
- **Cons:** Modifies a committed schema to buy a property Option A already provides. Adds a column
  that appears in the ERD, in every insert, and in every test fixture, and that an audience will ask
  about. The ERD's constraint notes are load-bearing precisely because the schema is meant to stay
  minimal.

### Option C: Random identifiers (UUID/ULID), ordering by `created_at` only
- **Pros:** Identifiers need no coordination with existing rows.
- **Cons:** Violates NFR-3 — seed data would differ every run and `make reset` would stop being
  reproducible. Does not even solve the problem it is adjacent to: ordering remains undefined
  between rows sharing a timestamp.

## Decision

**Option A.** Transaction identifiers are `txn-` followed by a **four-digit zero-padded decimal**,
assigned in insertion order. Newest-first is `ORDER BY created_at DESC, id DESC`.

Two properties of this decision are load-bearing and are stated here explicitly rather than left to
be inferred at implementation time:

1. **The sequence is global across the `transactions` table, not per account.** `id` is the table's
   `TEXT PRIMARY KEY`. Numbering per account would produce a `txn-0001` for every account and
   violate the primary key on the second account seeded. The next id is derived from the count of
   **all** existing transaction rows.

2. **Deriving the next id from a row count is sound only because nothing ever deletes a
   transaction.** Count-derived and max-id-derived numbering agree exactly as long as the table is
   append-only. This project specifies no delete capability — the ledger is an append-only log, and
   `make reset` drops the whole database file rather than deleting rows. Under those conditions a
   count is equivalent to a max and is the simpler of the two to read. **If a delete capability is
   ever added, count-derived ids collide and this decision must move to max-id-derived (or to
   Option B).**

## Consequences

### Positive
- `.docs/architecture/erd.md` is unchanged by this feature: no column added, no constraint added.
- Ordering is deterministic under `FixedClock`, so newest-first is a real assertion rather than a
  coincidence of insertion order.
- Seeded data is byte-identical across resets, satisfying FR-15 and NFR-3.
- Tests can assert exact identifiers (`txn-0007`), which reads clearly on a projector.

### Negative
- The lexicographic tiebreak holds **only while all ids share the same width**. At `txn-10000` the
  padding is exhausted and `txn-9999` would sort after it. Nothing in this project approaches that —
  seed data is 24–36 rows and a demo adds a handful — but the ordering guarantee is bounded by the
  padding width, and that bound is a property of a format string rather than of the schema.
- Two responsibilities ride on one column: identity and insertion order.
- Assigning an id requires reading existing state (a count) before the insert, so identifier
  assignment is not independent of the table's contents.

### Follow-up Actions
- [ ] Derive the next id from the count of all transaction rows, formatted `txn-%04d`.
- [ ] Order every transaction listing by `created_at DESC, id DESC` — both the page and the JSON
      route, so FR-11's "same order the page shows" holds.
- [ ] Add a test that writes at least three transactions under one `FixedClock` instant and asserts
      the exact newest-first id sequence.
- [ ] Seed accounts sequentially so seeded ids are stable and assertable.
