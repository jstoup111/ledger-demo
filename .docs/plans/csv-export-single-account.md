# Implementation Plan: Selected-account CSV download

**Date:** 2026-08-28
**Design:** `.docs/specs/2026-08-28-csv-export-single-account.md` (Approved)
**Stories:** `.docs/stories/csv-export-single-account.md` (accepted stories)
**Tier:** S (`.docs/complexity/csv-export-single-account.md`)
**Conflict check:** Skipped — Tier S

## Summary

Four short TDD tasks render transaction values as deterministic CSV, select that representation on
the existing transaction-listing handler, expose one selected-account page link, and style the link
as a projector-legible control. No route, schema, domain, dependency, command, or JavaScript change.

> **Amended 2026-09-02 by operator decision — `prd_audit` plan gaps S2.2, S2.4, S2.6, and S4.6:**
> the implementation remains unchanged, but the plan now contains eight tasks. Tasks 5–8 add the
> independent request-level, state-preservation, and static proof that the accepted stories required
> and the original four tasks did not own.

## Technical Approach

- Add a renderer in `internal/httpapi` that accepts an already-read `[]ledger.Transaction` and returns
  a complete CSV byte slice. It writes `id,amount_cents,description,created_at`, formats amounts with
  `strconv.FormatInt`, formats times with `CreatedAt.UTC().Format(time.RFC3339)`, and relies on
  `encoding/csv` for deterministic escaping.
- Extend the existing `GET /api/accounts/{id}/transactions` handler: complete the store read and
  existing error handling first, select CSV only when `format=csv`, then set
  `text/csv; charset=utf-8` and `Content-Disposition: attachment; filename="transactions.csv"`.
  Every other request remains on the current JSON branch.
- Add a download URL to page data only after a valid selected account has been resolved. Render it as
  a normal link labelled `Download CSV`; the missing-account branch returns before that value exists.
- Style that link using the existing static stylesheet. No JavaScript, form submission, new route,
  clock read, float conversion, or ledger write is involved.

> **Amended 2026-09-02 by operator decision — proof approach:** reuse the existing `routerTestStore`
> request harness and the existing Go AST inspection pattern in `internal/httpapi/router_test.go`.
> Compare CSV against the independently obtained JSON response, exercise two populated accounts,
> snapshot the complete fake-store state around a download, and inspect the two download-path
> function bodies for direct wall-clock, floating-point, or store-write calls. These are test-only
> additions; no production path or public behavior changes.

## Verify-Claims Ledger — plan — 2026-08-28

### Claims

- [verified] The current transaction handler completes the store read and handles expected and
  unexpected failures before serializing a success response.
- [verified] The current page handler resolves a valid selected account before populating page data;
  the missing-account branch renders and returns earlier.
- [verified] The transaction store already supplies stable newest-first order, and the existing JSON
  response already uses signed integer cents and UTC RFC 3339 recorded times.
- [verified] Query parameters do not create another `ServeMux` registration, so explicit
  representation selection preserves the tested five-endpoint count.

### Assumptions

- None. The static attachment name, explicit `format=csv` selection, ordinary-link control, and
  browser-only scope were approved in PR #24 and reconfirmed by the operator's format-refresh request.

### Verdict

CLEAR — no unconfirmed load-bearing assumptions remain.

## Verify-Claims Ledger — plan amendment — 2026-09-02

### Claims

- [verified] The 2026-09-01 `prd_audit` measured S2.2, S2.4, S2.6, and S4.6 behavior as currently
  correct but found no committed tests owning their named proof obligations.
- [verified] `internal/httpapi/router_test.go` already supplies a multi-account `routerTestStore`,
  request-level JSON assertions, and a Go AST inspection pattern, so the omitted proof can be added
  without a production-code change or new dependency.
- [verified] `ledger.Store` has one write operation, `Append`; full before/after snapshots of the fake
  store cover created, updated, and deleted account or transaction state, while AST inspection can
  reject a direct `Append`, `time.Now`, `float32`, or `float64` call on the download path.
- [verified] `web/web_test.go` already owns the no-new-dependency clause of S4.6.

### Assumptions

- None. After reviewing all four audit findings, the operator explicitly directed this recovery on
  2026-09-02. The story criteria, product behavior, and production design remain unchanged.

### Verdict

CLEAR — the amendment rests only on verified repository and audit evidence.

## Prerequisites

- None. No migration, dependency, seed change, or infrastructure setup is required.

## Tasks

### Task 1: Render transaction values as deterministic CSV

**Story:** Story 2 — document shape, values, escaping, and order; Story 3 — empty document shape;
Story 4 — determinism
**Type:** happy-path + negative-path

**Steps:**
1. Add table-driven tests for a renderer over `[]ledger.Transaction`: exact four-field header;
   positive and negative integer-cent strings; UTC RFC 3339 times; input order preserved; comma,
   quote, and line-break descriptions parsed back unchanged; empty input produces only the header;
   repeated renders are byte-identical.
2. Verify the scoped renderer tests fail because the renderer does not exist (RED).
3. Implement `internal/httpapi/csv.go` with `bytes.Buffer`, `encoding/csv`, `strconv.FormatInt`, and
   UTC RFC 3339 formatting. Return a copied complete byte slice and add no store, clock, float, or
   domain decision.
4. Verify the scoped renderer tests pass (GREEN).
5. Commit with subject `feat: render account transactions as CSV` and trailer `Task: 1`.

**Done when:**
- The renderer test parses every emitted record as exactly four fields and proves the exact header, signed integer cents, UTC recorded times, preserved input order, and special-character round trips.
- The empty-input case emits exactly the header row, and two renders of unchanged input are byte-identical.
- The production renderer contains no store access, wall-clock read, floating-point conversion, or third-party dependency.

**Files:** internal/httpapi/csv.go; internal/httpapi/csv_test.go

**Dependencies:** none

### Task 2: Serve CSV from the existing transaction-listing handler

**Story:** Story 2 — downloadable selected-account rows; Story 3 — empty and failure distinction;
Story 4 — ordinary JSON and route compatibility
**Type:** happy-path + negative-path

**Steps:**
1. Add request tests for explicit CSV selection: populated success has the exact CSV content type,
   static attachment name, and renderer body; empty success is header-only; missing account remains
   not found without CSV headers or body; unexpected store failure remains undisclosed and contains no
   partial CSV; absent and non-CSV format values retain the existing exact JSON response.
2. Verify the populated and empty CSV cases fail because the handler still returns JSON (RED).
3. Update `handleAccountTransactions` so the store read and existing error mapping finish first. On
   exact `format=csv`, render the full document, set download headers, and write it; otherwise execute
   the existing JSON serialization unchanged.
4. Verify the scoped handler suite passes (GREEN), including the existing five-route assertion and
   transaction-listing tests.
5. Commit with subject `feat: download transactions as CSV` and trailer `Task: 2`.

**Done when:**
- Populated and empty explicit CSV requests return status 200, the exact CSV and attachment headers, and respectively one row per transaction or only the header row.
- Missing-account and unexpected-store requests return their existing failure statuses without CSV headers, partial CSV, or disclosed internal errors.
- Requests without exact CSV selection retain the existing JSON status, content type, body, and ordering, while the router still declares exactly five endpoints.

**Files:** internal/httpapi/router.go; internal/httpapi/router_test.go

**Dependencies:** Task 1

### Task 3: Bind the download control to the valid selected account

**Story:** Story 1 — selected-account control; Story 2 — selected-account isolation; Story 3 — no
missing-account control
**Type:** happy-path + negative-path

**Steps:**
1. Add page-level tests proving populated and empty valid accounts each render exactly one
   `Download CSV` link for that account, switching accounts changes its target, and the missing-account
   page renders no download link.
2. Verify the scoped page tests fail because page data and the template have no download control (RED).
3. Add a selected-account download URL to `pageData` after valid account resolution and render it in
   `web/index.html.tmpl` as an ordinary link. Use the existing transaction-listing path with exact
   CSV selection; add no fallback link on missing-account pages.
4. Verify the scoped page tests pass (GREEN), including existing selection, empty-state, and invalid
   account assertions.
5. Commit with subject `feat: add selected-account CSV control` and trailer `Task: 3`.

**Done when:**
- Populated and empty valid-account pages each render exactly one `Download CSV` link whose target selects that account's CSV representation.
- Switching accounts changes the link target, and a missing-account page renders no download link.
- Existing page selection, transaction form, empty-state, and invalid-account behavior remain green.

**Files:** internal/httpapi/router.go; internal/httpapi/router_test.go; web/index.html.tmpl

**Dependencies:** Task 2

### Task 4: Style the download link as a projector control

**Story:** Story 1 — clearly labelled visible control
**Type:** happy-path

**Steps:**
1. Add a stylesheet test requiring the download link's active rule to use the existing page palette,
   readable spacing, and a visible inline-block button shape without animation, media queries, or an
   external asset.
2. Verify the focused stylesheet test fails because no download-control rule exists (RED).
3. Add a `download-control` class to the link and style it in `web/style.css` using the existing
   typography and colors. Preserve native link and keyboard behavior.
4. Verify the focused stylesheet and offline-asset tests pass (GREEN).
5. Commit with subject `style: emphasize the CSV download control` and trailer `Task: 4`.

**Done when:**
- The active stylesheet gives `.download-control` a visible inline-block control shape with readable padding, contrast, and spacing using only existing local assets and styles.
- The rendered control remains a native link labelled `Download CSV`, and existing offline/style constraints still pass.

**Files:** web/index.html.tmpl; web/style.css; web/web_test.go

**Dependencies:** Task 3

### Task 5: Prove CSV and JSON listing parity at the request boundary

**Story:** Story 2 S2.2 — CSV values and newest-first order agree with the ordinary programmatic
listing
**Type:** verification

**Steps:**
1. Add a request-level test with at least two transactions whose identifiers, signed cent amounts,
   descriptions, and recorded times are distinct and already ordered newest-first.
2. Issue the ordinary JSON request and the explicit CSV request for the same account. Decode the JSON
   response and parse the CSV response independently; do not call `renderTransactionsCSV` and do not
   construct the expected CSV rows from the store fixture.
3. Compare every CSV data row by index with the corresponding JSON element for identifier, decimal
   integer-cent string, description, RFC 3339 recorded time, and total row count.
4. Run the scoped request test and commit the test-only proof with subject
   `test: prove CSV and JSON listing parity` and trailer `Task: 5`.

**Done when:**
- A named request-level test parses independently requested JSON and CSV representations for the same
  two-transaction account and compares all four CSV fields element-for-element in response order.
- The test's expected side comes from the decoded JSON response, not the CSV renderer or store fixture,
  so changing either representation alone makes the comparison fail.
- The scoped test passes with no production-file change.

**Files:** internal/httpapi/router_test.go

**Verify-only:** yes

**Dependencies:** Task 4

### Task 6: Prove selected-account isolation across populated accounts

**Story:** Story 2 S2.4 — switching the selected account leaks no row from the previous account
**Type:** negative-path verification

**Steps:**
1. Add a request-level test whose fake store contains two populated accounts with distinct transaction
   identifiers, descriptions, amounts, and recorded times.
2. Download and parse the first account's CSV, then download and parse the second account's CSV.
3. Assert each parsed document equals only its selected account's exact rows and that the second
   document contains none of the first account's row values.
4. Run the scoped request test and commit the test-only proof with subject
   `test: prove CSV account isolation` and trailer `Task: 6`.

**Done when:**
- A named request-level test exercises two populated accounts in sequence and parses both documents.
- Each document's data rows exactly equal the selected account's fixture rows; the second document has
  the selected account's row count and no row equal to any first-account row.
- The scoped test passes with no production-file change.

**Files:** internal/httpapi/router_test.go

**Verify-only:** yes

**Dependencies:** Task 5

### Task 7: Prove a CSV download preserves complete ledger state

**Story:** Story 2 S2.6 and Story 4 S4.6 — downloading creates, updates, or deletes no ledger state
**Type:** negative-path verification

**Steps:**
1. Add a request-level test that deep-copies the fake store's complete accounts slice and per-account
   transaction map before an explicit CSV request.
2. Issue the download, require a successful parsed CSV response, then compare the store's complete
   accounts and transaction state with the snapshots using deep equality. A transaction-count-only
   assertion is insufficient because it cannot detect updates or balanced delete/create mutations.
3. Run the scoped request test and commit the test-only proof with subject
   `test: prove CSV downloads are read only` and trailer `Task: 7`.

**Done when:**
- A named request-level test snapshots all fake-store account and transaction values, performs a CSV
  download, and proves both complete structures are deeply equal afterward.
- The test would fail for an account or transaction create, update, or delete even when the total
  transaction count remained unchanged.
- The scoped test passes with no production-file change.

**Files:** internal/httpapi/router_test.go

**Verify-only:** yes

**Dependencies:** Task 6

### Task 8: Guard the download path against clock, float, and store-write calls

**Story:** Story 4 S4.6 — the download path reads no wall clock, performs no floating-point
conversion, adds no dependency, and mutates no ledger state
**Type:** negative-path verification

**Steps:**
1. Extend the existing Go AST test pattern to parse `internal/httpapi/router.go` and
   `internal/httpapi/csv.go`, locate `handleAccountTransactions` and `renderTransactionsCSV`, and
   fail if either function body directly calls `time.Now`, converts through `float32` or `float64`,
   or invokes the store write method `Append`.
2. Require both named function declarations to be found so a rename or extraction cannot make the
   guard silently pass over no code. Keep Task 7's full state snapshot as the behavioral mutation
   proof and the existing `web/web_test.go` dependency assertion as the dependency proof.
3. Run the scoped AST and request tests and commit the test-only proof with subject
   `test: guard CSV execution constraints` and trailer `Task: 8`.

**Done when:**
- A named static test finds both download-path functions and rejects direct `time.Now`, `float32`,
  `float64`, and `Append` calls in either body.
- Task 7's request-level state snapshot passes, and the existing dependency test still proves
  `modernc.org/sqlite` is the only direct non-stdlib dependency.
- The scoped tests pass with no production-file or dependency-manifest change.

**Files:** internal/httpapi/router_test.go

**Verify-only:** yes

**Dependencies:** Task 7

## Task Dependency Graph

```text
Task 1 (CSV renderer)
  -> Task 2 (existing handler representation)
       -> Task 3 (selected-account page link)
            -> Task 4 (projector control styling)
                 -> Task 5 (independent JSON/CSV parity proof)
                      -> Task 6 (two-account isolation proof)
                           -> Task 7 (complete state-preservation proof)
                                -> Task 8 (static execution-constraint guard)
```

## Integration Points

- After Task 2, the existing transaction-listing endpoint can return CSV while preserving JSON and
  failure behavior.
- After Task 3, the valid selected-account page reaches that representation without adding a route.
- After Task 4, the native link is visually legible for the live projector demo.
- After Task 5, the two programmatic representations are independently compared field-for-field and
  in response order.
- After Task 6, two populated selected accounts are proven isolated at the request boundary.
- After Task 7, a successful download is proven not to create, update, or delete ledger state.
- After Task 8, static and behavioral guards cover the remaining wall-clock, floating-point,
  dependency, and mutation constraints.

## Coverage Mapping

| Requirement / story behavior | Task(s) |
|---|---|
| FR-1 / Story 1 — selected-account control | 3, 4 |
| FR-2 / Story 2 — selected-account download and read-only behavior | 2, 3 |
| FR-3 / Story 2 — four fields and integer cents | 1, 2 |
| FR-4 / Story 2 — values and newest-first order agree | 1, 2 |
| FR-5 / Stories 1 and 3 — valid empty account | 1, 2, 3 |
| FR-6 / Stories 1 and 3 — missing account is not a CSV or fallback control | 2, 3 |
| FR-7 / Story 4 — deterministic bytes | 1, 2 |
| FR-8 / Story 4 — ordinary behavior and five endpoints unchanged | 2, 3, 4 |

> **Amended 2026-09-02 by operator decision — corrected proof ownership:** the original mapping
> identifies implementation ownership but omits the story's named independent proof. The active
> coverage additions are:

| Criterion / proof obligation | Owning task(s) |
|---|---|
| S2.2 / FR-4 — independently compare CSV and JSON values and order | 5 |
| S2.4 / FR-2 — prove isolation across two populated accounts | 6 |
| S2.6 / FR-2 — prove the complete ledger state is unchanged | 7 |
| S4.6 / FR-7, FR-8 — statically and behaviorally prove execution constraints | 7, 8 |

## Verification

- [x] Every happy and negative acceptance criterion maps to an owning task.
- [x] Each task owns scoped RED/GREEN work; no terminal catch-all validation task exists.
- [x] Every task has two or three falsifiable Done When checks.
- [x] Every Files declaration is repo-relative and limited to the task's evidence surface.
- [x] Dependencies are explicit, linear, and acyclic.

> **Amended 2026-09-02 by operator decision:** the 2026-09-01 `prd_audit` falsified the original
> claim that every criterion's named proof had an owning task. Tasks 5–8 now own S2.2, S2.4, S2.6,
> and S4.6 at the lowest sufficient request/static layers, with explicit dependencies and test-only
> file scope.
