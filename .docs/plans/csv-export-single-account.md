# Implementation Plan: One Account's Transactions as CSV

**Date:** 2026-08-09
**Design:** `.docs/specs/2026-08-09-csv-export-single-account.md` (Approved)
**Stories:** `.docs/stories/csv-export-single-account.md` (accepted stories)
**Tier:** S (`.docs/complexity/csv-export-single-account.md`)
**Conflict check:** Skipped — Tier S (see the complexity record)

## Summary

**Three tasks.** Add a CSV renderer over the existing store interface, wire it into the existing
subcommand dispatcher as `export <account-id>` writing to standard output, and prove end to end that
the exported rows agree with the JSON listing and that repeated exports are byte-identical. No new
route, no schema change, no domain change, no new dependency.

## Technical Approach

**One new production file and one edited one.** `cmd/server/export.go` holds the renderer;
`cmd/server/main.go` gains one dispatch case. Nothing under `internal/` changes, `web/` is untouched,
and the router is not opened.

**The renderer's shape.**

```go
func writeAccountCSV(w io.Writer, store ledger.Store, accountID string) error
```

It takes the domain interface (NFR-4) and an `io.Writer` (so unit tests use a buffer and the command
passes standard output). It does exactly three things, in this order:

1. `transactions, err := store.Transactions(accountID)`; on error, return it unchanged. **This call
   comes before the first byte is written, and that ordering is the whole of FR-5.** Verified by
   reading `internal/store/sqlite.go`: `Transactions` resolves the account first and returns
   `fmt.Errorf("account %q: %w", id, ledger.ErrAccountNotFound)` before it queries any row — so the
   error already names the requested identifier and already satisfies identity comparison against the
   sentinel. No new sentinel, and no window in which a header could be emitted for a missing account.
2. Write the header record `id,amount_cents,description,created_at` — the four field names the JSON
   listing already publishes for these values (PRD assumption A3).
3. Write one record per transaction: `transaction.ID`,
   `strconv.FormatInt(transaction.Amount, 10)`, `transaction.Description`, and
   `transaction.CreatedAt.UTC().Format(time.RFC3339)`. Then `Flush` and return the writer's error.

**Why the ordering and the formats are not choices to be re-made.** The store's query is already
`ORDER BY created_at DESC, id DESC`, and it is the *same call* the JSON handler and the page handler
make — so FR-4's three-way order agreement is structural, not something the renderer implements. The
two formatting expressions above are copied verbatim from `internal/httpapi/router.go`'s
`transactionResponse` construction (`AmountCents` as the raw `int64`, `CreatedAt.UTC().Format(time.RFC3339)`).
Reusing them, rather than inventing equivalents, is what makes the surfaces agree byte for byte.
`strconv.FormatInt` keeps NFR-3 (no float type, no float formatting) trivially true.

**Standard-library CSV defaults are used as-is** (PRD assumption A5): `encoding/csv` separates records
with `\n` (its `UseCRLF` field defaults to false) and quotes only fields that need it, which is both
deterministic (FR-7) and correct for a description containing a separator or a quote. No `Comma` and
no `UseCRLF` is set — configuring quoting is an explicit non-goal.

**The dispatcher change, and the one existing assertion it moves.** `run` currently has the signature
`run(command string) error` and `main()` reads only `os.Args[1]`. It becomes:

```go
func run(command string, args ...string) error
```

**Variadic on purpose.** Every existing call site in `cmd/server/main_test.go` is `run("serve")`,
`run("seed")`, or `run("frobnicate")`, and all of them keep compiling unchanged — verified by reading
the file. A `run(args []string)` signature would have forced edits to six existing test call sites for
no benefit. `main()` passes `os.Args[2:]...`.

The `export` case requires **exactly one** argument, opens the database through the same
`env("LEDGER_DB_PATH", defaultDBPath)` convention `serve` and `seed` already use, and writes through a
package-level `var stdout io.Writer = os.Stdout`. That seam mirrors the `var listenAndServe = http.ListenAndServe`
variable already in this file, and it exists so a dispatcher-level test can capture output without
polluting the test binary's own standard output.

The default branch's message becomes `unknown command %q (want: serve, seed, export)`. That breaks
exactly one existing assertion — the `for _, command := range []string{"serve", "seed"}` loop in
`TestRunRejectsUnknownCommandWithoutStartingServer` — and Task 2 updates it in the same commit to
`{"serve", "seed", "export"}` rather than letting the builder meet it cold. Grep confirms this string
appears in exactly one production location and is asserted in exactly one test.

**Sequencing.** Task 1 is the document, provable in isolation against an in-memory store. Task 2 is the
wiring, which needs Task 1's function to exist. Task 3 is the cross-surface agreement and determinism
spec, which needs the real binary to have the subcommand. Strictly linear; there is no parallelism to
exploit in three tasks this small.

**Why Task 3 is not a catch-all.** It proves two specifically named requirements that are only
observable across process and surface boundaries — FR-4's agreement between the export and the JSON
listing for the same database, and FR-7's byte-identical repeat — plus FR-5's exit code, which a
buffer-level test cannot see. It is scoped to those; it is not a "verify the feature and fix whatever
turns up" step, and it adds no implementation.

## Prerequisites

- None. No migration, no dependency change, no seed change. `go.mod` keeps its single pinned
  requirement — `encoding/csv` and `strconv` are standard library.

## Tasks

### Task 1: Render one account's transactions as a CSV document

**Story:** Story 1 — the header row, one row per transaction, integer cents, matching recorded times;
Story 2 — the unknown-account failure writes zero bytes
**Type:** happy-path

**Steps:**
1. Write failing test: create `cmd/server/export_test.go` with a table-driven test over an in-memory
   store (`store.Open(":memory:")`, matching the existing in-memory convention) and a
   `bytes.Buffer`, asserting the **exact full document** for three fixtures: an account with a positive
   and a negative transaction, an account with a description containing a comma and a quotation mark,
   and an existing account with no transactions (header line only). Assert the header line is exactly
   `id,amount_cents,description,created_at`; assert the amount fields are exactly `158579` and `-6450`;
   assert the recorded-time field equals `CreatedAt.UTC().Format(time.RFC3339)` for the fixture
   instant; assert the line count after the header equals the transaction count. Add two negative
   cases: an unknown account id must return an error satisfying `errors.Is(err, ledger.ErrAccountNotFound)`
   whose message contains the requested id, **and** must leave `buf.Len() == 0`; and rendering the same
   store twice must produce byte-identical buffers.
2. Verify test fails (RED) — `writeAccountCSV` does not exist.
3. Implement `cmd/server/export.go`: `writeAccountCSV(w io.Writer, store ledger.Store, accountID string) error`
   doing exactly the three steps in the Technical Approach — store read first and returned unchanged on
   error, then `csv.NewWriter(w)`, the header record, one record per transaction with
   `strconv.FormatInt(transaction.Amount, 10)` and `transaction.CreatedAt.UTC().Format(time.RFC3339)`,
   then `Flush` and return `writer.Error()`. Set no `Comma` and no `UseCRLF`. Use no float type.
4. Verify test passes (GREEN). Confirm the parameter type is `ledger.Store`, not `*store.SQLite`, and
   that the file imports no `internal/store` symbol other than what the test needs.
5. Commit with message: "feat: render one account's transactions as CSV"

**Files likely touched:**
- `cmd/server/export.go` — the renderer (new)
- `cmd/server/export_test.go` — document, quoting, empty-account, unknown-account, determinism cases (new)

**Wired-into:** nothing yet — the renderer is unreachable from `main` until Task 2. This is deliberate
and is closed by Task 2 in the same batch.

**Dependencies:** none

---

### Task 2: Dispatch `export <account-id>` to standard output

**Story:** Story 2 — the unknown-subcommand message names all three commands; argument-count and
database-path failures write nothing; the existing subcommands are unchanged
**Type:** negative-path

**Steps:**
1. Write failing test: in `cmd/server/main_test.go`, add tests that (a) `run("export", "acct-1")`
   against a seeded temporary database writes a document whose first line is the header and whose
   second line names the newest transaction, capturing output by swapping the package-level `stdout`
   variable for a buffer and restoring it with `t.Cleanup`; (b) a table-driven case over zero arguments
   and two arguments returns an error mentioning that exactly one account id is expected, with an empty
   buffer; (c) `run("export", "acct-nope")` returns an error naming the id with an empty buffer; and
   (d) an unopenable `LEDGER_DB_PATH` is reported with the path named and no file created, mirroring
   `TestSeedReportsMissingParentDatabaseDirectory`. **Update the existing assertion** in
   `TestRunRejectsUnknownCommandWithoutStartingServer` from `[]string{"serve", "seed"}` to
   `[]string{"serve", "seed", "export"}`.
2. Verify test fails (RED) — there is no `export` case and the message names two commands.
3. Implement in `cmd/server/main.go`: change `run(command string) error` to
   `run(command string, args ...string) error`; have `main()` pass `os.Args[2:]...`; add
   `var stdout io.Writer = os.Stdout`; add `case "export": return export(args)`; and change the default
   message to `unknown command %q (want: serve, seed, export)`. Write `export(args []string) error` in
   `cmd/server/export.go`: reject any argument count other than one with a message stating the
   expectation, open `env("LEDGER_DB_PATH", defaultDBPath)` with the same
   `fmt.Errorf("open database %q: %w", ...)` wrapping `serve` and `seed` use, `defer database.Close()`,
   and return `writeAccountCSV(stdout, database, args[0])`. Add no clock and no `time.Now()` call.
4. Verify test passes (GREEN). Confirm every other existing test in `cmd/server` passes **unmodified** —
   the serve port/database tests, the two seed determinism tests, and the seed missing-directory test.
5. Update the one factual line in `README.md` that describes the entry point's subcommands so it names
   all three.
6. Commit with message: "feat: add the export subcommand writing one account's CSV to stdout"

**Files likely touched:**
- `cmd/server/main.go` — variadic `run`, the `export` case, the `stdout` seam, the updated message
- `cmd/server/export.go` — the `export(args []string)` command function
- `cmd/server/main_test.go` — export dispatch, argument-count, unknown-account, database-path cases;
  the updated valid-command assertion
- `README.md` — the entry-point line naming the subcommands

**Wired-into:** `main()` → `run` → `case "export"` → `export` → `writeAccountCSV`, reachable by running
the built binary as `ledger-server export <account-id>`. This closes Task 1's deliberate gap.

**Dependencies:** Task 1

---

### Task 3: Prove the export agrees with the JSON listing and repeats byte for byte

**Story:** Story 1 — the export and the programmatic listing agree element for element, and repeated
exports are byte-identical; Story 2 — the unknown account exits non-zero with empty standard output
**Type:** negative-path

**Steps:**
1. Write failing test: add a spec to `test/acceptance/ledger_acceptance_test.go` driving the real
   compiled binary via the package's existing `serverBin` and `seedDB` helpers into a `t.TempDir()`
   database (never the default demo file). Add a small `exportCSV(t, dbPath, accountID)` helper beside
   the existing `seedDB` in `test/acceptance/harness_test.go` that runs
   `exec.Command(serverBin, "export", accountID)` with `LEDGER_DB_PATH` set, capturing standard output
   and standard error separately and returning them with the exit error. Then assert: (a) for the first
   seeded account, parsing the CSV with `encoding/csv` and comparing against
   `GET /api/accounts/{id}/transactions` on a server started over the same database yields the same
   identifiers in the same sequence, the same `amount_cents` values, and the same `created_at` strings,
   element for element; (b) exporting the same account twice returns byte-identical standard output;
   (c) the seeded account with no transactions exports the header line alone with a zero exit; and
   (d) exporting an unknown account exits **non-zero**, names the requested id on standard error, and
   leaves standard output exactly zero bytes long.
2. Verify test fails (RED) if the orders diverge, the bytes differ, or the failure path emits anything.
3. Implement: no production change expected — the store's single ordered query and the formats reused
   in Task 1 should already satisfy this. If any assertion fails, fix the renderer or the dispatcher
   rather than relaxing the assertion.
4. Verify test passes (GREEN). Confirm the suite has no `time.Sleep`, still passes with `-count=2`, and
   still completes under ten seconds; confirm the router still exposes exactly five routes and every
   existing page and listing assertion passes unmodified.
5. Commit with message: "test: assert the CSV export agrees with the JSON listing and repeats exactly"

**Files likely touched:**
- `test/acceptance/harness_test.go` — the `exportCSV` helper beside `seedDB`
- `test/acceptance/ledger_acceptance_test.go` — agreement, determinism, empty-account, and
  unknown-account specs

**Wired-into:** nothing new — this task drives the entry point Task 2 wired, through the real binary.

**Verify-only:** yes

**Dependencies:** Task 2

---

## Task Dependency Graph

```
Task 1 (CSV renderer over the store interface)
   └─▶ Task 2 (export subcommand + updated valid-command assertion + README line)
          └─▶ Task 3 (agreement with the JSON listing, determinism, non-zero exit)
```

Linear and acyclic. Three tasks, one root.

## Integration Points

- **After Task 2:** the feature is end-to-end usable — `make reset` then
  `LEDGER_DB_PATH=./ledger.db go run ./cmd/server export acct-1` prints the document. This is the point
  a presenter can rehearse with it, and the point PRD assumptions A2, A3, and A5 (the command word, the
  header names, and the default CSV punctuation) become confirmable by looking at one invocation.
- **After Task 3:** the three surfaces are asserted to agree and the failure path is asserted to be
  silent, over the real binary.

## Out of scope for BUILD

- No `Makefile` target. The export takes an argument that varies per invocation; the existing targets
  exist to make stateful operations single actions, and wrapping a parameterised command in one would
  obscure rather than simplify it. Decided in `/explore`.
- No new decision record. The Governing Decisions table in the PRD records the conformance check
  against all eight accepted decisions and none is amended, so there is nothing to write.

## Verification

- [ ] All happy path criteria covered by at least one task — Story 1 → Tasks 1, 3; Story 2 → Task 2
- [ ] All negative path criteria covered by at least one task — Story 1 → Tasks 1, 3; Story 2 →
      Tasks 2, 3
- [ ] Exactly three tasks; no task exceeds 5 minutes of work
- [ ] Dependencies are explicit and acyclic
- [ ] No terminal catch-all validation task — Task 3 is scoped to FR-4, FR-5, and FR-7 and adds no
      implementation
- [ ] Every task carries a `**Wired-into:**` line, and Task 1's deliberate gap is closed by Task 2
- [ ] The one existing assertion this feature moves is named and updated inside Task 2, not discovered
      during BUILD
- [ ] `gofmt` clean, `go vet` clean, suite under ten seconds, green on a repeated run, no `time.Sleep`,
      no new dependency, no float type or float formatting anywhere
- [ ] The five existing routes are unchanged in count and shape; no route serves CSV
- [ ] No test touches the default demo database file
- [ ] `time.Now()` still appears exactly once in the repository

## Coverage Mapping

| FR | Story | Task(s) |
|---|---|---|
| FR-1 | Story 1 | 2 |
| FR-2 | Story 1 | 1 |
| FR-3 | Story 1 | 1 |
| FR-4 | Story 1 | 1, 3 |
| FR-5 | Story 2 | 1, 2, 3 |
| FR-6 | Story 1 | 1, 3 |
| FR-7 | Story 1 | 1, 3 |
| FR-8 | Story 2 | 2, 3 |
