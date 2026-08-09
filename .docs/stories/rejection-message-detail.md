# Stories — Rejection Messages Name the Offending Value

**Status:** Accepted

**Feature:** rejection-message-detail · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-08-09-rejection-message-detail.md` (Approved, FR-1 … FR-9)
**Constrained by:** `.docs/decisions/adr-2026-08-09-rejection-message-composition.md` (Accepted),
`.docs/decisions/api-response-contract.md` (Accepted, amended 2026-08-09),
`.docs/decisions/styleguide.md`

Two stories, one per surface, covering all nine FRs. Grouped by surface rather than one-per-FR because
FR-4 — the two surfaces must say the identical thing — is only assertable by owning both.

**Test ownership, stated once so the same rule is not asserted three times:**

| Subject | Owned by |
|---|---|
| The exact sentence for every rule, and every degradation of a tampered value | Story 1 |
| That the programmatic message equals the page's message for the same rejection, and that no identifier or status moved | Story 2 |
| That a script-bearing value is inert | Both — the surfaces escape differently, so each owns its own proof |

Story 2 consumes Story 1's sentences rather than restating them.

## Negative-path categories evaluated

Every category is evaluated explicitly. Most do not apply, and saying so is more useful than inventing
a scenario.

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — this feature's whole subject is what a rejection says, plus a third client-supplied query value that can be tampered with |
| Data integrity | **Yes, as an invariant to preserve** — a rejection still records nothing; naming a value must not change that |
| Model-level immutability | No — no model is touched; no field is read or written |
| Invariant side-effect on alternate branches | **Yes** — the posting endpoint still has two response branches, and FR-4 requires them to agree character for character |
| Dependency unavailability | **Yes, narrowly** — a message that names the balance needs the balance derived; when it cannot be (no selected account), the sentence must degrade rather than break |
| Auth / permission failures | No — auth is an explicit non-goal; there is no principal |
| Timeouts & network errors | No — NFR-4 forbids network calls; nothing to time out |
| Resource exhaustion | **Yes, bounded** — an arbitrarily long value in the address is the resource risk, and the 32-character cap plus suppression is the answer |
| Partial failure & rollback | No — composing a sentence has no steps to roll back |
| Concurrent access | No — no new shared state; message composition is pure |
| Cascade deletion effects | No — nothing is deleted |
| Exception class hierarchy | No — Go sentinel errors compared with `errors.Is`, unchanged by this feature |
| Dedup / idempotency key analysis | No — there is no dedup or idempotency criterion anywhere in this spec, and introducing one is an explicit non-goal |

---

**Status:** Accepted

## Story 1: The panel on stage names the value that was rejected

**Requirement:** FR-1, FR-2, FR-5, FR-6, FR-7 (page surface), FR-8

As a presenter, I want the rejection panel to name the value that was just rejected, so the audience at
the back of the room reads what went wrong instead of taking my word for it.

### Acceptance Criteria

#### Happy Path

- Given a seeded database and `acct-1` with a balance of `$1,283.50`, when the presenter submits the
  form with amount `12.3.4` and any valid description, then the page's error panel reads
  `Amount is malformed. Submitted: 12.3.4.`
- Given the same account, when the presenter submits amount `0.00` with a valid description, then the
  panel reads `Amount must not be zero. Submitted: 0.00.`
- Given the same account, when the presenter submits a valid amount with a description of exactly 187
  characters, then the panel reads
  `Description is too long. Submitted: 187 characters; the limit is 140.` and the description itself
  appears nowhere in the panel.
- Given the same account, when the presenter submits amount `-2000.00` with a valid description, then
  the panel reads `Balance would go negative. Posting -$2,000.00 against a balance of $1,283.50.` —
  both figures formatted the way the page formats a balance, with a thousands separator and two decimal
  places.
- Given a submission whose amount would push the balance past the 64-bit cents ceiling, when it is
  rejected, then the panel reads `Balance would overflow.` followed by
  `Posting {amount} against a balance of {balance}.` with both figures formatted the same way.
- Given a submission against an account id that does not exist, when the page renders the rejection,
  then the panel reads `Account not found. Requested: {the id that was submitted}.`
- Given a submission with a blank description, when it is rejected, then the panel reads
  `Description must not be empty.` — unchanged from today, because emptiness has no value to name.
- Given any of the above, when the page renders, then the panel still sits after the balance and
  immediately before the form, with nothing between the panel and the form.
- Given any of the above, when the page renders, then the response is `200`, the account's balance is
  unchanged, and the transaction list contains no new row.

#### Negative Paths

- Given the address `/?account=acct-1&error=amount_malformed` with no value carried at all, when the
  page renders, then the panel reads exactly `Amount is malformed.` — a complete sentence, with no
  trailing `Submitted:` and no empty panel.
- Given a carried value of more than 32 characters for `amount_malformed`, when the page renders, then
  the panel reads exactly `Amount is malformed.` and the over-long value appears nowhere in the
  response body.
- Given a carried value containing a control character (for example a percent-encoded newline), when
  the page renders, then the panel reads exactly `Amount is malformed.` and the control character
  reaches no output.
- Given `error=description_too_long` with a carried value that is not 1–6 decimal digits (`abc`, an
  empty string, a 30-digit number), or that is a number not greater than 140 (`3`, `140`), when the
  page renders, then the panel reads exactly `Description is too long.`
- Given `error=balance_would_go_negative` with a carried value that is not an integer (`12.50`, `1e9`,
  `abc`) or does not fit a 64-bit signed integer, when the page renders, then the panel reads exactly
  `Balance would go negative.`
- Given `error=description_empty` with a carried value present anyway, when the page renders, then the
  panel reads exactly `Description must not be empty.` and the carried value appears nowhere in the
  response body.
- Given `error=not_a_real_code` with any carried value, when the page renders, then the panel is
  non-empty and generic — `Unable to post transaction.` — and neither the unrecognized code nor the
  carried value appears anywhere in the response body. Two different unrecognized codes produce the
  identical panel.
- Given `error=amount_malformed` and a carried value of `<script>alert(1)</script>`, when the page
  renders, then the value appears in the panel as escaped visible text, the body contains no raw
  `<script` sequence, and no attribute or element is created by the value.
- Given `/?account=acct-nope&error=balance_would_go_negative` with a valid carried amount — an unknown
  account, so no balance can be derived — when the page renders, then the panel reads exactly
  `Balance would go negative.` with no dangling `Posting` clause.
- Given a database seeded with zero accounts and the address `/?error=amount_zero&detail=0.00`, when
  the page renders, then the panel reads exactly `Amount must not be zero. Submitted: 0.00.` if the
  value validates and `Amount must not be zero.` otherwise — and in neither case is a posting form
  rendered.
- Given the amount field is submitted with a value longer than 32 characters, when the rejection
  redirect is built, then the redirect carries no value for it at all — not a truncated one.

### Done When

- [ ] A table-driven test covers all seven codes' page messages, asserting the full expected sentence
      by equality against the panel's text, not by substring.
- [ ] A table-driven test covers every degradation in the Negative Paths above, asserting the plain
      sentence for the code and asserting the tampered value is absent from the body.
- [ ] The existing assertion that the panel sits between the balance and the form still passes,
      unmodified.
- [ ] The existing assertions that an unrecognized code renders one static non-empty generic panel and
      is not echoed still pass, unmodified.
- [ ] A test asserts a script-bearing carried value renders escaped and produces no raw `<script`
      sequence.
- [ ] A test asserts the redirect built for an over-long submitted amount carries no value parameter.
- [ ] A rejection still records nothing: the transaction count before and after is equal.
- [ ] `grep` finds no second copy of any rejection sentence — the page no longer holds its own message
      table.
- [ ] No floating-point type and no float parsing appears anywhere on the message path.

---

**Status:** Accepted

## Story 2: A programmatic client reads the same sentence, against unmoved identifiers

**Requirement:** FR-3, FR-4, FR-7 (programmatic surface), FR-9

As a presenter showing `curl` next to the browser, I want the programmatic rejection to say exactly
what the projector says while its machine-readable identifier stays put, so nothing already written
against this API breaks and the two windows do not contradict each other on stage.

### Acceptance Criteria

#### Happy Path

- Given a JSON submission with amount `12.3.4`, when it is rejected, then the response is `400` with
  `Content-Type: application/json; charset=utf-8` and a body of exactly
  `{"error":{"code":"amount_malformed","message":"Amount is malformed. Submitted: 12.3.4."}}`.
- Given each of the seven rejection cases submitted as JSON, when each is rejected, then the `code` and
  HTTP status are exactly: `account_not_found`/`404`, `amount_zero`/`400`, `description_empty`/`400`,
  `description_too_long`/`400`, `amount_malformed`/`400`, `balance_would_go_negative`/`400`,
  `balance_overflow`/`400`.
- Given each of the seven cases, when the same input is submitted once as a form post and once as
  JSON, then the sentence the page renders and the `message` in the JSON body are **equal, character
  for character** — asserted by comparing the two captured strings to each other, not by comparing each
  to a literal.
- Given a JSON submission rejected for taking the balance below zero, when the response is read, then
  the `message` names both the attempted amount and the account's current balance, and the balance
  matches what `GET /api/accounts` reports for that account at that moment.
- Given a rejected JSON submission, when `GET /api/accounts/{id}/transactions` is read afterwards, then
  it returns the same rows as before, and the account's `balance_cents` is unchanged.
- Given the error body of any rejection, when its keys are enumerated, then `error` has exactly the two
  keys `code` and `message` — no third key was added to carry the value.

#### Negative Paths

- Given a JSON submission of amount `<script>alert(1)</script>`, when it is rejected, then the body
  parses as valid JSON, `code` is `amount_malformed`, the `message` contains the value in a form that
  cannot terminate the JSON string, and the raw byte sequence `<script` does not appear in the body.
- Given a JSON body that is not parseable at all, when it is rejected, then the response is `400` with
  code `amount_malformed` and message exactly `Amount is malformed.` — there was no submitted value to
  name, so the sentence stands alone.
- Given a JSON body containing two JSON values, when it is rejected, then the response is `400`,
  `amount_malformed`, message exactly `Amount is malformed.`
- Given a JSON submission with a blank description, when it is rejected, then the message is exactly
  `Description must not be empty.`
- Given an internal failure with no mapped sentinel, when it is written, then the response is `500`
  with code `internal_error` and message `Unable to post transaction.` — unchanged by this feature.
- Given a form submission, when it is rejected, then the response is still `303 See Other` with a
  `Location` whose `error` value is the same identifier the JSON branch would have returned, and the
  account id in that `Location` is escaped.
- Given the request method or path does not match a route, when the request is made, then the response
  is `405` or `404` with an empty body, and the route count is still exactly five.

### Done When

- [ ] A table-driven test asserts all seven `(code, HTTP status)` pairs by their literal identifier
      strings, so a renamed identifier fails the suite.
- [ ] A table-driven test asserts, per rejection case, that the page's rendered sentence and the JSON
      `message` are equal to each other.
- [ ] A test asserts the error body has exactly the keys `error.code` and `error.message`.
- [ ] A test asserts the script-bearing amount produces a parseable JSON body with no raw `<script`
      sequence.
- [ ] A test asserts the unparseable-body and two-values cases yield the plain
      `Amount is malformed.` sentence.
- [ ] A test asserts the balance named in a `balance_would_go_negative` message equals the
      `balance_cents` that `GET /api/accounts` reports.
- [ ] A rejection records nothing, asserted on both branches.
- [ ] The route count is asserted to be exactly five.
- [ ] `make test` passes under 10 seconds with `-count=2`; `gofmt -l .` is empty and `go vet ./...` is
      clean.
