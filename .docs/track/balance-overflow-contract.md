# Track: balance-overflow-contract

Track: technical

The shipped API emits a seventh error code, `balance_overflow` (HTTP 400), that the Accepted
`.docs/decisions/api-response-contract.md` does not document. Reconciling an internal contract
document with already-shipped behavior changes no user-facing behavior: no route is added or removed,
no error code string or HTTP status changes, and nothing a presenter can see on the projector is
different afterwards. There are therefore no product requirements to enumerate, so `/prd` is skipped
and acceptance criteria live directly in the stories.

## Why not the documentation-only route

`/explore` can deliver a purely documentation-only request directly, without authoring an SDLC
artifact set. That route was considered and rejected for two independent reasons:

1. **It fails closed without a source issue.** The route is defined for "an unambiguous
   documentation-only request **with a source issue**", and explicitly fails closed when "no source
   issue exists" — it must not emit a delivery result or claim delivery. This idea arrived from the
   operator directly; there is no originating GitHub issue, and inventing one to unlock the route
   would be fabrication.
2. **The work is not purely documentation.** Nothing in the repository binds the contract document to
   the shipped sentinel-to-code mapping. That missing binding is precisely how a seventh code reached
   production undocumented. Closing it is a testable functional surface — a test — which belongs on
   the normal technical route.

## Ownership of the contract amendment

`.docs/decisions/api-response-contract.md` is an Accepted DECIDE artifact whose assertion ("six error
codes") this DECIDE pass falsified. Under the harness's DECIDE Artifact Amendment Ownership rule,
DECIDE amends such an artifact **in place on the spec branch before the first BUILD entry**, additively
(`> **Amended YYYY-MM-DD by …:**` beside the original, never rewriting or deleting it), and **BUILD
never receives that mutation as a task**. The amendment is therefore part of this spec commit, not a
plan task — see Story 1.

## Discovery — every claim verified directly

| Claim | Verdict | Evidence |
|---|---|---|
| `checkedAdd` guard returns `ErrBalanceOverflow` on an overflowing fold | **Verified** | `internal/ledger/balance.go:26-31` — guards both directions against `math.MaxInt64` and `math.MinInt64` |
| Mapped to `{"code":"balance_overflow"}` with HTTP 400 | **Verified** | `internal/httpapi/errors.go:38-39` |
| Contract documents only six codes, never mentions overflow | **Verified** | `.docs/decisions/api-response-contract.md` — six table rows; `grep -i overflow` over `.docs/` returns zero hits |
| Mapping has test coverage from commit `85df875` | **Verified** | `git show --stat 85df875`; `internal/httpapi/errors_test.go:51-54` (unit), `internal/httpapi/router_test.go:649` `TestRouterMapsBalanceOverflowAtBothPostingBoundaries` drives a deposit to exactly `math.MaxInt64` cents and asserts both the JSON envelope and the rendered page panel |
| Page renders the overflow message rather than the generic one | **Verified** | `internal/httpapi/router.go:164`; asserted at `internal/httpapi/router_test.go:699` |
| Baseline is green | **Verified** | `go test ./...` all packages `ok`; `gofmt -l .` empty; `go vet ./...` clean |

The guard and its tests are sound. The gap is the document.

## Scope boundary — what is deliberately left alone

- **`.docs/specs/2026-08-08-base-ledger.md` FR-12 "six cases" is not stale.** FR-12 enumerates six
  *input validation rules*. `balance_overflow` is not one of them — it is an arithmetic guard on the
  derived-balance fold, discovered during implementation. The PRD's count is correct as written.
- **`.docs/stories/base-ledger.md` and `.docs/plans/base-ledger.md`** likewise refer to the six
  validation rules, which remain six. They belong to another feature and are protected artifacts that
  a plan task must not name.
- **The overflow guard is not touched.** Folding signed `int64` cents genuinely can overflow; removing
  or weakening the guard would let a balance wrap silently.
