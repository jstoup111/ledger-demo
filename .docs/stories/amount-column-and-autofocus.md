# Stories — Amount Column Alignment and Amount-Field Focus

**Status:** Accepted

**Feature:** amount-column-and-autofocus · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-08-09-amount-column-and-autofocus.md` (Approved, FR-1 … FR-17)
**Constrained by:** `.docs/decisions/styleguide.md` and the seven Accepted ADRs in
`.docs/decisions/`

Three stories covering all seventeen FRs. Tier S, so the rule is **at least one negative path per
story** rather than per acceptance criterion; where a story has a genuinely richer failure surface
(Story 3) more are written anyway.

Stories state observable behavior. **How** the amount column is targeted for alignment is a mechanism
choice deliberately left to `.docs/plans/amount-column-and-autofocus.md` (PRD Open Question 1); the
criteria below are written so that either resolution satisfies them.

The styleguide update that accompanies this work is intentionally **not** a story — documentation
accompanying functional work is carried as a plan task, not as acceptance criteria.

## Negative-path categories evaluated

The skill requires every category to be explicitly evaluated. Most do not apply to a
presentation-layer change, and saying so is more useful than inventing a scenario.

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — an unknown `account` query value must still render the not-found page with no form and no list (Story 3) |
| Invariant side-effect on alternate branches | **Yes** — the page has four render branches (no accounts, unknown account, selected account, selected account with no transactions); Story 3 asserts both changes are inert on every branch that lacks their target |
| Data integrity | No — this change reads nothing and writes nothing; no record is created, updated, or deleted |
| Model-level immutability | No — no model is touched; the transaction log is not read differently |
| Auth / permission failures | No — auth is an explicit non-goal; there is no principal to reject |
| Timeouts & network errors | No — NFR-1 forbids network access; the page and stylesheet are embedded in the binary |
| Dependency unavailability | No — no new dependency; existing database-failure paths return before the template renders and are unchanged |
| Concurrent access | No — no shared mutable state is read or written by either change |
| Resource exhaustion | No — no upload, no batch, no pool |
| Partial failure & rollback | No — nothing multi-step; rendering a page is not a transaction |
| Cascade deletion effects | No — nothing is deleted |
| Exception class hierarchy | No — no error is raised, caught, or classified by this change |
| Dedup / idempotency key analysis | No — there is no dedup or idempotency criterion anywhere in this spec, and adding one is an explicit non-goal |

---

## Story 1: Scan the amount column at projector distance

**Requirement:** FR-1, FR-2, FR-3, FR-4, FR-14, FR-15, FR-16

As an audience member reading the projected page from across the room, I want the transaction amounts
to line up vertically, so that I can scan the column and compare figures without effort.

### Acceptance Criteria

#### Happy Path
- Given a selected account whose transactions include `Paycheck` at `$1,000.00` and `Groceries` at
  `$283.50`, when the page renders, then the amount values are right-aligned within their column so
  the two figures' decimal points fall in the same horizontal position, while `Paycheck` and
  `Groceries` remain left-aligned in the description column.
- Given the transaction list renders, when the amount column's heading is inspected, then it carries
  the same right alignment as the values beneath it, so heading and column read as one unit.
- Given the transaction list renders, when the description and recorded-at columns are inspected,
  then both keep their existing left alignment; the right alignment applies to the amount column and
  to nothing else.
- Given a selected account with a negative amount alongside a positive one, when the page renders,
  then both are right-aligned by the same rule regardless of sign or digit count.
- Given the transaction list renders, when its content is compared against the current behavior, then
  the amount strings themselves are byte-identical to what is rendered today — same currency symbol,
  same thousands separator, same two decimal places, same sign convention — and the rows remain in
  their existing newest-first order with the existing column order preserved.
- Given the page renders, when the top-to-bottom order is inspected, then it remains heading →
  account selector → balance → post form → transaction list.

#### Negative Paths
- Given the embedded stylesheet after this change, when it is scanned for forbidden constructs, then
  it contains no `@media`, no `prefers-color-scheme`, no `@keyframes`, no `@font-face`, and no
  `@import`, and it introduces no colour value outside the six fixed palette entries — the alignment
  rule is alignment only.
- Given the transaction list renders, when the stylesheet's alignment rule is evaluated, then it does
  not right-align the description or recorded-at cells; a rule broad enough to catch a second column
  is a failure, not an acceptable approximation.

### Done When
- [ ] The active stylesheet declares a `text-align: right` rule that resolves for the amount column's
      heading and every amount cell, asserted by the regex-per-rule pattern established in
      `web/web_test.go`.
- [ ] A test proves the description and recorded-at cells are **not** right-aligned by that rule.
- [ ] A page-render test asserts the amount heading and amount cells are marked such that the rule
      targets them and only them, using the existing `strings.Contains` body-assertion style in
      `internal/httpapi/router_test.go`.
- [ ] The existing whole-row literal assertion at `internal/httpapi/router_test.go:288` is updated in
      the same change if and only if the chosen approach alters the row markup, and it still asserts
      the full row including description, formatted amount, and timestamp.
- [ ] The existing amount strings `$1,000.00` and `$283.50` still appear unchanged in the rendered
      page, cross-checked against the independently authored expected-amount fixture.
- [ ] The existing layout-order assertion (heading < account link < balance < form < transaction)
      still passes unmodified.
- [ ] The existing stylesheet guard test still passes: root type basis `20px`, `.balance` at `4rem`
      weight `700`, table `border-collapse: collapse`, `.error` colours unchanged.
- [ ] `gofmt` clean, `go vet` clean, full suite green in under ten seconds.

---

## Story 2: Type an amount the instant the page appears

**Requirement:** FR-6, FR-7, FR-8, FR-9, FR-10, FR-17

As a presenter operating the page live while narrating, I want the amount field to already hold the
caret when the page loads, so that my first keystroke lands in it without a click and no keystroke is
lost in front of an audience.

### Acceptance Criteria

#### Happy Path
- Given a selected account with a form to post against, when the page renders, then the amount input
  carries the plain HTML `autofocus` attribute so the browser places the caret in it on load, and the
  description input does not.
- Given the page renders, when it is scanned for scripts, then no `<script>` element and no inline
  event-handler attribute appears anywhere; the caret is placed by the delivered markup alone.
- Given the page renders with a form, when the whole document is scanned, then the `autofocus`
  attribute occurs exactly once — no second field, link, or button competes for initial focus.
- Given a submission is rejected and the page re-displays with the rejection message, when the
  re-displayed page renders, then the amount input again carries `autofocus` and the rejection panel
  is still positioned above the form.
- Given the page loads at 1280×720, when the initial viewport is inspected, then the balance element
  is within it and no scrolling is required to see it — placing the caret does not displace the
  viewport.
- Given the form renders, when its fields and target are inspected, then it keeps its existing method
  and submission target, both existing field names, and the required-input behavior on both fields;
  posting a transaction from the page behaves exactly as before.

#### Negative Paths
- Given a page state with no form (an unknown requested account, or no accounts at all), when the
  page renders, then the `autofocus` attribute appears nowhere in the response — the attribute must
  not be emitted outside the form it belongs to. (Detailed per state in Story 3.)
- Given the form renders, when the description input is inspected, then it does **not** carry
  `autofocus`; two autofocus candidates would make which field receives the caret
  browser-dependent, which is a determinism failure on stage.
- Given the rendered page, when it is checked for outbound references, then it still contains no
  `http://`, no `https://`, and no `<script`, so the page keeps rendering fully offline.

### Done When
- [ ] The amount input in `web/index.html.tmpl` carries the bare `autofocus` attribute; the
      description input does not.
- [ ] A page-render test asserts `autofocus` is present on the amount input, using the established
      form-attribute assertion style at `test/acceptance/ledger_acceptance_test.go:262-270`.
- [ ] A test asserts `autofocus` occurs exactly **once** in the rendered body (a count assertion, not
      a `Contains`), so a second focus candidate fails loudly.
- [ ] A test asserts the existing `name="amount"` and `name="description"` substrings and the
      `required` behavior are still present and unbroken by the added attribute.
- [ ] A test covers the post-rejection re-render asserting both `autofocus` on the amount input and
      the rejection panel positioned above the form.
- [ ] No JavaScript is introduced: the existing no-outbound-reference and no-`<script>` assertions
      still pass.
- [ ] FR-10 is confirmed by manual test at 1280×720 — the balance is visible on load without
      scrolling. This closes assumption A-1 in the PRD, the one load-bearing assumption in this
      feature.
- [ ] `gofmt` clean, `go vet` clean, full suite green in under ten seconds.

---

## Story 3: Both changes stay inert where their target does not exist

**Requirement:** FR-5, FR-11, FR-12, FR-13

As a presenter, I want the pages that have no transaction list and/or no form to render exactly as
they do today, so that a legibility tweak cannot turn into a broken page on stage.

The page has four render branches. Two of them lack the form, one lacks the list, and one has both.
Neither change may cause a missing element to appear, nor a present element to disappear.

### Acceptance Criteria

#### Happy Path
- Given a selected account with **no transactions** (`GET /?account=acct-3`), when the page renders,
  then it responds `200 text/html` showing a `$0.00` balance and the existing explicit empty-state
  message, and **no** transaction table is emitted — no table shell, no header row, no empty amount
  cell. The alignment rule simply has nothing to match.
- Given that same no-transaction account, when the page renders, then the post form **is** present
  and its amount input carries `autofocus` — this branch has a valid account to post against.

#### Negative Paths
- Given an **unknown** requested account (`GET /?account=acct-nope`), when the page renders, then it
  responds `200` with the account list and the not-found message **only**: no balance element, no
  transaction table, no post form, and consequently no `autofocus` attribute anywhere in the body.
- Given that same unknown-account page, when it is compared against current behavior, then it is
  unchanged by this feature — neither the alignment rule nor the focus attribute causes a form, a
  table, or a focus request to appear on a page that has no account to post against.
- Given **no accounts exist at all**, when the page renders, then it responds `200` with neither a
  post form nor a transaction table, and no `autofocus` attribute appears anywhere in the body.
- Given an `account` query value containing markup
  (`GET /?account=%3Cscript%3Ealert(1)%3C%2Fscript%3E`), when the page renders, then the value is
  HTML-escaped, no raw `<script>` tag appears, and no `autofocus` attribute appears — the
  unknown-account branch is taken and it has no form.
- Given any of the three form-less or list-less states, when the response is inspected, then its
  status code and `Content-Type` are exactly what they are today; this feature introduces no error
  path, no redirect, and no new response.

### Done When
- [ ] A table-driven test covers all four render branches — no accounts, unknown account, selected
      account with transactions, selected account with no transactions — asserting for each whether
      a table is present, whether a form is present, and whether `autofocus` is present.
- [ ] For every branch that renders no form, a test asserts `autofocus` appears **zero** times in the
      body.
- [ ] The existing unknown-account assertions still pass unmodified, including the negative
      assertions on `class="balance"`, `<form`, `<table`, and `aria-label="Transactions"`.
- [ ] The existing zero-transaction assertions still pass unmodified, including `class="balance">$0.00`
      and the empty-state message.
- [ ] The existing zero-account page test still passes unmodified.
- [ ] The existing escaped-injection assertion still passes, extended to assert no `autofocus`.
- [ ] Status codes and content types for all four branches are asserted unchanged.
- [ ] `gofmt` clean, `go vet` clean, full suite green in under ten seconds.
