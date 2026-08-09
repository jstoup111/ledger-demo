# Implementation Plan: Seeded transactions carry distinct recorded times

**Date:** 2026-08-09
**Design:** [2026-08-09-seed-recorded-times.md](../specs/2026-08-09-seed-recorded-times.md)
**Stories:** [seed-recorded-times.md](../stories/seed-recorded-times.md)
**Tier:** S (`.docs/complexity/seed-recorded-times.md`) — conflict-check, architecture-diagram,
architecture-review, and coherence-check are skipped by the tier rules
**Origin:** intake issue `jstoup111/ledger-demo#3`

> **Filename note:** this plan is `seed-recorded-times.md`, not the skill's default
> `YYYY-MM-DD-<feature>.md`, because the plan stem must match `.docs/complexity/seed-recorded-times.md`
> for the daemon to resolve the complexity tier at build time.

## Summary

Three tasks. Two are TDD cycles over the seed dataset and its test; the third updates the README. The
entire production diff lives in one file, `cmd/server/seed.go`, and consists of one new struct field,
twenty-one values for it, and four changed lines in the loading loop. No other package is touched.

## Technical Approach

**Each row declares when it was recorded, as a fixed offset before the injected instant.** The seed
command already receives its instant by injection. `loadSeedData` reads that instant **once** into an
anchor, and each seeded row declares a `time.Duration` saying how long *before* the anchor it was
recorded. The row is then posted through the existing `ledger.PostTransaction` with a fixed clock
carrying `anchor.Add(-offset)`.

This is the approach because it is the only one that puts the answer to "which two rows share an
instant?" on the same line as the rows themselves (NFR-4, projector legibility of the code). Three
alternatives were weighed and rejected:

| Alternative | Why not |
|---|---|
| Derive offsets arithmetically from the row's index (`-(n-1-i) * 26h`) | Smallest diff, but the deliberate shared-instant pair then needs a special-case branch inside the arithmetic — which reads on a projector as a bug, not as a decision. Spacing is also uniform, which reads as generated rather than as history. |
| Give `PostTransaction` an explicit recorded-at parameter | Changes a domain signature used by both HTTP posting paths to serve a seed-only concern, and weakens the injected-clock decision by adding a second way time enters the domain. |
| Write the rows first and adjust `created_at` afterwards, in the store | Rewrites rows in an append-only log and adds store surface that exists only for the fixture. Contradicts the premise the id-derivation decision rests on. |
| Hard-code each row's absolute instant with `time.Date(...)` | Deterministic, but the injected clock stops being the source of seeded time — injection becomes decorative, and the seed's reference instant is no longer one fact but twenty-one. |

**Direction is the load-bearing constraint.** The listing is `ORDER BY created_at DESC, id DESC`. Ids
are assigned in the sequence rows are recorded, so for the two clauses to agree, the declared offsets
must be **non-increasing down each account's list** — the first row is the furthest in the past, the
last row is the nearest to the anchor. Any row later in the list carrying a larger offset would sort
the listing differently from the id sequence, silently reordering the demo data. Assert this, do not
assume it.

**Nothing else in the dataset moves.** Row order within each account stays exactly as committed, which
is what keeps identifiers (`txn-0001` … `txn-0021`), the running balance each row was validated
against, and the first account's `128350`-cent total unchanged. Offsets are metadata on rows whose
sequence is fixed; they are not a re-sort of the dataset.

**Whole hours only.** Every offset is a whole number of days plus whole hours, so every seeded instant
is whole-second and the whole-second RFC 3339 rendering is lossless (NFR-3). No offset is sub-second.

**One name change.** `loadSeedData`'s current parameter is named `clock`, which shadows the `clock`
package. Since the body now refers to `clock.FixedClock`, rename the parameter to `seedClock`. This is
the only incidental edit in the change.

## Prerequisites

None. No dependency is added, no schema migration exists to run, and `make reset` already drops the
database file wholesale.

## Reference: the offset table to implement

Offsets are **before** the anchor. The anchor is the instant injected into the seed command, unchanged
at `2026-08-08T14:30:00Z`. Resulting instants are given so the executor can assert exact values instead
of recomputing them. `day` and `hour` are file-local constants (`24 * time.Hour`, `time.Hour`).

### `acct-1` — Everyday Checking (12 rows, in the committed order)

| # | Id | Description | Offset before anchor | Resulting instant |
|---|---|---|---|---|
| 1 | `txn-0001` | Paycheck | `32*day` | `2026-07-07T14:30:00Z` |
| 2 | `txn-0002` | Grocery market | `30*day + 3*hour` | `2026-07-09T11:30:00Z` |
| 3 | `txn-0003` | Electric bill | `27*day + 5*hour` | `2026-07-12T09:30:00Z` |
| 4 | `txn-0004` | Music subscription | `24*day + 1*hour` | `2026-07-15T13:30:00Z` |
| 5 | `txn-0005` | Neighborhood cafe | `20*day + 6*hour` | `2026-07-19T08:30:00Z` |
| 6 | `txn-0006` | Cashback reward | `17*day + 2*hour` | `2026-07-22T12:30:00Z` |
| 7 | `txn-0007` | Fuel station | `13*day + 7*hour` | `2026-07-26T07:30:00Z` |
| 8 | `txn-0008` | Phone bill | `10*day + 4*hour` | `2026-07-29T10:30:00Z` |
| 9 | `txn-0009` | Dinner with friends | `6*day + 5*hour` | `2026-08-02T09:30:00Z` |
| 10 | `txn-0010` | Tax refund | `3*day + 3*hour` | `2026-08-05T11:30:00Z` |
| 11 | `txn-0011` | Membership renewal | `1*day + 2*hour` in Task 1, **changed to `0` in Task 2** | `2026-08-07T12:30:00Z`, then `2026-08-08T14:30:00Z` |
| 12 | `txn-0012` | Concert tickets | `0` | `2026-08-08T14:30:00Z` |

Rows 11 and 12 are the deliberate shared-instant pair, introduced by Task 2. They sit at the **top** of
the newest-first listing, which is where a presenter can point at them without scrolling. Because they
share the anchor exactly, the reference instant used by the API response contract's worked example stays
present in seeded data.

### `acct-2` — Weekend Savings (9 rows, in the committed order, all distinct)

| # | Id | Description | Offset before anchor | Resulting instant |
|---|---|---|---|---|
| 1 | `txn-0013` | Opening balance | `60*day` | `2026-06-09T14:30:00Z` |
| 2 | `txn-0014` | Monthly savings contribution | `52*day + 2*hour` | `2026-06-17T12:30:00Z` |
| 3 | `txn-0015` | Travel envelope deposit | `45*day + 4*hour` | `2026-06-24T10:30:00Z` |
| 4 | `txn-0016` | Weekend plans savings | `38*day + 1*hour` | `2026-07-01T13:30:00Z` |
| 5 | `txn-0017` | Train tickets | `31*day + 6*hour` | `2026-07-08T08:30:00Z` |
| 6 | `txn-0018` | Cabin weekend savings | `24*day + 3*hour` | `2026-07-15T11:30:00Z` |
| 7 | `txn-0019` | Museum passes | `17*day + 5*hour` | `2026-07-22T09:30:00Z` |
| 8 | `txn-0020` | Next adventure savings | `10*day + 2*hour` | `2026-07-29T12:30:00Z` |
| 9 | `txn-0021` | Cabin deposit | `2*day + 8*hour` | `2026-08-06T06:30:00Z` |

### `acct-3` — Project Fund

Seeded empty. Unchanged, and nothing is added to it.

## Tasks

### Task 1: Declare a per-row recorded offset and derive each row's instant from the injected anchor
**Story:** Story 1 / FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8
**Type:** feature

The audience-facing outcome. At the end of this task every seeded row in both populated accounts
carries a **distinct** recorded time; the deliberate shared pair arrives in Task 2.

**Steps:**
1. Write failing tests in `cmd/server/seed_test.go`. Replace the existing per-row assertion that every
   `CreatedAt` equals the injected instant — that assertion is the specified behavior being changed, so
   it is removed, not worked around. In its place, extend `TestLoadSeedDataIsDeterministic` (or add a
   sibling table-driven test over the same snapshot) with:
   - **Bounded:** no seeded `CreatedAt` is after the anchor, and at least one equals it exactly.
   - **Whole-second:** for every row, `CreatedAt.Truncate(time.Second).Equal(CreatedAt)` — so nothing
     finer than the displayed resolution is stored.
   - **UTC:** every `CreatedAt` renders with a `Z` offset.
   - **Direction:** for each populated account, walking its rows in *ascending id* order (the order they
     were recorded), each row's `CreatedAt` is greater than or equal to the previous row's. Never less.
   - **Varying:** `acct-1`'s twelve rows contain more than one distinct `CreatedAt` value.
   - **All distinct in `acct-2`:** its nine rows contain nine distinct `CreatedAt` values.
   - **Span:** the earliest and latest seeded instants are at least 28 days apart, so the column reads
     as weeks of history rather than as one afternoon.
   - Do **not** assert that all of `acct-1`'s rows are distinct — Task 2 deliberately makes two equal,
     and this test must still pass afterwards.
   - Leave the existing assertions on account count, id pattern, global id sequence, per-account row
     counts, the `128350`-cent total, the forbidden-term check, and the two-snapshot `DeepEqual`
     untouched. They are the regression guard for FR-8 and must keep passing unmodified.
2. Verify the tests fail (RED) — every seeded row currently carries the anchor, so Direction passes
   vacuously while Varying, Span, and the `acct-2` distinctness check fail.
3. In `cmd/server/seed.go`, add file-local constants `day = 24 * time.Hour` and `hour = time.Hour`, and
   add a `recordedBefore time.Duration` field to `seedTransaction` with a doc comment stating the two
   invariants a reader must not break: offsets are **non-increasing down each account's list** (so
   `created_at DESC` agrees with the `id DESC` tiebreak), and no offset is negative (so nothing seeded
   can outrank a transaction recorded live during the demo).
4. Populate `recordedBefore` for all twenty-one rows from the Reference table above, using the Task 1
   value for row 11 of `acct-1` (`1*day + 2*hour`).
5. Change `loadSeedData` to read the anchor once (`anchor := seedClock.Now()`) before the loop, and to
   post each row with `clock.FixedClock{T: anchor.Add(-transaction.recordedBefore)}` instead of the
   injected clock directly. Rename the parameter from `clock` to `seedClock` so it stops shadowing the
   `clock` package. Do not reorder the dataset, do not touch `InsertAccount`, and do not change the
   error-wrapping messages.
6. Verify the tests pass (GREEN).
7. Confirm no wall-clock read was introduced: a repository search for `time.Now()` still returns exactly
   one hit, in `internal/clock/clock.go`.
8. Commit: "seed: give each seeded transaction its own recorded time"

**Files:** `cmd/server/seed.go`, `cmd/server/seed_test.go`
**Wired-into:** `cmd/server/main.go#seed` — the existing call site, whose signature and injected clock
are unchanged; no new wiring
**Dependencies:** none

### Task 2: Introduce the deliberate shared-instant pair and assert it drives the tiebreak
**Story:** Story 2 / FR-4, FR-5
**Type:** feature

The coverage outcome. Task 1 leaves every row distinct, which makes the listing's id tiebreak
unexercised against seeded data — the same defect the intake issue warned about, in mirror image. This
task puts exactly one pair back, deliberately, and asserts its presence so it cannot be removed
silently.

**Steps:**
1. Write failing tests in `cmd/server/seed_test.go`, over the newest-first listing of `acct-1` as
   returned by the store (not over a re-sorted copy):
   - **Exactly one pair:** counting `acct-1`'s twelve `CreatedAt` values by value, exactly one value
     occurs twice and every other occurs once. This asserts both the presence of the pair (so removing
     it fails loudly rather than making an ordering check vacuous) and that there is no second pair.
   - **No triple:** no `CreatedAt` value occurs three or more times.
   - **Adjacent and id-ordered:** walking the newest-first listing, the shared pair appears at
     consecutive positions, and the earlier-listed of the two carries the lexicographically greater id.
     Assert the exact ids — `txn-0012` then `txn-0011`.
   - **Tiebreak genuinely exercised:** the "equal times ⇒ descending ids" check over the listing is
     entered at least once. Count the comparisons where the two times are equal and assert that count
     is exactly one, so the check can never pass by never running.
   - **`acct-2` has no pair:** its nine `CreatedAt` values are nine distinct values (already asserted in
     Task 1; keep it, and read it here as the contrast case).
   - **Stable across resets:** the existing two-snapshot `DeepEqual` already covers this; add no
     duplicate assertion for it.
2. Verify the tests fail (RED) — after Task 1, `acct-1` has twelve distinct instants, so the
   exactly-one-pair count and the exercised-tiebreak count are both zero.
3. In `cmd/server/seed.go`, change `acct-1` row 11 (`Membership renewal`) from `1*day + 2*hour` to `0`,
   so it shares the anchor instant with row 12 (`Concert tickets`). This is the whole of the production
   change; nothing else moves.
4. Verify the tests pass (GREEN). `Concert tickets` (`txn-0012`) leads the listing and
   `Membership renewal` (`txn-0011`) follows it, at the same instant.
5. Confirm the pair introduces nothing else: no uniqueness constraint is added anywhere, the schema is
   untouched, no warning or annotation is rendered, and the template and stylesheet are unmodified.
   `git diff --stat` for this task shows only `cmd/server/seed.go` and `cmd/server/seed_test.go`.
6. Commit: "seed: keep one deliberate pair of equal recorded times so the id tiebreak stays covered"

**Files:** `cmd/server/seed.go`, `cmd/server/seed_test.go`
**Wired-into:** none — no new production surface; the pair is consumed by the existing
`ORDER BY created_at DESC, id DESC` in `internal/store/sqlite.go`, which is unchanged
**Dependencies:** Task 1

### Task 3: Describe the seeded history in the README
**Story:** Story 1 / FR-2, FR-9
**Type:** documentation

The demo's front door should say what a reset produces, so a presenter can predict the column before
projecting it. Required by the docs-track-features rule: this change alters what the page displays.

**Steps:**
1. Add a short **Seed data** subsection under Quick Start in `README.md` stating: three accounts; the
   first with twelve transactions and the second with nine, the third seeded empty; recorded times
   spanning roughly two months of history ending at a fixed reference instant; and that the first
   account deliberately holds one pair of transactions recorded at the same instant, which is what keeps
   the newest-first ordering rule's identifier tiebreak covered by a real test.
2. Do not restate the offset table and do not add a timestamp that will drift. Four to six lines.
3. Leave the stale "Status: Scaffold only" section alone — correcting it is not this feature's scope and
   editing it would put an unrelated claim in this diff.
4. Verify the formatting and vetting gates are still clean (no Go file changed here).
5. Commit: "docs: describe the seeded transaction history in the README"

**Files:** `README.md`
**Wired-into:** none (documentation)
**Dependencies:** Task 2

## Task Dependency Graph

```
Task 1  (seed offsets + varying-times tests)
   └── Task 2  (shared-instant pair + tiebreak-coverage tests)
          └── Task 3  (README seed-data note)
```

Strictly sequential. Tasks 1 and 2 edit the same two files and Task 2's RED depends on Task 1's data
being in place; Task 3 documents the end state, so it follows both.

## Verification (owned by the gates, not by a task)

Listed so the executor knows what will judge the work. None of these is additional implementation, and
no task exists to "make the feature pass" — a failure here returns to the owning task above.

- Story-level acceptance specs are authored at BUILD entry by `/writing-system-tests` from the two
  stories, before implementation. This plan deliberately authors none, and no task edits
  `test/acceptance/`.
- The existing acceptance spec that walks the first account's real seeded listing and checks
  "equal times ⇒ descending ids" becomes genuinely exercised by Task 2 rather than trivially true. It
  needs no edit.
- Aggregate suite, repeated-run determinism (`-count=2`), the 10-second budget, `gofmt`, and `go vet`
  are the project's standing gates.
- The single-wall-clock-read property is asserted inside Task 1, step 7, and is also a standing story
  criterion.

## Out of Scope for This Plan

- Any change to `internal/ledger`, `internal/store`, `internal/httpapi`, `internal/clock`, `web/`, or
  the route set. If a task appears to need one, stop — the approach above is wrong and this returns to
  DECIDE rather than expanding here.
- Any uniqueness constraint, dedup mechanism, idempotency key, or duplicate-detection reasoning
  prompted by the deliberate shared-instant pair. This is the project's primary non-goal and the pair
  must not become a reason to approach it.
- Any date-range filter, grouping, per-day subtotal, or sort control over the newly-varied times.
- Correcting the README's stale project-status section.
