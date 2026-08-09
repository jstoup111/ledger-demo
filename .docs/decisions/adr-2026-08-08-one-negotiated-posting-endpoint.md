# ADR: One content-negotiated endpoint serves both the browser form and JSON clients

**Date:** 2026-08-08
**Status:** Accepted
**Deciders:** james.stoup

## Context

The page carries **no JavaScript** — a hard rule in `.docs/decisions/styleguide.md`, motivated by
having no build step and nothing to debug on stage. A page with no JavaScript can only submit a
transaction as a real HTML form post.

The published HTTP surface (`internal/httpapi/router.go:17-19`, `.docs/architecture/sequences.md`)
declares one posting route, `POST /api/accounts/{id}/transactions`, returning `201` JSON. So the
browser form and a programmatic client are two callers with genuinely different needs: a browser
needs to end up back on a rendered page, a script needs a machine-readable body.

FR-9 in `.docs/specs/2026-08-08-base-ledger.md` constrains how those two callers may be served:
neither may bypass a validation rule the other enforces. FR-7 adds that reloading the result of a
successful form post must not record the transaction a second time.

## Options Considered

### Option A: One endpoint, response mode chosen by the request's content type
`application/x-www-form-urlencoded` → `303 See Other` back to the page;
`application/json` → `201` with the created transaction.
- **Pros:** The published surface stays at three API routes. FR-9 holds **structurally** — there is
  one handler and one validation path, so a bypass is impossible by construction rather than by
  discipline. The `303` gives POST-redirect-GET, which satisfies FR-7 for free.
- **Cons:** One handler carries a branch on content type and two response shapes — visible
  complexity in a file an audience reads on a projector. Rejections reach the page as a code in the
  redirect's query string rather than as a value passed directly to the template.

### Option B: Two endpoints, one per caller
Add `POST /accounts/{id}/transactions` for the browser, keeping the JSON route single-purpose.
- **Pros:** Each handler does exactly one thing and reads slightly more plainly in isolation.
- **Cons:** Four API routes instead of three, and the extra one appears in no committed document, so
  the published surface and the code diverge on day one. FR-9 becomes a **convention** — two entry
  points that must both call the same domain operation, with nothing preventing a future change to
  one from skipping a rule the other keeps.

### Option C: One endpoint that always answers with rendered HTML
The form posts and the handler renders the updated page directly with `200`.
- **Pros:** No redirect, no negotiation, the fewest moving parts of the three.
- **Cons:** Violates FR-7 — a reload re-submits the POST and records the transaction again, which is
  precisely the failure a presenter would trip over on stage. Also leaves programmatic clients
  parsing HTML, so FR-14's machine-readable rejection identifier has nowhere to live.

## Decision

**Option A.** `POST /api/accounts/{id}/transactions` inspects the request's content type and answers
accordingly:

| Request body | Success | Rejection |
|---|---|---|
| `application/x-www-form-urlencoded` | `303 See Other`, `Location: /?account={id}` | `303 See Other`, `Location: /?account={id}&error={code}` |
| `application/json` | `201`, the created transaction | `400` or `404`, `{"error":{"code","message"}}` |

Chosen primarily because it makes FR-9 structural. With one handler and one domain call, there is no
second path that could drift out of agreement with the first — which matters more here than handler
brevity, because the demo's whole point is that a live change to validation behaves identically
however it is exercised. Option B's cost is not the extra route but the standing invitation to
diverge. Option C is rejected outright on FR-7: duplicate-on-reload is a defect a presenter meets in
front of an audience.

The negotiation is on **content type of the request**, not an `Accept` header, because a browser form
post sends `application/x-www-form-urlencoded` unavoidably and sends an `Accept` header that also
lists HTML — content type is the signal that is actually reliable here.

## Consequences

### Positive
- The published surface in `internal/httpapi/router.go:17-19` remains accurate — three API routes,
  five routes total, unchanged by this feature.
- One validation path, so FR-9 cannot be violated by a later edit to a second handler.
- POST-redirect-GET satisfies FR-7 with no extra mechanism.

### Negative
- The posting handler contains a content-type branch and two response encoders. This is the single
  least simple part of the HTTP layer, and it is visible on a projector.
- A rejection's identity travels through a URL query parameter. The page therefore renders a message
  looked up **from a code supplied by the client**, which means an unrecognized code must degrade to
  a generic message rather than to a blank panel.
- The redirect target is constructed from the account id in the request path, so that id must be
  escaped when built into the `Location` header.

### Follow-up Actions
- [ ] Implement the single posting handler with a content-type branch in `internal/httpapi`.
- [ ] Map each domain sentinel to exactly one code from `.docs/decisions/api-response-contract.md`,
      once, at the HTTP boundary.
- [ ] Render an unrecognized `error` query value as a generic message, never as an empty panel.
- [ ] Add a test asserting the form-encoded path returns `303` with the expected `Location`, and a
      test asserting the JSON path returns `201`, for the same input.
