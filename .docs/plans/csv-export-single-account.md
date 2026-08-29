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

## Task Dependency Graph

```text
Task 1 (CSV renderer)
  -> Task 2 (existing handler representation)
       -> Task 3 (selected-account page link)
            -> Task 4 (projector control styling)
```

## Integration Points

- After Task 2, the existing transaction-listing endpoint can return CSV while preserving JSON and
  failure behavior.
- After Task 3, the valid selected-account page reaches that representation without adding a route.
- After Task 4, the native link is visually legible for the live projector demo.

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

## Verification

- [x] Every happy and negative acceptance criterion maps to an owning task.
- [x] Each task owns scoped RED/GREEN work; no terminal catch-all validation task exists.
- [x] Every task has two or three falsifiable Done When checks.
- [x] Every Files declaration is repo-relative and limited to the task's evidence surface.
- [x] Dependencies are explicit, linear, and acyclic.
