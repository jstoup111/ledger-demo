# Implementation Plan: Acceptance Harness Hygiene

**Date:** 2026-08-09
**Design:** none — technical track, no PRD (`.docs/track/acceptance-harness-hygiene.md`)
**Stories:** `.docs/stories/acceptance-harness-hygiene.md` (accepted stories)
**Conflict check:** Skipped — Tier S (`.docs/complexity/acceptance-harness-hygiene.md`). Three
stories owning disjoint concerns in one file, no shared state, no ordering relationship.
**Tier:** S (`.docs/complexity/acceptance-harness-hygiene.md`)
**Governing ADR:** `.docs/decisions/adr-2026-08-09-bounded-yield-in-test-readiness-probes.md` (Accepted)

> **Filename note:** this plan is `acceptance-harness-hygiene.md`, not the skill's default
> `YYYY-MM-DD-<feature>.md`, because the plan stem must match
> `.docs/complexity/acceptance-harness-hygiene.md` for the daemon to resolve the complexity tier at
> build time — the same reason `.docs/plans/base-ledger.md` is named as it is.

> **Ownership note:** `.docs/plans/base-ledger.md` records these defects and states both are
> "queued for a separate pass". This plan **is** that pass, so it takes ownership of
> `test/acceptance/harness_test.go` deliberately rather than by accident. No production behavior is
> re-opened, and the acceptance specs in `test/acceptance/ledger_acceptance_test.go` are not touched.

## Summary

Repairs three verified defects in the acceptance suite's own scaffolding in **11 tasks**, all inside
`test/acceptance/harness_test.go`. No production file changes, no dependency is added, and every
existing helper signature is preserved so the 13 call sites in `ledger_acceptance_test.go` compile
untouched.

## Technical Approach

Three defects, three mechanisms, one file. They are sequenced readiness → lifetime → port, because the
port retry (Story 3) consumes the error-returning readiness introduced for Story 1 and the child
bookkeeping introduced for Story 2.

**1. One `child` value owns each process, and exactly one goroutine calls `Wait`.**
This is the subtlest part of the change and the easiest to get wrong. Today `stop` calls
`cmd.Process.Kill()` then `cmd.Wait()`. Readiness now also needs to *observe* the child exiting, and a
second `cmd.Wait()` returns `exec: Wait was already called`. The fix is a single waiter goroutine that
publishes its result through a **closed channel** rather than a sent value, so any number of readers
can observe the exit:

```go
type child struct {
    cmd     *exec.Cmd
    logs    *safeBuffer
    exited  chan struct{} // closed by the sole waiter goroutine, after waitErr is set
    waitErr error         // written before close(exited); read only after <-exited
    once    sync.Once     // guards stop
}

// started immediately after cmd.Start():
go func() {
    c.waitErr = c.cmd.Wait()
    close(c.exited)
}()
```

Writing `waitErr` before `close` and reading it only after receiving from the closed channel is
correctly ordered by the channel's happens-before edge, so `-race` stays clean. A buffered
send-once channel would deadlock the second reader; that is why this is a close, not a send.

**2. Readiness polls on a ticker and watches three things, returning an error instead of failing.**
`waitReady` becomes a pure function returning `(attempts int, err error)` so Story 3 can *classify* the
failure and retry. It dials first, then blocks in a `select` over the deadline, the child's `exited`
channel, and a `time.Ticker` — so an already-listening server returns on attempt 1, and a child that
dies is reported immediately with its own output instead of after the 15-second ceiling.
`readyPollInterval = 25 * time.Millisecond` against measured 32–123 ms startups gives 2–5 attempts per
start (baseline: ~720). Per the governing ADR this is a bounded yield, not a `time.Sleep`; no literal
`time.Sleep` is introduced anywhere.

**3. Attempt counts are recorded so "it yields" is assertable.**
A package-level `atomic.Int64` accumulates attempts across the package, and `waitReady`'s return value
gives per-start counts. Without this, "does not busy-wait" is unfalsifiable and would silently
regress.

**4. Termination is registry-driven so a signal can do what `t.Cleanup` cannot.**
`t.Cleanup` never runs when the binary dies by signal (verified: stdlib `testing` imports `os/signal`
nowhere). A mutex-guarded package registry of live children plus a `signal.Notify` handler in
`TestMain` closes that branch. `stop` keeps its `sync.Once` and its `t.Cleanup` registration, so the
normal path is unchanged; it additionally untracks. `terminateAll` kills and reaps every tracked child
and is a safe no-op over already-dead ones because it waits on the closed `exited` channel.

**5. The port race is retried, not eliminated, and the retry is made deterministically testable.**
`freePort`'s window cannot be closed from the test side — and it is wider than it looks: it reserves
`127.0.0.1:0` while the server binds `":"+PORT` on all interfaces, so the reservation never proved the
child's actual bind was free. Rather than perfect an unperfectable probe, `startServer` gains a
**port-source seam** (`func() string`) and a bounded 3-attempt retry that re-picks on
`address already in use` *and only on that*. Any other startup failure — an unopenable database, say —
fails immediately, so a retry loop can never mask a genuine defect. The seam is what makes the race
reproducible in a test with no sleeps and no reliance on real contention.

**6. The public helper surface is frozen.** `startServer(t, dbPath) (base string, stop func())`,
`newApp`, `newAppAt`, and `seedDB` keep their exact signatures. New capability arrives as new
lower-level functions the existing ones call.

**Layering:** `tryStartServer` returns `(base, stop, []attempt, error)` and holds all logic;
`startServerPorts` wraps it with `t.Fatalf`; `startServer` calls `startServerPorts` with the real port
source. Tests assert against `tryStartServer`'s error directly — that is how a "the harness fails
correctly" criterion gets tested without failing the test that asserts it.

**Explicitly not in scope:** nothing on the project's non-goal list is approached. No dedup, no
idempotency key, no uniqueness constraint, no dedup window — retrying a *port* replays no HTTP request
and posts no transaction twice. The five routes, the domain, the schema, the seed data, the page, and
`go.mod` are untouched. Per the ADR, the SIGKILL residual is documented, not fixed by editing the
production server.

## Prerequisites

None. All imports are stdlib and all but `os/signal`, `sync/atomic`, and `syscall` are already
imported by the file. `syscall` is used only for the `syscall.SIGTERM` constant, which exists on every
Go platform, so the file still compiles everywhere.

## Tasks

### Batch A — Readiness yields (Story 1)

### Task 1: Ticker-based readiness returning attempts and an error
**Story:** Story 1 — "uses a deadline plus an interval channel"; "returns as soon as the port accepts"
**Type:** infrastructure

**Steps:**
1. Write failing test `TestWaitReadyPollsOnAnInterval`: hold a listener on a port, call `waitReady`
   against it, and assert it returns `attempts == 1` with a nil error (already-listening ⇒ one dial).
   Then, for a port nothing listens on, start a goroutine that begins listening after a short
   ticker-driven delay and assert the returned attempt count satisfies the interval-bounded ceiling
   `attempts <= int(elapsed/readyPollInterval)+2` — the amended Story 1 assertion.
2. Verify test fails (RED) — `waitReady` currently returns nothing and calls `t.Fatalf`.
3. Implement: add `readyPollInterval = 25 * time.Millisecond`; change the signature to
   `waitReady(addr string, c *child) (attempts int, err error)`. Dial first, then `select` over
   `deadline.C` (a `time.NewTimer(readyTimeout)`) and `ticker.C` (a
   `time.NewTicker(readyPollInterval)`), both `defer`-stopped. Return a wrapped error on deadline
   carrying the address, `readyTimeout`, and `c.logs.String()`. Keep `readyTimeout` at 15s.
4. Verify test passes (GREEN).
5. Commit: "acceptance: poll readiness on an interval instead of busy-waiting"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** none

### Task 2: Readiness fails immediately when the child exits
**Story:** Story 1 negative path — "exits immediately … fails as soon as the exit is observed"
**Type:** negative-path

**Steps:**
1. Write failing test `TestWaitReadyDetectsChildExit`: start the built server binary with
   `LEDGER_DB_PATH` pointing at a path inside a directory that does not exist (so `store.Open` fails
   and the child exits non-zero within milliseconds), wrap it in a `child`, and assert `waitReady`
   returns a non-nil error whose message contains the child's own captured output. Assert on **message
   content, not elapsed time**, per the governing ADR's ban on duration-dependent assertions.
2. Verify test fails (RED) — readiness currently watches only the clock and would wait 15s.
3. Implement: introduce the `child` struct exactly as shown in Technical Approach §1 — a single waiter
   goroutine setting `waitErr` then `close(c.exited)`. Add `case <-c.exited:` to `waitReady`'s
   `select`, returning an error naming the address, `c.waitErr`, and `c.logs.String()`.
4. Verify test passes (GREEN).
5. Commit: "acceptance: report a server that exits before it listens"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 1

### Task 3: Record readiness attempts for assertion
**Story:** Story 1 — per-start interval-bounded ceiling and the package-wide `< 500` tripwire
**Type:** infrastructure

**Steps:**
1. Write failing test `TestReadinessAttemptsStayBounded`: read the package counter, call `newApp(t)`
   (a real start through the real path), read it again, and assert the delta satisfies the
   interval-bounded ceiling for that start's elapsed time. Add `TestReadinessAttemptTotal` asserting
   the package total stays under 500 after the suite's starts (baseline 6,470; measured expectation
   ~45).
2. Verify test fails (RED) — no counter exists.
3. Implement: add `var readinessAttempts atomic.Int64`; have the single place that calls `waitReady`
   do `readinessAttempts.Add(int64(attempts))` on every outcome, success or failure, so a regression
   cannot hide in the failure path. Import `sync/atomic`.
4. Verify test passes (GREEN).
5. Commit: "acceptance: record readiness dial attempts so yielding is assertable"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 1

### Task 4: Correct the package header and pin the no-sleep rule
**Story:** Story 1 — header "cites that ADR"; `grep -rn 'time.Sleep' .` returns zero hits
**Type:** refactor

**Steps:**
1. Write failing test `TestNoLiteralTimeSleep`: walk the repository's `.go` files from the package
   directory (using `filepath.WalkDir` over `../..`, skipping `.git`) and assert no file contains the
   literal string `time.Sleep`. This makes the convention mechanically enforced rather than asserted
   in a comment.
2. Verify test fails only if a sleep exists (it must pass immediately — confirm by temporarily adding
   one, seeing RED, then removing it, so the test is proven non-tautological rather than assumed).
3. Implement: rewrite the package header's no-sleep bullet to state the scoped rule and cite
   `.docs/decisions/adr-2026-08-09-bounded-yield-in-test-readiness-probes.md`, replacing the current
   "no time.Sleep anywhere (NFR-4) … Readiness therefore uses a deadline-bounded dial loop" wording
   that encoded the falsified literal reading. Record the SIGKILL residual limitation here too.
4. Verify test passes (GREEN).
5. Commit: "acceptance: scope the no-sleep rule to its purpose per ADR"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 1

### Batch B — Children die with the test binary (Story 2)

### Task 5: Child registry with a terminate-all entry point
**Story:** Story 2 happy path — "terminate-all … the port it held is immediately re-bindable"
**Type:** infrastructure

**Steps:**
1. Write failing test `TestTerminateAllKillsALiveServer`: start a real server via the harness, capture
   its port, call `terminateAll()`, then assert the process is gone by **successfully binding a
   listener on that port** — proving reaping, not just signalling. Close the listener.
2. Verify test fails (RED) — no registry and no `terminateAll` exist.
3. Implement: add a package-level `var live = struct{ mu sync.Mutex; set map[*child]struct{} }{...}`
   with `trackChild`, `untrackChild`, and `terminateAll`. `terminateAll` snapshots the set under the
   mutex, releases it, then for each child calls `Process.Kill()` and waits `<-c.exited`. Never hold
   the mutex while waiting, or a concurrent `untrackChild` deadlocks the handler.
4. Verify test passes (GREEN).
5. Commit: "acceptance: track live server children and add terminateAll"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** none

### Task 6: Terminate-all is exhaustive and safe over dead children
**Story:** Story 2 — "**every** tracked child"; "already stopped … a no-op that neither panics,
blocks, nor reports an error"; the `-race` criterion
**Type:** negative-path

**Steps:**
1. Write failing test `TestTerminateAllIsExhaustive`: start three servers, call `terminateAll()` once,
   assert all three ports re-bind. Add `TestTerminateAllOverStoppedChildIsNoOp`: start one, call its
   `stop`, then call `terminateAll()` and assert it returns without panic and the registry is empty.
   Add `TestRegistryIsRaceFree`: start and stop servers from several goroutines while a goroutine
   calls `terminateAll`, so `-race` exercises the registry.
2. Verify test fails (RED).
3. Implement: make `stop` call `untrackChild` inside its existing `sync.Once`, so a stopped child
   leaves the registry exactly once. Confirm `terminateAll` tolerates a closed `exited` and a
   `Process.Kill()` on a finished process (both return errors that must be discarded, not reported).
4. Verify test passes (GREEN), including `go test ./test/acceptance/ -race`.
5. Commit: "acceptance: make terminateAll exhaustive and safe over reaped children"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 5

### Task 7: Wire interrupt and terminate signals in TestMain
**Story:** Story 2 negative path — the branch that bypasses `t.Cleanup` today
**Type:** negative-path

**Steps:**
1. Write failing test `TestSignalHandlerIsInstalled`: assert the handler is installed before any
   server can start, by checking the package-level flag `signalHandlerReady` (set by `TestMain` after
   `signal.Notify` returns) is true. Delivering a real SIGINT to the test process cannot be asserted
   from inside it — it would kill the run — so the *effect* is covered by Tasks 5–6 against
   `terminateAll` and the *wiring* is covered here.
2. Verify test fails (RED).
3. Implement: in `TestMain`, before `m.Run()`, create `sigs := make(chan os.Signal, 1)`, call
   `signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)`, set `signalHandlerReady = true`, and start a
   goroutine: on receipt, call `terminateAll()`, remove the temp build dir, then `os.Exit(1)`.
   Register it before the build step so an interrupt during `go build` is also handled. Import
   `os/signal` and `syscall`.
4. Verify test passes (GREEN).
5. Commit: "acceptance: kill server children when the test binary is signalled"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 5, 6

### Batch C — A lost port race is retried (Story 3)

### Task 8: Port-source seam and an error-returning start path
**Story:** Story 3 — "when a test supplies a sequence of ports … consumes them in order"
**Type:** infrastructure

**Steps:**
1. Write failing test `TestStartServerConsumesSuppliedPortsInOrder`: build a port source returning a
   recorded sequence, call `tryStartServer` with it, and assert the returned base URL carries the
   first port and that exactly one port was consumed.
2. Verify test fails (RED) — the seam does not exist.
3. Implement: split into three layers. `tryStartServer(t *testing.T, dbPath string, ports func() string)
   (base string, stop func(), attempted []string, err error)` holds all logic and returns errors;
   `startServerPorts` wraps it and calls `t.Fatalf` on error; `startServer(t, dbPath)` calls
   `startServerPorts` with the real `freePort`-backed source. **`startServer` keeps its exact existing
   signature** so all 13 call sites compile untouched. `tryStartServer` registers `t.Cleanup(stop)`
   only for the child it ultimately returns.
4. Verify test passes (GREEN).
5. Commit: "acceptance: add a port-source seam and an error-returning start path"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 2

### Task 9: Retry a lost port race
**Story:** Story 3 happy paths — retry succeeds; no retry on the normal path; retry leaks nothing
**Type:** happy-path

**Steps:**
1. Write failing test `TestStartServerRetriesAPortTakenBeforeBind`: hold a real listener on port P,
   supply a source yielding `P` then a genuinely free port, and assert `tryStartServer` returns a nil
   error, that a `GET /api/accounts` through the returned base URL succeeds, and that the base URL
   names the second port. Add `TestStartServerDoesNotRetryWhenTheFirstPortIsFree` asserting exactly
   one port is consumed and one child created. Add `TestRetryLeavesNoAbandonedChild` asserting the
   abandoned attempt's process is reaped (its port re-binds).
2. Verify test fails (RED).
3. Implement: loop `portAttempts = 3` times inside `tryStartServer`. On a readiness error, kill and
   `<-c.exited` the child and `untrackChild` it **before** the next attempt, then continue only if
   `isBindConflict(c.logs.String())` reports `strings.Contains(out, "address already in use")`.
   Accumulate each attempted port in `attempted`.
4. Verify test passes (GREEN).
5. Commit: "acceptance: retry a server start whose port was taken before bind"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 8, 5

### Task 10: Exhaustion and non-port failures are diagnosed, never masked
**Story:** Story 3 negative paths — exhaustion names the ports; failure identified from child output;
a non-port failure does not retry
**Type:** negative-path

**Steps:**
1. Write failing test `TestStartServerExhaustsBoundedPortAttempts`: hold listeners on three ports,
   supply exactly those, and assert the error names all three attempted ports, contains
   `address already in use`, and that the source was consumed exactly `portAttempts` times (bounded,
   not infinite). Add `TestStartServerDoesNotRetryANonPortFailure`: point `dbPath` inside a
   nonexistent directory, supply a source that would yield several ports, and assert the error is
   returned after **one** attempt with the child's own output — proving a genuine defect is not masked
   by the retry loop.
2. Verify test fails (RED).
3. Implement: on exhaustion, return an error listing `attempted` and the last child's output. Ensure
   the non-conflict branch returns immediately without consuming another port.
4. Verify test passes (GREEN).
5. Commit: "acceptance: diagnose port exhaustion and never retry a real startup failure"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 9

### Task 11: Record the widened reservation hole beside `freePort`
**Story:** Story 3 root cause — the probe/bind address mismatch
**Type:** refactor

**Steps:**
1. No test — this task records a verified finding at the site that causes it, so the next reader does
   not "fix" the retry by tightening the probe.
2. Implement: add a comment on `freePort` stating that it reserves `127.0.0.1:0` while the server
   binds `":"+PORT` on all interfaces (`cmd/server/main.go`), so the reservation cannot prove the
   child's bind will succeed; the retry in `tryStartServer`, not a better probe, is the mitigation.
   Note the rejected alternative (server accepts `PORT=0` and prints its bound address) and why it was
   rejected: production scope for a test-only benefit.
3. Verify `gofmt -l .` is empty and `go vet ./...` is clean.
4. Commit: "acceptance: document why the port reservation cannot be made authoritative"

**Files:** `test/acceptance/harness_test.go`
**Wired-into:** none (no new production surface)
**Dependencies:** 9

## Task Dependency Graph

```
Batch A (readiness)          Batch B (lifetime)
  1 ──┬── 2 ──┐                5 ── 6 ── 7
      ├── 3   │                │
      └── 4   │                │
              │                │
              └──── 8 ─── 9 ───┤  (9 depends on 8 and 5)
                         ├── 10
                         └── 11
```

Batch A and Batch B are independent and may proceed concurrently. Batch C joins them: Task 8 needs
Task 2's error-returning readiness, and Task 9 needs Task 5's registry for reaping abandoned attempts.

## Integration Points

- **After Task 3:** the readiness path is fully replaced and measurable end-to-end — the whole
  existing suite runs green with attempt counts assertable and ~99% of the dials gone.
- **After Task 7:** an interrupted run is safe; SIGINT/SIGTERM kills every tracked child.
- **After Task 10:** all three defects are closed, and the port race is deterministically reproduced
  in a test rather than waited for in production use.

## Coverage Mapping

| Story | Criterion | Task(s) |
|---|---|---|
| 1 | Interval-bounded attempts per start (amended) | 1, 3 |
| 1 | Package total under 500 (amended) | 3 |
| 1 | Returns as soon as the port accepts; no settling delay | 1 |
| 1 | Deadline + interval channel; zero `time.Sleep` hits | 1, 4 |
| 1 | Header cites the ADR | 4 |
| 1 | Never-listening child → deadline failure with address, ceiling, output | 1 |
| 1 | Immediately-exiting child → failure with child output | 2 |
| 1 | Must not busy-spin (regression fails the attempt assertion) | 1, 3 |
| 2 | Normal finish kills and reaps as today; `stop` stays idempotent | 5, 6 |
| 2 | Terminate-all ⇒ process gone, port re-bindable | 5 |
| 2 | Every tracked child terminated | 6 |
| 2 | Already-stopped child ⇒ safe no-op | 6 |
| 2 | `TestMain` wires signals before the first server | 7 |
| 2 | Catchable signal kills every child | 7 (wiring) + 5, 6 (effect) |
| 2 | Dead entry does not stall the sweep | 6 |
| 2 | Registry race-free under `-race` | 6 |
| 2 | SIGKILL residual documented, out of scope | 4 |
| 3 | Occupied port ⇒ retry, then serves | 9 |
| 3 | Retried app talks only to the survivor | 9 |
| 3 | Free first port ⇒ no retry, one child | 9 |
| 3 | Retry cleans up; no abandoned survivor | 9 |
| 3 | Port seam consumed in order, no sleeps | 8 |
| 3 | Exhaustion names attempted ports + `address already in use` | 10 |
| 3 | Failure identified from child output, not the deadline | 2, 10 |
| 3 | Non-port failure ⇒ no retry | 10 |
| 3 | Abandoned child reaped before the next attempt | 9 |
| 3 | Probe/bind mismatch recorded | 11 |
| all | `-count=2`, under 10s, `gofmt`, `go vet` | verified per task; Task 11 closes on lint |

Every acceptance criterion maps to at least one task. Scope check: **11 tasks** — normal range,
no split needed.

## Verification

- [ ] All happy-path criteria covered by at least one task (see Coverage Mapping)
- [ ] All negative-path criteria covered by explicit negative-path tasks, not a catch-all
- [ ] No task exceeds 5 minutes of work
- [ ] Dependencies are explicit and acyclic
- [ ] No terminal catch-all validation task — Task 11 records a finding and closes on lint; it does
      not re-prove the feature
- [ ] Every task carries a `**Wired-into:**` line (all `none` — this feature adds no production
      surface whatsoever)
- [ ] No task names another feature's sealed artifact under `.docs/`
- [ ] `startServer`, `newApp`, `newAppAt`, `seedDB` signatures unchanged; all 13 existing call sites
      compile untouched
- [ ] `go.mod` and `go.sum` unchanged; no production file under `cmd/` or `internal/` modified
- [ ] The five HTTP routes are unchanged
