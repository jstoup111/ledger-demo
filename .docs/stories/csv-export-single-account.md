**Status:** Accepted

# Stories — Selected-account CSV download

## Story 1: Show the download action for the valid selected account

**Requirement:** FR-1, FR-5

As a presenter, I want one obvious download action for the account currently on screen so that I can
obtain its rows without leaving the projected flow.

### Acceptance Criteria

#### Happy Path

- Given a populated valid account is selected, when the page renders, then exactly one clearly
  labelled download control is associated with that account.
- Given a valid empty account is selected, when the page renders, then the same download control is
  present even though the transaction list is empty.

#### Negative Paths

- Given the requested account does not exist, when the error page renders, then no download control is
  presented for a fallback account.
- Given the presenter switches between two valid accounts, when each page renders, then each control
  is associated only with the currently selected account.

### Done When

- [ ] Page-level tests prove populated and empty valid accounts each render exactly one selected-account
      download control.
- [ ] Page-level tests prove a missing-account page renders no download control and switching accounts
      changes the control's selected-account association.

---

## Story 2: Download the selected account's raw transaction rows

**Requirement:** FR-2, FR-3, FR-4

As a presenter, I want the selected account's transactions as a simple CSV document so that the same
ledger values are readable as rows and columns.

### Acceptance Criteria

#### Happy Path

- Given a populated valid account is selected, when the control is activated, then the response is a
  downloadable CSV containing one header row and exactly one row per selected-account transaction.
- Given the document is parsed, when its rows are compared with the ordinary programmatic listing,
  then transaction identifiers, signed integer-cent amounts, descriptions, recorded times, and
  newest-first order agree element for element.
- Given a description contains a comma, quotation mark, or line break, when the CSV is parsed, then
  the row still has exactly four fields and the description round-trips unchanged.

#### Negative Paths

- Given the presenter changes the selected account before downloading, when the document is parsed,
  then it contains no row from the previously selected account.
- Given positive and negative amounts are downloaded, when their fields are read, then they remain
  bare signed integer-cent values with no currency or floating-point formatting.
- Given a download succeeds, when the ledger is inspected afterward, then no account or transaction
  was created, updated, or deleted.

### Done When

- [ ] Request-level tests prove the response is a downloadable four-field CSV whose parsed rows match
      the ordinary listing value-for-value and newest-first.
- [ ] Table-driven rendering tests prove positive and negative cents plus comma, quote, and line-break
      descriptions round-trip without field corruption.
- [ ] Selection and read-only tests prove the document contains only the selected account's rows and
      leaves ledger state unchanged.

---

## Story 3: Distinguish an empty account from a missing account

**Requirement:** FR-5, FR-6

As a presenter, I want empty and missing accounts to produce visibly different outcomes so that a
failed lookup cannot be mistaken for a valid account with no activity.

### Acceptance Criteria

#### Happy Path

- Given a valid account has no transactions, when its download is requested, then the request
  succeeds with the header row and no data rows.

#### Negative Paths

- Given the requested account does not exist, when a download is requested, then the response is not
  found and carries neither a CSV content type nor a header-only document.
- Given the transaction read fails unexpectedly, when a download is requested, then the response is
  an internal failure with no partial CSV and no disclosure of the underlying error.

### Done When

- [ ] A request-level test proves a valid empty account downloads exactly the header row.
- [ ] Missing-account and unexpected-read tests prove failure responses contain no CSV headers or
      partial CSV body and do not disclose internal errors.

---

## Story 4: Preserve deterministic ordinary ledger behavior

**Requirement:** FR-7, FR-8

As a presenter, I want the download addition isolated from existing ledger behavior so that the live
demo remains repeatable.

### Acceptance Criteria

#### Happy Path

- Given the ledger is unchanged, when the same selected account is downloaded twice, then the two
  documents are byte-identical.
- Given no download is requested, when the ordinary transaction listing is read, then its status,
  content type, JSON body, and newest-first behavior are unchanged.
- Given the existing account page and posting form are used, when accounts are selected and
  transactions are posted, then their visible behavior and redirects are unchanged.

#### Negative Paths

- Given CSV support is present, when the public endpoint registrations are counted, then there is no
  sixth endpoint.
- Given a query does not explicitly request a download, when the ordinary listing is read, then it is
  not silently changed to CSV.
- Given downloads are repeated, when their execution is inspected and exercised, then they read no
  wall clock, perform no floating-point conversion, and mutate no ledger state.

### Done When

- [ ] Determinism tests prove repeated downloads from unchanged state are byte-identical.
- [ ] Existing JSON-listing, page-selection, posting, redirect, and five-endpoint tests remain green,
      with a focused request test proving non-download listing requests stay JSON.
- [ ] Static and behavioral checks prove the download path introduces no wall-clock read,
      floating-point handling, dependency, or ledger mutation.

## Verify-Claims Ledger — stories — 2026-08-28

### Claims

- [verified] FR-1 through FR-8 in the approved PRD are covered by Stories 1 through 4.
- [verified] Each story contains concrete happy and negative paths plus independently verifiable Done
  When outputs.
- [verified] The current architecture introduces no external call, queue, lock, authentication, or
  new mutable state requiring an additional architecture-induced negative path.

### Assumptions

- None. The scenarios rest on the approved PRD and verified current application behavior.

### Verdict

CLEAR — no unconfirmed load-bearing assumptions remain.

