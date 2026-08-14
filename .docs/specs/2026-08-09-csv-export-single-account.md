# PRD — One account's transactions as CSV from a single command

**Status:** Approved
**Date:** 2026-08-09
**Feature:** csv-export-single-account · **Tier:** S · **Track:** product
**Origin:** intake issue `jstoup111/ledger-demo#14`
**Supersedes:** nothing. Amends no accepted requirement.

## Browser-download amendment — 2026-08-13

**Amendment Status:** Approved

> **Amended 2026-08-13 by operator:** The browser-download requirements in this section supersede
> every original statement below that requires a command-line invocation, standard-output delivery,
> or no page/HTTP change. The original text remains as decision history. Its narrow authorization,
> raw four-field CSV content, integer-cents rule, newest-first order, empty-versus-missing-account
> distinction, and determinism requirements remain in force where this amendment does not replace
> them.

### Problem / Background

PR #24 gives a terminal user a narrow CSV export, but the live demo is driven from the account page.
A presenter should not have to leave that page, identify a command, or redirect output just to obtain
the selected account's rows. The missing visible action makes the feature undiscoverable on a
projector and leaves the intended demo flow incomplete.

### Goals

1. A presenter can download the selected account's transactions as CSV directly from the account
   page through one clearly labelled control.
2. The downloaded rows agree with the existing page and programmatic listing in values and order.
3. The interaction remains deterministic and distinguishes an empty account from an account that
   does not exist.
4. The feature stays small enough for a 30–40 minute live full-loop demonstration.

### Non-Goals

- A command-line export or any second invocation surface.
- A new standalone HTTP endpoint; the published endpoint count remains five.
- JavaScript or client-side CSV construction.
- Exporting multiple accounts, date filtering, column selection, configurable formatting, or a
  user-chosen destination.
- Statements, totals, running balances, summaries, grouping, reporting, or any broader exception to
  the project's non-goals.
- Any write to accounts, transactions, seed data, or schema.

### Users / Personas

- **Presenter:** operates the account page on a projector and needs one obvious action that downloads
  the currently selected account's rows.

### Amended Functional Requirements

- **FR-1** Whenever a valid account is selected, the page presents one clearly labelled
  `Download CSV` control associated with that account.
- **FR-2** Activating the control downloads a CSV document for exactly the selected account without
  requiring a terminal step and without changing ledger state.
- **FR-3** The document contains one header row followed by one row per transaction, with exactly four
  fields per row: transaction identifier, signed amount in integer cents, description, and recorded
  time.
- **FR-4** Transaction rows appear newest first and carry the same identifiers, amounts, descriptions,
  and recorded times as the existing programmatic listing for that account.
- **FR-5** A valid account with no transactions still offers the control; its download succeeds with
  the header row and no data rows.
- **FR-6** A request naming an account that does not exist fails as not found and does not return a
  CSV document that could be mistaken for an empty account.
- **FR-7** Repeated downloads against the same unchanged ledger produce byte-identical documents.
- **FR-8** Existing behavior remains unchanged outside an explicit download: the account page still
  selects and displays accounts normally, transaction posting is unchanged, and the programmatic
  transaction listing retains its existing JSON response.

### Non-Functional Requirements

- **NFR-1 — Existing HTTP contract.** The published surface remains exactly five endpoints; existing
  status codes and ordinary response behavior remain compatible.
- **NFR-2 — Money integrity.** Amounts remain signed `int64` cents throughout, with no floating-point
  arithmetic or formatting.
- **NFR-3 — Determinism.** The feature reads no wall clock, introduces no randomness, and preserves
  the existing stable transaction order.
- **NFR-4 — Dependency boundary.** The project keeps its single existing third-party dependency and
  requires no network service.
- **NFR-5 — Demo presentation.** The control remains legible and operable in the existing
  projector-oriented page.
- **NFR-6 — Test budget.** The deterministic full suite remains under ten seconds.

### Acceptance Criteria / Success Metrics

- Select a populated account and activate `Download CSV`: the browser downloads one CSV document;
  its data rows match that account's programmatic transaction listing element for element and in the
  same order.
- Switch to another account and download again: only the newly selected account's rows appear.
- Select the seeded empty account: the control remains visible and the download contains only the
  header row.
- Request a download for a missing account: the response is not found and no misleading CSV document
  is returned.
- Request the ordinary programmatic transaction listing: its status, content type, and JSON body are
  unchanged.
- Repeated downloads for unchanged data are byte-identical; the endpoint count remains five; no
  ledger row changes.

### Scope

**In:** one selected account, one visible download control, one raw CSV document, existing ordering
and values, empty- and missing-account behavior, preservation of the ordinary page and JSON listing.

**Out:** CLI delivery, additional endpoints, JavaScript, additional export choices, statements,
summaries, calculations, authentication, infrastructure, and every unrelated non-goal in
`CLAUDE.md`.

### Key Product Decisions & Rationale

- **Browser-only delivery:** chosen over keeping both browser and CLI surfaces because the visible
  demo outcome is the goal and duplicate invocation paths threaten the time box.
- **Selected account only:** keeps the action unambiguous and prevents the exception from expanding
  into reporting.
- **Raw rows only:** preserves PR #24's narrow authorization; no totals or narrative are added.
- **Existing listing semantics:** one ordering and one set of values prevent the page, download, and
  programmatic listing from disagreeing during a demo.

### Dependencies

- The account page already has a single selected-account context.
- The existing programmatic listing already exposes the four transaction values and deterministic
  newest-first order the download must match.
- The approved architecture and tests fix the published HTTP surface at five endpoints.
- The project already distinguishes missing accounts from valid accounts with empty histories.

### Open Questions

None. The operator selected browser-only delivery, Product track, and Tier S; the inherited five-route
constraint and existing response contract bound the remaining behavior.

### Verify-Claims Ledger — PRD amendment

#### Claims

- [verified] The page has one selected-account context and already renders that account's transaction
  list — read `internal/httpapi/router.go` and `web/index.html.tmpl`.
- [verified] The ordinary programmatic listing returns the same stored rows in deterministic
  newest-first order — read `handleAccountTransactions` and `store.Transactions`.
- [verified] Exactly five HTTP endpoints are a governing constraint, not merely an observation — read
  `.docs/architecture/containers.md` and `TestNewRouterDeclaresExactlyFiveRoutes`.

#### Assumptions

- [load-bearing, 90%] A valid empty account should still show the control and download a header-only
  document.
  - Basis: inferred from PR #24's accepted empty-account requirement and the operator's choice to
    replace only the delivery surface.
  - Impact if wrong: FR-5 and its story scenarios change.
  - Confirm by: operator approval of this amendment.
  - **Status: APPROVED by operator 2026-08-13**
- [load-bearing, 95%] PR #24's four-field, newest-first, integer-cents CSV semantics remain desired.
  - Basis: inferred from the request to add a button to PR #24 and the selection of the approach that
    replaces delivery rather than document content.
  - Impact if wrong: FR-3, FR-4, and the implementation tasks change.
  - Confirm by: operator approval of this amendment.
  - **Status: APPROVED by operator 2026-08-13**

#### Verdict

CLEAR — the operator approved the amendment and both load-bearing assumptions on 2026-08-13.

## Authorization — read before the Non-Goals section

"Statements, exports, or reporting" is on this project's non-goals list, and that list is the
live-demo payload rather than an oversight. This feature is nonetheless in scope because the
originating intake issue, authored by the operator, **explicitly authorizes spending this one item**
as a rehearsal: the intent is to time a narrow build loop end to end and then revert the
implementation.

That authorization is narrow and is not transitive. It covers a plain dump of one account's existing
transaction rows and nothing else. It does not authorize statements, summaries, totals, reports, date
grouping, or any other item on the non-goals list, and the Non-Goals section below is written to close
those doors rather than leave them ajar.

## Problem

There are two ways to read transaction data out of this ledger, and both assume a reader who is
comfortable with the medium:

```
GET /api/accounts/{id}/transactions   → a JSON array
GET /?account={id}                    → an HTML table in a browser
```

Neither is a plain tabular document. A presenter who wants to put ledger data in front of an audience
in the form audiences already recognise — rows and columns, openable in a spreadsheet — has no way to
get one, and reading raw JSON off a projector is exactly the kind of friction this project exists to
avoid.

The honest statement of impact is small: this is a rehearsal instrument first and a convenience
second. It exists to measure a narrow loop end to end. Its secondary value is real but modest.

## Goals

1. A presenter obtains one account's transactions as CSV in a single command, without starting the
   server and without adding anything to the HTTP surface.
2. The exported document agrees with what the page and the programmatic listing already show — same
   rows, same order, same recorded times, same amounts — so the three surfaces cannot disagree in
   front of an audience.
3. Asking for an account that does not exist fails loudly and produces no document at all, rather
   than an empty file that reads as "this account has no history".
4. The export is deterministic: the same database produces the same bytes every time.

## Non-Goals

Restated in full because this project's non-goals are load-bearing, and because this feature is the
one authorized incursion into them — which makes it exactly the place where scope would creep. This
feature must not introduce, hint at, or leave a hook for any of the following:

- **Duplicate or double-charge detection of any kind.** No idempotency key, no dedup window, no
  "same amount and description" grouping or flagging in the output, and **no uniqueness constraint of
  any kind beyond the existing primary key**. An export that surfaced repeated rows adjacently must
  not draw any conclusion from that adjacency; identical-looking rows are simply rows.
- Overdraft allowance, fees, or percentage calculations.
- Pending transactions or holds; available-versus-posted balance. The export carries posted
  transaction rows, and there is no second category for it to distinguish.
- **Anything resembling a statement, a report, or a summary** — no total row, no running balance
  column, no opening or closing balance, no per-period subtotal, no date grouping, no header block
  naming the account or the period, and no title. The authorization covers a row dump, not a
  document with a narrative.
- Interest or rounding rules. Amounts are copied, not computed.
- Authentication, users, sessions, or multi-tenancy.
- Transfers between accounts, or any balancing counter-entry.
- Containerization, continuous integration, or deployment tooling.
- Metrics, tracing, or structured logging beyond what the standard library provides.

Also out of scope, and named by the intake issue itself: exporting all accounts at once; date
filtering; column selection; writing to a file path given as an argument; configurable quoting or
escaping; and a CSV HTTP route.

## Functional Requirements

- **FR-1** A presenter can export one account's transactions as CSV with a single command against the
  existing demo database, naming the account, without starting the server. The output goes to standard
  output so the presenter can read it directly or redirect it with the shell they already have.
- **FR-2** The document carries one header row followed by one row per transaction, with exactly four
  columns in this order: the transaction's identifier, its signed amount, its description, and its
  recorded time. The header names match the field names the programmatic listing already publishes for
  those values, so a reader who has seen one surface recognises the other.
- **FR-3** Amounts appear as **integer cents**, signed, exactly as stored — no currency symbol, no
  thousands separator, no decimal point, and no floating-point formatting anywhere in the path. A
  credit of `158579` appears as `158579` and a debit of `-6450` appears as `-6450`.
- **FR-4** Rows appear newest first, in exactly the same order the page and the programmatic listing
  use, and each row's recorded time is rendered in the same format those surfaces already use. For any
  account, the three surfaces list the same transactions in the same sequence with the same recorded
  times and the same amounts.
- **FR-5** Requesting an account that does not exist fails: the command exits non-zero, reports the
  requested account identifier in its message, and writes **nothing** to standard output — not an
  empty document and not a header-only document.
- **FR-6** Requesting an account that exists but has no transactions succeeds, exits zero, and emits
  the header row and nothing after it. This is the deliberate counterpart to FR-5: an empty history and
  a missing account are different answers and must look different.
- **FR-7** Two runs against the same unchanged database produce byte-identical output.
- **FR-8** Existing behavior is untouched: the HTTP surface stays at exactly five routes, the page and
  the programmatic listing are unchanged, the seeded dataset is unchanged, and the existing
  subcommands continue to behave exactly as they do today. The only existing assertion that moves is
  the one that reads the unknown-subcommand message and requires it to name every valid command.

## Non-Functional Requirements

- **NFR-1 — Determinism (inherited, non-negotiable).** No randomness and no wall-clock read anywhere
  in the export path. Repeated runs are indistinguishable; the suite stays deterministic and passes
  repeated runs with no ordering dependency between tests and no sleeping.
- **NFR-2 — Exactly one wall-clock read in the repository (inherited external constraint).** The
  export reads recorded times off stored rows and needs no clock at all. A repository-wide search for a
  system-clock read must still return exactly one hit, in the one place the injected-clock decision
  permits.
- **NFR-3 — No floating-point arithmetic or formatting.** Money stays `int64` cents from the row to
  the emitted field. No `float32`, no `float64`, and no float parsing or formatting in the export path.
- **NFR-4 — Reads go through the existing domain store interface.** The export gains no knowledge of
  SQLite and issues no query of its own.
- **NFR-5 — One dependency, unchanged.** `encoding/csv` is standard library. The module's single
  pinned requirement is untouched.
- **NFR-6 — Suite budget and hygiene.** The full suite stays under ten seconds and fully
  deterministic. Tests use in-memory storage for unit coverage and a temporary directory for anything
  file-backed; the default demo database file is never touched by a test.
- **NFR-7 — Lint clean.** Formatting and vetting gates pass with no findings.
- **NFR-8 — The HTTP surface stays at exactly five routes.** Stated as a non-functional requirement
  and not merely as a non-goal, because it is the constraint that decided this feature's shape. A
  feature currently in BUILD in this repository carries the same requirement; a sixth route added now
  would fail that feature's requirement audit at ship time. This is why the export is a subcommand.

## Acceptance Criteria

- Reset the demo database, run the export for the first seeded account, and read the output: a header
  row naming the four columns, then the account's transactions newest first, amounts as bare signed
  integers.
- The first data row is the same transaction the page shows at the top of its table and the same
  element the programmatic listing returns first, with the same recorded time and the same amount in
  cents.
- The row count after the header equals the number of transactions the programmatic listing returns
  for that account.
- Run the export for the seeded account that has no transactions: it exits zero and emits the header
  row alone.
- Run the export for an account identifier that does not exist: it exits non-zero, its message names
  the identifier that was asked for, and standard output is completely empty.
- Run the export twice against the same database and compare: the two outputs are byte-identical.
- Run the export with no account identifier, and again with two: each fails with a message that states
  what the command expects.
- Ask the binary for an unknown subcommand: the message names all three valid subcommands.
- The server still exposes exactly five routes; the page and the programmatic listing are unchanged.
- The suite passes, passes again on a repeated run, completes under ten seconds, and the formatting and
  vetting gates report nothing. No float type or float formatting appears anywhere in the repository.

## Governing Decisions (conformance check)

Checked before any effort is spent, per the design-conformance rule. Every governing decision below is
`Status: Accepted` and **none is amended by this feature**.

| Decision | What it constrains | Conformance |
|---|---|---|
| `adr-2026-08-08-money-as-int64-cents.md` | Money is `int64` cents; no floats anywhere | **Conforms, and FR-3 restates it as an output requirement.** Amounts are copied from the row to the field as integers. The export performs no arithmetic on money at all. |
| `adr-2026-08-08-injected-clock.md` | Time is injected; only the system clock wrapper reads the wall clock | **Conforms.** The export reads stored recorded times and needs no clock. It adds no call site (NFR-2). |
| `adr-2026-08-08-store-interface-in-domain-package.md` | The store interface is declared in the domain package; the domain knows nothing of SQLite | **Conforms.** The export depends on the interface, not on the implementation (NFR-4). |
| `adr-2026-08-08-deterministic-transaction-ids-and-ordering.md` | Sequential zero-padded identifiers; newest-first is a total order tie-broken by identifier | **Conforms, and this feature depends on it.** FR-4's order requirement and FR-7's byte-identical requirement are both consequences of that total order; the export introduces no ordering of its own. |
| `adr-2026-08-08-sentinel-errors-for-domain-failures.md` | Domain failures are sentinel errors wrapped for identity comparison | **Conforms.** FR-5's failure is the existing account-not-found sentinel, wrapped with the requested identifier by the store. No new sentinel is introduced and no error string is compared by value. |
| `api-response-contract.md` | Recorded times are RFC 3339 in UTC; amounts are published as an integer cents field | **Conforms, and FR-2/FR-4 are written to match it.** The export reuses both conventions rather than inventing a second pair, which is what makes the surfaces agree. |
| `adr-2026-08-08-one-negotiated-posting-endpoint.md` | One posting endpoint, content-negotiated | **Not touched.** The export writes nothing and posts nothing. |
| `styleguide.md` | Page and stylesheet constraints | **Not touched.** No template or stylesheet change is in scope. |

No new ADR is authored. This feature introduces no seam, no dependency direction, and no schema
decision that the decisions above do not already govern; at Tier S `/architecture-review` is skipped
and there is nothing left for it to decide.

## Dependencies

Pre-existing constraints this feature works within, named as product reality rather than as chosen
mechanism:

- The stored transaction record already carries exactly the four values FR-2 asks for. Nothing must be
  derived, joined, or computed to produce a row.
- The newest-first listing is already a stable total order — recorded time descending, then identifier
  descending — and it is the same listing the page and the programmatic surface both read. FR-4's
  agreement requirement and FR-7's determinism requirement are both properties of that existing order
  rather than new work.
- Reading an account's transactions already fails with the account-not-found sentinel, naming the
  requested identifier, before any transaction row is read. FR-5 is therefore achievable without a new
  error and without any window in which a partial document could be written.
- The existing subcommands locate the database through one environment convention with a documented
  default. The export inherits that convention rather than introducing a second one.

## Assumption Ledger

Recorded because the operator was unavailable to confirm these interactively. Each is stated with its
confidence, its basis, its impact if wrong, and how to confirm it. **These are assumed, operator
unavailable — not operator-approved.** The authorization to spend the exports non-goal is the one item
here that is *not* an assumption: it is stated in writing by the operator in intake issue
`jstoup111/ledger-demo#14`.

| # | Assumption | Confidence | Basis | Impact if wrong | How to confirm |
|---|---|---|---|---|---|
| A1 | Tier **S** is correct, so `/architecture-diagram`, `/architecture-review`, `/conflict-check`, and `/coherence-check` are legitimately skipped and no ADR is written. | 90% | `inferred` from the signal table in `.docs/complexity/csv-export-single-account.md`, and consistent with the `size: S` label already on the originating intake issue. | The skipped DECIDE steps would have to run before landing. No requirement here changes; only the paperwork. | Operator confirms the tier. |
| A2 | The command word is `export` and it takes the account identifier as its single positional argument. | 80% | `inferred` — the intake issue's own hypothesis names `export <account-id>`, and it is the shortest form consistent with the two existing subcommands, which take no arguments. | The command word and argument shape change. Every FR still holds; only the spelling in the criteria and in the valid-command message changes. | Operator reads FR-1 and says the word. |
| A3 | The four column headers should be the field names the programmatic listing already publishes for those values, in the order FR-2 states. | 85% | `verified` that those four field names exist and are already published for exactly these values; `inferred` that reusing them is preferable to inventing spreadsheet-friendly titles. | Only the header row's text changes. FR-2's column set and order, and every other FR, are unaffected. | Operator reads one header row. |
| A4 | Emitting no document at all — not even a header — is the right failure for an unknown account, and a header-only document is the right success for an empty account. | 90% | `verified` that the intake issue states the unknown-account case must not produce an empty or headers-only file; the empty-account counterpart in FR-6 is `inferred` from that same distinction. | FR-5 and FR-6 would collapse into one behavior, losing the distinction between "no such account" and "no history". | Operator reads FR-5 and FR-6 together. |
| A5 | The standard library's default CSV conventions — its record separator and its automatic quoting of fields that need it — are acceptable as-is, since the intake issue puts quoting configuration out of scope. | 85% | `inferred` from the issue's explicit exclusion of quoting and escaping configuration, plus the determinism requirement, which the defaults satisfy. | Only the emitted punctuation changes for descriptions that contain a separator or a quote. No FR changes. | Operator opens one exported file in a spreadsheet. |

> **Amended 2026-08-13 by operator:** A1 is confirmed by the approved Tier S decision. A2 and A5 are
> superseded because the CLI delivery no longer exists. A3 and A4 are confirmed by approval of the
> browser-download amendment's FR-3 through FR-6. The original assumption table remains as history;
> it contains no pending input to the amended scope.

## Corrections to the Intake Issue's Claims

Recorded rather than silently adopted.

- The issue frames the sixth-route alternative as merely risky to sequence. It is **stronger than
  that**: the route-count constraint is written as a non-functional requirement of a feature currently
  in BUILD, so a sixth route would fail that feature's requirement audit outright rather than merely
  arrive at an awkward time. NFR-8 records the constraint on this feature directly so the decision does
  not depend on remembering another feature's paperwork.
- The issue lists "the recorded time" as a column without stating its format. Left unstated that would
  be a silent disagreement between the export and the two existing surfaces, so FR-4 pins the format to
  the one those surfaces already use rather than leaving it to the implementation.
- The issue says the export adds "no dependency". Confirmed by reading the module requirements: the
  single pinned requirement is unchanged (NFR-5).

## Open Questions

None blocking. The two a reviewer is most likely to raise are answered above rather than left open:
whether an ADR is needed (no — see the Governing Decisions table) and what the recorded time's format
is (FR-4 — the format both existing surfaces already use).
