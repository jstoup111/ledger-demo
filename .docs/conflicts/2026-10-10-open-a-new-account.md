# Conflict Check: open-a-new-account

**Date:** 2026-10-10
**Stories checked:** `.docs/stories/open-a-new-account.md` (Stories 1–7) against every file in
`.docs/stories/` (acceptance-harness-hygiene, account-totals-row, account-transaction-count,
amount-column-and-autofocus, average-transaction-amount, balance-overflow-contract, base-ledger,
csv-export-single-account, rejection-message-detail, seed-recorded-times) and the active specs.
**Result:** PASS — 2 blocking conflicts found and resolved in DECIDE; 1 degrading conflict accepted.

## ADR corpus

- **Examined and compared:** `adr-2026-10-10-sequential-account-ids-ordered-by-sequence`,
  `adr-2026-08-08-one-negotiated-posting-endpoint`, `adr-2026-08-09-rejection-message-composition`,
  `adr-2026-08-08-sentinel-errors-for-domain-failures`, `adr-2026-08-08-store-interface-in-domain-package`,
  `adr-2026-08-08-deterministic-transaction-ids-and-ordering`, `adr-2026-08-08-money-as-int64-cents`,
  `api-response-contract`.
- **Narrowed out (no overlapping subject):** `adr-2026-08-08-injected-clock` (opening reads no time),
  `adr-2026-08-09-bounded-yield-in-test-readiness-probes` (test harness only), `styleguide` (no new
  visual rule; the form reuses existing styles), `architecture-review-2026-08-08-base-ledger`
  (review record, decisions captured in the ADRs above). None is superseded.

No ADR-versus-story conflict: every story is consistent with the ADR decisions it cites.

## Conflict 1: `POST /api/accounts` is asserted to be `405`

**Stories involved:** base-ledger Story 1 vs open-a-new-account Story 4
**Files:** `.docs/stories/base-ledger.md` vs `.docs/stories/open-a-new-account.md`
**Type:** contradiction
**Severity:** blocking

**Description:** base-ledger Story 1 asserted "Given `POST /api/accounts`, when the request is made,
then the response is `405` with an empty body." Story 4 makes that same request open an account with
`201`. Both cannot hold. The assertion is also executable today: the acceptance subtest
"POST /api/accounts is not allowed" in `test/acceptance/ledger_acceptance_test.go` and
`TestNewRouterDeclaresExactlyFiveRoutes` in `internal/httpapi/router_test.go`.

**Resolution Options:**
1. Replace the base-ledger assertion in place with `PUT`/`DELETE` on `/api/accounts` → `405`, and
   let open-a-new-account own `POST`; the plan updates both tests in the task that adds the route.
2. Keep `POST /api/accounts` as `405` and open accounts at a different path — diverges from the
   collection-POST shape every other route follows and from the PRD's single obvious endpoint.

**Resolution applied:** Option 1. base-ledger Story 1 now reads "Given `PUT` or `DELETE` on
`/api/accounts` … `405`", pointing at Story 4 for `POST`. Story 4 carries the `PUT` → `405` negative
path. Root: story phrasing — the original assertion protected "no unplanned route", which this spec
plans. That story file's five pre-existing amendment blocks were folded into its current text in the
same pass, as the stories rule requires.

## Conflict 2: an extra form on form-less pages

**Stories involved:** base-ledger Story 1 (unknown-account page renders the list and not-found message
only) and amount-column-and-autofocus Story 3 (form-less states contain no `<form`) vs the
first draft of open-a-new-account Story 6 / PRD FR-7 ("the form appears wherever the page renders,
including zero accounts")
**Files:** `.docs/stories/base-ledger.md`, `.docs/stories/amount-column-and-autofocus.md` vs
`.docs/stories/open-a-new-account.md`
**Type:** contradiction (plus an overlap: existing tests locate the posting form as the page's first
`<form`)
**Severity:** blocking

**Description:** Rendering an open-account form on the unknown-account and zero-account pages
contradicts their "no form" assertions, which existing router and acceptance tests enforce. Rendering
it before the posting form would also move the page's first `<form`, which several existing tests use
to locate the posting form and its error panel.

**Resolution Options:**
1. Render the open-account form only on the selected-account page, after the transactions section;
   leave the unknown- and zero-account pages unchanged.
2. Render it everywhere and rewrite the existing "no form" assertions and tests.

**Resolution applied:** Option 1. PRD FR-7, ADR D4, and Story 6 now place the form on the
selected-account page only, after the transactions; Story 6 adds a negative path asserting no `<form`
on the unknown- and zero-account pages, and that the name input carries no `autofocus` (so the
amount-column-and-autofocus "exactly one autofocus" rule still holds if that spec is ever built).
Root: the PRD requirement, corrected in DECIDE; the zero-account case is unreachable from a reset
ledger, and Story 1 still proves opening the first account (`acct-1`) in the domain.

## Accepted degrading conflict: "exactly five endpoints" NFRs

**Specs involved:** account-totals-row, account-transaction-count, average-transaction-amount,
csv-export-single-account, rejection-message-detail, seed-recorded-times (NFR "the public HTTP
surface remains exactly five endpoints"); balance-overflow-contract Story Done When ("Exactly five
HTTP routes remain registered").
**Type:** overlap
**Severity:** degrading

**Description:** Each of those statements is scoped to its own feature's diff ("this change adds no
route") and none appears in a story acceptance criterion except the shipped balance-overflow Done When,
which was true at its ship. After this feature the surface is six routes; the route-count test moves to
six in the task that adds the route. In-flight feature branches that edit `internal/httpapi/router.go`
or the page template (open PRs #72, #79, #84) will textually conflict with this one at rebase; their
own behavior does not contradict any story here.

**Accepted:** yes (operator pre-authorized DECIDE outputs). No artifact change.

## Pairs examined with no conflict (both directions)

- Story 7 (sequence order) vs base-ledger Story 1 ("ordered by `id` ascending", "first account by id
  ascending"): for the seeded `acct-1`..`acct-3` text and sequence order coincide, and Story 7's
  negative path pins that the seeded order is unchanged. Satisfying either keeps the other.
- Stories 5/6 vs rejection-message-detail Stories 1–2: the new codes add rows; no existing code,
  status, or sentence changes, and the existing detail-validation rule is reused unchanged.
- Story 4 vs csv-export-single-account: different paths; CSV is per-account read-only.
- Stories 1–3 vs base-ledger Story 6 (seed identity): seed data is not changed; opened accounts exist
  only until the next reset.
- No oscillating pair: no two stories gate the same behavior in opposite directions.
