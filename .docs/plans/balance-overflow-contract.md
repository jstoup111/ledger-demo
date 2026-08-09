# Implementation Plan: Balance Overflow Contract

**Date:** 2026-08-09
**Design:** none — technical track, no PRD. Intent and discovery in
[balance-overflow-contract.md](../track/balance-overflow-contract.md)
**Stories:** [balance-overflow-contract.md](../stories/balance-overflow-contract.md) (Accepted)
**Conflict check:** skipped — Tier S
**Architecture review:** skipped — Tier S. No ADR authored, so no ADR can be DRAFT at the land gate.
**Tier:** S ([balance-overflow-contract.md](../complexity/balance-overflow-contract.md))

> **Filename note:** this plan is `balance-overflow-contract.md` so its stem matches
> `.docs/complexity/balance-overflow-contract.md` and the daemon resolves the tier at build time.

## Summary

**Two tasks.** Story 1 — the accepted contract documenting the seventh error code — is already
delivered in this spec commit and is **not** a task below. Story 2 is the whole of BUILD: one
test-only addition that binds the contract document to the shipped sentinel-to-code mapping, so a
future eighth code cannot ship undocumented.

No production file is touched. No route is added or removed. No error code string, HTTP status, or
message changes. `internal/ledger/balance.go` and its guard are untouched.

## Already delivered at DECIDE — not a BUILD task

`.docs/decisions/api-response-contract.md` is an Accepted DECIDE artifact whose "six error codes"
assertion this DECIDE pass falsified. The harness's DECIDE Artifact Amendment Ownership rule requires
DECIDE to amend it in place on the spec branch, additively, before the first BUILD entry — and states
that **BUILD never receives that mutation as a task**. The amendment is therefore in this spec commit:
three dated `> **Amended 2026-08-09 …**` notes plus one added error-table row. BUILD must not re-edit
that document; Task 2 only *reads* it.

## Technical Approach

**The failure mode being closed.** `internal/httpapi/errors.go#codeFor` is the single place a domain
sentinel becomes a wire code. The contract document lists those codes for humans. Nothing connects the
two, so when `85df875` added a seventh case to `codeFor` the document silently fell out of date. The
fix is to make the document a test input.

**Everything lives in `_test.go`.** Both helpers and both assertions go in
`internal/httpapi/errors_test.go`. `codeFor` and `codedError` are package-private, so the test must be
in package `httpapi`; keeping it test-only means the production surface of this feature is zero lines.

**Two derived sets, compared both ways.**

- `mappedErrorCodes()` folds the domain sentinels through `codeFor` and collects `code` values. The
  sentinel list is written out explicitly, mirroring the existing list in
  `internal/ledger/errors_test.go`.
- `documentedErrorCodes(markdown string)` scans the `## Error responses` section only — from that
  heading to the next `## ` heading — and takes the first backticked token of each table row. Scoping
  to the section is what keeps the Conventions table (`| Money | Integer cents … |`) and the Non-JSON
  responses table out of the result. Rows are recognised by a leading `|`, which skips the amendment
  blockquotes (leading `>`), the header row, and the `|---|` separator.

Both directions are asserted: a code in the mapping but not the document, and a code in the document
but not the mapping, each fail with the offending code named. A one-directional check would not have
caught `85df875`'s omission from the document's side alone.

**RED is genuine, not a tautology.** The document is already correct as of this spec commit, so a test
that only reads the real file would pass the moment it compiles and would prove nothing. Task 1
therefore builds the negative case first, against an inline **stale six-row fixture** — a copy of the
pre-amendment table — and requires the comparison to report `balance_overflow` as undocumented. That
negative case is permanent: it keeps proving the comparison has teeth without depending on the real
document ever being wrong. Task 2 then points the same, already-proven comparison at the real file.

**Determinism.** One `os.ReadFile` of a committed file. No clock, no network, no sleep, no ordering
dependency, no database. `-count=2` safe.

## Task Dependency Graph

```
Task 1 (comparison + stale-fixture negative case)
   └── Task 2 (live assertion against the real contract document)
```

Strictly sequential — Task 2 uses the helpers Task 1 introduces.

---

### Task 1: Contract-to-mapping comparison, proven by a stale fixture
**Story:** Story 2 (a code at the boundary but missing from the contract must fail, and the reverse)
**Type:** test

**Steps:**
1. Write failing test `TestContractCodeSetComparisonDetectsDrift` in
   `internal/httpapi/errors_test.go`. Table-driven over inline markdown fixtures:
   - **stale six-row fixture** — the pre-amendment table, verbatim: `account_not_found`, `amount_zero`,
     `description_empty`, `description_too_long`, `amount_malformed`, `balance_would_go_negative`.
     Expect the comparison to report exactly `balance_overflow` as documented-nowhere.
   - **extra-row fixture** — the seven real rows plus a fabricated `| \`not_a_real_code\` | \`400\` |
     … |`. Expect exactly `not_a_real_code` reported as present in the document but unmapped.
   - **matching fixture** — the seven real rows. Expect no discrepancy in either direction.
   - **decoy-section fixture** — the seven real rows preceded by a Conventions-style table and an
     amendment blockquote inside the `## Error responses` section, and followed by a `## Non-JSON
     responses` table. Expect the same result as the matching fixture, proving the section scoping and
     the blockquote skip.
2. Verify test fails (RED) — `documentedErrorCodes` and `mappedErrorCodes` do not exist, so the
   package does not compile.
3. Implement both helpers in `internal/httpapi/errors_test.go`:
   - `mappedErrorCodes() []string` — fold the seven `ledger.Err*` sentinels through `codeFor`, each
     wrapped once with `fmt.Errorf("%w", …)` so the test exercises the same `errors.Is` path the
     handlers do, and collect `code`. Fail if any sentinel yields the zero `codedError` (a sentinel
     falling through to the generic `500`).
   - `documentedErrorCodes(markdown string) []string` — scope to `## Error responses` up to the next
     `## `; for each line beginning `|`, skip the header and `|---|` separator rows and take the first
     backticked token; return them in document order.
   - a comparison helper returning the two discrepancy slices (mapped-but-undocumented,
     documented-but-unmapped).
4. Verify test passes (GREEN)
5. Confirm `gofmt -l .` is empty and `go vet ./...` is clean.
6. Commit: "httpapi: prove the contract-to-mapping code comparison against a stale fixture"

**Files:** `internal/httpapi/errors_test.go`
**Wired-into:** `internal/httpapi/errors.go#codeFor` (the mapping under comparison — read, not modified)
**Dependencies:** none

---

### Task 2: Assert the real contract document matches the shipped mapping
**Story:** Story 2 (the documented set and the emitted set are equal, and `balance_overflow` is in both)
**Type:** test

**Steps:**
1. Write failing test `TestContractDocumentsEveryEmittedErrorCode` in
   `internal/httpapi/errors_test.go`. It must:
   - read `filepath.Join("..", "..", ".docs", "decisions", "api-response-contract.md")`, failing with
     the resolved path if the file is missing, so a moved document is a loud failure rather than a
     silent skip;
   - run Task 1's comparison over the real content and fail on either discrepancy slice, naming the
     offending codes and pointing at `.docs/decisions/api-response-contract.md`;
   - assert `balance_overflow` appears in both the documented set and the mapped set, so the specific
     code this feature exists for is pinned by name and not merely by set equality;
   - assert the documented set has exactly seven entries and no duplicates.
2. Verify test fails (RED) — temporarily assert an eighth documented code is expected, observe the
   failure message names it correctly, then remove that temporary line. This confirms the assertion is
   wired to the real file rather than passing vacuously.
3. Make it pass against the document as amended in this spec commit. **No edit to any file under
   `.docs/` is permitted in this task** — the document is already correct; if this test fails, the
   discrepancy is in `codeFor` and must be reported, not papered over by editing the contract.
4. Verify test passes (GREEN)
5. Confirm by name that `TestRouterMapsBalanceOverflowAtBothPostingBoundaries` in
   `internal/httpapi/router_test.go` and the `balance overflow` case in
   `TestCodeForMapsWrappedDomainErrors` are both still present and passing — neither deleted, skipped,
   nor loosened.
6. Confirm `git diff` touches no file under `internal/ledger/`, leaving the `checkedAdd` guard exactly
   as shipped, and that `go.mod` is unchanged.
7. Run `go test ./... -count=2`; confirm it passes, stays under 10 seconds, and that `gofmt -l .` is
   empty and `go vet ./...` is clean.
8. Commit: "httpapi: assert the API response contract documents every emitted error code"

**Files:** `internal/httpapi/errors_test.go`
**Wired-into:** `internal/httpapi/errors.go#codeFor`; reads
`.docs/decisions/api-response-contract.md`
**Dependencies:** Task 1

---

## Out of scope — do not do these

- **Do not edit `.docs/decisions/api-response-contract.md` during BUILD.** DECIDE owns that amendment
  and already made it in this spec commit.
- **Do not touch `internal/ledger/balance.go` or `checkedAdd`.** Folding signed `int64` cents genuinely
  can overflow; the guard is load-bearing and stays.
- **Do not change any existing error code string, HTTP status, or message.** The six originally
  documented codes are a stable contract.
- **Do not add or remove an HTTP route.** Exactly five remain.
- **Do not touch `.docs/specs/2026-08-08-base-ledger.md`, `.docs/stories/base-ledger.md`, or
  `.docs/plans/base-ledger.md`.** Their "six validation rules" wording is correct and they are another
  feature's protected artifacts.
- **Do not add a dependency.** stdlib plus the pinned `modernc.org/sqlite` only; this feature adds
  nothing.
- **Do not build any non-goal.** No dedup or idempotency of any kind, no overdraft or fees, no
  pending/holds, no statements or exports, no interest or rounding, no auth, no transfers, no Docker or
  CI, no metrics or tracing.
