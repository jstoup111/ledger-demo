# PRD — Amount Column Alignment and Amount-Field Focus

**Status:** Approved
**Date:** 2026-08-09
**Track:** product (`.docs/track/amount-column-and-autofocus.md`)
**Tier:** S (`.docs/complexity/amount-column-and-autofocus.md`)
**Stem:** `amount-column-and-autofocus`

## Problem / Background

`ledger-demo` is a stage prop. Its single measure of success, per
`.docs/decisions/styleguide.md`, is that a room full of people can read the page from a distance
while a presenter talks over it. On this project legibility and presenter ergonomics are therefore
**functional requirements, not preferences** — a thing the audience cannot read is a broken feature.

Two concrete defects remain in that surface:

1. **Amounts do not line up.** Every cell in the transaction table is left-aligned, so a column of
   money reads as a ragged right edge: `$1,000.00` sits directly above `$283.50` with their decimal
   points in different places. An audience at 1280×720 from across a room cannot scan the column or
   compare magnitudes at a glance — the one thing a money column exists to let them do.
2. **The presenter must click before typing.** On page load no field holds the caret. To post a
   transaction the presenter has to find and click the amount box before the first keystroke lands.
   On stage, mid-sentence, that is a fumble in front of an audience — and a keystroke aimed at an
   unfocused page is silently lost.

Both are small, self-contained presentation defects. Neither touches the ledger's behavior: no
amount, balance, ordering, validation rule, or stored value changes.

## Goals

- An audience can scan the amount column at projector distance and compare figures without
  effort, because the figures align.
- A presenter can begin typing an amount the instant the page appears, with no click and no lost
  keystroke.
- Neither change alters what the ledger computes, records, rejects, or displays as a value.
- Neither change adds a failure mode to a live demo.

## Non-Goals

Excluded **by design**, not deferred. The features added live on stage are drawn from this list; if
any already exists, the demo is ruined.

- Duplicate or double-charge detection of any kind; idempotency keys, dedup windows, or any
  uniqueness constraint beyond the primary key.
- Overdraft allowance, fees, or percentage calculations.
- Pending transactions or holds; available-versus-posted balance.
- Statements, exports, or reporting.
- Interest or rounding rules.
- Authentication, users, sessions, or multi-tenancy.
- Transfers between accounts, or any balancing counter-entry.
- Containerization, continuous integration, or deployment tooling.
- Metrics, tracing, or structured logging beyond what the standard library provides.

Additionally out of scope for *this* feature specifically:

- Any dark theme, responsive breakpoint, animation, or webfont.
- Any new colour value beyond the six fixed palette entries.
- Re-ordering, renaming, adding, or removing a table column.
- Changing the fixed page layout order.
- Re-aligning any column other than the amount column, or any other table on the page (there is
  only one).
- Changing which fields the form has, their names, or their validation.

## Users / Personas

- **The presenter** — operates the page live while narrating. Needs to type an amount immediately
  and to trigger a rejection deliberately without losing their place.
- **The audience** — reads the projected page at 1280×720 from across a room. Never interacts.
  Reads the balance and scans the transaction list.

## Functional Requirements

### Amount column alignment

- **FR-1** In the transaction list, the amount values are aligned to the right-hand edge of their
  column, so that the figures in consecutive rows line up vertically and the column can be scanned
  down its right edge.
- **FR-2** The heading of the amount column is aligned consistently with the values beneath it, so
  the heading and its column read as one unit rather than as two differently-positioned elements.
- **FR-3** Every other column keeps its existing left alignment. The change is confined to the
  amount column.
- **FR-4** The alignment applies to every amount regardless of sign, magnitude, or digit count —
  including negative amounts and amounts whose formatted length differs from their neighbours'.
- **FR-5** When the selected account has no transactions, the page renders its existing empty-state
  message and no transaction list at all. Alignment is a property of a list that is present; its
  introduction must not cause an empty account to render a list, a table shell, an empty header row,
  or an error.

### Immediate amount entry

- **FR-6** When the page loads with a form to post against, the amount field holds the input caret,
  so the presenter's first keystroke is entered into it without any click or keyboard navigation.
- **FR-7** The caret is placed by the markup as delivered, requiring no script to run. (See
  Dependencies — the project permits no JavaScript at all.)
- **FR-8** Exactly one element on the page requests initial focus. No second field, link, or button
  competes for it.
- **FR-9** After a submission is rejected, the re-displayed page again places the caret in the
  amount field, so the presenter can correct the amount and resubmit without reaching for the mouse.
  The rejection message remains visible above the form.
- **FR-10** Placing the caret in the amount field must not displace the page's viewport: the balance
  — the single largest element and the thing the audience watches — remains visible without
  scrolling when the page loads at 1280×720.

### Pages that have no form and no list

These are the three states in which one or both target elements are absent. The feature must be
inert in each, not merely non-crashing.

- **FR-11** On the page for a requested account that does not exist, the page continues to render
  the account list and the not-found message **only** — no balance, no transaction list, and no
  post form. Neither the alignment change nor the focus change causes any of those to appear, and
  no element on that page requests focus.
- **FR-12** When no accounts exist at all, the page continues to render without a form and without a
  transaction list, and no element on that page requests focus.
- **FR-13** In every state above, the page is served with its existing status code and content type.
  Neither change introduces an error path, a redirect, or a new response.

### Preserved behavior

- **FR-14** The amount values themselves are unchanged: same formatting, same currency rendering,
  same sign convention, same two-decimal presentation.
- **FR-15** The transaction list keeps its existing newest-first stable order, its existing columns
  in their existing order, and its existing per-row bottom rule.
- **FR-16** The page keeps its fixed top-to-bottom order: heading → account selector → balance →
  post form → transaction list.
- **FR-17** The form keeps its existing fields, its existing required-input behavior, and its
  existing submission target. Posting a transaction from the page behaves exactly as before.

## Non-Functional Requirements

Only those that bear on this change:

- **NFR-1 Offline.** The page must render fully with no network access. No external stylesheet,
  font, script, or image may be introduced.
- **NFR-2 Single fixed theme.** Light theme only. No dark-mode variant, no colour-scheme
  preference query, no responsive breakpoint, no animation, and no webfont may be introduced.
- **NFR-3 Fixed palette.** The six fixed palette values are the whole palette. This change
  introduces no new colour value; it is an alignment and focus change, which needs none.
- **NFR-4 Type scale preserved.** The root type basis that makes the page projector-legible, and
  the established balance / heading / table-content sizes, are unchanged.
- **NFR-5 Determinism.** The rendered page for a given seeded state stays byte-identical run to
  run. No time, randomness, or environment-dependent value is introduced.
- **NFR-6 Test discipline.** Tests remain standard-library, table-driven, and deterministic, with
  no added sleeps, and the full suite stays under ten seconds.
- **NFR-7 Surface stability.** The count and shape of the served endpoints is unchanged.

## Acceptance Criteria / Success Metrics

- Projected at 1280×720, a column of mixed-magnitude amounts reads with its figures vertically
  aligned; the decimal points of consecutive two-decimal amounts fall in the same place.
- On page load the presenter types digits and they appear in the amount field, with no click.
- The balance is visible on load without scrolling.
- The unknown-account page, the no-accounts page, and the no-transactions page each render exactly
  as they do today.
- The existing suite passes, extended to cover the alignment, the initial caret, and each of the
  three absent-element states.

## Scope

**In scope**

- Right alignment of the amount column's values and its heading.
- Initial caret placement in the amount field, delivered by markup alone.
- Test coverage for both, including the three states where the form and/or the list is absent.
- Recording the alignment rule and the initial-caret rule in the governing frontend styleguide, so
  a future change finds the decision at the point of use. This is an amendment to an accepted
  decision record and is therefore made during DECIDE on the spec branch, not as build work.

**Out of scope**

- Everything under Non-Goals.
- Any change to Go domain, store, clock, or routing logic.
- Any change to the database schema or seed data.

## Key Decisions & Rationale

Product-level only.

- **Alignment is applied to the amount column alone.** Right-aligning descriptions or timestamps
  would hurt scanning, not help it — text reads from its left edge, money compares on its right.
- **The heading follows its values.** A left-aligned heading over right-aligned figures reads as a
  misplaced label at distance and undercuts the alignment it labels.
- **The caret goes to the amount field, not the description.** Amount is the first field in the
  form and the value the presenter is narrating when they start typing; a description can be typed
  after a single tab.
- **The rejection path keeps the caret.** A deliberately triggered rejection is a scripted moment in
  the demo. Returning the caret to the amount field makes the recovery keystroke-only.
- **Both changes are deliberately inert where their target is missing.** Three page states have no
  form, no list, or neither. Specifying them as requirements rather than discovering them later is
  what keeps a legibility tweak from becoming a stage failure.

## Dependencies

Pre-existing external constraints this feature must live within — none chosen here:

- **The project permits no JavaScript anywhere.** This is a standing project rule, not a decision
  of this feature, and it is why initial caret placement must be expressed declaratively in the
  delivered markup rather than by a script on load.
- **The governing frontend styleguide** (`.docs/decisions/styleguide.md`) fixes the theme, palette,
  type scale, and layout order, and forbids dark mode, breakpoints, animation, and webfonts.
- **The single existing page and its embedded stylesheet** are the entire surface being changed.
- **The existing transaction list already has an amount column** in a fixed column order; this
  feature aligns it rather than introducing it.

## Assumptions

Recorded per the harness correctness gate. The operator was unavailable at authoring time; each is
labelled **assumed, operator unavailable** and none is presented as confirmed.

| # | Assumption | Confidence | Basis | Impact if wrong | How to confirm |
|---|---|---|---|---|---|
| A-1 | Placing the caret in the amount field does not scroll the balance out of view at 1280×720, because the form sits high in the fixed layout order and the viewport is taller than the content above the form. | 88% | inferred — fixed layout order plus known viewport; not yet observed on a projector | **High for the demo.** If the browser scrolls to the focused field, the audience loses the balance at the moment of load — the opposite of this feature's goal. Mitigation: FR-10 makes it a testable requirement, so it fails loudly rather than on stage. | Load the page at 1280×720 and confirm the balance is visible without scrolling. Covered by FR-10 and by manual test. |
| A-2 | Right alignment alone makes the figures line up, because every amount is presented with exactly two decimal places. | 97% | verified — the existing amount presentation is uniformly two-decimal in the shipped page and its fixtures | Low-to-moderate. If some amount rendered with a different decimal count, the right edge would still align but the decimal points would not, weakening FR-1's promise. | Assert alignment against a fixture with mixed magnitudes and a negative amount (FR-4). |
| A-3 | No new colour, preference query, breakpoint, animation, webfont, or script is needed for either change. | 99% | verified — alignment and initial focus are both expressible within the existing fixed theme | None if right. If wrong, NFR-2/NFR-3 would be breached, which is a hard stop and an open question, never a silent addition. | The existing stylesheet guard test already fails on any such token. |
| A-4 | The amount field is the right field to focus. | 92% | inferred — it is the form's first field and the operator asked for it specifically | Very low; trivially retargeted. | Presenter confirmation at rehearsal. |

None of these changes a requirement's substance if it turns out otherwise, except **A-1**, which is
why it is written as FR-10 — a requirement to be proven — rather than left as an assumption to be
trusted.

## Open Questions

1. **How the amount column is targeted for alignment** — by its position in the column order, or by
   marking the amount cells as amount cells. The trade-off is coupling to column position versus a
   small markup change and one existing test assertion to update. This is the one mechanism choice
   the feature contains. Tier S skips `/architecture-review`, so it is resolved in the
   implementation plan and recorded there with its rationale, not decided here.
2. **Whether to also request tabular (fixed-width) figures** so integer digits align in columns and
   not merely at the decimal point. This would sharpen scanning further at distance. It is a
   deliberate addition beyond what was asked, is not required to satisfy FR-1, and is **not adopted
   here** — recorded as an open question for a future pass rather than folded into this one.
3. **Whether the governing styleguide's "Current state" paragraph should be corrected.** It states
   that the balance, error, and table rules are "present but commented out"; they are now active.
   This is pre-existing drift from the base-ledger build, not something this feature introduces, and
   correcting it is outside this scope. Flagged so it is not mistaken for a new inconsistency.
