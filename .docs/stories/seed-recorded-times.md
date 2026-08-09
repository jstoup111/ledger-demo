# Stories — Seeded transactions carry distinct recorded times

**Status:** Accepted

**Feature:** seed-recorded-times · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-08-09-seed-recorded-times.md` (Approved, FR-1 … FR-9)
**Origin:** intake issue `jstoup111/ledger-demo#3`
**Constrained by:** the Accepted injected-clock and transaction-id/ordering decisions in
`.docs/decisions/`, and the API response contract

Two stories. The first is the audience-facing outcome (the column varies and still reads correctly);
the second is the coverage outcome (the shared-instant tiebreak stays demonstrable and asserted against
real seeded data). They are separated because they fail for different reasons and are verified by
different observations, not because they touch different code.

## Negative-path categories evaluated

Every category is evaluated explicitly. Most do not apply, and saying so is more useful than inventing
a scenario for a change that alters twenty-one timestamps in a fixture.

| Category | Applies? |
|---|---|
| Invalid input | No — nothing here reads user input. The recorded time is not a validation input, and the six rejection rules do not consult it |
| Data integrity | **Yes** — the seeded dataset's identifiers, amounts, descriptions, membership, and balances must survive the change untouched (Story 1) |
| Model-level immutability | **Yes** — the log stays append-only; no row is rewritten after insertion to adjust its time (Story 1) |
| Invariant side-effect on alternate branches | **Yes** — the page's listing and the programmatic listing are two renderings of one order and must still agree exactly (Story 1) |
| Ordering / total-order collapse | **Yes** — this is the whole of Story 2: the primary sort clause must now do real work without the tiebreak clause going unexercised |
| Dependency unavailability | No — no new dependency; the seed's failure modes on an unopenable database are unchanged and already covered |
| Auth / permission failures | No — auth is an explicit non-goal; there is no principal |
| Timeouts & network errors | No — no network calls exist; the demo runs offline |
| Resource exhaustion | No — twenty-one fixture rows |
| Partial failure & rollback | No — seeding is a sequence of single inserts, unchanged by this feature |
| Concurrent access | No — one presenter, one browser; the accepted unlocked-balance-check characteristic is untouched and no requirement here asks for it |
| Cascade deletion effects | No — nothing is deleted; a reset drops the database file wholesale |
| Exception class hierarchy | No — sentinel errors compared with `errors.Is`, and this feature adds none |
| Dedup / idempotency key analysis | No, and deliberately so. Story 2 creates a pair of rows sharing one recorded instant. That is ordering coverage. It is **not** a duplicate to be detected, and no uniqueness criterion of any kind may be introduced over it |
| Determinism regression | **Yes** — a varying column is worthless if it varies between resets. Both stories assert reproducibility |

---

**Status:** Accepted

## Story 1: The Recorded column reads as real history

**Requirement:** FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8, FR-9

As a presenter, I want the transaction table's Recorded column to show a different time on each row, so
that an audience reading it from across the room sees plausible history instead of a column of twelve
identical values that looks like broken data.

### Acceptance Criteria

#### Happy Path

- Given a freshly reset database, when a client requests the first account's transactions, then more
  than one distinct recorded value appears among them, and the values span at least several weeks
  ending at or before the seed's fixed reference instant.
- Given a freshly reset database, when a client requests the second account's transactions, then every
  row's recorded time differs from every other row's in that account — no two are equal.
- Given a freshly reset database, when the page renders an account, then each visible row's Recorded
  cell shows the same value the programmatic listing reports for that row, and consecutive rows in the
  first account show different values except for the single deliberate pair covered by Story 2.
- Given a freshly reset database, when either listing is read, then the first row is the most recently
  recorded one and each subsequent row's recorded time is earlier than or equal to the row above it —
  never later.
- Given a freshly reset database, when both the page and the programmatic listing are read for the same
  account, then they present the rows in the same order, row for row.
- Given a freshly reset database, when a presenter records a transaction from the page, then that new
  row appears at the top of the list, above every seeded row.
- Given a freshly reset database, when every seeded recorded time is compared to the seed's fixed
  reference instant, then none is later than it, and at least one equals it exactly — so the reference
  instant used in the API response contract's worked example is still present in seeded data.
- Given a seeded recorded time, when it is rendered for display, then the rendered value is exact: the
  stored instant carries no finer component that the display rounds away, so what the audience reads is
  the whole truth about that row.

#### Negative Path

- Given two consecutive resets, when every seeded row from each is compared field for field — including
  recorded times and identifiers — then the two sets are indistinguishable. A recorded time that varied
  between resets fails this story outright.
- Given the suite is run twice in one invocation (`-count=2`), when it completes, then it passes both
  times with no ordering dependency and no sleeping anywhere.
- Given a freshly reset database, when the first account's balance is read, then it is still exactly
  `128350` cents — varying recorded times must not have reordered the sequence rows were recorded in,
  because that sequence fixes the running balance each row was validated against.
- Given a freshly reset database, when the seeded identifiers are collected, then they still form one
  unbroken global sequence with no per-account restart, no gaps, and no duplicates, and each row still
  carries the same identifier it carried before this change.
- Given a freshly reset database, when the third account is requested, then it still returns an empty
  listing and a zero balance — it remains seeded empty.
- Given a freshly reset database, when the seeded amounts, descriptions, and account membership are
  compared to what they were before this change, then all three are identical. Only recorded times
  changed.
- Given the repository, when it is searched for a system-clock read, then exactly one hit is returned,
  in the one place permitted to hold it. A second call site fails this story.
- Given the seeded data, when it is inspected for randomness, then there is none — no shuffling, no
  pseudo-random offsets, no dependence on the order a map is iterated in.
- Given the page and stylesheet, when compared before and after, then neither changed, the route count
  is still exactly five, and the page still carries no JavaScript.

### Done When

- [ ] The first account's seeded recorded values contain more than one distinct value; the second
      account's are all distinct.
- [ ] Recorded times never run counter to the recording sequence: within an account, an earlier-recorded
      row never carries a later time than a later-recorded one.
- [ ] No seeded recorded time is later than the seed's fixed reference instant, and at least one equals
      it.
- [ ] Every seeded recorded instant is exact at the resolution it is displayed at, so no displayed value
      is a rounded rendering of something finer.
- [ ] Two consecutive resets produce identical rows, compared including recorded times and identifiers.
- [ ] The first account still sums to `128350` cents; the third account is still empty; identifiers,
      amounts, descriptions, and membership are unchanged.
- [ ] A repository search for a system-clock read returns exactly one hit.
- [ ] The suite passes, passes again with `-count=2`, and completes under 10 seconds.
- [ ] Formatting and vetting gates are clean.
- [ ] Route count is five; the page has no JavaScript; the stylesheet is unchanged.

---

**Status:** Accepted

## Story 2: The shared-instant ordering rule stays demonstrable on stage

**Requirement:** FR-4, FR-5

As a presenter, I want exactly one pair of seeded rows to share a recorded instant, so that I can point
at two real rows on the projector and show that rows recorded at the same moment still hold a fixed,
repeatable order — and so the test that proves it is asserting something that actually occurs in seeded
data rather than a condition that is never true.

### Acceptance Criteria

#### Happy Path

- Given a freshly reset database, when the first account's listing is read, then exactly one pair of
  rows in it shares an identical recorded time, and every other row in that account carries a time no
  other row in it shares.
- Given that pair, when the listing is walked from the top, then the two rows are adjacent — nothing
  sorts between them — and the one with the higher identifier appears first.
- Given two consecutive resets, when the pair is located in each, then it is the same two rows in the
  same relative order both times.
- Given a freshly reset database, when the first account's listing is walked, then a check of the form
  "if two neighbouring rows share a recorded time, the earlier-listed one must carry the higher
  identifier" is exercised at least once with its condition genuinely true, rather than passing because
  the condition never occurs.
- Given a freshly reset database, when the first account's recorded values are counted by value, then
  exactly one value appears twice and every other appears once.
- Given the second account, when its listing is read, then it contains no shared-instant pair — so the
  two accounts between them demonstrate both the varying case and the tie case.

#### Negative Path

- Given the first account's listing, when it is examined, then it contains **no more than one**
  shared-instant pair, and no instant is shared by three or more rows. A second pair is a defect: it
  makes the column look repetitive again and adds no coverage the first pair does not already give.
- Given the shared pair, when its two rows are inspected, then they are **not** treated as duplicates in
  any way: no uniqueness constraint exists over the recorded time, no dedup or idempotency mechanism is
  introduced, no warning is rendered, and the pair is not annotated, highlighted, collapsed, or
  filtered anywhere in the page or the listing. Two rows recorded at one instant are ordinary data.
- Given the pair shares an instant, when either listing is read, then the relative order of the two is
  never observed to differ between reads, between resets, or between the page and the programmatic
  listing. An order that varies means the total order collapsed and fails this story.
- Given every seeded row were made distinct in recorded time, when the ordering assertion above is run,
  then it must be reported as vacuous rather than passing silently — a test whose condition never occurs
  is the defect this story exists to prevent, and it must fail loudly if the pair is ever removed.
- Given the third account, when it is requested, then it is still empty — no pair is added to it, and it
  remains the empty-history demonstration.

### Done When

- [ ] The first account's seeded listing contains exactly one shared-instant pair; no instant is shared
      by three or more rows.
- [ ] The pair's two rows are adjacent in the listing and ordered by descending identifier.
- [ ] A test asserts the pair's *presence* in the first account's seeded listing, so removing the pair
      fails the suite rather than quietly making an ordering assertion vacuous.
- [ ] The tiebreak assertion over the first account's real seeded listing runs with its condition true.
- [ ] The second account has no shared-instant pair; the third account is still empty.
- [ ] No uniqueness constraint, dedup mechanism, idempotency key, warning, annotation, or filter is
      introduced anywhere on account of the shared pair.
- [ ] Two consecutive resets place the same pair in the same relative order.
