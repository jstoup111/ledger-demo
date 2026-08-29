# Complexity — csv-export-single-account

Tier: S

## Signals

| Signal | Current value | Reads as |
|---|---|---|
| Models / schema | No changes | S |
| Integrations / dependencies | No new integration or dependency | S |
| Auth / users / sessions | None | S |
| State machines | None; the feature is read-only | S |
| Stories | Four narrow stories | S |
| Blast radius | Existing HTTP and page presentation surfaces with focused tests | S |
| Routes | One existing route gains an explicit representation; zero routes added | S |
| New interfaces / seams | None | S |

## Rationale

The feature renders values already returned by the store, selects that representation on an existing
read handler, and adds one selected-account page control. It adds no domain decision, persistence,
schema, dependency, authentication, state machine, or route registration. Small tier therefore skips
architecture-diagram, architecture-review, conflict-check, and coherence-check.

## Stem

`csv-export-single-account` matches the plan, track, intake, and stories filenames used by daemon
discovery.

