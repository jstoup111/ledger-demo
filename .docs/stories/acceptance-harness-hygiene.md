# Stories — Acceptance Harness Hygiene

**Status:** Accepted

**Feature:** acceptance-harness-hygiene · **Tier:** S · **Track:** technical
**Source:** technical intent — three defects verified in `test/acceptance/harness_test.go`
(no PRD; the technical track puts acceptance criteria here)
**Constrained by:** `.docs/decisions/adr-2026-08-09-bounded-yield-in-test-readiness-probes.md`
(Accepted) and the six Accepted ADRs already in `.docs/decisions/`

Three stories, one per defect. They share a file but not a concern: Story 1 owns *how the harness
waits*, Story 2 owns *how the child dies*, Story 3 owns *which port it binds*. No story depends on
another, and each is independently verifiable.

## Scope boundary

The subject is the suite's own scaffolding. Explicitly unchanged by all three stories:

- the five HTTP routes — none added, none removed, none renamed;
- every production package (`internal/...`, `cmd/server`, `web/`) — byte-for-byte untouched;
- `go.mod` / `go.sum` — no dependency added; `os/exec`, `os/signal`, `net`, and `time` are stdlib;
- the signatures of `newApp`, `newAppAt`, `seedDB`, and `startServer` — so the 13 existing call sites
  in `test/acceptance/ledger_acceptance_test.go` compile with no edit;
- the specs' shape — they still drive the REAL `./cmd/server` binary over real HTTP against
  file-backed SQLite in `t.TempDir()`. Nothing moves in-process.

## Negative-path categories evaluated

Tier S requires at least one negative path per story; the categories are still each evaluated, because
saying which do not apply is more useful than inventing a scenario.

| Category | Applies? |
|---|---|
| **Dependency unavailability** | **Yes** — the child server process is the dependency. Stories 1 and 3 cover it never becoming ready and dying at startup |
| **Resource exhaustion** | **Yes** — this is the point of Story 1: ~23,400 `connect()` syscalls/sec is exhaustion of CPU and ephemeral ports |
| **Concurrent access** | **Yes** — Story 3 is a check-then-act race against any other process on the machine for a port |
| **Partial failure & rollback** | **Yes** — Story 3's retry must leave no half-started child behind when it re-picks a port |
| **Invariant side-effect on alternate branches** | **Yes** — Story 2: `stop` runs on the `t.Cleanup` branch but is bypassed when the binary dies by signal; Story 3 adds a second bypass branch (the abandoned first attempt) |
| **Timeouts & network errors** | **Yes** — Story 1's deadline is the timeout path, and it must fail loudly with the server's own output |
| **Data integrity** | No — no story writes to the database; `seedDB` is unchanged |
| **Invalid input** | No — no user input exists on any path here; the only inputs are a temp path and a port the harness itself chose |
| **Auth / permission failures** | No — auth is an explicit non-goal; there is no principal |
| **Cascade deletion effects** | No — nothing is deleted; `t.TempDir()` is removed by the framework |
| **Model-level immutability** | No — no model is involved |
| **Exception class hierarchy** | No — Go sentinel errors, no exception tree; these helpers call `t.Fatalf` rather than returning typed errors |
| **Dedup / idempotency key analysis** | No — **and deliberately so.** No story here introduces a dedup key, an idempotency window, or a uniqueness constraint of any kind. Retrying a *port* is not retrying a *request*: no HTTP call is ever replayed, and no transaction is ever posted twice |

---

## Story 1: The harness yields while waiting for a server to become ready

**Requirement:** Defect D1 (technical track — no FR).
Governed by `adr-2026-08-09-bounded-yield-in-test-readiness-probes.md`.

As the engineer running the suite, I want the readiness probe to wait on an interval instead of
spinning, so a suite run stops burning a core and a failed start is diagnosed in milliseconds instead
of after a 15-second stall.

**Measured baseline (this machine, go1.26.5 darwin/arm64):** 9 server starts per run, each ready in
32–123 ms, costing **6,470 dial attempts and 423 ms of pure spin per run** at ~23,400 attempts/sec.
A single failed start would cost ~350,000 `connect()` syscalls.

> **Note on a falsified premise.** The defect was originally reported as also "piling up TIME_WAIT
> sockets". That part is **wrong** and is not a criterion below: 4,600 refused connects moved this
> machine's TIME_WAIT count 18 → 19, because a refused connect is answered with RST and never enters
> TIME_WAIT. Only the one *successful* probe per start is an active close. The defect stands entirely
> on the CPU and syscall evidence above.

### Acceptance Criteria

#### Happy Path
- Given a server that becomes ready in the normal 30–130 ms, when the harness waits for it, then it
  makes **at most 8 connection attempts** for that start — the count is recorded by the harness and
  asserted, rather than inferred.

  > **Amended 2026-08-09 by acceptance-harness-hygiene `/plan`:** the assertion is now
  > *interval-bounded* rather than a fixed count: attempts for a start must be
  > **≤ (elapsed / poll interval) + 2**. A fixed ceiling of 8 was falsified as machine-dependent — at
  > the chosen 25 ms interval it holds for the measured 32–123 ms starts, but a slower machine taking
  > 400 ms would make 17 legitimate attempts and fail the story rather than the code. The derived
  > bound tests the property that actually matters (the probe waits an interval between attempts) and
  > is machine-independent, while a regression to the yield-free loop produces thousands of attempts
  > and still fails loudly.
- Given the full acceptance package, when it is run, then the total connection attempts spent on
  readiness across all 9 server starts is **under 100** (baseline: 6,470).

  > **Amended 2026-08-09 by acceptance-harness-hygiene `/plan`:** the ceiling is raised to **under
  > 500** for the same machine-dependence reason (9 starts × a slow-machine 22 attempts ≈ 198 would
  > breach 100 legitimately). 500 is still a 13× margin below the 6,470 baseline, so it remains a
  > effective tripwire against a reintroduced busy-wait; the per-start interval-bounded assertion
  > above is the precise check.
- Given a server that is ready, when the harness waits for it, then the wait returns as soon as the
  port accepts — no fixed settling delay is introduced, and no assertion anywhere depends on how long
  the wait took.
- Given the readiness wait, when it polls, then it uses a deadline plus an interval channel; a literal
  `time.Sleep` appears **nowhere** in the repository, so `grep -rn 'time.Sleep' .` returns zero hits.
- Given `.docs/decisions/adr-2026-08-09-bounded-yield-in-test-readiness-probes.md`, when the package
  header comment of `harness_test.go` is read, then it cites that ADR instead of asserting the old
  literal reading of the no-sleep rule.

#### Negative Paths
- Given a child process that starts but never listens, when the readiness deadline expires, then the
  harness fails the test with a message naming the address, the elapsed ceiling, **and** the child's
  captured stdout+stderr — the diagnostic that exists today is preserved, not traded away.
- Given a child process that exits immediately without ever listening, when the harness is waiting,
  then it fails **as soon as the exit is observed** rather than waiting out the deadline, and the
  failure message contains the child's own output. (Asserted by message content, not by timing.)
- Given the readiness probe, when it is waiting, then it must not busy-spin: the recorded attempt
  count for a start that takes ~100 ms stays bounded by the interval, so a regression back to a
  yield-free loop fails this story's attempt-count assertion rather than passing silently.

### Done When
- [ ] Readiness attempt counts are observable to a test, and a single normal start asserts the
      interval-bounded ceiling `≤ (elapsed / poll interval) + 2` (amended 2026-08-09; originally a
      fixed `≤ 8`, which was machine-dependent).
- [ ] Whole-package readiness attempts assert < 500 (amended 2026-08-09; originally < 100).
- [ ] `grep -rn 'time.Sleep' .` returns zero hits.
- [ ] A never-listening child produces a failure naming the address, the ceiling, and the child output.
- [ ] An immediately-exiting child produces a failure containing the child's own output.
- [ ] `go test ./...` passes with `-count=2` and the full suite stays under 10 seconds.
- [ ] `gofmt -l .` is empty and `go vet ./...` is clean.

---

## Story 2: An interrupted test run leaves no server process behind

**Requirement:** Defect D2 (technical track — no FR).

As the engineer running the suite, I want every server the harness started to die when the test binary
dies, so an interrupted run cannot leave a `ledger-server` holding a port. In a real incident an
orphan reparented to `launchd` held port 8080 for 90 minutes.

**Root cause, verified:** the stdlib `testing` package imports `os/signal` **nowhere** and installs no
handler, so a signal that terminates the test binary skips every `t.Cleanup` — including the `stop`
that kills the child. `cmd/server` has no signal handling of its own either, so nothing on either side
of the pair cleans up.

### Acceptance Criteria

#### Happy Path
- Given a server started by the harness, when the test that started it finishes normally, then the
  child is killed and reaped exactly as today — `stop` stays idempotent and stays registered with
  `t.Cleanup`, so no existing call site changes behavior.
- Given a server started by the harness, when the harness's terminate-all entry point is invoked
  directly, then that child is no longer running and the port it held is immediately re-bindable by
  the test — proving the process is gone, not merely signalled.
- Given several servers started and still live, when terminate-all is invoked once, then **every**
  tracked child is terminated, not just the most recent.
- Given a child already stopped through its own `stop`, when terminate-all is invoked afterwards, then
  it is a no-op that neither panics, blocks, nor reports an error on the already-reaped process.
- Given `harness_test.go`, when it is inspected, then `TestMain` registers a handler for interrupt and
  terminate signals that runs terminate-all before exiting, and it does so **before** the first server
  can be started.

#### Negative Paths
- Given the test binary is terminated by a catchable signal (SIGINT or SIGTERM) while a server is
  live, when the handler runs, then every tracked child is killed before the process exits — this is
  the branch that bypasses `t.Cleanup` today and is the whole point of the story.
- Given a child that has already exited on its own, when the signal handler sweeps the registry, then
  the sweep completes without error and still exits — a dead entry must not stall the handler and
  strand the remaining live children.
- Given the registry is mutated from a test goroutine while the signal handler sweeps it, when both
  run, then access is guarded so `go test -race` reports no data race on the registry.
- Given the test binary is killed with SIGKILL, when it dies, then an orphan **may** remain — this is
  acknowledged as unfixable in-harness (a handler cannot run, and darwin has no `Pdeathsig`) and is
  **out of scope**; it must not be papered over by editing the production server to watch its parent.

### Done When
- [ ] A test starts a real server, invokes terminate-all, and asserts the process is gone by
      successfully re-binding the port it held.
- [ ] A test asserts terminate-all kills multiple live children in one call.
- [ ] A test asserts terminate-all over an already-stopped child is a safe no-op.
- [ ] `TestMain` wires interrupt and terminate signals to terminate-all before any server starts.
- [ ] `go test ./... -race` is clean, and `-count=2` passes.
- [ ] The SIGKILL residual limitation is recorded in the file's comments, not silently ignored.
- [ ] No production file under `cmd/` or `internal/` is modified by this story.

---

## Story 3: A port taken between reservation and bind does not fail the run

**Requirement:** Defect D3 (technical track — no FR).

As the engineer running the suite, I want a lost port race to be retried rather than fatal, so the
suite stops failing with `bind: address already in use` after burning the readiness ceiling.

**Root cause, verified:** `freePort` listens on `127.0.0.1:0`, reads the number, closes the listener,
and only later does the child bind — any process on the machine can take it in that window.
**A second, wider hole:** the reservation probes `127.0.0.1:0` while the server binds `":"+PORT` (all
interfaces), so a port free on loopback may be unavailable for the child's actual bind. The
reservation therefore never proved what it appeared to prove, which is why the fix retries rather than
trying to perfect the probe.

### Acceptance Criteria

#### Happy Path
- Given the port the harness picked is occupied by the time the child binds, when the harness starts a
  server, then it observes the child's bind failure, picks a different port, starts again, and returns
  a base URL that serves requests — the spec that called it passes with no awareness a retry happened.
- Given a retry occurred, when the returned app is used, then it talks to the surviving server only:
  a request through the returned base URL succeeds and the abandoned attempt's process is not running.
- Given the first port is free, when a server is started, then **no** retry occurs and exactly one
  child process is created — the retry path adds no cost to the normal case.
- Given a retry occurred, when the test finishes, then `stop` and terminate-all clean up correctly and
  no process from the abandoned attempt survives (no leak introduced by the new branch).
- Given the port-selection seam, when a test supplies a sequence of ports, then the harness consumes
  them in order — making the whole race deterministically reproducible with no sleeps and no
  dependence on real contention.

#### Negative Paths
- Given every port the harness tries is occupied, when it exhausts its bounded attempt budget, then it
  fails with a message that names the attempted ports, states that each bind was refused, and includes
  the child's own `address already in use` output — it must not retry forever and must not report a
  generic timeout that hides the real cause.
- Given a child that fails to bind, when the harness detects it, then the failure is identified from
  the **child's observed exit and output**, not from the readiness deadline expiring — asserted by the
  message containing `address already in use`, so a regression to "burn 15 s then report" fails here.
- Given a child that exits at startup for a reason **other** than a taken port (for example an
  unopenable database), when the harness sees the exit, then it does **not** retry — it fails
  immediately with that child's output, so a genuine defect is never masked by a retry loop.
- Given the abandoned attempt of a retried start, when the retry proceeds, then the abandoned child is
  reaped before the next attempt begins, leaving no zombie and no second process contending for the
  database file.

### Done When
- [ ] A test forces a first-attempt bind collision by holding the port, and the resulting app serves a
      successful request.
- [ ] A test asserts the normal path creates exactly one child and performs no retry.
- [ ] A test asserts total exhaustion fails with the attempted ports and `address already in use`
      in the message.
- [ ] A test asserts a non-port startup failure fails immediately with no retry.
- [ ] The port-selection seam lets a test supply ports deterministically, with no sleeps.
- [ ] No abandoned child survives a retry, asserted by process state or port re-bindability.
- [ ] `go test ./...` passes with `-count=2`, the suite stays under 10 seconds, `gofmt -l .` is empty,
      and `go vet ./...` is clean.

---

## Assumptions recorded (operator unavailable; labelled, never approved on his behalf)

| # | Assumption | Confidence | Impact if wrong |
|---|---|---|---|
| 1 | Unix/darwin-only process handling is acceptable — no Windows support goal (macOS Makefile, no CI, no Docker, explicit non-goals) | 95% | **Low.** The design deliberately uses only `os/signal` and `os.Process.Kill`, both portable, and avoids `syscall` and process groups entirely, so almost nothing rides on this |
| 2 | SIGKILL of the test binary is out of scope (no handler can run; darwin has no `Pdeathsig`; a parent-watchdog would mean editing the production demo binary) | 99% | **Low.** A residual orphan remains for `kill -9` only, and it is documented rather than hidden |
| 3 | A bounded retry (3 attempts) is sufficient for the port race; eliminating it would require the server to accept `PORT=0` and print its bound address — rejected in `/explore` as production scope for a test-only benefit | 90% | **Low.** The race becomes rare rather than impossible; the escalation path is recorded in the explore notes and in this file |
| 4 | Recording readiness attempt counts for assertion is acceptable test-only instrumentation | 92% | **Low.** If rejected, Story 1's attempt-count criteria would fall back to a source-level assertion that the probe polls on an interval, which is weaker but sufficient |

All four are **assumed, operator unavailable** — none is presented as operator-approved.
