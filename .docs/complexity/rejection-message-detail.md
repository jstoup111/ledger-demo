# Complexity — rejection-message-detail

Tier: S

## Signals

| Signal | Value | Reads as |
|---|---|---|
| Models | 0 — no new type persisted, no schema change, no migration | S |
| Integrations | 0 — no new dependency; `modernc.org/sqlite` stays the only third-party module | S |
| Auth / users / sessions | None (explicit non-goal) | S |
| State machines | None — a rejection is one stateless response | S |
| Story count | 2 | S |
| Blast radius | 2 files of production code (`internal/httpapi/errors.go`, `internal/httpapi/router.go`), plus their tests and the acceptance specs. No package boundary moves, no route added or removed, no template structure change | S |
| Published contract touched | 1 — `.docs/decisions/api-response-contract.md` gains a documented query parameter on the rejection redirect. The seven `code` values and the error body shape are unchanged | S, but it is why an ADR is recorded |
| Untrusted-input surface | 1 new client-supplied value rendered on the page, joining the 2 already carried (`account`, `error`) under the same escaping rule and an existing precedent test | S |

## Rationale

Every signal reads Small. This is a change to text composition inside one existing package: the
sentinel→code mapping already happens exactly once at the HTTP boundary, so the seam this feature
needs is already there and does not have to be created or moved. There is no new persistence, no new
route (the surface stays at exactly five), no JavaScript, no new colour, and no new dependency.

The two things that make it more than a copy edit are both narrow and both fully specified before
BUILD: the offending value must survive a `303` redirect to reach the page, and it is therefore
client-supplied and must be validated and escaped. Neither expands the blast radius past two
production files.

**Tier: S recorded on the operator's instruction**, consistent with the signals above. The Small tier
skips `/architecture-diagram`, `/architecture-review`, `/conflict-check`, and `/coherence-check`.

## What Small skips, and why each is safe to skip here

- **`/architecture-diagram`** — no component, container, or data-flow shape changes. The diagrams in
  `.docs/architecture/` describe packages and routes; this feature adds neither.
- **`/architecture-review`** — no new component, boundary, or dependency to weigh. One decision *is*
  load-bearing (how the offending value reaches the page without a second render path), so it is
  recorded as a standalone Accepted ADR,
  `.docs/decisions/adr-2026-08-09-rejection-message-composition.md`, rather than left implicit. An ADR
  at Small tier is a record of a decision, not a reinstatement of the skipped review step.
- **`/conflict-check`** — two stories that touch the same two files in the same direction, with test
  ownership assigned explicitly in the plan's Technical Approach. There is no second writer to
  contend with.
- **`/coherence-check`** — the plan carries a Coverage Mapping table covering all FRs; a separate
  committed traceability artifact is a Medium/Large requirement.

## Stem

`rejection-message-detail` — matches `.docs/plans/rejection-message-detail.md` so the daemon resolves
the tier at build time. `.docs/specs/2026-08-09-rejection-message-detail.md` carries the date prefix
that `/prd` uses, following the same split as `base-ledger`.
