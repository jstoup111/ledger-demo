# Track: csv-export-single-account

Track: product

> **Amended 2026-08-13 by operator:** The approved product surface is now a visible CSV download
> control for the selected account, served by extending the existing transaction-listing route's
> response behavior. This supersedes the command-line delivery described below. The scope remains
> one account and one raw four-column document, with no sixth route, CLI export, JavaScript, filters,
> date ranges, multi-account export, summaries, statements, or other reporting. The operator chose
> this browser-only approach to keep the live full loop within 30–40 minutes.

Scope boundary: One visible CSV download control for the valid selected account, using the existing
transaction-listing HTTP surface; no sixth route, CLI export, JavaScript, filters, date ranges,
multi-account export, summaries, statements, or other reporting.

The change adds a capability a person invokes and reads. A presenter types one command and gets a
table of ledger data back in a format an audience already understands; the acceptance signal is what
that output contains and what order it is in, not how it is produced internally. That is a user-facing
requirement judged by observable output, so `/prd` runs and the FRs are stated as properties of the
emitted document and of the command's exit behavior.

It is deliberately a *narrow* product surface: one account, one destination (standard output), no
options. The PRD is short because the requirement is short.

## Discovery notes (ephemeral — not a design doc)

Verified by reading the code at the branch point, not assumed:

- `cmd/server/main.go` dispatches exactly two subcommands, `serve` and `seed`, through
  `run(command string) error`, and `main()` reads only `os.Args[1]`. Passing an account id therefore
  requires the dispatcher to accept arguments.
- `store.Transactions(accountID)` already resolves the account first and returns
  `ledger.ErrAccountNotFound` wrapped with the requested id (`account %q: ...`) before it reads any
  transaction row. The unknown-account outcome is therefore obtainable without a new sentinel error
  and without any chance of a partially-written document.
- That same call already orders rows `created_at DESC, id DESC`, and it is the identical call the
  JSON listing handler and the HTML page both make. Order agreement across the three surfaces is
  structural if the export reads through the same interface.
- Both existing surfaces render the recorded time as `CreatedAt.UTC().Format(time.RFC3339)` and
  expose the amount as an integer field named `amount_cents`. Matching those two conventions is what
  makes the three surfaces agree rather than merely coincide.
- `encoding/csv` is in the standard library, so the single pinned dependency is unchanged.
- One existing assertion reads the unknown-subcommand message and requires it to name every valid
  command (`cmd/server/main_test.go`, the `[]string{"serve", "seed"}` loop). Adding a third command
  moves that assertion. It is the only existing assertion this feature moves.

## Approaches weighed

The intake issue offered two candidates. Both were evaluated on merits; neither was adopted just
because it was written down.

1. **A `cmd/server` subcommand writing to standard output.** No new route, no new HTTP surface, no
   template, no schema change. Testable at two levels that already exist in this repo: an in-memory
   store unit test for the document itself, and the acceptance package's real-binary harness for the
   command. Reuses the existing `LEDGER_DB_PATH` convention that `serve` and `seed` already follow.
2. **A sixth HTTP route serving `text/csv`.** Rejected. The route count is an active, in-flight
   constraint elsewhere in this repository: a feature currently in BUILD carries a non-functional
   requirement fixing the HTTP surface at exactly five routes, and adding a sixth while that work is
   unmerged would fail its audit at ship time. The route form also buys nothing the subcommand does
   not already give a presenter, and it would put a data-extraction surface on the demo's public
   surface area, which is the opposite of narrow.
3. **A third option, not in the issue: a `make export` target wrapping the subcommand.** Considered
   and dropped as unnecessary — the two existing Make targets exist to make *stateful* operations
   (reset, dev) single actions. Export takes an argument that varies per invocation, which a Make
   target obscures rather than simplifies. The subcommand is already one command.

**Chosen: approach 1.** This is settled and recorded here so it is not reopened downstream.

> **Amended 2026-08-13 by operator:** Approach 1 above is superseded. The chosen approach reuses the
> existing account-transactions GET route for an explicitly requested CSV representation and adds a
> visible download control to the selected-account page. A separate sixth route was rejected because
> the five-route contract is still governing; retaining both the CLI and browser delivery was rejected
> because it adds a second invocation surface and a shared rendering seam without improving the live
> demo outcome.
