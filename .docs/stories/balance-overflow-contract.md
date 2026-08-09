# Stories — Balance Overflow Contract

**Status:** Accepted

**Feature:** balance-overflow-contract · **Tier:** S · **Track:** technical
**Source:** operator idea, no originating GitHub issue. Technical track, so acceptance criteria live
here rather than in a PRD.
**Constrained by:** `.docs/decisions/api-response-contract.md` (Accepted, amended 2026-08-09) and
`.docs/decisions/adr-2026-08-08-sentinel-errors-for-domain-failures.md`

Two stories. Story 1 is the outcome — the accepted contract tells the truth about the shipped API —
and is **delivered at DECIDE**, in this spec commit, because the harness assigns amendment of an
Accepted DECIDE artifact to DECIDE and forbids handing it to BUILD as a task. Story 2 is the only
BUILD-owned work: the assertion that stops this class of drift recurring.

## Negative-path categories evaluated

Every category is evaluated explicitly; saying a category does not apply is more useful than
inventing a scenario for it.

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — Story 2's negative case feeds the parser a deliberately stale six-row contract table and requires the comparison to fail |
| Data integrity | **Yes** — the overflow guard exists so a wrapped `int64` fold can never report a wrong balance; Story 2 asserts the code that reports it stays mapped |
| Model-level immutability | No — no model is added or changed |
| Invariant side-effect on alternate branches | **Yes, already covered** — `internal/httpapi/router_test.go:649` asserts the overflow rejection on both the JSON and form branches, including that the maximum balance is unchanged. Story 2 must not weaken it |
| Dependency unavailability | No — the contract document is read from the working tree by a test in the same repository; no external dependency |
| Auth / permission failures | No — auth is an explicit non-goal |
| Timeouts & network errors | No — no network call on any path |
| Resource exhaustion | No — one small file read |
| Partial failure & rollback | No — nothing is written at runtime |
| Concurrent access | No — no runtime behavior changes |
| Cascade deletion effects | No — nothing is deleted |
| Exception class hierarchy | No — Go sentinel errors compared with `errors.Is` |
| Dedup / idempotency key analysis | No — there is no dedup or idempotency criterion anywhere in this feature, and dedup is an explicit non-goal |

---

**Status:** Accepted

## Story 1: The accepted contract documents the code the API actually emits

**Delivered at:** DECIDE (this spec commit). **No BUILD task** — the harness's DECIDE Artifact
Amendment Ownership rule requires DECIDE to amend an Accepted DECIDE artifact in place on the spec
branch and forbids routing that mutation to BUILD.

As an engineer reading `.docs/decisions/api-response-contract.md` to learn what the API can return,
I want every error code the API actually emits to be listed there, so that I do not have to read
`internal/httpapi/errors.go` to discover a seventh code the contract never mentions.

### Scenarios

- **Given** the accepted contract, **when** its error-response table is read, **then** it lists
  `balance_overflow` with HTTP `400` and the rule "folding the account's signed `int64` cents would
  overflow `int64`; nothing is recorded".
- **Given** that `balance_overflow` was introduced during implementation after the contract was
  accepted on 2026-08-08, **when** the row is read, **then** an adjacent dated amendment note says so
  and names the guard (`internal/ledger/balance.go`), the mapping (`internal/httpapi/errors.go`), and
  the commit (`85df875`), so a reader can tell a documentation gap from a behavior change.
- **Given** the original text asserting "six error codes" and "not one of the six codes below",
  **when** the amended document is read, **then** each stale assertion carries a dated amendment note
  beside it correcting the count to seven, **and** the original sentence is still present and
  unaltered.
- **Given** the amendment, **when** the document is compared against the shipped API, **then** no
  existing error code string, HTTP status, or rule has been changed, and no row has been removed.

### Done When
- [ ] `.docs/decisions/api-response-contract.md` has seven rows in its error table, the seventh being
      `balance_overflow` / `400`.
- [ ] Three dated `> **Amended 2026-08-09 …**` notes are present: at the "six error codes" summary, at
      the "not one of the six codes below" page rule, and below the error table.
- [ ] Every one of the six originally documented code strings and HTTP statuses is byte-for-byte
      unchanged, confirmed by `git diff` showing only additions in the error table.
- [ ] The document's own `**Status:** Accepted` and `**Date:** 2026-08-08` header lines are unchanged.
- [ ] `grep -c 'balance_overflow' .docs/decisions/api-response-contract.md` is non-zero.

---

## Story 2: The contract and the shipped mapping cannot silently drift apart again

As the engineer who has to trust this contract on stage, I want a test that fails when the documented
set of error codes and the set the HTTP boundary actually produces disagree, so that the next code
added at the boundary cannot ship undocumented the way `balance_overflow` did.

### Scenarios

- **Given** the amended contract and the shipped `codeFor` mapping, **when** the documented code set
  is compared with the set produced by mapping every domain sentinel, **then** the two sets are equal
  and the test passes.
- **Given** a code present at the HTTP boundary but absent from the contract's error table, **when**
  the sets are compared, **then** the test fails and its message names the undocumented code. This is
  asserted against a deliberately stale six-row fixture, so the negative case is permanent and does
  not depend on the real document ever being wrong.
- **Given** a code present in the contract's error table but absent from the boundary mapping, **when**
  the sets are compared, **then** the test fails and its message names the undocumented-in-code entry.
- **Given** `ledger.ErrBalanceOverflow`, **when** it is wrapped and passed to `codeFor`, **then** the
  result is still HTTP `400` with code `balance_overflow` and message "Balance would overflow." —
  unchanged from what commit `85df875` shipped.
- **Given** the existing overflow assertions in `internal/httpapi/errors_test.go` and
  `internal/httpapi/router_test.go`, **when** this story is complete, **then** all of them are still
  present and still passing; none is deleted, skipped, or loosened.

### Done When
- [ ] A test in `internal/httpapi` parses the error-code column of the "Error responses" table in
      `.docs/decisions/api-response-contract.md` and compares it, as a set, against the codes
      `codeFor` returns for the domain sentinels.
- [ ] The comparison is **bidirectional** — documented-but-unmapped and mapped-but-undocumented each
      fail with a message naming the offending code.
- [ ] A table-driven negative case drives the same comparison over an inline stale six-row fixture and
      requires a failure naming `balance_overflow`.
- [ ] The parser is scoped to the `## Error responses` section, so the Conventions and Non-JSON
      response tables in the same document cannot be mistaken for error rows.
- [ ] The amendment blockquotes inside that section are ignored by the parser.
- [ ] `TestRouterMapsBalanceOverflowAtBothPostingBoundaries` and the `balance overflow` case in
      `TestCodeForMapsWrappedDomainErrors` still exist and still pass, verified by name.
- [ ] `internal/ledger/balance.go` is unchanged — `git diff` touches no file under `internal/ledger`.
- [ ] Exactly five HTTP routes remain registered; no route added or removed.
- [ ] No new module dependency: `go.mod` is unchanged and the test imports only stdlib plus the
      project's own packages.
- [ ] No `time.Sleep`, no wall-clock read, no test-ordering dependency; the test passes with
      `-count=2`.
- [ ] `make test` (or `go test ./...`) passes in under 10 seconds; `gofmt -l .` is empty and
      `go vet ./...` is clean.

---

## Assumption ledger

Recorded rather than guessed at, per the harness correctness gate. The operator was unavailable, so
each is labelled **assumed, operator unavailable** and proceeds; none is presented as operator
approval.

| # | Assumption | Confidence | Impact if wrong | How to confirm |
|---|---|---|---|---|
| A1 | The amendment identifier may cite the spec branch instead of an issue number. The harness's amendment form is `> **Amended YYYY-MM-DD by #NNN:**`, and no issue exists — the operator directed that none be invented. The branch name is used as the traceability handle. | 90% — verified that no issue exists; the substitution itself is inferred | Cosmetic. A reviewer wanting an issue number would ask for one to be filed and the three notes retitled. No behavior or artifact structure depends on it. | Operator confirms the branch reference is acceptable, or files an issue and supplies its number. **Assumed, operator unavailable.** |
| A2 | Story 2's drift guard is wanted. The operator's stated outcome was "the code remains asserted", which minimally means the existing assertions survive. Story 2 goes one step further and adds a doc-to-code binding, because the absent binding is the direct cause of this gap and because it is the only work BUILD legitimately owns once DECIDE takes the amendment. | 70% — the intent is inferred from the outcome statement, not stated | If unwanted, Story 2 is dropped and the feature completes at DECIDE with the amendment alone. Nothing in Story 1 depends on Story 2. Cost of being wrong is one small test file, reversible in one commit. | Operator confirms or drops Story 2. **Assumed, operator unavailable.** |
| A3 | Reading a `.docs/` markdown file from a test in `internal/httpapi` is acceptable coupling. The path is `../../.docs/decisions/api-response-contract.md`; it is deterministic, offline, and stdlib-only, but it does couple a package test to the repository layout. | 75% — mechanically certain to work; the taste judgement is inferred | If judged too brittle, the alternative is a hand-maintained list of seven codes in the test, which catches less drift, or dropping Story 2 per A2. | Operator or `/code-review` rules on the coupling. **Assumed, operator unavailable.** |
| A4 | FR-12's "six cases" in `.docs/specs/2026-08-08-base-ledger.md` is not stale and is deliberately untouched — FR-12 enumerates six input-validation rules, and the overflow guard is not one of them. | 95% — verified by reading FR-12 and the guard | If FR-12 were meant to cover the overflow guard, that spec would need its own amendment. It is also a protected artifact belonging to another feature, so it could not be a plan task here regardless. | Read `.docs/specs/2026-08-08-base-ledger.md` FR-12 alongside `internal/ledger/balance.go`. **Assumed, operator unavailable.** |

## Explicit non-goals for this feature

None of the following is built, hinted at, or left a hook for. Each is a live-demo payload item whose
presence would ruin the presentation.

Duplicate or double-charge detection, idempotency keys, dedup windows, any uniqueness constraint
beyond the primary key; overdraft, fees, percentages; pending transactions or holds,
available-vs-posted balance; statements, exports, reporting; interest or rounding rules; auth, users,
sessions; transfers or counter-entries; Docker, CI, deployment tooling; metrics, tracing, or
structured logging beyond stdlib `log`.

Additionally out of scope here: adding or removing any HTTP route, changing any existing error code
string or HTTP status, and altering the overflow guard itself.
