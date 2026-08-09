# Complexity — amount-column-and-autofocus

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 — no domain type changes; `Account` and `Transaction` untouched | S |
| Integrations | 0 — no new dependency; `go.mod` unchanged | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None | S |
| Story count | 2 | S |
| Blast radius | 2 production files (`web/index.html.tmpl`, `web/style.css`) plus test updates | S |
| Validation surface | 0 new rules; no new sentinel errors | S |
| Persistence | None — no schema change, no migration, no seed change | S |
| Routes | Unchanged; the five existing routes are neither added to nor removed | S |

## Rationale

Every signal reads Small. This is a presentation-layer change confined to the embedded template and
stylesheet: one column gains right alignment, one input gains the plain HTML `autofocus` attribute.
No Go logic, no domain type, no store method, no route, and no schema is touched. There is no new
dependency and no new validation rule. Two stories cover it.

The one non-trivial element is not complexity but *coupling to existing tests*: an established
assertion in `internal/httpapi/router_test.go` pins the transaction row as a whole-string literal,
so a markup change there is a deliberate, named test update rather than an incidental edit. That is
a planning detail, not a tier signal.

Tier chosen by the operator (pre-granted decision, operator unavailable).

## What Small requires — and what it skips

Skipped for Small, per the DECIDE phase table:

- `/architecture-diagram` — no component, container, or data-flow change to redraw.
- `/architecture-review` — no design trade-off with architectural blast radius; no new ADR is
  authored, so the DRAFT-ADR land gate has nothing to catch.
- `/conflict-check` — two stories touching disjoint markup regions (table cell vs. form input).
- `/coherence-check` — S tier is exempt from the committed traceability mapping.

Run: `/explore` → complexity → `/prd` (product track) → `/stories` → `/plan`.

## Stem

`amount-column-and-autofocus` — matches `.docs/plans/amount-column-and-autofocus.md` so the daemon
resolves this tier at build time.
