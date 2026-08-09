# Coherence Mapping — base-ledger

**Date:** 2026-08-08
**Tier:** M — required for M and L
**Track:** product
**Plan stem:** base-ledger

Maps [2026-08-08-base-ledger.md](../specs/2026-08-08-base-ledger.md) (FR-1 … FR-16) →
[base-ledger.md](../stories/base-ledger.md) (6 stories) →
[base-ledger.md](../plans/base-ledger.md) (32 tasks).

Row classes present: `fr`, `story`, `task`. The `outcome` class is omitted because this spec is
chat-origin — `conduct-ts engineer claim` returned empty, and there is no intake marker or staged
outcomes file — so no intake outcome bullets exist. Omission is correct, not a gap.

Every `covered` verdict was confirmed by reading the counterpart artifact file.

| Row class | Cited id(s) | Counterpart id(s) | Verdict | Notes |
|---|---|---|---|---|
| fr | fr-1 | story-1, task-6, task-26 | covered | Account list and account selection asserted in task-26. |
| fr | fr-2 | story-1, task-10, task-26 | covered | task-10 derives balance as a fold; task-26 asserts it renders prominently. |
| fr | fr-3 | story-2, task-8, task-26 | covered | task-8 asserts the exact newest-first id sequence under one clock instant. |
| fr | fr-4 | story-1, task-9, task-26 | covered | Page-level zero balance and empty-state message added to task-26 by this check; previously no task asserted it. |
| fr | fr-5 | story-3, task-26 | covered | task-26 asserts the post form and its action. |
| fr | fr-6 | story-3, task-19, task-24 | covered | task-19 is the dollars-to-cents table including signed values. |
| fr | fr-7 | story-3, task-24 | covered | Reload-does-not-double assertion added to task-24 by this check; the 303 alone did not assert it. |
| fr | fr-8 | story-4, task-24 | covered | 201 with the created transaction. |
| fr | fr-9 | story-4, task-24 | covered | The both-content-types equivalence table is the executable form of this FR. |
| fr | fr-10 | story-1, task-21 | covered | Accounts with integer balances. |
| fr | fr-11 | story-2, task-22 | covered | task-22 asserts JSON order equals page order. |
| fr | fr-12 | story-5, task-11, task-12, task-13, task-14, task-15, task-18, task-19 | covered | Six enumerated cases each individually covered: a to task-14, b to task-11, c to task-12, d to task-13, e to task-19, f to task-15; task-18 is the rule-semantics table over all six. |
| fr | fr-13 | story-3, task-27 | covered | Error panel rendered above the form. |
| fr | fr-14 | story-4, task-20 | covered | Sentinel-to-code mapping plus the error body shape. |
| fr | fr-15 | story-6, task-29, task-31 | covered | task-29 asserts the seeded shape; task-31 asserts two resets are byte-identical. |
| fr | fr-16 | story-6, task-30 | covered | Port honored; unknown subcommand exits non-zero. |
| story | story-1 | task-2, task-6, task-10, task-21, task-26 | covered | Five tasks cite Story 1 on their Story line. |
| story | story-2 | task-8, task-9, task-16, task-22, task-23, task-26 | covered | task-16 cites Story 2 for the global id sequence. |
| story | story-3 | task-19, task-24, task-26, task-27, task-28 | covered | |
| story | story-4 | task-20, task-24, task-25 | covered | |
| story | story-5 | task-3, task-11, task-12, task-13, task-14, task-15, task-17, task-18, task-19 | covered | task-18 is the single owner of rule semantics. |
| story | story-6 | task-1, task-5, task-7, task-29, task-30, task-31, task-32 | covered | task-32 added by this check to cover the offline-rendering criterion, which had no task. |
| task | task-1 | story-6 | covered | Infrastructure: injectable clock, forced by NFR-3 and fr-15's byte-identical resets. |
| task | task-2 | story-1 | covered | Infrastructure: the domain types fr-2 and fr-3 operate on. |
| task | task-3 | story-5 | covered | Infrastructure: forced by fr-12's six independently identifiable rejections and fr-14. |
| task | task-4 | story-1 | covered | Infrastructure: reads forced by fr-2, fr-3, fr-10. The interface's placement in the domain is ADR-mandated rather than FR-mandated; stated plainly rather than overclaimed. |
| task | task-5 | story-1 | covered | Infrastructure: persistence forced by fr-15 and fr-2. Also asserts the absence of any uniqueness constraint beyond the primary keys. |
| task | task-6 | story-1 | covered | |
| task | task-7 | story-2 | covered | Infrastructure: append forced by fr-5 and fr-8; the global count specifically by condition C1. |
| task | task-8 | story-2 | covered | |
| task | task-9 | story-2 | covered | Negative path: empty list versus null, unknown account not-found. |
| task | task-10 | story-1 | covered | |
| task | task-11 | story-5 | covered | fr-12b |
| task | task-12 | story-5 | covered | fr-12c |
| task | task-13 | story-5 | covered | fr-12d, boundary asserted from both sides. |
| task | task-14 | story-5 | covered | fr-12a |
| task | task-15 | story-5 | covered | fr-12f, zero-balance boundary asserted from both sides. |
| task | task-16 | story-2 | covered | Infrastructure: id assignment forced by fr-3's total order and NFR-3's no-randomness rule. |
| task | task-17 | story-5 | covered | |
| task | task-18 | story-5 | covered | Owns rule semantics; introduces no production surface. |
| task | task-19 | story-3 | covered | fr-6 and fr-12e |
| task | task-20 | story-4 | covered | fr-14 |
| task | task-21 | story-1 | covered | fr-10 |
| task | task-22 | story-2 | covered | fr-11 |
| task | task-23 | story-2 | covered | Negative path: method and unknown-path edges, and append-only immutability. |
| task | task-24 | story-3 | covered | fr-7, fr-8, fr-9 |
| task | task-25 | story-4 | covered | Negative path: malformed JSON and a numeric amount rejected, keeping floats off the money path. |
| task | task-26 | story-1 | covered | fr-1, fr-2, fr-4, fr-5 |
| task | task-27 | story-3 | covered | fr-13, condition C2, resolution F2. |
| task | task-28 | story-3 | covered | NFR-1 stylesheet rules. |
| task | task-29 | story-6 | covered | fr-15 |
| task | task-30 | story-6 | covered | fr-16 |
| task | task-31 | story-6 | covered | fr-15, resolution F3. |
| task | task-32 | story-6 | covered | NFR-2. |

## Architecture-review conditions

All three conditions from the approved review trace to tasks:

- **C1** — ids numbered globally across the table, not per account: task-7 counts the whole table with
  no account filter, task-16 asserts `txn-0006` from five rows spread across two accounts, task-29
  asserts the seeded sequence does not restart per account.
- **C2** — an unrecognized error value renders a generic message, never an empty panel: task-27.
- **C3** — all ids share one width: task-8 (ordering under equal timestamps), task-16 and task-29
  (four-digit id assertion).

## Conflict-check resolutions

All three are honored in the plan, not merely recorded in the stories:

- **F1** — test ownership. Confirmed by reading all 32 task bodies: tasks 11 through 15 each
  implement one rule, task-18 is the single rule-semantics table, task-24 asserts equivalence only
  and explicitly consumes task-18's codes rather than restating them, task-27 uses one representative
  rejection. No rule is asserted three times.
- **F2** — the unknown-account page renders the account list and message only: task-27 asserts no
  balance element, no transaction list, and no form.
- **F3** — file-backed tests use a per-test temporary directory, never the default database path:
  task-31.

## Non-functional requirements

- **NFR-1** projector legibility: task-26 and task-28.
- **NFR-2** fully offline: task-32.
- **NFR-3** determinism: task-1, task-8, task-29, task-31, plus the Batch F gate asserting a single
  `time.Now()` call site.
- **NFR-4** suite budget, repeat-run stability, test ratio, negative case per rule: Batch F gate plus
  tasks 11 through 15 and task-18.
- **NFR-5** lint clean: Batch F gate.
- **NFR-6** code legibility on a projector: no task, by nature — a review property rather than a
  testable assertion. Recorded so its absence is deliberate; the simplify and code-review gates judge
  it.

## Non-goals drift scan

No task or story criterion introduces anything from the non-goals list. Two findings stated
positively: task-5 asserts the **absence** of any uniqueness constraint beyond the primary keys by
querying the schema catalogue, which is the correct shape when an absence is itself a requirement;
and task-25 rejects a numeric amount, keeping floating point off the money path rather than admitting
a rounding rule.

The unlocked read-then-write balance check correctly has no task. It is recorded under Known
Characteristics in the PRD and its absence from the plan is intended.

## Result

54 rows — 16 `fr`, 6 `story`, 32 `task` — all `covered`. Zero `gap` rows.

This verdict is clean only because three holes found during the check were closed in the plan rather
than recorded as gaps:

1. **fr-4** — story-1 required a page-level zero balance and empty-state message for an account with
   no transactions; no task asserted it. Added to task-26.
2. **fr-7** — the plan asserted the redirect but not the property fr-7 states, that reloading does not
   record a second transaction. Added to task-24.
3. **NFR-2** — fully offline had no covering task or gate anywhere. Added as task-32.

Left unfixed, rows fr-4, fr-7, and story-6 would each have been `gap`.
