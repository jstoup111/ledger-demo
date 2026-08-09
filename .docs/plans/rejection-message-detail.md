# Implementation Plan: Rejection Messages Name the Offending Value

**Date:** 2026-08-09
**Design:** [2026-08-09-rejection-message-detail.md](../specs/2026-08-09-rejection-message-detail.md)
**Stories:** [rejection-message-detail.md](../stories/rejection-message-detail.md)
**Decision:** [adr-2026-08-09-rejection-message-composition.md](../decisions/adr-2026-08-09-rejection-message-composition.md)
(Accepted)
**Contract:** [api-response-contract.md](../decisions/api-response-contract.md) (Accepted, amended
2026-08-09 by this DECIDE pass)
**Conflict check:** skipped — Tier S (`.docs/complexity/rejection-message-detail.md`)
**Architecture review:** skipped — Tier S. The one load-bearing decision is recorded as the Accepted
ADR above.
**Coherence check:** skipped — Tier S. The Coverage Mapping table below carries the traceability.
**Tier:** S (`.docs/complexity/rejection-message-detail.md`)

> **Filename note:** this plan is `rejection-message-detail.md`, not the skill's default
> `YYYY-MM-DD-<feature>.md`, because the plan stem must match
> `.docs/complexity/rejection-message-detail.md` for the daemon to resolve the complexity tier at
> build time.

## Summary

Enriches the seven rejection messages so each names the value that was rejected, in **19 tasks across
four batches**. Every task is a TDD cycle at 2–5 minute granularity.

## Technical Approach

**The domain is not touched.** `internal/ledger` keeps exactly the sentinels, rules, and signatures it
has. Everything this feature needs is already in the HTTP handler's hands at the moment of rejection:
the submitted amount string, the submitted description (so its character count), the parsed amount in
cents, the account id, and the store (so the derived balance). No new argument crosses into the domain
and no sentinel is added, renamed, or removed. If a task in this plan finds itself editing
`internal/ledger`, it has gone wrong.

**Two mappings, each in one place.** Sentinel → identifier stays exactly where it is, in
`internal/httpapi/errors.go#codeFor`, per `adr-2026-08-08-sentinel-errors-for-domain-failures.md`. This
feature adds a **second, separate** step — identifier + carried value → sentence — in a new
`internal/httpapi/message.go`. The first mapping is not scattered and not moved; the second is created
already-single rather than created twice. `router.go#pageErrorMessage`'s duplicate message table is
**deleted**, not updated: it is the drift hazard NFR-7 names, and it exists today.

**The composer is pure, so almost all of the suite needs no HTTP.** Its whole signature is
(identifier, carried value, account id, balance-if-known) → sentence. Batch A builds and fully covers
it with plain table-driven unit tests — no recorder, no store, no router. Batches B and C then wire the
two surfaces to it, and their tests only have to prove the wiring and the URL round-trip, not the
sentences again.

**The enriched sentence is the old sentence plus one more.** So `Amount is malformed.` remains a prefix
of `Amount is malformed. Submitted: 12.3.4.` This is what makes FR-5's fallback trivially correct (drop
the second sentence) and it is why existing tests asserting today's text keep passing. Assertions for
the new text are written as **equality** against the panel or the `message`, not `strings.Contains`, so
a missing or duplicated clause fails rather than silently passing on the prefix.

**Money stays integer cents.** The two balance messages carry the attempted amount as decimal integer
cents and render it through the existing `router.go#formatDollars`. `strconv.ParseInt` reads it back.
There is no `float64`, no `float32`, and no `ParseFloat` anywhere on this path — a grep for those is a
Done-When item.

**Everything carried in the address is untrusted on read.** Including on the programmatic branch, which
composes from values it holds directly — the same validator runs there, so there is one code path and
no "trusted caller" variant to keep in agreement. Validation failure means the plain sentence, whole;
never a partial sentence, never an empty panel.

**Test ownership, to avoid asserting the same rule three times** (the discipline base-ledger's
conflict-check imposed, applied here without the step): Batch A owns every sentence and every
degradation rule. Batch B and Batch C own only that their surface reaches the composer and that the
value survives the round trip. Task 17 owns FR-4 and asserts *equality between the two surfaces* — it
restates no sentence of its own. Task 18 owns the frozen-contract invariants.

## Prerequisites

None. No new dependency, no configuration change, no schema change, no new route. The suite is green at
this branch point (`go test ./...` across all seven packages, verified 2026-08-09).

## BUILD-entry artifacts not authored by any plan task

| Artifact | Owner | Constrains |
|---|---|---|
| `test/acceptance/ledger_acceptance_test.go` (additions) | `/writing-system-tests` (BUILD entry, before implementation) | Stories 1–2; the RED targets for Tasks 16–18 |

Acceptance specs are authored at BUILD entry by the acceptance-spec step, not by an implementation
task, and ownership is not reassigned to one. Existing acceptance cases in that file that assert
today's message text remain valid — the enriched sentence contains them — and any that assert a
message by **equality** must be updated by that step to the FR-1 text.

---

## Batch A — the message composer (pure, no HTTP)

### Task 1: Free-text carried value validator
**Story:** Story 1 (degradation of a tampered value)
**Type:** infrastructure

**Steps:**
1. Write failing test, table-driven: accepted — `0.00`, `12.3.4`, `abc`, a 32-character value, a value
   containing `<script>alert(1)</script>`. Rejected — empty string, a 33-character value, a value
   containing `\n`, `\r`, `\t`, or `\x00`.
2. Verify test fails (RED)
3. Implement a validator in `internal/httpapi/message.go` returning the value and an ok flag: non-empty,
   `utf8.RuneCountInString` at most 32, and no rune satisfying `unicode.IsControl`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: validate a free-text carried rejection value"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** `internal/httpapi/message.go#messageFor` (consumed by the `amount_zero` and
`amount_malformed` branches)
**Dependencies:** none

### Task 2: Character-count carried value validator
**Story:** Story 1 (degradation of a tampered count)
**Type:** infrastructure

**Steps:**
1. Write failing test, table-driven: accepted — `141`, `187`, `999999`. Rejected — empty, `abc`, `3`,
   `140`, `-5`, `1.5`, a 7-digit value, a 30-digit value, `0141`.
2. Verify test fails (RED)
3. Implement: 1–6 ASCII decimal digits with no leading zero, parsed with `strconv.Atoi`, accepted only
   when strictly greater than 140.
4. Verify test passes (GREEN)
5. Commit: "httpapi: validate a carried description character count"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** `internal/httpapi/message.go#messageFor` (the `description_too_long` branch)
**Dependencies:** none

### Task 3: Integer-cents carried value validator
**Story:** Story 1 (degradation of a tampered amount)
**Type:** infrastructure

**Steps:**
1. Write failing test, table-driven: accepted — `1`, `-1`, `200000`, `-200000`,
   `9223372036854775807`, `-9223372036854775808`. Rejected — empty, `abc`, `12.50`, `1e9`, `+5`, `-`,
   `9223372036854775808`, a 40-digit value.
2. Verify test fails (RED)
3. Implement: optional single leading `-`, then 1–19 ASCII decimal digits, parsed with
   `strconv.ParseInt(value, 10, 64)`. No `ParseFloat`, no float type.
4. Verify test passes (GREEN)
5. Commit: "httpapi: validate a carried amount in integer cents"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** `internal/httpapi/message.go#messageFor` (the `balance_would_go_negative` and
`balance_overflow` branches)
**Dependencies:** none

### Task 4: The seven plain sentences, and the generic fallback
**Story:** Story 1 (FR-2, FR-6) · **base-ledger Condition C2** is inherited here
**Type:** happy-path

**Steps:**
1. Write failing test, table-driven with **no** carried value: each of the seven identifiers returns
   exactly the sentence it returns today — `Account not found.`, `Amount must not be zero.`,
   `Description must not be empty.`, `Description is too long.`, `Amount is malformed.`,
   `Balance would go negative.`, `Balance would overflow.` An unrecognized identifier returns exactly
   `Unable to post transaction.`, and two different unrecognized identifiers return the identical
   string. An **empty** identifier returns the empty string, so a page with no rejection renders no
   panel.
2. Verify test fails (RED)
3. Implement `messageFor` in `internal/httpapi/message.go` taking an identifier and a small context
   struct (carried value, account id, balance, balance-known flag), returning the plain sentence.
4. Verify test passes (GREEN)
5. Commit: "httpapi: compose rejection sentences in one place"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** `internal/httpapi/errors.go#writeJSONError`, `internal/httpapi/router.go#handlePage`
**Dependencies:** none

### Task 5: `account_not_found` names the requested id
**Story:** Story 1 (FR-1)
**Type:** happy-path

**Steps:**
1. Write failing test: with account id `acct-9`, the message equals
   `Account not found. Requested: acct-9.` With an empty account id it equals `Account not found.`
   A carried value is ignored for this identifier — supplying one changes nothing.
2. Verify test fails (RED)
3. Implement the `account_not_found` branch, sourcing the id from the context and appending the second
   sentence only when it is non-empty.
4. Verify test passes (GREEN)
5. Commit: "httpapi: name the requested account in the not-found message"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** same as Task 4
**Dependencies:** 4

### Task 6: `amount_zero` and `amount_malformed` name the submitted amount
**Story:** Story 1 (FR-1)
**Type:** happy-path

**Steps:**
1. Write failing test, table-driven: `amount_zero` with `0.00` equals
   `Amount must not be zero. Submitted: 0.00.`; `amount_malformed` with `12.3.4` equals
   `Amount is malformed. Submitted: 12.3.4.` Each with a value the Task 1 validator rejects (empty,
   33 characters, containing `\n`) equals the plain sentence, and the rejected value does not appear in
   the returned string at all.
2. Verify test fails (RED)
3. Implement both branches through the Task 1 validator.
4. Verify test passes (GREEN)
5. Commit: "httpapi: name the submitted amount in zero and malformed messages"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** same as Task 4
**Dependencies:** 1, 4

### Task 7: `description_too_long` names the character count
**Story:** Story 1 (FR-1)
**Type:** happy-path

**Steps:**
1. Write failing test: with `187` the message equals
   `Description is too long. Submitted: 187 characters; the limit is 140.` With a value the Task 2
   validator rejects (`abc`, `3`, `140`, empty, 7 digits) it equals `Description is too long.` and the
   rejected value does not appear in the returned string.
2. Verify test fails (RED)
3. Implement the branch through the Task 2 validator, with `140` written as the literal limit the
   domain enforces.
4. Verify test passes (GREEN)
5. Commit: "httpapi: name the description length against the limit"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** same as Task 4
**Dependencies:** 2, 4

### Task 8: The two balance messages name the amount and the balance
**Story:** Story 1 (FR-1)
**Type:** happy-path

**Steps:**
1. Write failing test, table-driven: `balance_would_go_negative` with carried `-200000` and a known
   balance of `128350` equals
   `Balance would go negative. Posting -$2,000.00 against a balance of $1,283.50.`;
   `balance_overflow` with the same inputs produces the matching `Balance would overflow. Posting …`
   sentence. When the balance is **not** known, each equals its plain sentence with no dangling
   `Posting` clause. When the carried value fails the Task 3 validator, each equals its plain sentence.
2. Verify test fails (RED)
3. Implement both branches through the Task 3 validator, rendering both figures with the existing
   `formatDollars`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: name the attempted amount against the balance"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** same as Task 4
**Dependencies:** 3, 4

### Task 9: A carried value is ignored where it has no meaning
**Story:** Story 1 (FR-5, FR-6)
**Type:** negative-path

**Steps:**
1. Write failing test, table-driven: for `description_empty` and for two different unrecognized
   identifiers, a carried value of `0.00`, of `<script>alert(1)</script>`, and of a 500-character
   string each produce the identifier's plain sentence, and in every case the carried value does not
   appear anywhere in the returned string. Both unrecognized identifiers still return the identical
   string.
2. Verify test fails (RED)
3. Implement by dispatching on the identifier first and only then consulting the carried value, so an
   identifier with no defined value never reads one.
4. Verify test passes (GREEN)
5. Commit: "httpapi: ignore a carried value where the rule defines none"

**Files:** `internal/httpapi/message.go`, `internal/httpapi/message_test.go`
**Wired-into:** same as Task 4
**Dependencies:** 4

> **Gate after Batch A:** `go vet ./...` clean; `gofmt -l .` empty; every FR-1 sentence and every FR-5
> degradation is covered by a unit test with no HTTP involved; `grep -n 'float\|ParseFloat'` over
> `internal/httpapi/message.go` returns nothing.

---

## Batch B — the programmatic surface

### Task 10: `codeFor` sheds its message; `writeJSONError` composes
**Story:** Story 2 (FR-3, FR-4)
**Type:** infrastructure

**Steps:**
1. Write failing test: for each of the seven sentinels, `codeFor` still returns the same
   `(status, identifier)` pair asserted by its **literal identifier string**, and
   `writeJSONError` writes a body whose `message` comes from `messageFor`. An unmapped error still
   yields `500` / `internal_error` / `Unable to post transaction.`
2. Verify test fails (RED)
3. Remove the `message` field from `codedError`, keep `status` and `code` and keep the zero-status
   "is this mapped?" behaviour that `handleAccountTransactions` and `handlePostTransaction` rely on,
   and have `writeJSONError` call `messageFor`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: compose the JSON error message through one composer"

**Files:** `internal/httpapi/errors.go`, `internal/httpapi/errors_test.go`
**Wired-into:** `internal/httpapi/router.go#handlePostTransaction`,
`internal/httpapi/router.go#handleAccountTransactions` (both call `writeJSONError`)
**Dependencies:** 4

### Task 11: The posting handler builds the rejection context
**Story:** Story 2 (FR-1 on the programmatic surface)
**Type:** happy-path

**Steps:**
1. Write failing test: a JSON post of amount `12.3.4` returns `400`, identifier `amount_malformed`, and
   body exactly
   `{"error":{"code":"amount_malformed","message":"Amount is malformed. Submitted: 12.3.4."}}`. A JSON
   post of `0.00` names it likewise. A JSON post with a 187-character description returns the FR-1
   count sentence. A post against a missing account names the requested id. A blank description returns
   exactly `Description must not be empty.`
2. Verify test fails (RED)
3. In `handlePostTransaction`, build the context at the point of rejection from what it already holds:
   the submitted amount string, the submitted description's `utf8.RuneCountInString`, the parsed amount
   in cents, and the path's account id. Pass it through to `writeJSONError`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: carry the offending value into the JSON rejection"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/errors.go`,
`internal/httpapi/router_test.go`
**Wired-into:** `internal/httpapi/message.go#messageFor` (the context it consumes)
**Dependencies:** 6, 7, 10

### Task 12: The balance is derived on the programmatic rejection path
**Story:** Story 2 (FR-1, FR-4)
**Type:** happy-path

**Steps:**
1. Write failing test: against a fixture account whose derived balance is `128350` cents, a JSON post
   of `-2000.00` returns `400`, `balance_would_go_negative`, and message
   `Balance would go negative. Posting -$2,000.00 against a balance of $1,283.50.` The balance in that
   sentence equals the `balance_cents` that `GET /api/accounts` reports for the account. A store whose
   balance read fails yields the plain `Balance would go negative.` sentence and still `400`.
2. Verify test fails (RED)
3. On the two balance identifiers only, derive the balance with `ledger.Balance(store, accountID)` on
   the rejection path and put it in the context; on error leave the balance unknown. No change to the
   success path.
4. Verify test passes (GREEN)
5. Commit: "httpapi: derive the balance for a balance-rejection message"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 8, 11

### Task 13: A script-bearing amount is inert in the JSON body
**Story:** Story 2 (FR-7, programmatic surface)
**Type:** negative-path

**Steps:**
1. Write failing test: a JSON post of amount `<script>alert(1)</script>` returns `400`, the body
   unmarshals as valid JSON into the two-key error shape, its `message` names the value, and the raw
   byte sequence `<script` does **not** appear in the response body. An unparseable body and a body
   holding two JSON values each return `400` / `amount_malformed` / exactly `Amount is malformed.`
   A key enumeration of `error` yields exactly `code` and `message`.
2. Verify test fails (RED)
3. No production change is expected — the assertion pins that the standard library's default JSON
   string escaping is what makes this safe, and that no bypass (raw byte writing, pre-escaping,
   `SetEscapeHTML(false)`) was introduced. Make whatever minimal change the RED reveals.
4. Verify test passes (GREEN)
5. Commit: "httpapi: prove a script-bearing amount is inert in the error body"

**Files:** `internal/httpapi/errors_test.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 10 (assertions only — consumes existing surface)
**Dependencies:** 11, 12

> **Gate after Batch B:** the programmatic surface is complete and named; every identifier and status
> in FR-3's table is asserted by literal string; the error body still has exactly two keys.

---

## Batch C — the page surface

### Task 14: The rejection redirect carries the offending value
**Story:** Story 1 (FR-1), Story 2 (FR-9)
**Type:** happy-path

**Steps:**
1. Write failing test, table-driven over the seven rules: the form-encoded rejection responds `303`
   with `Location` equal to `/?account={escaped id}&error={identifier}` plus
   `&detail={percent-encoded value}` for exactly the five identifiers that define one — the submitted
   amount for `amount_zero` and `amount_malformed`, the decimal character count for
   `description_too_long`, the decimal integer cents for the two balance rules — and **no** `detail`
   for `account_not_found` and `description_empty`. A submitted amount longer than 32 characters
   produces a `Location` with no `detail` at all, not a truncated one. The account id is still escaped.
2. Verify test fails (RED)
3. Extend `writePostError` to append the parameter from the same context Task 11 built, gated by the
   same validators, `url.QueryEscape`d.
4. Verify test passes (GREEN)
5. Commit: "httpapi: carry the offending value on the rejection redirect"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** `internal/httpapi/router.go#handlePage` (reads the parameter back on the next request)
**Dependencies:** 6, 7, 8, 11

### Task 15: The page composes through the one composer; the duplicate table is deleted
**Story:** Story 1 (FR-1, FR-2, FR-8)
**Type:** happy-path

**Steps:**
1. Write failing test: `GET /?account=acct-1&error=amount_malformed&detail=12.3.4` renders a panel whose
   text equals `Amount is malformed. Submitted: 12.3.4.` The same for the count and the two balance
   rules, the latter using the fixture's derived balance. `GET /?account=acct-nope` renders
   `Account not found. Requested: acct-nope.` The panel still sits after the balance element and
   immediately before the `<form>`, with nothing between them. A page with no `error` parameter renders
   no panel.
2. Verify test fails (RED)
3. Delete `pageErrorMessage` and its table. Compose `pageData.ErrorMessage` with `messageFor`, passing
   the `error` value, the `detail` value, the requested account id, and the selected account's derived
   balance. Order the handler so the balance is derived before the message is composed on the
   selected-account branch; leave the balance unknown on the unknown-account and zero-account branches.
4. Verify test passes (GREEN)
5. Commit: "httpapi: render the page rejection through the one composer"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** `web/index.html.tmpl` (`{{.ErrorMessage}}`, unchanged markup)
**Dependencies:** 5, 9, 14

### Task 16: The page degrades and escapes
**Story:** Story 1 (FR-5, FR-6, FR-7 page surface)
**Type:** negative-path

**Steps:**
1. Write failing test, table-driven over tampered addresses: `detail` absent, 33 characters, containing
   a percent-encoded newline, `detail=abc` under `description_too_long`, `detail=3` under
   `description_too_long`, `detail=12.50` under `balance_would_go_negative`, `detail` supplied under
   `description_empty`, and `detail` supplied under `error=not_a_real_code`. Each renders the plain
   sentence for its identifier — or the generic sentence for the unrecognized one — and in every case
   the tampered value appears nowhere in the response body. Separately,
   `detail=%3Cscript%3Ealert%281%29%3C%2Fscript%3E` under `amount_malformed` renders the value as
   escaped visible text with no raw `<script` sequence in the body. Finally,
   `/?account=acct-nope&error=balance_would_go_negative&detail=-200000` renders exactly
   `Balance would go negative.` — no balance is derivable, so no `Posting` clause.
2. Verify test fails (RED)
3. Make whatever minimal change the RED reveals; the intended production behaviour is already in place
   from Batch A and Task 15.
4. Verify test passes (GREEN)
5. Commit: "httpapi: prove the page degrades and escapes a tampered rejection value"

**Files:** `internal/httpapi/router_test.go`
**Wired-into:** same as Task 15 (assertions only — consumes existing surface)
**Dependencies:** 15

### Task 17: The two surfaces say the identical thing
**Story:** Story 2 (FR-4)
**Type:** happy-path

**Steps:**
1. Write failing test, table-driven over all seven rules: submit the same input twice — once
   form-encoded, once as JSON — follow the form branch's `303` to the page, extract the panel's text,
   unmarshal the JSON branch's `message`, and assert the **two captured strings are equal to each
   other**. No expected literal appears in this test; it consumes the sentences the earlier tasks
   fixed.
2. Verify test fails (RED)
3. Make whatever minimal change the RED reveals. If the two disagree, the disagreement is the bug and
   the fix belongs in the composer or in whichever surface skipped it — never in this test's
   expectations.
4. Verify test passes (GREEN)
5. Commit: "httpapi: assert page and JSON rejections read identically"

**Files:** `internal/httpapi/router_test.go`
**Wired-into:** same as Task 15 (assertions only — consumes existing surface)
**Dependencies:** 13, 15

> **Gate after Batch C:** both surfaces are named and provably identical; the panel is still between
> the balance and the form; `grep -rn 'pageErrorMessage'` returns nothing.

---

## Batch D — invariants and documentation

### Task 18: The frozen contract, the route count, and nothing recorded
**Story:** Story 1, Story 2 (FR-3, FR-9)
**Type:** negative-path

**Steps:**
1. Write failing test: a table of the seven `(identifier, HTTP status)` pairs asserted by literal
   identifier string, so a rename fails the suite; the router serves exactly five routes and an
   unmatched method or path still returns `405`/`404` with an empty body; and for every one of the
   seven rules, on both branches, the account's transaction count and derived balance are identical
   before and after the rejection.
2. Verify test fails (RED)
3. Make whatever minimal change the RED reveals; these are invariants the feature must not have
   disturbed, not new behaviour.
4. Verify test passes (GREEN)
5. Commit: "httpapi: pin the frozen identifiers, route count, and nothing-recorded invariant"

**Files:** `internal/httpapi/router_test.go`, `internal/httpapi/errors_test.go`
**Wired-into:** same as Task 10 (assertions only — consumes existing surface)
**Dependencies:** 10, 15

### Task 19: Document the rejection messages
**Story:** Story 1, Story 2 (the harness "docs track features" rule)
**Type:** infrastructure

**Steps:**
1. Add a short **Rejections** section to `README.md` giving the seven identifiers, the sentence shape
   `<rule sentence>. <value sentence>`, one worked example, and the one-line statement that the
   identifiers are a frozen contract while the message is not.
2. Correct the stale first sentence of the README's **Status** section, which still reads "Scaffold
   only" and claims `internal/ledger`, `internal/store`, and `internal/clock` are "deliberately empty".
   That was true before base-ledger shipped and would flatly contradict the new section. Leave the
   non-goals paragraph that follows it **verbatim** — it is the load-bearing part of that section and
   is still exactly correct.
3. Verification is documentation-only: no Go file changes, and `make test` still passes.
4. Commit: "docs: document the rejection message shape and refresh the status note"

**Files:** `README.md`
**Wired-into:** none (documentation — no production surface)
**Dependencies:** 17, 18

> **Gate after Batch D:** `make test` passes under 10 seconds with `-count=2`; `gofmt -l .` is empty;
> `go vet ./...` is clean; `grep -rn 'float64\|float32\|ParseFloat' internal/ web/ cmd/` returns
> nothing; the README documents what a rejection now says.

---

## Checkpoints

- **After Task 9** — every sentence and every degradation rule is proven with no HTTP involved. If
  FR-1's copy is wrong, this is the cheapest point to find out.
- **After Task 13** — the programmatic surface is complete, named, and inert.
- **After Task 17** — the feature is demonstrable end to end and the two surfaces are provably in
  agreement. This is the point a presenter can rehearse against.

## Task Dependency Graph

```
1 ─┐
2 ─┼─ (validators, independent)
3 ─┘
4 ─┬─ 5 ──────────────┐
   ├─ 6 (with 1) ──┐  │
   ├─ 7 (with 2) ──┤  │
   ├─ 8 (with 3) ──┤  │
   ├─ 9 ───────────┼──┤
   └─ 10 ──────────┤  │
                   ├─ 11 ─┬─ 12 ─┬─ 13 ─────────┐
                   │      │      │              │
                   └─ 14 (with 6,7,8,11) ─┐     │
                                          │     │
                              5, 9, 14 ─── 15 ─┬─ 16
                                               ├─ 17 (with 13)
                                               └─ 18 (with 10)
                                        17, 18 ─── 19
```

Acyclic. Tasks 1–4 have no dependencies and are the entry frontier.

## Coverage Mapping

| Story | Requirements | Tasks |
|---|---|---|
| 1 — the panel names the value | FR-1, FR-2, FR-5, FR-6, FR-7 (page), FR-8 | 1, 2, 3, 4, 5, 6, 7, 8, 9, 14, 15, 16, 19 |
| 2 — the programmatic surface agrees | FR-3, FR-4, FR-7 (JSON), FR-9 | 10, 11, 12, 13, 17, 18, 19 |

| Requirement | Covered by |
|---|---|
| FR-1 (every message names its value) | 5, 6, 7, 8, 11, 12, 15 |
| FR-2 (additive sentence; empty description unchanged) | 4, 9 |
| FR-3 (identifiers and statuses frozen) | 10, 18 |
| FR-4 (page and JSON identical) | 17 |
| FR-5 (degrade to the plain sentence) | 1, 2, 3, 6, 7, 8, 16 |
| FR-6 (unrecognized identifier stays generic, not echoed) | 4, 9, 16 |
| FR-7 (echoed value is inert) | 13 (JSON), 16 (page) |
| FR-8 (panel placement unchanged) | 15 |
| FR-9 (nothing else about a rejection changes) | 14, 18 |
| NFR-2 (no float) | Batch A gate, Batch D gate |
| NFR-5 / NFR-6 (suite and lint) | Batch D gate |
| NFR-7 (one source of message text) | 10, 15 (`pageErrorMessage` deleted) |
| NFR-8 (five routes) | 18 |

Every happy-path and negative-path criterion in both stories maps to at least one task. Negative paths
are their own tasks (9, 13, 16, 18), never folded into a cleanup task. There is **no terminal
catch-all validation task** — the batch gates plus `/manual-test` and `/prd-audit` own completed-feature
validation.

## Non-Goals Check

Checked task by task against the PRD's Non-Goals and `CLAUDE.md`:

- No task adds duplicate or double-charge detection, an idempotency key, a dedup window, or any
  uniqueness constraint. **No task touches the schema at all.**
- No task computes a shortfall, a fee, or a percentage. The two balance messages state the attempted
  amount and the balance as two independent figures and perform no arithmetic between them.
- No task introduces a pending state, a hold, or an available-versus-posted distinction. There is one
  balance.
- No task adds a statement, an export, or a report; no interest or rounding rule; no auth, user, or
  session; no transfer or counter-entry; no container, CI, or deployment file; no metric, trace, or
  logging beyond the stdlib `log` calls that already exist.
- No task adds a route (the surface stays at five, asserted by Task 18), a dependency, a JavaScript
  file, a colour, a media query, a keyframe, or a font face.
- No task edits `internal/ledger`, `internal/store`, `internal/clock`, `cmd/server`, or
  `web/style.css`.

## Verification

- [ ] All happy path criteria covered by at least one task
- [ ] All negative path criteria covered by at least one task
- [ ] No task exceeds 5 minutes of work
- [ ] Dependencies are explicit and acyclic
- [ ] Every task carries a `**Wired-into:**` line
- [ ] No task targets another feature's sealed artifact under `.docs/architecture/`, `.docs/plans/`,
      `.docs/specs/`, or `.docs/stories/`
- [ ] No task introduces anything from the non-goals list
