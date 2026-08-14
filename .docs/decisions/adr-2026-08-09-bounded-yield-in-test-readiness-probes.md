# ADR: The no-sleep rule forbids sleeping to mask races, not yielding while polling

**Date:** 2026-08-09
**Status:** APPROVED
**Deciders:** james.stoup

## Context

Project convention 6 (CLAUDE.md) requires the suite to be deterministic and says **no
`time.Sleep`**. `adr-2026-08-08-injected-clock.md` states the same in its consequences: "the suite can
run in parallel with no shared state and no `time.Sleep`". The header of
`test/acceptance/harness_test.go` repeats it as "no time.Sleep anywhere (NFR-4)".

The author of that file read the rule as *never block, ever*, and wrote a readiness probe that dials
in a tight loop with no yield of any kind:

```go
timeout := time.After(readyTimeout)
for {
    select {
    case <-timeout:
        t.Fatalf(...)
    default:            // falls straight through
    }
    conn, err := net.Dial("tcp", addr)   // ~23,400 attempts/sec
    ...
}
```

Measured on this machine: **~23,400 `connect()` syscalls per second**, **6,470 wasted dials and
423 ms of pure spin across the 9 server starts in one suite run**, and roughly **350,000 syscalls** if
a start ever fails and the 15-second ceiling is reached. The fix is to poll on an interval — which
looks, to anyone grepping for the convention, exactly like the thing the rule bans.

Two readings of the rule cannot both stand, and the wrong one is already costing CPU. Since a locked
convention is being reinterpreted rather than a component designed, the decision is recorded here so
it is not re-litigated at review time or by a future agent reading only CLAUDE.md.

## Options Considered

### Option A: Scope the rule to its purpose — ban sleeping *as a synchronization mechanism*, permit a bounded yield while polling a real condition
- **Pros:** Matches why the rule exists. Every determinism hazard the rule targets — a sleep standing
  in for a happens-before edge, a sleep padding the suite to hide a race, a sleep whose duration is
  tuned until CI goes green — is still banned. A poll that waits on an *observable* condition
  (the port accepts, or the child exited) with a hard deadline adds no nondeterminism: the outcome is
  decided by the condition, never by the interval. It also keeps the letter of the rule intact,
  because a `time.Ticker` in a `select` is not a `time.Sleep` call.
- **Cons:** The rule stops being a pure grep and needs a sentence of judgment attached.

### Option B: Keep the literal reading — no blocking of any kind, keep the busy-wait
- **Pros:** Mechanically checkable with one grep; nothing to explain.
- **Cons:** Costs a spinning core for 30–120 ms per server start and would cost ~350,000 syscalls on a
  failed start, for zero determinism benefit. It also makes the harness *worse* at determinism, not
  better: the spin starves the very child process it is waiting for on a loaded machine, and it
  cannot notice a child that has already died, so a bind conflict burns the full 15 seconds before
  reporting. The rule was written to protect determinism, and the literal reading is now harming it.

### Option C: Poll the injected `clock.Clock` instead of the wall clock
- **Pros:** Superficially consistent with the injected-time convention.
- **Cons:** Category error. `clock.Clock` exists so *recorded domain timestamps* are reproducible;
  `FixedClock` never advances, so a readiness loop driven by it would never time out. Process startup
  latency is real elapsed time, not domain time — it is not the clock convention's subject.

## Decision

**Option A.** The no-sleep rule is scoped to its purpose:

> **Banned:** sleeping to create a happens-before edge, to paper over a race, or to pad the suite —
> any wait whose *duration* is load-bearing for the result.
> **Permitted:** a bounded yield while polling an observable condition, provided the wait is bounded
> by an explicit deadline, every exit is decided by the observed condition (or the deadline, which
> fails loudly with diagnostics), and no assertion depends on how long the wait took.

Concretely, `test/acceptance/harness_test.go` may poll readiness on a `time.Ticker` inside a `select`
that also watches an explicit deadline and a child-exited channel. Literal `time.Sleep` remains
**banned repo-wide**, so convention 6 stays greppable exactly as written — this ADR narrows what the
prohibition *means*, and narrows nothing about what it *matches*.

`SystemClock` remains the only permitted caller of `time.Now()`
(`adr-2026-08-08-injected-clock.md` is unaffected). Readiness polling uses `time.Ticker`, `time.After`,
and `time.NewTimer`, none of which read the current time.

## Consequences

### Positive
- Removes ~6,470 wasted `connect()` syscalls and ~423 ms of spin per suite run, and ~350,000 syscalls
  from the failure path.
- The readiness probe can watch for child exit, so a bind conflict is reported in milliseconds with
  the server's own output instead of after a 15-second stall.
- The stated rule and the code stop contradicting each other, so the next reader is not forced to
  guess which one wins.

### Negative
- Convention 6 now carries a scope clause; "no `time.Sleep`" alone no longer settles every argument.
- A ticker interval is a tuning knob, and a future contributor could grow it into the very thing the
  rule bans. Mitigated by the ban on duration-dependent assertions: an interval can only affect how
  *fast* a spec settles, never whether it passes.

### Follow-up Actions
- [x] Record the scope clause here so CLAUDE.md convention 6 is read against it.
- [ ] Update the `test/acceptance/harness_test.go` package header, which currently asserts the
      literal reading ("no time.Sleep anywhere (NFR-4)"), to cite this ADR.
