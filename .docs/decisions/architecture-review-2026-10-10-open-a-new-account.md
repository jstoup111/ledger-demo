# Architecture Review: Open a new account

**Date:** 2026-10-10
**Mode:** Lightweight (Medium tier — Feasibility and Alignment only; complexity assessed at
`.docs/complexity/open-a-new-account.md`, domain integrity handled per-cycle by the TDD domain
reviewer)
**Reviewed:** `.docs/specs/2026-10-10-open-a-new-account.md` (Approved, FR-1 – FR-9). Stories and
plan do not exist yet — this review runs before `/stories`.
**Verdict:** APPROVED

## Feasibility

| Check | Assessment |
|---|---|
| **Stack compatibility** | Clear. Stdlib `net/http`, `html/template`, `unicode/utf8`, and the existing SQLite store. No new dependency; `go.mod` unchanged. |
| **Prerequisites** | None external. The SQLite store already has `InsertAccount(ledger.Account) error` (verified, `internal/store/sqlite.go`), used today by seeding; the domain `Store` interface only needs to declare it. |
| **Integration surface** | Four internal packages (`ledger`, `store`, `httpapi`, `web`), one binary, zero external systems. |
| **Data implications** | None. The `accounts` table already holds `id` and `name`; no column, constraint, or migration. `make reset` still restores pristine state. |
| **Performance risk** | Opening reads the account list once to count it. Demo scale is single-digit accounts. |
| **Worktree isolation** | Unchanged. No new port, file, or service. |

No infeasibility found.

## Alignment

**Domain boundaries** — Respected. Name validation and id assignment live in `internal/ledger`
(`adr-2026-08-08-store-interface-in-domain-package.md`: the consumer declares the interface, so the
interface grows by one method the domain needs). The HTTP layer parses the request and maps
sentinels; it decides nothing about names or ids.

**Pattern consistency** — Every mechanism reuses an APPROVED decision: sentinels
(`adr-2026-08-08-sentinel-errors-for-domain-failures.md`), request negotiation and
redirect-after-post (`adr-2026-08-08-one-negotiated-posting-endpoint.md`), message composition and
validated `detail` (`adr-2026-08-09-rejection-message-composition.md`), integer cents
(`adr-2026-08-08-money-as-int64-cents.md` — the new account's balance is `0`, derived). The one
uncovered structural decision — account identity and the listing order it forces — is recorded in
`adr-2026-10-10-sequential-account-ids-ordered-by-sequence.md`.

Focused local pattern basis — **the posting route is the precedent for the opening route.** Role: a
negotiated POST serving both the page form and a JSON client. Traits to preserve: negotiation on
request `Content-Type` only; `303` to the page on form success; `201` with the resource on JSON
success; coded error envelope on JSON rejection; `error`/`detail` redirect on form rejection;
messages composed once by `messageFor` for both branches. Allowed variation: the redirect for a
rejection carries no `account` parameter (there is no account yet). Rediscovery hints:
`internal/httpapi/router.go` `handlePostTransaction`, `postRequest`, `writePostError`,
`postRedirectDetail`; `internal/httpapi/message.go` `messageFor`, `characterCountCarriedValue`.

**State management** — No state. An account is an id and a name; there is no open/closed flag, and
none is added (closing is out of scope).

**Diagram accuracy** — `.docs/architecture/sequences.md` gains flow 3 and
`.docs/architecture/components.md` a note; no relationship changes. `render-diagrams --check` passes.
ERD unchanged.

**Security boundaries** — Auth is a non-goal. New untrusted inputs: the `name` field (validated in
the domain, escaped on output by `html/template`) and the existing `error`/`detail` query parameters,
which gain two codes; `detail` for `account_name_too_long` is validated on read exactly like the
existing character-count detail, so an injected value falls back to the plain sentence.

**Production DI defaults** — Unchanged; file-backed SQLite in production.

**Contract change** — `.docs/decisions/api-response-contract.md` is amended (additive) for the new
route and the two codes. Existing codes, statuses, and shapes are untouched.

## Wiring Surface

| New surface | Wired into |
|---|---|
| `ledger.Store.InsertAccount` (interface method) | Already implemented by `store.SQLite`; called by `ledger.OpenAccount`. |
| `ledger.OpenAccount` | Called by the new `POST /api/accounts` handler registered in `httpapi.NewRouter`. Delivered in stack child 1; its production caller arrives in child 2. |
| `ledger.ErrAccountNameEmpty`, `ledger.ErrAccountNameTooLong` | Returned by `OpenAccount`; mapped in `httpapi.codeFor`. |
| Account ordering helper (domain) | Called by `handleAccounts` and `handlePage` in `internal/httpapi/router.go`, replacing their string comparisons. |
| `POST /api/accounts` | Registered in `httpapi.NewRouter`; posted to by the page's open-account form and by JSON clients. |
| Open-account form | Rendered by `web/index.html.tmpl` on the selected-account page only, after the transactions section (ADR D4). |

Overlap scan (`ai-conductor overlap-scan` over these paths): no unmerged overlapping work reported
(advisory). Open feature PRs that also edit `internal/httpapi/router.go` and the page template are
assessed in `/conflict-check`.

## Risks

| Risk | Type | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| A later account listing forgets the ordering helper | Technical | Low | Low | One helper in the domain; both current listings covered by tests with ten or more accounts |
| Concurrent opens compute the same id | Data | Very low | Low | Primary key rejects the duplicate (500); documented in the ADR as a known property |
| Rejection message shown above the wrong form | Technical | Medium | Medium | ADR D4 fixes placement by code family; covered end to end |

## ADRs Created

- `adr-2026-10-10-sequential-account-ids-ordered-by-sequence.md` — APPROVED (operator
  pre-authorized DECIDE outputs for this spec).
