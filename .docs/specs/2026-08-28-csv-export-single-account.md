# PRD: Selected-account CSV download

**Date:** 2026-08-28
**Status:** Approved

## Problem / Background

The live demo is operated from the account page, but its transaction rows can only be read in the
page or through the existing programmatic listing. A presenter needs one obvious way to obtain the
selected account's raw rows in a spreadsheet-friendly document without leaving the projected flow.

This is an operator-authorized rehearsal exception to the project's export non-goal. The exception is
limited to the behavior below and does not authorize statements, reports, summaries, or a reusable
export subsystem.

## Goals & Non-Goals

**Goals**

- Let a presenter download the valid selected account's raw transaction rows from the account page.
- Keep the downloaded values and order aligned with the existing programmatic listing.
- Distinguish a valid empty account from an account that does not exist.
- Keep repeated downloads deterministic and read-only.

**Non-Goals**

- Command-line delivery or a second invocation surface.
- A sixth HTTP endpoint.
- JavaScript or client-side document construction.
- Multiple accounts, filters, date ranges, column choices, or configurable formatting.
- Statements, totals, running balances, summaries, grouping, or reporting.
- Any ledger, seed-data, or schema mutation.

## Users / Personas

- **Presenter:** operates the ledger account page on a projector and needs one discoverable action
  that downloads the rows currently in view.

## Functional Requirements

- **FR-1:** When a valid account is selected, the page presents exactly one clearly labelled download
  control associated with that account.
- **FR-2:** Activating the control downloads a CSV document containing only the selected account's
  transactions without changing ledger state.
- **FR-3:** The document contains one header row followed by one row per transaction, with exactly four
  fields per row: transaction identifier, signed amount in integer cents, description, and recorded
  time.
- **FR-4:** Downloaded rows carry the same values and newest-first order as the existing programmatic
  listing for that account.
- **FR-5:** A valid account with no transactions still presents the control and downloads a
  header-only document successfully.
- **FR-6:** A requested account that does not exist does not present a download control and does not
  return a CSV document that could be mistaken for a valid empty account.
- **FR-7:** Repeated downloads against the same unchanged ledger are byte-identical.
- **FR-8:** Without an explicit download, the account page, transaction posting, and ordinary
  programmatic transaction listing retain their existing behavior.

## Non-Functional Requirements

- The public HTTP surface remains exactly five endpoints.
- Amounts remain signed integer cents throughout; no floating-point conversion is introduced.
- The feature reads no wall clock, introduces no randomness, and preserves stable transaction order.
- The project gains no third-party dependency or external service.
- The control remains keyboard-operable and legible in the existing projector-oriented page.
- The deterministic full test suite remains under ten seconds.

## Acceptance Criteria / Success Metrics

- A populated selected account renders one download control, and its document matches that account's
  ordinary transaction listing value-for-value and in the same order.
- Switching accounts changes the document source without including rows from the prior selection.
- The seeded empty account renders the control and downloads only the header row.
- A missing-account page renders no download control, and a missing-account download is not a CSV.
- Two downloads from unchanged state are byte-identical and leave account and transaction data
  unchanged.
- Existing page, posting, JSON-listing, and five-endpoint assertions remain green.

## Scope

### In Scope

- One valid selected account, one visible download control, one raw four-field CSV document.
- Populated, empty, missing-account, deterministic, and existing-behavior cases.

### Out of Scope

- CLI delivery, additional endpoints, JavaScript, export choices, statements, summaries,
  calculations, authentication, infrastructure, and every unrelated project non-goal.

## Key Decisions & Rationale

- **Browser-only delivery:** keeps the capability inside the live projected flow and avoids a second
  invocation surface.
- **Selected account only:** keeps the action unambiguous and prevents expansion into reporting.
- **Raw rows only:** exposes recorded data without adding totals, narratives, or derived decisions.
- **Preserve existing listing semantics:** prevents the page, download, and programmatic listing from
  disagreeing during a demo.

## Dependencies

- The account page already has a single selected-account context.
- The existing programmatic listing already exposes the transaction values and stable newest-first
  order the document must match.
- The existing application contract fixes the public HTTP surface at five endpoints.

## Open Questions

None.

## Verify-Claims Ledger — PRD — 2026-08-28

### Claims

- [verified] The current page resolves one valid selected account before rendering its transaction
  rows; missing accounts render a separate error state.
- [verified] The current programmatic listing reads the same stored transactions and publishes them in
  their existing stable order.
- [verified] The current router and its tests enforce exactly five endpoint registrations.
- [verified] PR #24 and the present operator request approve the browser-only scope captured here.

### Assumptions

- None. Every functional requirement is preserved from the approved browser amendment in PR #24.

### Verdict

CLEAR — no unconfirmed load-bearing assumptions remain.
