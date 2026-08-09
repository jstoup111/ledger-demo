# Complexity — acceptance-harness-hygiene

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 — no domain type is added or changed | S |
| Integrations | 0 — no new dependency; `os/exec`, `os/signal`, `net`, `time` are all stdlib | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None | S |
| Story count | 3 — one per defect | S |
| Blast radius | **1 file** — `test/acceptance/harness_test.go`. No production package, no schema, no route, no template, no seed data | S |
| Validation surface | 0 new rules — no user input is involved anywhere | S |
| Public contract change | None — `startServer`, `seedDB`, `newApp`, and `newAppAt` keep their signatures, so all 9 existing call sites compile untouched | S |

## Rationale

Every signal reads Small. This is a single-file repair of the test suite's own scaffolding: no
production code is touched, no dependency is added, no HTTP route or domain behavior changes, and the
existing helper signatures are preserved so the 13 call sites across
`test/acceptance/ledger_acceptance_test.go` need no edits.

The work is genuinely small but not trivially safe — it touches process lifetime and signal
handling, where a mistake makes the suite flaky rather than broken, and flakiness in the harness is
what this change exists to remove. That risk is answered by the stories' acceptance criteria
(determinism under `-count=2`, suite wall-clock ceiling, no orphan after an interrupt) rather than by
raising the tier: the concern is *care within one file*, not architectural breadth, and Medium's
extra artifacts (a conflict matrix over 3 non-overlapping stories, C4 diagrams for a change that
alters no component) would produce no signal.

Tier chosen by the operator.

## What Small skips

Per the harness's Small tier, these are deliberately **not** produced:

- `/prd` — skipped by the **technical** track (`.docs/track/acceptance-harness-hygiene.md`); no
  product requirements exist. Acceptance criteria live in the stories.
- `/architecture-diagram` — the change alters no component, container, or relationship, so the
  existing `.docs/architecture/` diagrams remain accurate as-is.
- `/architecture-review` — no architectural decision is at stake.
- `/conflict-check` — three stories, each owning a disjoint concern in one file, with no shared
  state, no resource contention, and no ordering relationship between them.
- `/coherence-check` — Medium and Large only.

**One ADR is nevertheless included and Accepted:**
`.docs/decisions/adr-2026-08-09-bounded-yield-in-test-readiness-probes.md`. It is not an
architecture-review product — it exists because the fix appears to contradict a *locked project
convention* (CLAUDE.md convention 6, "no `time.Sleep`"), and an unrecorded reinterpretation of a
locked convention is exactly what gets re-litigated later. It is `Status: Accepted`, so the land gate
is satisfied.

## Stem

`acceptance-harness-hygiene` — matches `.docs/plans/acceptance-harness-hygiene.md` and
`.docs/stories/acceptance-harness-hygiene.md` so the daemon resolves the tier at build time.
