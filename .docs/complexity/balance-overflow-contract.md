# Complexity — balance-overflow-contract

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 — no type added or changed | S |
| Integrations | 0 — no dependency added; stdlib only | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None | S |
| Story count | 2 | S |
| Blast radius | One Accepted decision document (amended at DECIDE) plus one test file | S |
| Runtime behavior changed | None — no route, code string, HTTP status, or message changes | S |
| New validation surface | None | S |

## Rationale

Every signal reads Small. Nothing that runs on the projector behaves differently afterwards: the
seven error codes, five routes, and every HTTP status are exactly as shipped. The change is one
additive amendment to an accepted contract document, plus one test that binds that document to the
mapping it describes so the two cannot drift apart again.

Tier confirmed by the operator.

## What Small skips

- `/architecture-diagram` — no component, container, or data-flow change; `.docs/architecture/`
  already describes the shipped shape.
- `/architecture-review` — no design decision is being made. The decision (guard the fold, map the
  sentinel) was already made and shipped in commit `85df875`; this work records it. No ADR is
  authored, so no ADR can be DRAFT at the land gate.
- `/conflict-check` — two stories, no shared mutable state, no resource contention.
- `/coherence-check` — S tier only; traceability is carried inline by the stories and plan.

## Stem

`balance-overflow-contract` — matches `.docs/plans/balance-overflow-contract.md` so the daemon
resolves the tier at build time.
