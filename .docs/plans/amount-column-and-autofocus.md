# Implementation Plan: Amount Column Alignment and Amount-Field Focus

**Date:** 2026-08-09
**Design:** `.docs/specs/2026-08-09-amount-column-and-autofocus.md` (Approved)
**Stories:** `.docs/stories/amount-column-and-autofocus.md` (accepted stories)
**Tier:** S (`.docs/complexity/amount-column-and-autofocus.md`)
**Conflict check:** Skipped — Tier S (see the complexity record)

## Summary

Eight tasks. Right-align the transaction table's amount column by marking its heading and cells with
an `amount` class and adding one class-scoped stylesheet rule, and place the caret in the amount input
on load with the plain HTML `autofocus` attribute. No Go logic, route, schema, or seed data changes.

## Technical Approach

**Two production files change:** `web/index.html.tmpl` and `web/style.css`. Both are embedded into
the binary through the existing `web.FS` (`//go:embed index.html.tmpl style.css`), so no wiring,
loader, or handler change is needed — the template is already executed by `handlePage` and the
stylesheet is already served by the existing `GET /style.css` route. The five routes, the `pageData`
view model in `internal/httpapi/router.go`, and every Go function are untouched.

**Resolution of PRD Open Question 1 — how the amount column is targeted.** Two candidates were
weighed:

- *Positional:* `th:nth-child(2), td:nth-child(2) { text-align: right }`. CSS-only, touches no markup
  and breaks no existing test.
- *Semantic:* `class="amount"` on the amount `<th>` and `<td>`, plus `.amount { text-align: right }`.

**Chosen: semantic class.** Three reasons, in order of weight:

1. **The table is the live-demo target.** This repo exists so a feature can be added to it on stage.
   A change that adds or reorders a table column would leave `nth-child(2)` silently aligning the
   wrong column — a wrong-but-plausible page in front of an audience, which is the worst possible
   failure mode here. A class follows the data it names.
2. **It matches the stylesheet's established idiom.** Every non-element rule in `web/style.css` is
   class-based (`.balance`, `.error`, `.selected-account`, `.stub`). A positional selector would be
   the only structural selector in the file.
3. **It is assertable in both layers** — the markup carries the class and the stylesheet carries the
   rule, so each half fails independently and loudly.

A third candidate, `<colgroup>` with `<col class="amount">`, was **rejected on merits**:
`text-align` is not among the properties that apply to `<col>` (only border, background, width, and
visibility do), so it cannot produce the required alignment at all.

**Specificity is already correct and needs no `!important`.** The existing `th, td { text-align:
left }` rule has specificity `(0,0,1)`; `.amount` has `(0,1,0)`, which wins regardless of source
order. Verified by reading `web/style.css:56-60`.

**The one existing test this breaks — handled explicitly, not discovered later.**
`internal/httpapi/router_test.go:288` builds a whole-row literal:

```go
row := "<tr><td>" + transaction.Description + "</td><td>" + expectedAmount + "</td><td>" + transaction.CreatedAt + "</td></tr>"
```

Adding `class="amount"` to the amount `<td>` makes that literal stop matching. Task 2 updates it in
the same commit as the markup change, keeping it a full-row assertion. Two neighbouring assertions
were checked and are **unaffected**: `<th>Recorded</th>` (router_test.go:255) and
`"<td>"+transaction.CreatedAt+"</td>"` (router_test.go:281) both target the third column.
`name="amount"` (ledger_acceptance_test.go:266) also stays contiguous once `autofocus` is appended
after `required`.

**Sequencing.** The alignment work (Tasks 1–3) and the focus work (Tasks 4–6) are independent and
touch disjoint markup regions — a table cell versus a form input — so they can proceed in either
order or concurrently. They converge at Task 7, the four-branch render matrix that asserts both
changes are inert where their target is absent, and Task 8 hardens the remaining absent-element
assertions.

**FR-10 is not a plan task.** "The balance remains visible without scrolling at 1280×720" cannot be
asserted by a stdlib Go test with no browser. It is carried as a `/manual-test` item and closes
PRD assumption A-1, the single load-bearing assumption in this feature.

## Prerequisites

- None. No migration, no dependency change, no seed change. `go.mod` stays at its single pinned
  requirement.

## Tasks

### Task 1: Add the class-scoped right-alignment rule to the stylesheet

**Story:** Story 1 — "the amount values are right-aligned within their column"
**Type:** happy-path

**Steps:**
1. Write failing test: in `web/web_test.go`, add an `"amount column alignment"` entry to the existing
   rule map in `TestEmbeddedStylesheetActivatesBalanceErrorAndTableRules`, with a regex matching
   `\.amount\s*\{[^}]*text-align:\s*right;[^}]*\}` against `activeCSS` (the comment-stripped source),
   following that test's established regex-per-rule pattern.
2. Verify test fails (RED) — the rule does not exist yet.
3. Implement: append to `web/style.css` a rule `.amount { text-align: right; }` with a brief comment
   stating it is the projector-legibility alignment for the money column.
4. Verify test passes (GREEN), and confirm the existing forbidden-token assertions (`@media`,
   `prefers-color-scheme`, `@keyframes`, `@font-face`, `@import`) still pass and no new colour value
   was introduced.
5. Commit with message: "style: right-align the amount column for projector legibility"

**Files likely touched:**
- `web/web_test.go` — new rule assertion in the existing map
- `web/style.css` — the `.amount` rule

**Wired-into:** none (no new production surface)

**Dependencies:** none

---

### Task 2: Mark the amount heading and amount cells in the template

**Story:** Story 1 — heading aligned with its values; amounts unchanged; existing row assertion kept
**Type:** happy-path

**Steps:**
1. Write failing test: in `internal/httpapi/router_test.go`'s markup test, assert the body contains
   `<th class="amount">Amount</th>` and, for a seeded transaction, `<td class="amount">$1,000.00</td>`,
   using the file's existing `strings.Contains` body-assertion style.
2. Verify test fails (RED).
3. Implement: in `web/index.html.tmpl`, add `class="amount"` to the amount `<th>` in the `<thead>` row
   and to the amount `<td>` in the `range` row. Keep the `range` row on one line — the existing
   whole-row assertion depends on there being no whitespace between cells.
4. Update the existing whole-row literal at `internal/httpapi/router_test.go:288` so its amount cell
   is `<td class="amount">`, keeping it a full-row assertion across description, amount, and
   timestamp.
5. Verify test passes (GREEN), and confirm `<th>Recorded</th>` (line 255) and the standalone
   timestamp-cell assertion (line 281) still pass untouched.
6. Commit with message: "feat: mark the amount column cells so alignment targets only that column"

**Files likely touched:**
- `web/index.html.tmpl` — `class="amount"` on the amount `<th>` and `<td>`
- `internal/httpapi/router_test.go` — new class assertions; updated whole-row literal

**Wired-into:** none (no new production surface)

**Dependencies:** Task 1

---

### Task 3: Prove the alignment does not leak to the other two columns

**Story:** Story 1 — negative path: "a rule broad enough to catch a second column is a failure"
**Type:** negative-path

**Steps:**
1. Write failing test: in `internal/httpapi/router_test.go`, assert the rendered body contains
   `<td>` + description + `</td>` and `<td>` + timestamp + `</td>` **without** a `class="amount"`
   attribute, so only the middle cell carries the class. In `web/web_test.go`, assert the active
   stylesheet contains no bare `td { ... text-align: right }` or `nth-child` right-alignment rule, so
   the alignment stays class-scoped.
2. Verify test fails (RED) if either guard is absent.
3. Implement: no production change expected — Task 2's markup should already satisfy this. If it does
   not, narrow the selector or the markup until it does.
4. Verify test passes (GREEN).
5. Commit with message: "test: assert amount alignment is scoped to the amount column only"

**Files likely touched:**
- `internal/httpapi/router_test.go` — per-column negative assertions
- `web/web_test.go` — selector-scope guard

**Wired-into:** none (no new production surface)

**Verify-only:** yes

**Dependencies:** Task 2

---

### Task 4: Place the caret in the amount input on load

**Story:** Story 2 — "the amount input carries the plain HTML `autofocus` attribute"
**Type:** happy-path

**Steps:**
1. Write failing test: in `test/acceptance/ledger_acceptance_test.go`, extend the form-attribute
   subtest (currently asserting `method="post"`, `name="amount"`, `name="description"`) with
   `mustContain(t, body, "autofocus", "post form")`, and assert the amount input's tag contains both
   `name="amount"` and `autofocus` in the same element.
2. Verify test fails (RED).
3. Implement: in `web/index.html.tmpl`, add the bare `autofocus` attribute to the amount input,
   after `required`, leaving `name="amount"` contiguous. Add nothing to the description input.
4. Verify test passes (GREEN), and confirm the existing `name="amount"`, `name="description"`, and
   no-`<script>` assertions still pass.
5. Commit with message: "feat: autofocus the amount field so the presenter can type immediately"

**Files likely touched:**
- `web/index.html.tmpl` — `autofocus` on the amount input
- `test/acceptance/ledger_acceptance_test.go` — form-attribute assertions

**Wired-into:** none (no new production surface)

**Dependencies:** none

---

### Task 5: Prove exactly one element requests initial focus

**Story:** Story 2 — negative path: "two autofocus candidates would make focus browser-dependent"
**Type:** negative-path

**Steps:**
1. Write failing test: in `internal/httpapi/router_test.go`, assert
   `strings.Count(body, "autofocus") == 1` for a selected-account page, and assert the description
   input's element does not contain `autofocus`.
2. Verify test fails (RED) if a second occurrence is ever introduced.
3. Implement: no production change expected — Task 4 adds exactly one. If the count is not 1, remove
   the extra.
4. Verify test passes (GREEN).
5. Commit with message: "test: assert exactly one element requests initial focus"

**Files likely touched:**
- `internal/httpapi/router_test.go` — count assertion and description-input negative

**Wired-into:** none (no new production surface)

**Verify-only:** yes

**Dependencies:** Task 4

---

### Task 6: Keep the caret in the amount field after a rejected submission

**Story:** Story 2 — "the re-displayed page again places the caret in the amount field"
**Type:** happy-path

**Steps:**
1. Write failing test: in `internal/httpapi/router_test.go`, reuse the existing post-then-follow-
   redirect pattern from the boundary test to render a page carrying a rejection, then assert the
   body contains `autofocus` on the amount input and that the `class="error"` panel still precedes
   `<form` by `strings.Index` ordinal comparison.
2. Verify test fails (RED) before Task 4's template change is present.
3. Implement: no production change expected — the same template serves the rejection re-render.
4. Verify test passes (GREEN).
5. Commit with message: "test: assert the amount field keeps focus after a rejected post"

**Files likely touched:**
- `internal/httpapi/router_test.go` — rejection re-render assertions

**Wired-into:** none (no new production surface)

**Verify-only:** yes

**Dependencies:** Task 4

---

### Task 7: Assert both changes are inert across all four render branches

**Story:** Story 3 — the four-branch matrix; FR-5, FR-11, FR-12, FR-13
**Type:** negative-path

**Steps:**
1. Write failing test: add a table-driven test to `internal/httpapi/router_test.go` over the four
   render branches — no accounts, unknown account (`?account=acct-nope`), selected account with
   transactions, selected account with no transactions (`?account=acct-3`) — with expectations per
   row for: `<table` present, `<form` present, and `strings.Count(body, "autofocus")`. Expect
   autofocus count `0` for the no-accounts and unknown-account rows and `1` for both selected-account
   rows, and expect no `<table` for the no-transaction row while its `<form` is present.
2. Verify test fails (RED) for any row whose expectation is not yet met.
3. Implement: no production change expected — the template's `{{if .AccountNotFound}}` /
   `{{else if .HasSelectedAccount}}` / `{{if .FormAction}}` / `{{if .Transactions}}` guards already
   produce this. If any row fails, fix the template guard rather than the expectation.
4. Verify test passes (GREEN), and confirm the existing zero-transaction assertions
   (`class="balance">$0.00`, the empty-state message) and the existing zero-account page test still
   pass unmodified.
5. Commit with message: "test: assert alignment and autofocus stay inert on form-less and list-less pages"

**Files likely touched:**
- `internal/httpapi/router_test.go` — four-branch render matrix

**Wired-into:** none (no new production surface)

**Verify-only:** yes

**Dependencies:** Task 2, Task 4

---

### Task 8: Harden the remaining absent-element and escaping assertions

**Story:** Story 3 — escaped-injection branch; status code and content type unchanged
**Type:** negative-path

**Steps:**
1. Write failing test: extend the existing escaped-injection case
   (`?account=%3Cscript%3Ealert(1)%3C%2Fscript%3E`) to also assert `autofocus` appears zero times,
   alongside its existing no-raw-`<script>` assertion. In
   `test/acceptance/ledger_acceptance_test.go`, extend the unknown-account subtest's existing
   `mustNotContain` set (`class="balance"`, `<form`, `<table`) with `autofocus`. Assert the status
   code and `Content-Type` for each of the four branches are unchanged.
2. Verify test fails (RED) if any branch emits an unexpected attribute or response.
3. Implement: no production change expected.
4. Verify test passes (GREEN).
5. Commit with message: "test: assert no focus request and unchanged responses on form-less pages"

**Files likely touched:**
- `internal/httpapi/router_test.go` — escaped-injection and response assertions
- `test/acceptance/ledger_acceptance_test.go` — unknown-account negative set

**Wired-into:** none (no new production surface)

**Verify-only:** yes

**Dependencies:** Task 7

---

## Task Dependency Graph

```
Task 1 (stylesheet rule)
   └─▶ Task 2 (template classes + fix row literal)
          ├─▶ Task 3 (alignment does not leak)
          └─────────────┐
                        ├─▶ Task 7 (four-branch matrix) ─▶ Task 8 (escaping + responses)
Task 4 (autofocus)  ────┘
   ├─▶ Task 5 (exactly one focus request)
   └─▶ Task 6 (focus survives rejection)
```

Two independent roots — Task 1 and Task 4 — converging at Task 7. Acyclic.

## Integration Points

- **After Task 2:** the amount column visibly right-aligns in a browser; the alignment half is
  end-to-end verifiable.
- **After Task 4:** the caret lands in the amount field on load; the ergonomics half is end-to-end
  verifiable. This is also the point at which FR-10 becomes checkable — load the page at 1280×720 and
  confirm the balance is still visible without scrolling (closes PRD assumption A-1 via
  `/manual-test`).
- **After Task 8:** every render branch is asserted for both changes.

## Out of scope for BUILD

- The amendment to `.docs/decisions/styleguide.md` recording the amount-column alignment rule is a
  **DECIDE-phase amendment made on this spec branch**, not a BUILD task. Per the harness
  documentation boundary, plans carry no documentation tasks; per DECIDE artifact amendment
  ownership, an accepted decision record is amended in DECIDE before the first BUILD entry.
- `README.md` needs no change: it documents neither the table's columns nor the form's fields
  (verified by grep).

## Verification

- [ ] All happy path criteria covered by at least one task — Story 1 → Tasks 1, 2; Story 2 → Tasks 4,
      6; Story 3 → Task 7
- [ ] All negative path criteria covered by at least one task — Story 1 → Tasks 1, 3; Story 2 →
      Tasks 5, 8; Story 3 → Tasks 7, 8
- [ ] No task exceeds 5 minutes of work
- [ ] Dependencies are explicit and acyclic
- [ ] No terminal catch-all validation task
- [ ] Every task carries a `**Wired-into:**` line
- [ ] `gofmt` clean, `go vet` clean, suite under ten seconds, no `time.Sleep`, no new dependency
- [ ] The five existing routes are unchanged in count and shape

## Coverage Mapping

| FR | Story | Task(s) |
|---|---|---|
| FR-1 | Story 1 | 1, 2 |
| FR-2 | Story 1 | 2 |
| FR-3 | Story 1 | 3 |
| FR-4 | Story 1 | 1, 2 |
| FR-5 | Story 3 | 7 |
| FR-6 | Story 2 | 4 |
| FR-7 | Story 2 | 4 |
| FR-8 | Story 2 | 5 |
| FR-9 | Story 2 | 6 |
| FR-10 | Story 2 | `/manual-test` (not unit-testable; closes assumption A-1) |
| FR-11 | Story 3 | 7, 8 |
| FR-12 | Story 3 | 7 |
| FR-13 | Story 3 | 8 |
| FR-14 | Story 1 | 2 |
| FR-15 | Story 1 | 2, 3 |
| FR-16 | Story 1 | 2 |
| FR-17 | Story 2 | 4 |
