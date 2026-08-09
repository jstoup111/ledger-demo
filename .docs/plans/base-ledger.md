# Implementation Plan: Base Ledger

**Date:** 2026-08-08
**Design:** [2026-08-08-base-ledger.md](../specs/2026-08-08-base-ledger.md)
**Stories:** [base-ledger.md](../stories/base-ledger.md)
**Conflict check:** Clean as of 2026-08-08 (`.docs/conflicts/2026-08-08-base-ledger.md` — 0 blocking,
3 degrading resolved)
**Architecture review:** APPROVED WITH CONDITIONS
(`.docs/decisions/architecture-review-2026-08-08-base-ledger.md` — 3 conditions, tracked below)
**Tier:** M (`.docs/complexity/base-ledger.md`)

> **Filename note:** this plan is `base-ledger.md`, not the skill's default
> `YYYY-MM-DD-<feature>.md`, because the plan stem must match `.docs/complexity/base-ledger.md` for
> the daemon to resolve the complexity tier at build time.

## Summary

Builds the complete base ledger — five packages that currently hold only doc comments, plus the page
markup and deterministic seed data — in **32 tasks** across six batches. Every task is a TDD cycle at
2–5 minute granularity.

## Technical Approach

**Dependency direction drives the batch order.** `internal/store` depends on `internal/ledger`, never
the reverse (`adr-2026-08-08-store-interface-in-domain-package.md`), so the domain's types, sentinels,
and `Store` interface come first (Batch A), the SQLite implementation second (Batch B), then the
domain operations that use it (Batch C), then the HTTP layer (Batches D–E), then wiring and seed
(Batch F). Nothing in Batch A imports anything from Batches B–F, which is what lets the domain
unit-test with no database.

**Money never becomes a float.** The dollars→cents parse lives at the HTTP boundary
(`internal/httpapi`), decided in the architecture review: FR-12e rejects malformed *input*, a property
of an untrusted string, while the other five rules are properties of a well-formed transaction. The
domain's `PostTransaction` therefore takes an `int64` and cannot be handed a malformed amount at all.
Parsing splits the string on `.` and uses `strconv.ParseInt` on each half — no `ParseFloat`, no
`float64` anywhere on the path.

**Identity and ordering are one mechanism.** Transaction ids are `fmt.Sprintf("txn-%04d", n+1)` where
`n` is the count of **all** rows in the transactions table — global, not per account, because `id` is
the table's primary key and per-account numbering would duplicate `txn-0001` on the second account
seeded (Condition C1). Constant width makes the id sort lexicographically in insertion order, so
`ORDER BY created_at DESC, id DESC` is a total order even though `FixedClock` gives every transaction
in a test the same timestamp.

**One posting handler, two response encoders.** `POST /api/accounts/{id}/transactions` branches on the
request's `Content-Type`: `application/json` returns `201` JSON, and **everything else, including a
missing header**, takes the form branch and returns `303 See Other`
(`adr-2026-08-08-one-negotiated-posting-endpoint.md`). Validation happens before the branch, so FR-9
holds structurally — there is one path, not two that must be kept in agreement.

**Test isolation.** Everything uses in-memory SQLite except the reset/seed determinism tests, which
must be file-backed because `reset` deletes a file. Those set `LEDGER_DB_PATH` to a `t.TempDir()` path
and never touch the default `./ledger.db` (conflict-check F3).

**Test ownership, to avoid triplicated rule tests** (conflict-check F1): Task 18 owns rule semantics
(sentinel + code + nothing recorded, all six rules). Task 24 asserts *equivalence only* — the same
rule fires through both content types — consuming Task 18's codes. Task 26 uses one representative
rejection because its subject is that a URL code becomes a visible panel.

## Prerequisites

None beyond the existing checkout. `modernc.org/sqlite` is already pinned in `go.mod`/`go.sum` by the
blank import in `internal/store/sqlite.go`, deliberately so the build needs no network.

> **Amended 2026-08-09 by operator review — build_review scope gap `test:build-review-config`:** the
> assertion above is retained, and was too narrow. This feature does require one harness-configuration
> change, scoped by `Task harness-config` below. `build_review` correctly flagged
> `.ai-conductor/config.yml` as modified with no plan task describing it.

### Task harness-config: Declare the aggregate test command and the acceptance-spec location
**Story:** Story 6 / FR-15 and FR-16 (the reset/run commands the demo is operated by)
**Type:** infrastructure

Recorded after the fact by operator review, not authored ahead of BUILD. It scopes a change BUILD had
already made out of plan; it is written here so the diff is described by the plan rather than to
request new work.

**Steps:**
1. Declare `test_suite.command` so the pre-SHIP aggregate gate has one project-owned command
   composing the whole authoritative suite. Without it that gate returns
   `missing_config: Project config must declare test_suite` and the SHIP tail cannot complete.
2. Record where acceptance specs live, so the `acceptance_specs` gate resolves them. The gate's
   built-in globs are Ruby/JS-shaped and match `test/acceptance/**/*` but not a top-level
   `acceptance/`, which cost this feature two provider attempts before the specs were relocated.
3. Verification is limited to harness configuration: `conduct-ts` loads the config without error and
   the two gates above resolve. No product behavior is touched and no Go file changes.

**Files:** `.ai-conductor/config.yml`
**Wired-into:** none (no new production surface — consumed by the engine's own gates, not by this
project's code)
**Verify-only:** yes
**Dependencies:** none

Marked verify-only because this task records a change BUILD had already committed, so it will never
produce a commit carrying its own `Task:` trailer. Without the marker, `build_review`'s advisory
work-happened floor flags it as a gap on every run (observed 2026-08-09 04:57:39).

## BUILD-entry artifacts not authored by any plan task

> **Added 2026-08-09 by operator review — build_review scope gap `test:build-review-acceptance`:**
> `build_review` correctly found ~1,180 lines in the two files below present in no plan task's
> `Files:` list.

| Artifact | Owner | Constrains |
|---|---|---|
| `test/acceptance/ledger_acceptance_test.go` | `/writing-system-tests` (BUILD entry, before implementation) | Stories 1–6; the RED targets for Tasks 10, 18, 19–28 |
| `test/acceptance/harness_test.go` | `/writing-system-tests` (BUILD entry, before implementation) | Shared fixture for the above: builds `./cmd/server`, runs `seed`/`serve` on a free port |

These are **not** work any implementation task performs, and ownership is not reassigned to one. They
are the acceptance-spec artifacts the `acceptance_specs` step authors at BUILD entry, and they are
recorded here only so the plan describes the diff. Regenerating them, or changing product behavior to
suit them, is explicitly out of scope for this remediation.

Two known defects in `harness_test.go` are deliberately **not** fixed here, to avoid adding further
unplanned acceptance-spec diff to a scope finding: `waitReady` busy-waits on `net.Dial` with no yield
for up to 15s per server start, and `startServer`'s child survives an interrupted `go test` (it
stranded a server on port 8080 for 90 minutes). Both are queued for a separate pass.

## Tasks

### Batch A — Domain shape and injectable time

### Task 1: Clock interface with SystemClock and FixedClock
**Story:** Story 6 (`grep 'time.Now()'` returns exactly one hit) · underpins all determinism
**Type:** infrastructure

**Steps:**
1. Write failing test: `FixedClock{T: <fixed instant>}.Now()` returns that exact instant on two
   successive calls; `SystemClock{}.Now()` returns a non-zero time.
2. Verify test fails (RED)
3. Implement `Clock interface { Now() time.Time }`, `SystemClock`, `FixedClock` in `internal/clock`.
   `SystemClock.Now` is the only `time.Now()` call site in the repository.
4. Verify test passes (GREEN)
5. Commit: "clock: add Clock interface with SystemClock and FixedClock"

**Files:** `internal/clock/clock.go`, `internal/clock/clock_test.go`, `internal/clock/doc.go`
**Wired-into:** `cmd/server/main.go#serve`, `cmd/server/main.go#seed` (constructed there and injected
inward, per architecture-review Wiring Surface)
**Dependencies:** none

### Task 2: Account and Transaction domain types
**Story:** Story 1 (account with balance), Story 2 (transaction fields)
**Type:** infrastructure

**Steps:**
1. Write failing test: a `Transaction` literal round-trips its fields; `Amount` is `int64`.
2. Verify test fails (RED)
3. Implement `Account{ID, Name string}` and
   `Transaction{ID, AccountID string; Amount int64; Description string; CreatedAt time.Time}` in
   `internal/ledger`. No balance field on `Account`.
4. Verify test passes (GREEN)
5. Commit: "ledger: add Account and Transaction types"

**Files:** `internal/ledger/ledger.go`, `internal/ledger/ledger_test.go`, `internal/ledger/doc.go`
**Wired-into:** `internal/store/sqlite.go#InsertAccount`, `internal/store/sqlite.go#Append`
**Dependencies:** none

> **Amended 2026-08-09 by operator review — build_review scope gap `test:build-review-plan`:**
>
> **Original approved assertion, preserved verbatim as the record of what DECIDE authorized:**
> `**Wired-into:** internal/store/sqlite.go#scanTransaction, internal/httpapi/router.go#NewRouter`
>
> That assertion was wrong on both symbols. `scanTransaction` is a name this plan invented and the
> implementation never created, and `NewRouter` consumes the router's dependencies rather than being
> where these types are wired. The `Wired-into:` line above therefore now carries the **as-built**
> call sites, `internal/store/sqlite.go#InsertAccount` and `internal/store/sqlite.go#Append`, which
> is the reconciliation this amendment authorizes.
>
> The live line must hold the effective contract, not the superseded one: `wiring_check` parses the
> `Wired-into:` line directly and does not read this note
> (`.pipeline/wiring-evidence.json` recorded `contract = …#scanTransaction, …#NewRouter` while the
> original was live, leaving an unresolvable symbol and a stale contract). An earlier revision of
> this amendment kept the original on the live line and put the correction here, which halted the
> feature; preserving the original as quoted text above satisfies the record without feeding the
> verifier a symbol that does not exist.
>
> No product behavior changes — only the recorded wiring contract.

### Task 3: Six distinct sentinel errors
**Story:** Story 5 (each rule reports its own sentinel)
**Type:** infrastructure

**Steps:**
1. Write failing test: all six sentinels are non-nil, pairwise distinct under `errors.Is`, and each
   survives one `fmt.Errorf("%w", …)` wrap.
2. Verify test fails (RED)
3. Implement `ErrAccountNotFound`, `ErrAmountZero`, `ErrDescriptionEmpty`, `ErrDescriptionTooLong`,
   `ErrAmountMalformed`, `ErrBalanceWouldGoNegative` in `internal/ledger`.
4. Verify test passes (GREEN)
5. Commit: "ledger: add the six domain sentinel errors"

**Files:** `internal/ledger/errors.go`, `internal/ledger/errors_test.go`
**Wired-into:** `internal/httpapi/errors.go#codeFor` (mapped to wire codes at the boundary)
**Dependencies:** none

### Task 4: Store interface declared in the domain
**Story:** Story 1, Story 2 (domain reads accounts and transactions without knowing SQLite)
**Type:** infrastructure

**Steps:**
1. Write failing test: a trivial in-package fake satisfies `ledger.Store` — compile-time assertion
   `var _ Store = (*fakeStore)(nil)`.
2. Verify test fails (RED)
3. Declare `Store` in `internal/ledger` with `Accounts`, `Account`, `Transactions`,
   `CountTransactions`, and `Append`. No SQLite import in this package, ever.
4. Verify test passes (GREEN)
5. Commit: "ledger: declare the Store interface in the domain package"

**Files:** `internal/ledger/store.go`, `internal/ledger/store_test.go`
**Wired-into:** `internal/store/sqlite.go#Store` (implements it), `cmd/server/main.go#serve` (concrete
value constructed and passed inward)
**Dependencies:** 2, 3

> **Gate after Batch A:** `go vet ./...` clean; `internal/ledger` and `internal/clock` import nothing
> from `internal/store`; `grep -rn 'time.Now()'` returns exactly one hit.

### Batch B — SQLite persistence

### Task 5: Schema creation
**Story:** Story 1 (no stored balance column), Story 6 (seed starts from an empty schema)
**Type:** infrastructure

**Steps:**
1. Write failing test: after opening an in-memory database and creating the schema, querying
   `sqlite_master` shows exactly the `accounts` and `transactions` tables, and the `transactions`
   DDL contains no `UNIQUE` beyond its primary key and no balance column.
2. Verify test fails (RED)
3. Implement `Open(dsn string)` (in-memory when the DSN says so, file-backed otherwise) and schema
   creation: `accounts(id TEXT PRIMARY KEY, name TEXT NOT NULL)` and
   `transactions(id TEXT PRIMARY KEY, account_id TEXT NOT NULL REFERENCES accounts(id), amount INTEGER NOT NULL, description TEXT NOT NULL, created_at TEXT NOT NULL)`.
4. Verify test passes (GREEN)
5. Commit: "store: create the accounts and transactions schema"

**Files:** `internal/store/sqlite.go`, `internal/store/sqlite_test.go`, `internal/store/doc.go`
**Wired-into:** `cmd/server/main.go#serve`, `cmd/server/main.go#seed` (invoked on open)
**Dependencies:** 4

### Task 6: Insert and read accounts
**Story:** Story 1 (three accounts listed)
**Type:** happy-path

**Steps:**
1. Write failing test: inserting three accounts then calling `Accounts()` returns them ordered by
   `id` ascending; `Account("acct-2")` returns that one; `Account("nope")` returns
   `ledger.ErrAccountNotFound`.
2. Verify test fails (RED)
3. Implement `InsertAccount`, `Accounts`, and `Account` on the store, wrapping the not-found case as
   `ErrAccountNotFound`.
4. Verify test passes (GREEN)
5. Commit: "store: insert and read accounts"

**Files:** `internal/store/sqlite.go`, `internal/store/sqlite_test.go`
**Wired-into:** same as Task 5
**Dependencies:** 5

### Task 7: Append a transaction and count all rows
**Story:** Story 2 (global id sequence), Story 6 (24–36 seeded rows) · **Condition C1**
**Type:** happy-path

**Steps:**
1. Write failing test: `CountTransactions()` returns the total across **all** accounts, not per
   account — insert 2 rows for `acct-1` and 3 for `acct-2`, assert `5`. Then `Append` a row and
   assert the count is `6` and the row is readable.
2. Verify test fails (RED)
3. Implement `Append(ledger.Transaction)` and `CountTransactions()` — the latter counts the whole
   table with no `WHERE account_id`.
4. Verify test passes (GREEN)
5. Commit: "store: append transactions and count all rows globally"

**Files:** `internal/store/sqlite.go`, `internal/store/sqlite_test.go`
**Wired-into:** same as Task 5
**Dependencies:** 6

### Task 8: Read transactions newest first, total order
**Story:** Story 2 (stable newest-first) · **Condition C3**
**Type:** happy-path

**Steps:**
1. Write failing test: insert `txn-0001`, `txn-0002`, `txn-0003` **all with the same
   `created_at`**; assert `Transactions("acct-1")` returns exactly
   `[txn-0003, txn-0002, txn-0001]`. Run the assertion twice in the same test to prove stability.
2. Verify test fails (RED)
3. Implement `Transactions(accountID)` with `ORDER BY created_at DESC, id DESC`.
4. Verify test passes (GREEN)
5. Commit: "store: read transactions newest first with a total order"

**Files:** `internal/store/sqlite.go`, `internal/store/sqlite_test.go`
**Wired-into:** same as Task 5
**Dependencies:** 7

### Task 9: Empty and unknown account read behavior
**Story:** Story 2 (empty → `[]`, unknown → not-found)
**Type:** negative-path

**Steps:**
1. Write failing test: `Transactions` for an existing account with no rows returns a non-nil empty
   slice (so it encodes as `[]`, never `null`); `Transactions` for an unknown account returns
   `ledger.ErrAccountNotFound`.
2. Verify test fails (RED)
3. Implement: initialize the result slice to a non-nil empty slice; check account existence first.
4. Verify test passes (GREEN)
5. Commit: "store: empty account returns an empty slice, unknown account returns not-found"

**Files:** `internal/store/sqlite.go`, `internal/store/sqlite_test.go`
**Wired-into:** same as Task 5
**Dependencies:** 8

> **Gate after Batch B:** store tests all use in-memory SQLite; no test touches `./ledger.db`;
> `sqlite_master` assertion confirms no uniqueness constraint beyond the primary keys.

### Batch C — Domain operations and the six rules

### Task 10: Derived balance
**Story:** Story 1 (balance computed, never stored)
**Type:** happy-path

**Steps:**
1. Write failing test: table-driven over an empty log → `0`; a single `+2500` → `2500`; a mixed log
   `[+128350, -4250]` → `124100`. Uses a fake `Store`, no database.
2. Verify test fails (RED)
3. Implement `Balance(accountID)` as a fold over `Transactions`. No caching, no stored field.
4. Verify test passes (GREEN)
5. Commit: "ledger: derive balance as a fold over the transaction log"

**Files:** `internal/ledger/balance.go`, `internal/ledger/balance_test.go`
**Wired-into:** `internal/httpapi/router.go#handleAccounts`, `internal/httpapi/router.go#handlePage`
**Dependencies:** 4

### Task 11: Reject a zero amount
**Story:** Story 5 / FR-12b
**Type:** negative-path

**Steps:**
1. Write failing test: `PostTransaction` with amount `0` returns an error satisfying
   `errors.Is(err, ErrAmountZero)`, and the fake store recorded nothing.
2. Verify test fails (RED)
3. Implement the zero-amount check.
4. Verify test passes (GREEN)
5. Commit: "ledger: reject a zero amount"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** `internal/httpapi/router.go#handlePostTransaction`
**Dependencies:** 10

### Task 12: Reject an empty or whitespace-only description
**Story:** Story 5 / FR-12c
**Type:** negative-path

**Steps:**
1. Write failing test: descriptions `""`, `"   "`, and `"\t\n"` each return
   `errors.Is(err, ErrDescriptionEmpty)` and record nothing.
2. Verify test fails (RED)
3. Implement the check against the `strings.TrimSpace`d description.
4. Verify test passes (GREEN)
5. Commit: "ledger: reject an empty or whitespace-only description"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 11

### Task 13: Reject a description over 140 characters
**Story:** Story 5 / FR-12d
**Type:** negative-path

**Steps:**
1. Write failing test: a 141-character description returns `errors.Is(err, ErrDescriptionTooLong)`
   and records nothing; a **140**-character description is accepted. Both sides of the boundary.
2. Verify test fails (RED)
3. Implement the length check as inclusive at 140.
4. Verify test passes (GREEN)
5. Commit: "ledger: reject a description longer than 140 characters"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 12

### Task 14: Reject an unknown account
**Story:** Story 5 / FR-12a
**Type:** negative-path

**Steps:**
1. Write failing test: `PostTransaction` against an account the store does not have returns
   `errors.Is(err, ErrAccountNotFound)` and records nothing.
2. Verify test fails (RED)
3. Implement the existence check as the first rule evaluated.
4. Verify test passes (GREEN)
5. Commit: "ledger: reject a transaction against an unknown account"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 13

### Task 15: Reject a transaction that would take the balance below zero
**Story:** Story 5 / FR-12f
**Type:** negative-path

**Steps:**
1. Write failing test: against a `1000`-cent balance, `-1001` returns
   `errors.Is(err, ErrBalanceWouldGoNegative)` and records nothing; `-1000` is **accepted** and the
   resulting balance is exactly `0`. Both sides of the boundary.
2. Verify test fails (RED)
3. Implement the check as `balance + amount < 0`.
4. Verify test passes (GREEN)
5. Commit: "ledger: reject a transaction that would take the balance below zero"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 14

### Task 16: Assign a global sequential id and append
**Story:** Story 2, Story 6 · **Conditions C1, C3**
**Type:** happy-path

**Steps:**
1. Write failing test: with 5 rows already present across two accounts, a successful
   `PostTransaction` produces id `txn-0006` — proving the sequence is global, not per account. Assert
   the id matches `^txn-\d{4}$`.
2. Verify test fails (RED)
3. Implement id assignment as `fmt.Sprintf("txn-%04d", CountTransactions()+1)`, stamp `CreatedAt`
   from the injected `Clock`, then `Append`.
4. Verify test passes (GREEN)
5. Commit: "ledger: assign globally sequential zero-padded transaction ids"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 15

### Task 17: Accept a valid transaction end to end in the domain
**Story:** Story 5 (happy path: a valid transaction passes all six rules)
**Type:** happy-path

**Steps:**
1. Write failing test: a `-1000` transaction with a 20-character description against a
   `128350`-cent balance is recorded, returns the created transaction with `CreatedAt` equal to the
   `FixedClock` instant, and the recomputed balance is `127350`. Also assert `0.01`-equivalent
   (`1` cent) is accepted.
2. Verify test fails (RED)
3. Implement the success path ordering: account exists → amount non-zero → description non-empty →
   description length → balance check → assign id → stamp time → append.
4. Verify test passes (GREEN)
5. Commit: "ledger: accept a valid transaction and return it"

**Files:** `internal/ledger/post.go`, `internal/ledger/post_test.go`
**Wired-into:** same as Task 11
**Dependencies:** 16

### Task 18: Rule-semantics table — the six rules, sentinel and nothing-recorded
**Story:** Story 5 (**sole owner of rule semantics**, per conflict-check F1)
**Type:** negative-path

**Steps:**
1. Write failing test: one table with a case per rule, each asserting the specific sentinel via
   `errors.Is` **and** that the store's row count is unchanged. Assert the six sentinels are pairwise
   distinct. This is the single place rule semantics are asserted — Tasks 24 and 26 consume its codes
   rather than restating them.
2. Verify test fails (RED)
3. Implement any gaps the table exposes (expected: none, since Tasks 11–15 built each rule).
4. Verify test passes (GREEN)
5. Commit: "ledger: table-driven coverage of all six validation rules"

**Files:** `internal/ledger/post_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 17

> **Gate after Batch C:** every rule has a negative test asserting its own sentinel; the domain
> package still imports nothing from `internal/store`; suite runs with no database.

### Batch D — HTTP reads and error mapping

### Task 19: Dollars-to-cents parsing without floats
**Story:** Story 3 (`3.50` → `350`), Story 5 / FR-12e
**Type:** happy-path

**Steps:**
1. Write failing test: table-driven — `"25"`→`2500`, `"3.50"`→`350`, `"-42.50"`→`-4250`,
   `"0.01"`→`1`, `"0"`→`0`; and rejecting `"abc"`, `"1.2.3"`, `"1,000"`, `"$5"`, `"1.234"`, `""`,
   `"  "` with `ErrAmountMalformed`.
2. Verify test fails (RED)
3. Implement by splitting on `.` and using `strconv.ParseInt` on each part. No `ParseFloat`, no
   `float64`, no `float32`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: parse dollar amounts to integer cents without floats"

**Files:** `internal/httpapi/money.go`, `internal/httpapi/money_test.go`
**Wired-into:** `internal/httpapi/router.go#handlePostTransaction`
**Dependencies:** 3

### Task 20: Sentinel-to-code mapping and the JSON error shape
**Story:** Story 4 / FR-14
**Type:** infrastructure

**Steps:**
1. Write failing test: each of the six sentinels maps to its documented code and status —
   `account_not_found`/`404`, `amount_zero`/`400`, `description_empty`/`400`,
   `description_too_long`/`400`, `amount_malformed`/`400`, `balance_would_go_negative`/`400` — and
   the encoded body is exactly `{"error":{"code":…,"message":…}}` with a non-empty message.
2. Verify test fails (RED)
3. Implement the mapping once, at the boundary, plus the error encoder.
4. Verify test passes (GREEN)
5. Commit: "httpapi: map domain sentinels to wire codes and encode the error shape"

**Files:** `internal/httpapi/errors.go`, `internal/httpapi/errors_test.go`
**Wired-into:** `internal/httpapi/router.go#handlePostTransaction`,
`internal/httpapi/router.go#handleTransactions`, `internal/httpapi/router.go#handlePage`
**Dependencies:** 3, 19

### Task 21: GET /api/accounts
**Story:** Story 1 / FR-10
**Type:** happy-path

**Steps:**
1. Write failing test: returns `200`, `Content-Type: application/json; charset=utf-8`, and an array
   of `{id, name, balance_cents}` ordered by `id` ascending, where each `balance_cents` is an integer
   equal to that account's summed amounts (expected totals written out literally in the fixture).
2. Verify test fails (RED)
3. Implement the handler and register the route.
4. Verify test passes (GREEN)
5. Commit: "httpapi: serve GET /api/accounts with derived balances"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** `internal/httpapi/router.go#NewRouter` (registered on the mux, itself called from
`cmd/server/main.go#serve`)
**Dependencies:** 10, 20

### Task 22: GET /api/accounts/{id}/transactions
**Story:** Story 2 / FR-11
**Type:** happy-path

**Steps:**
1. Write failing test: returns `200` and an array of
   `{id, account_id, amount_cents, description, created_at}` newest first with `created_at` in
   RFC 3339 UTC; an account with no transactions returns literal `[]`; an unknown account returns
   `404` with code `account_not_found`.
2. Verify test fails (RED)
3. Implement the handler and register the route.
4. Verify test passes (GREEN)
5. Commit: "httpapi: serve GET /api/accounts/{id}/transactions newest first"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 8, 9, 21

### Task 23: Method and path edge cases
**Story:** Story 1, Story 2 (405 on wrong method, 404 on unknown path)
**Type:** negative-path

**Steps:**
1. Write failing test: `POST /api/accounts` → `405` empty body; `PUT`, `PATCH`, `DELETE` on
   `/api/accounts/acct-1/transactions` → `405` empty body and the stored row unchanged;
   `GET /nope` → `404` empty body.
2. Verify test fails (RED)
3. Implement by relying on `ServeMux` method patterns; add explicit assertions rather than new code
   where the mux already yields the right status.
4. Verify test passes (GREEN)
5. Commit: "httpapi: assert method and unknown-path edge cases"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 22

> **Gate after Batch D:** `grep -rn 'float64\|float32\|ParseFloat' internal/` returns nothing on the
> money path; all read routes covered.

### Batch E — Posting and the page

### Task 24: POST both branches, negotiated on Content-Type
**Story:** Story 3 / FR-7, Story 4 / FR-8 and FR-9
**Type:** happy-path · integration point

**Steps:**
1. Write failing test: with `Content-Type: application/json` and body
   `{"amount":"-42.50","description":"Coffee beans"}` → `201` with the created transaction and
   `amount_cents: -4250`. With `application/x-www-form-urlencoded` and the same values → `303` with
   `Location: /?account=acct-1`. With **no** `Content-Type` → the form branch. Then the
   **equivalence** table: the same invalid input through both content types produces the same rule,
   identified by wire code (codes consumed from Task 18, not restated), and the row count is
   unchanged after each. Finally, **FR-7's reload property**: after a successful form post, issue the
   redirect's `GET` **twice** and assert the transaction count is unchanged — the `303` is what makes
   a reload replay only the `GET`.
2. Verify test fails (RED)
3. Implement one handler: parse amount at the boundary, call the single domain operation, then branch
   on `Content-Type` for the response only. `application/json` → JSON; everything else → `303`.
   Escape the account id when building `Location`.
4. Verify test passes (GREEN)
5. Commit: "httpapi: post a transaction through one content-negotiated endpoint"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 17, 19, 20, 23

### Task 25: Malformed JSON request bodies
**Story:** Story 4 (invalid JSON, missing amount, numeric amount)
**Type:** negative-path

**Steps:**
1. Write failing test: a body that is not valid JSON → `400` `amount_malformed`; a body omitting
   `amount` → `400` `amount_malformed`; a body with `"amount": -42.50` as a JSON **number** → `400`
   `amount_malformed` (accepting a number would put a float on the money path); an unrecognized extra
   field is ignored and the transaction is created. Row count unchanged on each rejection.
2. Verify test fails (RED)
3. Implement by decoding `amount` as a `string` field only, so a JSON number fails to decode.
4. Verify test passes (GREEN)
5. Commit: "httpapi: reject malformed JSON bodies and numeric amounts"

**Files:** `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 24

### Task 26: Page markup — selector, balance, form, list
**Story:** Story 1, Story 2, Story 3 / FR-5
**Type:** happy-path

**Steps:**
1. Write failing test: `GET /` renders `200 text/html` containing all three account names as links;
   `GET /?account=acct-1` renders the balance as `$1,283.50` inside an element with class `balance`,
   a `<form method="post" action="/api/accounts/acct-1/transactions">`, and the transaction rows
   newest first in the same order the JSON route returns. No `account` param renders the first
   account by id ascending. Response contains no `<script>` tag. **FR-4:** an account with no
   transactions renders a `$0.00` balance and an explicit empty-state message in the transaction
   area — not an error and not a blank region.
2. Verify test fails (RED)
3. Implement the template in styleguide layout order — heading → account selector → balance → post
   form → transaction list — and the page handler.
4. Verify test passes (GREEN)
5. Commit: "web: render the account page with selector, balance, form, and log"

**Files:** `web/index.html.tmpl`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 10, 22, 24

### Task 27: Visible error panel, generic fallback, unknown account
**Story:** Story 3 / FR-13 · **Condition C2**, conflict-check **F2**
**Type:** negative-path

**Steps:**
1. Write failing test: `GET /?account=acct-1&error=description_empty` renders the matching message
   inside an element with class `error`, positioned above the form in the output.
   `error=not_a_real_code` renders a **non-empty generic** message, never an empty panel.
   `error=<script>alert(1)</script>` is escaped, with no raw `<script>` in the body.
   `?account=acct-nope` renders the account list and a not-found message **only** — no `balance`
   element, no transaction list, and **no form**.
2. Verify test fails (RED)
3. Implement the error-panel block with a lookup that falls back to a generic message, and the
   unknown-account branch that omits balance, list, and form.
4. Verify test passes (GREEN)
5. Commit: "web: render rejections visibly with a generic fallback for unknown codes"

**Files:** `web/index.html.tmpl`, `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
**Wired-into:** same as Task 21
**Dependencies:** 26

### Task 28: Activate the reserved stylesheet rules
**Story:** Story 3 (legibility criteria)
**Type:** happy-path

**Steps:**
1. Write failing test: `web/style.css` contains active `.balance`, `.error`, and table rules, and
   contains no `@media`, no `prefers-color-scheme`, no `@keyframes`, and no `@font-face`.
2. Verify test fails (RED)
3. Uncomment the reserved block in `web/style.css` — `.balance` at `4rem`/700, `.error` with
   `#fdecea` and a 6px `#b3261e` left border, tables with `border-collapse: collapse`. Values come
   from the styleguide unchanged.
4. Write a failing stylesheet regression assertion in `web/web_test.go` that `html` **or** `:root`
   declares `font-size: 20px` — the assertion must target the root element specifically, because a
   `20px` rule on `body` does not change what `rem` resolves against and is exactly the state that
   shipped unnoticed. Verify it fails (RED).
5. Set the root `rem` basis in `web/style.css` — `html { font-size: 20px }` (or `:root`) — so
   `table { font-size: 1rem }` is 20px, `h1` (`2rem`) is 40px, and `.balance` (`4rem`) is 80px, the
   scale the styleguide's "Body 20px (`1rem` base)" line intends. Verify GREEN.

> **Amended 2026-08-09 by operator decision:** step 4 added. `manual_test` found no `html`
> `font-size` rule, so `1rem` resolves against the browser default of 16px, not the `20px` on `body`
> — `rem` inherits from the root element, never from `body`. That puts `table { font-size: 1rem }`
> at 16px, below the projector legibility floor, and silently shrinks `h1` (`2rem`) and `.balance`
> (`4rem`) too. The styleguide's type scale states "Body 20px (`1rem` base)", so 20px is the intended
> basis and this is a compliance bug, not a change to the design.
4. Verify test passes (GREEN)
5. Commit: "web: activate the balance, error, and table styles"

**Files:** `web/style.css`, `web/web_test.go`
**Wired-into:** none (no new production surface — served by the existing `GET /style.css` route)
**Dependencies:** 27

> **Gate after Batch E:** all five routes serve; page renders in styleguide order; no JavaScript
> anywhere in the rendered output.

### Batch F — Seed and command wiring

### Task 29: Deterministic seed data
**Story:** Story 6 / FR-15
**Type:** happy-path

**Steps:**
1. Write failing test: seeding an in-memory database produces exactly 3 accounts and between 16 and
   24 transactions; the first two accounts carry 8–12 transactions each and the third carries
   **none**; the first account's amounts sum to exactly `128350` cents; every id matches
   `^txn-\d{4}$`; the id set is globally unique and forms one unbroken sequence with no per-account
   restart; all `created_at` values come from the injected clock. Seeding twice into two fresh
   databases yields identical rows.
2. Verify test fails (RED)
3. Implement the seed as literal data — 3 accounts, the first two with 8–12 plausible transactions
   each summing to `128350` cents for the first, the third deliberately empty, fixed timestamps, no
   randomness, no `time.Now()`.
4. Verify test passes (GREEN)
5. Commit: "cmd/server: load deterministic seed data"

> **Amended 2026-08-09 by operator decision (FR-15 reconciliation):** originally "3 accounts, 8–12
> plausible transactions each" and "between 24 and 36 transactions", which is retained above in the
> preceding revision's wording via the FR-15 amendment record. The third account is now seeded empty
> so FR-4's empty state is reachable on stage, and the first account's sum is pinned to `128350`
> cents so Story 1's and the API contract's worked examples hold against seed data. `manual_test`
> kicked this task back twice against the old shape.
>
> **Seeded descriptions must not name anything on the non-goals list.** `manual_test` found the
> current fixture showing "Interest credit", "Transfer fee", "Account fee", and "Automatic transfer"
> (×5) — interest, fees, and transfers are all explicit non-goals, and these would appear on the
> projector in a demo whose whole premise is that those features do not exist. Three of them were
> padding that nets to zero, added only to keep a balance assertion alive. Replace them with
> descriptions drawn from ordinary deposit-account activity, keeping each account's count in range
> and `acct-1`'s sum at exactly `128350` cents.

**Files:** `cmd/server/seed.go`, `cmd/server/seed_test.go`
**Wired-into:** `cmd/server/main.go#seed` (the `seed` subcommand `make seed`/`make reset` already run)
**Dependencies:** 16, 5

### Task 30: Wire serve and seed to the store and clock
**Story:** Story 6 / FR-16
**Type:** infrastructure · integration point

**Steps:**
1. Write failing test: `serve` builds a router against a store opened at `LEDGER_DB_PATH` and honors
   `PORT`; an unknown subcommand exits non-zero naming `serve` and `seed`; a `LEDGER_DB_PATH` whose
   directory does not exist exits non-zero with the path in the message.
2. Verify test fails (RED)
3. Implement: open the database, construct `SystemClock`, pass both into `httpapi.NewRouter`, and
   replace the stub `seed` body with a call to the real loader.
4. Verify test passes (GREEN)
5. Commit: "cmd/server: wire the store and clock into serve and seed"

**Files:** `cmd/server/main.go`, `cmd/server/main_test.go`, `.env.example`, `CLAUDE.md`
**Wired-into:** `cmd/server/main.go#main` (the process entry point dispatching `serve` and `seed`)

> **Amended 2026-08-09 by operator review — build_review gaps `test:build-review-env-scope` and
> `test:build-review-doc-scope`:** `.env.example` and `CLAUDE.md` added to `Files:`. Wiring `serve`
> settled how the port and database path are actually supplied, and both files documented the
> superseded contract — `.env.example:10-13` described per-worktree override files and
> `CLAUDE.md:102-112` described a `.env.local`, when `make` loads neither and the supported form is an
> invocation-time override (`make dev PORT=<port>`, with `LEDGER_DB_PATH` alongside it when a
> worktree needs a distinct database). `build_review` correctly flagged both as changed by a diff no
> `Files:` line assigned.
>
> These belong to this task rather than a separate documentation task: both files document *this
> task's* contract, and the plan skill's documentation boundary forbids splitting documentation that
> accompanies functional work into its own task. Scoping them here describes the diff without
> creating one.
**Dependencies:** 29, 24

### Task 31: File-backed reset determinism
**Story:** Story 6 / FR-15 · conflict-check **F3**
**Type:** happy-path

**Steps:**
1. Write failing test: with `LEDGER_DB_PATH` set to a **`t.TempDir()`** path, run seed, capture all
   rows, delete the file, run seed again, and assert every account row and transaction row is
   identical including `created_at` and ids. Never touches the default `./ledger.db`.
2. Verify test fails (RED)
3. Implement whatever the assertion exposes — expected: nothing beyond Task 29, since determinism is
   already structural.
4. Verify test passes (GREEN)
5. Commit: "cmd/server: assert two resets produce byte-identical data"

**Files:** `cmd/server/seed_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 30

### Task 32: Assert the demo is fully offline
**Story:** Story 6 (renders fully with no network available) · **NFR-2**
**Type:** negative-path

**Steps:**
1. Write failing test: the rendered `GET /` body contains no `http://` or `https://` reference and no
   `<script>` tag; `web/style.css` contains no `@import` and no `@font-face`; and `go.mod` declares
   exactly one non-standard-library requirement (`modernc.org/sqlite`).
2. Verify test fails (RED)
3. Implement whatever the assertion exposes — expected: nothing, since the styleguide already forbids
   webfonts and the page carries no scripts. The test exists so a later change cannot quietly
   introduce an outbound fetch, which would break the demo on a disconnected projector laptop.
4. Verify test passes (GREEN)
5. Commit: "web: assert the page makes no outbound requests"

**Files:** `web/web_test.go`, `internal/httpapi/router_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 28, 31

> **Gate after Batch F:** `make check` clean (`gofmt -l .` empty, `go vet ./...` silent); `make test`
> under 10 seconds and passing with `-count=2`; `grep -rn 'time.Now()'` returns exactly one hit;
> test-to-implementation line ratio at roughly 4:1.

## Task Dependency Graph

```
Batch A (domain shape, no dependencies between 1/2/3)
  1  clock                      (independent)
  2  domain types               (independent)
  3  sentinels                  (independent)
  4  Store interface            ← 2, 3

Batch B (persistence, strictly linear)
  5  schema                     ← 4
  6  accounts                   ← 5
  7  append + global count      ← 6
  8  newest-first ordering      ← 7
  9  empty / unknown reads      ← 8

Batch C (domain ops; 11→18 linear because each adds a rule to one function)
  10 derived balance            ← 4
  11 amount zero                ← 10
  12 description empty          ← 11
  13 description too long       ← 12
  14 unknown account            ← 13
  15 balance below zero         ← 14
  16 global sequential id       ← 15
  17 valid transaction          ← 16
  18 rule-semantics table       ← 17

Batch D (HTTP reads)
  19 money parsing              ← 3          (parallel with Batch B/C)
  20 sentinel → code            ← 3, 19
  21 GET /api/accounts          ← 10, 20
  22 GET .../transactions       ← 8, 9, 21
  23 method / path edges        ← 22

Batch E (posting and page)
  24 POST both branches         ← 17, 19, 20, 23
  25 malformed JSON bodies      ← 24
  26 page markup                ← 10, 22, 24
  27 error panel + unknown acct ← 26
  28 stylesheet rules           ← 27

Batch F (seed and wiring)
  29 seed data                  ← 16, 5
  30 wire serve + seed          ← 29, 24
  31 reset determinism          ← 30
  32 fully-offline assertion    ← 28, 31
```

Acyclic. Tasks 1, 2, 3 have no dependencies and may run in any order; Task 19 depends only on Task 3,
so Batch D's parsing work can proceed alongside Batches B and C.

## Integration Points

- **After Task 9** — the store satisfies `ledger.Store` end to end; persistence is testable alone.
- **After Task 18** — the domain is complete and fully covered with no database involved.
- **After Task 24** — the first point the whole stack is exercised: HTTP → parse → domain → store →
  response, through both content types.
- **After Task 28** — the page is complete; the demo is visually operable against a hand-seeded
  database.
- **After Task 31** — `make reset` + `make dev` is the documented boot sequence and works.

## Conditions Tracked From Architecture Review

| Condition | Tasks that satisfy it |
|---|---|
| C1 — ids numbered globally, not per account | 7 (global count), 16 (`txn-0006` from 5 existing rows across two accounts), 29 (unbroken seeded sequence) |
| C2 — unrecognized `error` renders a generic message, never an empty panel | 27 |
| C3 — all ids share one width | 8 (ordering under equal timestamps), 16 and 29 (`^txn-\d{4}$`) |

## Coverage Mapping

| Story | Requirements | Tasks |
|---|---|---|
| 1 — account and balance | FR-1, 2, 4, 10 | 2, 6, 10, 21, 26 |
| 2 — stable newest-first log | FR-3, 11 | 8, 9, 22, 23, 26 |
| 3 — record from the page | FR-5, 6, 7, 13 | 19, 24, 26, 27, 28 |
| 4 — record programmatically | FR-8, 9, 14 | 20, 24, 25 |
| 5 — six rules reject | FR-12a–f | 11, 12, 13, 14, 15, 17, 18, 19 |
| 6 — reset and run | FR-15, 16 | 1, 29, 30, 31, 32 |

Coherence-check verified this mapping in both directions against the story and plan files and closed
two holes it found: FR-4's page-level empty-state assertion was added to Task 26, FR-7's
reload-does-not-double assertion was added to Task 24, and NFR-2 (fully offline) gained Task 32,
which had no covering task at all.

Every happy-path and negative-path criterion maps to at least one task. Negative paths are their own
tasks (9, 11–15, 18, 23, 25, 27), never folded into a cleanup task. There is no terminal catch-all
validation task — the batch gates plus `/manual-test` and `/prd-audit` cover the assembled feature.

## Verification

- [ ] All happy path criteria covered by at least one task
- [ ] All negative path criteria covered by at least one task
- [ ] No task exceeds 5 minutes of work
- [ ] Dependencies are explicit and acyclic
- [ ] Every task carries a `**Wired-into:**` line
- [ ] No task targets another feature's sealed artifact
- [ ] No task introduces anything from the non-goals list
