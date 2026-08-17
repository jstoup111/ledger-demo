**Status:** Accepted

# Stories — One Account's Transactions as CSV

**Status:** Accepted

> **Amended 2026-08-13 by operator:** The leading status marker is canonical for this amendment. This
> original Accepted marker is retained as the approval history of the CLI-only stories.

**Feature:** csv-export-single-account · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-08-09-csv-export-single-account.md` (Approved, FR-1 … FR-8)
**Constrained by:** the eight Accepted decision records in `.docs/decisions/`

Two stories covering all eight FRs. Tier S, so the rule is **at least one negative path per story**;
Story 2 is mostly negative paths by nature and carries more.

Stories state observable behavior. The command word, the function boundaries, and the file layout are
mechanism choices left to `.docs/plans/csv-export-single-account.md`; the criteria below name the
command word `export` only because the PRD's FR-1 does (assumption A2), and are otherwise written so
that a different spelling would still satisfy them.

## Browser-download amendment — 2026-08-13

> **Amended 2026-08-13 by operator:** Story identifiers 1 and 2 remain stable, but their amended
> criteria below are authoritative. Every original criterion and Done-When item requiring a command,
> standard output, command arguments, or an unchanged page is superseded. The raw four-field document,
> integer-cents, newest-first, empty-versus-missing-account, determinism, read-only, and five-endpoint
> guarantees remain in force.

### Verify-Claims Ledger

- [verified] The amended PRD is Approved and enumerates FR-1 through FR-8 for browser delivery.
- [verified] A valid selected account exists in the page data for both populated and empty accounts;
  missing accounts render a distinct error state.
- [verified] The transaction listing already distinguishes expected not-found failures from unexpected
  store failures before it serializes a successful response.
- [approved] Empty accounts retain a visible control and produce a header-only document — operator
  approved the PRD amendment on 2026-08-13.
- [approved] The four-field, integer-cents, newest-first document semantics remain unchanged — operator
  approved the PRD amendment on 2026-08-13.

**Verdict:** CLEAR — no unconfirmed load-bearing assumptions remain.

## Negative-path categories evaluated

The skill requires every category to be explicitly evaluated. Most do not apply to a read-only
output command, and saying so is more useful than inventing a scenario.

> **Amended 2026-08-13 by operator:** The original CLI-focused table below is superseded by this
> browser-download evaluation and remains only as history.

| Category | Current applicability |
|---|---|
| Invalid input | **Yes** — a missing account must not look like a valid empty-account CSV (Story 2) |
| Auth / permission failures | No — authentication remains an explicit non-goal |
| Timeouts & network errors | No — the feature introduces no external network dependency |
| Concurrent access | No — the feature is read-only and adds no shared mutable state |
| Resource exhaustion | No — one account's existing rows are bounded demo data; there is no upload or batch |
| Partial failure & rollback | **Yes** — an unexpected store failure must not emit a partial CSV (Story 2) |
| Dependency unavailability | **Yes** — an unexpected transaction-read failure returns an internal error without leaking details (Story 2) |
| Data integrity | **Yes** — success and failure must leave ledger rows and schema unchanged (Stories 1 and 2) |
| Cascade deletion effects | No — nothing is deleted |
| Model-level immutability | **Yes** — transaction records remain read-only throughout download (Story 2) |
| Exception class hierarchy | **Yes** — the existing missing-account domain failure must still map to not found rather than an internal error (Story 2) |
| Dedup / idempotency analysis | No — duplicate detection remains an explicit non-goal; every stored row is exported independently |
| Invariant side-effect on alternate branches | **Yes** — explicit download behavior must not alter the ordinary JSON branch or posting behavior (Story 2) |

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — an unknown account identifier, a missing account identifier, and more than one account identifier (Story 2) |
| Invariant side-effect on alternate branches | **Yes** — adding a third subcommand must not alter what the two existing subcommands do, and the failure branch must not emit a partial document (Stories 1 and 2) |
| Data integrity | **Yes, as a guarantee to preserve** — the export reads only; no row is created, updated, or deleted, and the seeded dataset is byte-for-byte unchanged after an export (Story 2) |
| Model-level immutability | **Yes** — the transaction log is read through the existing interface and the balance is not consulted; no model is touched (Story 2) |
| Auth / permission failures | No — auth is an explicit non-goal; there is no principal to reject |
| Timeouts & network errors | No — the export makes no network call and does not start or contact the server |
| Dependency unavailability | **Yes** — a database path that cannot be opened must fail with a message naming the path, the same way the existing seed subcommand already does (Story 2) |
| Concurrent access | No — the command reads a local file-backed store and exits; no shared mutable state is written |
| Resource exhaustion | No — no upload, no batch, no pool; the output is one account's existing rows |
| Partial failure & rollback | **Yes, as the central negative path** — the failure case must leave standard output completely empty rather than a header-only document (Stories 1 and 2) |
| Cascade deletion effects | No — nothing is deleted |
| Exception class hierarchy | **Yes** — the unknown-account failure must be the existing account-not-found sentinel, identifiable by wrapping rather than by string comparison (Story 2) |
| Dedup / idempotency key analysis | No — there is no dedup or idempotency criterion anywhere in this spec, and adding one is an explicit non-goal. Two rows that happen to look alike are two rows; the export draws no conclusion from their adjacency |

---

## Story 1: Get one account's history as a spreadsheet-shaped document

> **Amended 2026-08-13 by operator:** This story is now satisfied through a visible `Download CSV`
> control for the selected account, not through a command. The amended criteria and Done-When items
> immediately below replace the original CLI-specific sections retained later in this story.

**Amended Requirement:** FR-1, FR-2, FR-3, FR-4, FR-5, FR-7

As a presenter operating the account page on a projector, I want one obvious control that downloads
the selected account's raw transaction rows as CSV, so I never have to leave the visible demo flow.

### Amended Acceptance Criteria

#### Happy Path

- Given a valid populated account is selected, when the page renders, then one clearly labelled
  `Download CSV` control is visible and associated with that selected account.
- Given that control, when the presenter activates it, then the browser receives a downloadable CSV
  document containing one header row followed by exactly that account's transaction rows.
- Given the downloaded rows, when they are compared with the ordinary programmatic listing for the
  selected account, then identifiers, signed integer-cent amounts, descriptions, recorded times, and
  newest-first order agree element for element.
- Given the presenter switches to a different valid account, when the new control is activated, then
  the document contains only the newly selected account's rows.
- Given a valid selected account with no transactions, when its page renders and the control is
  activated, then the control is present and the downloaded document contains the header row only.
- Given an unchanged ledger, when the same selected account is downloaded twice, then the two
  documents are byte-identical.

#### Negative Paths

- Given no valid account is selected because the requested account does not exist, when the error page
  renders, then no download control is presented for another account.
- Given transaction descriptions containing a comma, quotation mark, or line break, when the document
  is parsed as CSV, then each row still has exactly four fields and each description round-trips
  without corruption.
- Given the presenter changes the selected account before downloading, when the document is read, then
  no row from the previously selected account appears.
- Given a valid empty account, when its document is read, then it is not mistaken for a missing account:
  the header is present and the download succeeds.
- Given the download is repeated, when its implementation is inspected and exercised, then it performs
  no wall-clock read, floating-point conversion, or ledger write that could change the bytes or state.

### Amended Done When

- [ ] A page-level test proves a populated selected account and the seeded empty account each render
      exactly one `Download CSV` control associated with that account.
- [ ] A request-level test activates the control's target and asserts a downloadable CSV response with
      the exact four-field header and one row per selected-account transaction.
- [ ] A table-driven CSV test covers positive and negative integer-cent amounts, special-character
      descriptions, a populated account, and an empty account.
- [ ] A comparison test proves the download and ordinary programmatic listing agree element for
      element and newest-first for the same account.
- [ ] A selection test proves switching accounts changes the download source without leaking rows from
      the prior selection.
- [ ] An invalid-selection page test proves the error state does not render a control for a fallback
      account.
- [ ] A determinism test proves two downloads from unchanged data are byte-identical and the store
      records no mutation.

**Requirement:** FR-1, FR-2, FR-3, FR-4, FR-6, FR-7

As a presenter about to put ledger data on a projector, I want one account's transactions as CSV from
a single command, so that my audience reads rows and columns instead of JSON.

### Acceptance Criteria

#### Happy Path
- Given the seeded demo database, when the presenter runs the export for the first seeded account
  without starting the server, then the command exits zero and writes a CSV document to standard
  output.
- Given that document, when its first line is read, then it is a header row of exactly four columns
  naming, in order, the transaction identifier, the amount in cents, the description, and the recorded
  time — using the same field names the programmatic listing already publishes for those values.
- Given that document, when the lines after the header are read, then there is exactly one line per
  transaction in the account and no additional line of any kind: no total, no running balance, no
  blank separator, and no trailing summary.
- Given a transaction credited `158579` cents and another debited `-6450` cents, when their rows are
  read, then their amount fields are exactly `158579` and `-6450` — bare signed integers with no
  currency symbol, no thousands separator, and no decimal point.
- Given the account's transactions, when the document's row order is compared with the programmatic
  listing's order for the same account, then the two agree element for element: same identifiers in
  the same sequence, newest first, with the same recorded times and the same amounts in cents.
- Given a row's recorded time, when it is compared with the same transaction's recorded time in the
  programmatic listing, then the two strings are identical.
- Given a seeded account that has **no** transactions, when the export runs for it, then the command
  exits zero and emits the header row and nothing after it.
- Given the same unchanged database, when the export is run twice for the same account, then the two
  outputs are byte-identical.

#### Negative Paths
- Given a transaction whose description contains a field separator or a quotation mark, when its row
  is read, then the description is delimited so the row still parses as four fields and round-trips
  back to the original description — the export does not silently corrupt or truncate it.
- Given the export path, when it is inspected for money handling, then no floating-point type, no
  float parsing, and no float formatting appears anywhere in it; the amount reaches the output as the
  stored integer.
- Given the export path, when it is inspected for time handling, then it reads no wall clock: a
  repository-wide search for a system-clock read still returns exactly one hit, in the one place the
  injected-clock decision permits.
- Given the export, when its reads are inspected, then they go through the domain store interface; the
  export issues no query of its own and gains no knowledge of the storage engine.

### Done When
- [ ] A table-driven test drives the CSV-writing function with an in-memory store and captures its
      output in a buffer, asserting the exact header line and the exact full document for a fixture of
      several transactions including a positive amount, a negative amount, and an account with none.
- [ ] A test asserts the document has exactly one line per transaction after the header — a count
      assertion, so an added total row fails loudly.
- [ ] A test asserts the amount field of a positive and a negative transaction are the exact integer
      strings, not formatted currency.
- [ ] A test asserts the recorded-time field is byte-identical to the format the programmatic listing
      publishes for the same instant.
- [ ] A test covers a description containing a separator and a quotation mark, asserting the emitted
      row parses back to four fields with the original description intact.
- [ ] A test asserts an existing account with no transactions yields the header line and nothing else.
- [ ] A test asserts two consecutive renders of the same store produce byte-identical output.
- [ ] The export function takes the domain store interface, not the storage implementation.
- [ ] `gofmt` clean, `go vet` clean, full suite green in under ten seconds, no `time.Sleep`, no new
      dependency, no float type anywhere in the repository.

---

## Story 2: A bad request fails loudly and nothing else changes

> **Amended 2026-08-13 by operator:** This story now covers missing-account download failure and
> preservation of the ordinary page, JSON listing, posting behavior, and five-endpoint contract. Its
> original subcommand and argument-validation criteria below are superseded.

**Amended Requirement:** FR-6, FR-8

As a presenter, I want a missing-account download to fail clearly while ordinary ledger behavior
remains unchanged, so the CSV feature cannot misrepresent an error or destabilize the live demo.

### Amended Acceptance Criteria

#### Happy Path

- Given no explicit download is requested, when the existing transaction listing is read, then its
  status, content type, JSON body, and newest-first behavior are unchanged.
- Given the account page and transaction form, when accounts are selected and transactions are posted,
  then their existing visible behavior and redirects are unchanged.
- Given the feature is present, when the published HTTP surface is counted, then it remains exactly
  five endpoints.

#### Negative Paths

- Given an account identifier that does not exist, when its CSV download is requested, then the
  response is not found and does not carry a CSV document or a header-only body that could be mistaken
  for an empty account.
- Given the store fails unexpectedly while transactions are read, when a CSV download is requested,
  then the response is an internal failure rather than a partial CSV document, and the underlying
  failure is not disclosed to the browser.
- Given a download succeeds or fails, when the store is inspected afterwards, then no account,
  transaction, balance, seed row, or schema has changed.
- Given the route registrations are inspected, when CSV support is present, then no sixth registration
  or standalone CSV endpoint exists.

### Amended Done When

- [ ] A missing-account request test asserts not-found status and the absence of a CSV content type or
      misleading header-only document.
- [ ] An unexpected-store-failure test asserts an internal-error response, no partial CSV bytes, and
      no disclosure of the underlying error.
- [ ] Existing ordinary JSON-listing, page-selection, form-posting, and redirect tests pass unchanged.
- [ ] The existing route-count test still asserts exactly five registrations.
- [ ] Read-only tests prove both successful and failed downloads leave all ledger rows and schema
      unchanged.

**Requirement:** FR-5, FR-8

As a presenter, I want a mistyped account to fail visibly with an empty output and everything else to
behave exactly as it did before, so that a rehearsal instrument cannot quietly produce a document that
misrepresents the ledger or break the demo it was added to.

### Acceptance Criteria

#### Happy Path
- Given the export is added, when the two existing subcommands are run, then each behaves exactly as
  it does today: the server serves on its configured port against its configured database, and the
  seed loads the same deterministic dataset.
- Given an unknown subcommand, when the binary rejects it, then the message names **all three** valid
  subcommands, so the export is discoverable from the error a presenter is most likely to see first.

#### Negative Paths
- Given an account identifier that does not exist, when the export runs, then the command exits
  non-zero, its message names the identifier that was requested, and standard output is **completely
  empty** — zero bytes, not an empty document and not a header-only one.
- Given that same failure, when the error is inspected programmatically, then it is identifiable as the
  existing account-not-found domain failure by wrapping rather than by comparing message text, and no
  new domain failure has been introduced.
- Given the export is run with **no** account identifier, when the command rejects it, then it exits
  non-zero with a message stating that exactly one account identifier is expected, and writes nothing
  to standard output.
- Given the export is run with **two** account identifiers, when the command rejects it, then it
  likewise exits non-zero with that same expectation stated, and writes nothing to standard output.
- Given a configured database path that cannot be opened, when the export runs, then it exits non-zero
  with a message naming that path, and creates no database file — matching how the existing seed
  subcommand already reports the same condition.
- Given an export has been run, when the database is inspected afterwards, then it is byte-for-byte
  unchanged: no row created, updated, or deleted, and no schema change.
- Given the server is started after this feature, when its routes are counted, then there are exactly
  five, unchanged in count and in shape; no route serves CSV and no route was added, removed, or
  renamed.
- Given the page and the programmatic listing, when their responses are compared against current
  behavior, then both are unchanged — same status codes, same content types, same bodies for the same
  database.
- Given the existing test suite, when it is run after this feature, then the only assertion that
  changed is the one reading the unknown-subcommand message; every other existing assertion passes
  unmodified.

### Done When
- [ ] A test asserts the unknown-account export returns an error that satisfies identity comparison
      against the existing account-not-found sentinel, and that the error message contains the
      requested identifier.
- [ ] A test asserts that on the unknown-account path the output buffer receives **zero** bytes — a
      length assertion, so a header written before the account check fails loudly.
- [ ] A table-driven test covers the argument-count rejections (none, two) asserting a non-zero
      outcome, a message naming the expectation, and no output.
- [ ] A test asserts an unopenable configured database path is reported with the path named and no
      database file created, mirroring the existing seed-subcommand test for that condition.
- [ ] The existing unknown-subcommand assertion is updated in the same change to require all three
      valid command words, and still fails if any is missing from the message.
- [ ] The existing subcommand tests — serve against a configured port and database, seed determinism,
      seed missing-directory reporting — pass unmodified.
- [ ] An end-to-end spec drives the real compiled binary: seed into a temporary directory, export, and
      assert the exported rows agree with the programmatic listing's rows for the same database, then
      export an unknown account and assert a non-zero exit with empty standard output.
- [ ] A test asserts the router still exposes exactly five routes, and the existing page and listing
      assertions pass unmodified.
- [ ] No test touches the default demo database file; file-backed tests use a temporary directory.
- [ ] `gofmt` clean, `go vet` clean, full suite green in under ten seconds, and green again on a
      repeated run.
