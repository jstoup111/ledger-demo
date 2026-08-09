# PRD — Seeded transactions carry distinct recorded times

**Status:** Approved
**Date:** 2026-08-09
**Feature:** seed-recorded-times · **Tier:** S · **Track:** product
**Origin:** intake issue `jstoup111/ledger-demo#3`
**Supersedes:** nothing. Amends no accepted requirement.

## Problem

The transaction table has three columns — Description, Amount, **Recorded** — and on a freshly reset
database the Recorded column shows one value repeated down every seeded row:

```
$ curl -s localhost:8080/api/accounts/acct-1/transactions | jq -r '.[].created_at' | sort | uniq -c
  12 2026-08-08T14:30:00Z
```

Nothing is incorrect and no test is wrong. Every seeded row is stamped from the same injected instant,
deliberately and under test. But this project's entire purpose is to be projected in front of an
audience, and a column of twelve identical timestamps is the one thing an audience can see is wrong
from across a room. It reads as broken data at exactly the moment the demo is asking to be trusted.

It also wastes the seeded dataset's best illustration of a real ordering rule. The newest-first listing
is a stable total order in which two rows recorded at the same instant still hold a defined position
relative to each other. Today *every* seeded row shares an instant, so that tiebreak is doing all of
the sorting work and none of it is visible or meaningful.

## Goals

1. The Recorded column reads as genuine history: within an account, the displayed times vary row to
   row, spanning a plausible stretch of past weeks.
2. Every determinism guarantee the demo depends on survives untouched — two resets remain
   indistinguishable, and the suite remains reproducible.
3. The "two rows at the same instant still order predictably" guarantee stays **demonstrable from
   seeded data**, on stage, and stays covered by a test that runs against seeded data rather than only
   against a hand-built fixture.

## Non-Goals

Restated in full because this project's non-goals are load-bearing and this change touches the one
column that might tempt someone toward them. This feature must not introduce, hint at, or leave a hook
for any of the following:

- **Duplicate or double-charge detection of any kind.** In particular: this feature deliberately
  creates a pair of seeded rows sharing one recorded instant. That pair is ordering coverage, and it is
  **not** a duplicate, not a near-duplicate, and not a signal to be detected. No idempotency key, no
  dedup window, no "same amount within N seconds" reasoning, and **no uniqueness constraint of any kind
  beyond the existing primary key** — least of all one on the recorded time.
- Overdraft allowance, fees, or percentage calculations.
- Pending transactions or holds; available-versus-posted balance.
- Statements, exports, or reporting — including any date-range filter, date grouping, "this month"
  heading, or per-day subtotal over the newly-varied times.
- Interest or rounding rules.
- Authentication, users, sessions, or multi-tenancy.
- Transfers between accounts, or any balancing counter-entry.
- Containerization, continuous integration, or deployment tooling.
- Metrics, tracing, or structured logging beyond what the standard library provides.

Also out of scope, though not project non-goals: any change to the page's markup, its stylesheet, the
set of HTTP routes, the displayed time *format*, sorting or filtering controls, or the amounts,
descriptions, account membership, or identifiers of any seeded row.

## Functional Requirements

### What the audience sees

- **FR-1** Within an account, seeded transactions show **different** recorded times from one another,
  so the Recorded column varies row to row rather than repeating one value. Concretely: the distinct
  recorded values in the first account's listing number more than one, and in the second account every
  row's recorded time differs from every other row's.
- **FR-2** The seeded recorded times read as a plausible history rather than as generated noise: they
  span a stretch of past weeks, they run in the same direction as the order the rows were recorded in
  (older rows carry earlier times), and each row's displayed time is an exact value rather than a
  truncated or rounded rendering of a finer one.
- **FR-3** The newest-first listing is unchanged in its observable order: the row a viewer sees first
  is still the most recently recorded one, on the page and in the programmatic listing alike, and the
  two still agree exactly.

### What the presenter can demonstrate

- **FR-4** **Exactly one** pair of seeded rows in the first account shares an identical recorded time,
  so the "same instant, still a defined relative order" guarantee is demonstrable live against seeded
  data. Because two rows sharing an instant can have nothing sort between them, that pair appears as
  two adjacent rows in the listing, and their relative order is fixed and repeatable across resets.
  Every other seeded row in that account carries a recorded time no other row in it shares.
- **FR-5** The second account's seeded rows are all distinct in recorded time, so the presenter has one
  account that shows a fully-varying column and one that also shows the shared-instant case.
- **FR-6** A transaction recorded live during the demo still appears at the top of the list. Every
  seeded recorded time is at or before the seed's fixed reference instant, and that instant is in the
  past relative to any live demo, so nothing seeded can outrank a live entry.

### What must not change

- **FR-7** Two consecutive resets produce indistinguishable seeded state, including every recorded
  time, with no run-to-run variation of any kind.
- **FR-8** The seeded dataset is otherwise byte-for-byte the same dataset: three accounts; the first
  and second carrying the same rows in the same sequence with the same amounts, descriptions, and
  identifiers as before this change; the third seeded empty. The first account's transactions still sum
  to exactly `128350` cents, so the worked examples in the base ledger's stories and API response
  contract continue to hold against seed data.
- **FR-9** A reset remains a single action, and running the demo remains a single action.

## Non-Functional Requirements

- **NFR-1 — Determinism (inherited, non-negotiable).** No randomness anywhere. Recorded times are
  reproducible; two resets produce indistinguishable state; the suite passes repeated runs
  (`-count=2`) with no ordering dependency between tests and no sleeping.
- **NFR-2 — Exactly one wall-clock read in the repository (inherited external constraint).** The
  project's injected-clock decision permits exactly one call site reading the system clock, inside
  `SystemClock`. This feature adds none: a search of the repository for a system-clock read must still
  return exactly one hit, in that one place. Seeded times continue to derive from the instant injected
  into the seed command.
- **NFR-3 — Projector legibility of the data.** The varied times must be legible and distinguishable at
  1280×720 from across a room, which means neighbouring rows must differ by an amount a viewer can see
  at the displayed resolution — not by a fraction of a second that the display rounds away.
- **NFR-4 — Projector legibility of the code.** The seeded dataset is read on a projector by an
  audience following a live diff. The declaration of what each row's recorded time is must be readable
  next to the row it belongs to, so a viewer can see at a glance which two rows deliberately share an
  instant. This is a real constraint on the solution, not a preference.
- **NFR-5 — Suite budget.** The full suite stays under 10 seconds and stays fully deterministic.
- **NFR-6 — Lint clean.** Formatting and vetting gates pass with no findings.
- **NFR-7 — Surface unchanged.** Exactly five HTTP routes, before and after. No JavaScript on the page.
  No stylesheet change, and therefore no change to the fixed palette, the 20px base, or the light-only
  theme.

## Acceptance Criteria

- Reset, run, open the page, and read the Recorded column: consecutive rows show different values, and
  the visible span of the column covers past weeks rather than one instant.
- `curl -s localhost:8080/api/accounts/acct-1/transactions | jq -r '.[].created_at' | sort | uniq -c`
  reports more than one distinct value, with exactly one value counted twice.
- The programmatic listing's first element is the same row the page shows first, for every account.
- Recording a transaction from the page puts it at the top of the list.
- Two consecutive resets produce identical data, compared including recorded times and identifiers.
- The first account's balance is still exactly `128350` cents; the third account still returns an empty
  listing.
- The suite passes, passes again with `-count=2`, completes under 10 seconds, and the formatting and
  vetting gates report nothing.
- A repository-wide search for a system-clock read still returns exactly one hit.

## Governing Decisions (conformance check)

Checked before any effort is spent, per the design-conformance rule. Both governing decisions are
`Status: Accepted` and **neither is amended by this feature**:

| Decision | What it constrains | Conformance |
|---|---|---|
| `adr-2026-08-08-injected-clock.md` | Time is injected; only `SystemClock` reads the system clock | **Conforms.** Seeded times continue to originate from the instant injected into the seed command. Deriving fixed, declared offsets from an injected instant is not a clock read, and adds no call site. |
| `adr-2026-08-08-deterministic-transaction-ids-and-ordering.md` | Sequential zero-padded ids; newest-first is a total order tie-broken by id | **Conforms, and this feature protects it.** The ADR's stated worry was that identical timestamps make "newest first" an assertion about something undefined. Varying the times exercises the primary sort clause for the first time, and FR-4 keeps a deliberate pair on the tiebreak clause so the second is still exercised too. Identifiers, their sequence, and their global numbering are untouched. |
| `api-response-contract.md` | Timestamps are RFC 3339 in UTC; the worked example uses `2026-08-08T14:30:00Z` | **Conforms.** Times remain RFC 3339 UTC. The seed's reference instant remains present in seeded data, so the document's worked example stays truthful without amendment. |
| `styleguide.md` | 20px base, light theme only, six fixed colours, no media/keyframe/font-face rules | **Conforms.** No stylesheet or template change is in scope. |

No new ADR is authored. This feature introduces no seam, no dependency direction, no schema decision,
and no mechanism the two ADRs above do not already govern; at Tier S `/architecture-review` is skipped
and there is nothing for it to decide.

## Dependencies

Pre-existing constraints this feature must work within (external to the change, named here as product
reality rather than as chosen mechanism):

- The transaction record already carries a recorded instant, stored at sub-second resolution and
  displayed at whole-second RFC 3339 UTC resolution. Times closer together than the displayed
  resolution would be indistinguishable to a viewer — NFR-3 exists because of this.
- The newest-first listing is ordered by recorded time descending, then by identifier descending. Both
  the page and the programmatic listing use it, which is why FR-2's direction constraint and FR-3's
  agreement guarantee are the same constraint seen twice.
- The recorded time is not a validation input: the six rejection rules do not read it, and the balance
  is a sum over amounts, so varying recorded times cannot change any balance or any rejection.
- Seeded rows are recorded through the same path a live transaction uses, so the sequence they are
  recorded in fixes their identifiers and the running balance each one is validated against. FR-8's
  "same rows in the same sequence" is what keeps identifiers and balances stable.

## Assumption Ledger

Recorded because the operator was unavailable to confirm these interactively. Each is stated with its
confidence, its impact if wrong, and how to confirm it. **These are assumed, operator unavailable — not
operator-approved.**

| # | Assumption | Confidence | Basis | Impact if wrong | How to confirm |
|---|---|---|---|---|---|
| A1 | Deriving fixed per-row offsets from the *injected* instant conforms to the injected-clock decision and needs no ADR amendment. | 90% | `inferred` — the ADR's rule is about *reading* the clock, and the decision text names `SystemClock` as the only permitted reader; the seed already receives its instant by injection. Verified that the repository contains exactly one system-clock call site. | An ADR amendment would be required before BUILD, and the tier would arguably rise to M to carry `/architecture-review`. No requirement here changes; only the paperwork. | Operator reads FR-2/NFR-2 and the Governing Decisions table and confirms, or asks for an amendment. |
| A2 | Tier **S** is correct, so `/architecture-diagram`, `/architecture-review`, `/conflict-check`, and `/coherence-check` are legitimately skipped. | 85% | `inferred` from the signal table in `.docs/complexity/seed-recorded-times.md`, and consistent with the `size: S` label already on the originating intake issue. | Skipped DECIDE steps would have to be run before landing; the spec content itself would not change. | Operator confirms the tier. |
| A3 | Day-and-hour scale spacing is the right granularity — plausible as a real statement, and coarse enough to be visible at the displayed whole-second resolution. | 80% | `inferred` from NFR-3 plus the displayed resolution; the intake filer's own hypothesis suggested day-scale offsets. | Only the chosen spacing values change. Every FR still holds; the plan's offset table would be retuned. | Operator looks at the rendered column once and says whether it reads as plausible. |
| A4 | The deliberate shared-instant pair belongs in the **first** account, adjacent, near the top of its listing. | 85% | `verified` that the first account is the one the acceptance ordering specs walk (they take the first account by ascending id), so a pair placed elsewhere leaves the tiebreak assertion vacuous there. Position "near the top" is a stage-value judgement, not a verified constraint. | If the pair went elsewhere, FR-4's coverage goal would be met in letter but the existing ordering spec would still not exercise the tiebreak — the original defect in a new form. | Operator confirms, or the plan's test explicitly asserts the pair is present in the first account's listing (it does). |
| A5 | It is acceptable that the pair's two descriptions have no narrative reason to share an instant. | 75% | `unverified` — any chosen pair is arbitrary; two purchases in one sitting is the most plausible reading available. | Cosmetic only: which two rows share the instant. No FR changes. | Operator glances at the pair and swaps it if it reads oddly. |

## Corrections to the Intake Issue's Claims

Recorded rather than silently adopted, because the issue's framing is load-bearing for FR-4.

- The issue says the equal-timestamp tiebreak risks "becoming dead code". **That overstates it.** The
  tiebreak is independently covered at the store level by a test that builds equal-timestamp rows as a
  fixture, and that test remains valid whatever the seed does. What genuinely goes vacuous once every
  seeded row is distinct is the **acceptance-level** assertion that walks the first account's real
  seeded listing and checks the tiebreak — its condition would simply never be true. FR-4 exists to
  keep that acceptance-level assertion meaningful, not to rescue the rule from having no coverage at
  all.
- The issue reports twelve seeded rows in the first account plus "a thirteenth row from a live form
  post during manual inspection". The thirteenth row is an artifact of that inspection session, not of
  the seed; seeded state is twelve rows in the first account, nine in the second, none in the third.

## Open Questions

None blocking. The two questions a reviewer is most likely to raise are answered above rather than left
open: whether an ADR amendment is needed (A1 — no) and how far apart the times should be (A3 —
day-and-hour scale, retunable without touching any requirement).
