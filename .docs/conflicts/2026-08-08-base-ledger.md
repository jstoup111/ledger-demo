# Conflict Check — Base Ledger

**Date:** 2026-08-08
**Scope:** `.docs/stories/base-ledger.md` (6 stories, FR-1 … FR-16) — the only stories file in the
repository, so the cross-feature conflict surface is nil. All work here is internal consistency among
the six stories and consistency against the Accepted ADRs and API contract.
**Result:** PASSED CLEAN — 0 blocking, 3 degrading, all 3 resolved
**Re-check after resolution:** clean

## Pairs examined

All 15 story pairs were reasoned through, not assumed compatible. The five pairs with genuine
interaction surface:

| Pair | Shared surface | Outcome |
|---|---|---|
| 3 × 4 × 5 | `POST /api/accounts/{id}/transactions` | **F1** — overlap, resolved |
| 1 × 2 | `GET /` rendering | **F2** — state ambiguity, resolved |
| 2 × 6 | Seeded transaction ids | Clean — both imply a global sequence; consistent |
| 5 × 3, 5 × 4 | Zero-balance boundary | Clean — zero permitted, only below-zero rejects |
| 6 × all | Test fixture state, database path | **F3** — resource contention, resolved |

The ten remaining pairs share no entity, endpoint, or fixture and cannot interact.

## Findings

### F1: Three stories assert the same six validation rules

**Stories involved:** Story 3 (page) vs Story 4 (programmatic) vs Story 5 (rules)
**Type:** overlap
**Severity:** degrading

**Description:** Story 5 asserts each rule's sentinel, wire code, and the nothing-recorded invariant.
Story 4's both-content-types matrix asserts all six rules again through both branches. Story 3
asserts `description_empty` and `amount_malformed` a third time on the form branch. Every expected
outcome agrees, so this is not a contradiction — but ownership is undefined, so `/plan` would derive
three overlapping sets of tests from it, and a later change to a rule would need finding in three
places. The suite has a 10-second budget and a 4:1 ratio target; triplicated rule tests spend both
for nothing.

**Resolution applied — Option 1 (assign ownership explicitly).**
- **Story 5** is the sole owner of rule semantics: which sentinel, which code, nothing recorded.
- **Story 4** asserts *equivalence only* — that the same rule fires for both content types,
  identified by wire code — consuming Story 5's codes as input rather than restating them. Scoped
  this way the matrix cannot drift out of agreement with Story 5.
- **Story 3** keeps one representative rejection because its real subject is that a code arriving in
  the URL becomes a visible panel, not that the rules are correct.

Amendment notes added beside the original assertions in all three stories; originals preserved.

### F2: Undefined page state for an account that does not exist

**Stories involved:** Story 1 (page rendering) vs Story 3 (the post form)
**Type:** state-conflict
**Severity:** degrading

**Description:** Story 1 asserts an unknown account renders `200` with a not-found message and no
balance element, but says nothing about the transaction list or the post form. Story 3 asserts the
form's action targets the selected account. Combined, the two admit a page that offers a form whose
action points at a nonexistent account — a submission that can only fail, and one a presenter could
trip over live. The ambiguity is in what Story 1 leaves unsaid rather than in a contradiction between
them.

**Resolution applied — Option 1 (message only).** For an unknown account the page renders the account
list and the not-found message only: no balance element, no transaction list, no post form. Amendment
note added to Story 1.

### F3: Reset tests contend on the default database path

**Stories involved:** Story 6 (reset and run) vs every other story's tests
**Type:** resource-contention
**Severity:** degrading

**Description:** Story 6 asserts `make reset` removes and recreates the database and that two
consecutive resets are byte-identical. Both assertions are genuinely file-backed — `reset` deletes a
file, which in-memory SQLite cannot express — yet the project convention is in-memory SQLite for
tests, and Story 6 simultaneously requires the suite to pass with `-count=2`. Left unqualified, two
runs of a reset test against the default `./ledger.db` contend with each other, with tests in other
packages, and with a `make dev` server holding that file open.

**Resolution applied — Option 1 (`t.TempDir()` per test).** Tests needing a real file set
`LEDGER_DB_PATH` to a `t.TempDir()` path and never touch the default `./ledger.db`. These are the one
sanctioned exception to the in-memory convention, and confining them to a per-test temporary
directory removes the contention entirely. Amendment note added to Story 6.

## Checked and clean

- **Page `200` vs API `404` for an unknown account** — a deliberate asymmetry, not a contradiction.
  The page degrades to a readable message (Story 1 explicitly asserts "not a `404`"); the JSON
  transactions route returns `404 account_not_found` (Story 2). No story asserts the page `404`s.
- **Zero balance** — consistent throughout. Story 5 accepts `-1000` against a `1000`-cent balance,
  landing exactly on `0`; Story 1 shows `$0.00` for an empty account. Only a balance *below* zero
  rejects, so zero is a valid state everywhere it appears.
- **Seeded identifiers** — Story 2 (`^txn-\d{4}$`, globally unique) and Story 6 (one unbroken global
  sequence, 24–36 rows) agree and reinforce each other. Nothing anywhere implies per-account
  numbering.
- **Single `time.Now()` call site** — Story 6 requires exactly one, inside `SystemClock`. No other
  story's criteria need a second; every timestamp reaches the domain through the injected clock.
- **Transaction-count assertions** — Story 3's reload-does-not-double and Story 4's
  nothing-recorded-on-rejection assert counts in disjoint scenarios (success then re-`GET` versus a
  rejected submission). No tension.

## Architecture-review conditions — coverage verified

All three conditions from `.docs/decisions/architecture-review-2026-08-08-base-ledger.md` are carried
by at least one story criterion:

| Condition | Covered by |
|---|---|
| C1 — ids numbered globally, not per account | Story 6 ("one unbroken globally sequential run … not restarting per account") and Story 2 (globally unique across all three accounts) |
| C2 — unrecognized `error` value renders a generic message, never an empty panel | Story 3 (`error=not_a_real_code` renders a non-empty generic message) |
| C3 — all ids share one width | Story 2 (`^txn-\d{4}$`, plus the scenario documenting that mixed widths break the tiebreak) |

## Not treated as conflicts

The unlocked read-then-write balance check is a recorded accepted characteristic
(`.docs/architecture/sequences.md`; "Known Characteristics" in the PRD). No story asserts
serialisation, by design, and the absence of a concurrency story is not a gap.

## Verdict

**Zero blocking conflicts.** Three degrading findings, all resolved by amendment rather than accepted
as compromises. No resolution changed an architectural decision, so no ADR was superseded. Proceed to
`/plan`.
