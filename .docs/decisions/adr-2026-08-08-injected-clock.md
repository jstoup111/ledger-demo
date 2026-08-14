# ADR: Time is injected via a Clock interface, never read directly

**Date:** 2026-08-08
**Status:** APPROVED
**Deciders:** james.stoup

## Context

Transactions carry a `CreatedAt` timestamp, and the transaction list is ordered newest first.
Both the seed data and the test suite are required to be **fully deterministic** — identical
data and identical results on every run.

Anything that calls `time.Now()` directly is untestable without either freezing the system
clock or asserting loosely ("the timestamp is roughly now"). Loose assertions are exactly what
produces a flaky test, and a flaky test that fails while a live audience watches is the worst
possible outcome for this project.

## Options Considered

### Option A: A `Clock` interface, injected
```go
type Clock interface { Now() time.Time }
```
with `SystemClock` for production and `FixedClock` for tests.
- **Pros:** Determinism is structural, not a convention. Tests assert exact timestamps.
  Trivially small — one method. Standard, widely recognized Go pattern.
- **Cons:** A `Clock` must be threaded through constructors into anything that stamps time.

### Option B: Call `time.Now()` directly, assert loosely in tests
- **Pros:** No plumbing at all; less code.
- **Cons:** Tests cannot assert exact values, so they assert ranges or ignore timestamps —
  weakening exactly the tests that ordering depends on. Seed data would differ every run,
  breaking `make reset` reproducibility.

### Option C: A package-level clock variable that tests monkey-patch
- **Pros:** No constructor plumbing; tests override one variable.
- **Cons:** Global mutable state. Tests that set it are order-dependent and cannot run in
  parallel — directly violating the "no test-ordering dependencies" requirement.

## Decision

**Option A.** Define `clock.Clock` with the single method `Now() time.Time`, plus
`SystemClock` and `FixedClock`. **`SystemClock` is the only place in the entire codebase
permitted to call `time.Now()`.** Every test uses `FixedClock`.

Chosen because determinism here is a hard requirement rather than a preference, and Option A
is the only one of the three that makes it structural. The rule is phrased as an absolute
prohibition on `time.Now()` so that it is mechanically checkable by grep: exactly one match,
in `SystemClock`. Option C is rejected specifically because global mutable state is
incompatible with the parallel, order-independent suite this project requires.

## Consequences

### Positive
- Seed data has fixed timestamps and is byte-identical on every run.
- Tests assert exact `CreatedAt` values, so newest-first ordering is genuinely covered.
- The suite can run in parallel with no shared state and no `time.Sleep`.

### Negative
- Constructors gain a `Clock` parameter, which is visible plumbing an audience will see.
- The "only `SystemClock` may call `time.Now()`" rule needs enforcement by review or grep;
  nothing in the compiler prevents a violation.

### Follow-up Actions
- [ ] Implement `Clock`, `SystemClock`, and `FixedClock` in `internal/clock`.
- [ ] Inject a `Clock` wherever a timestamp is stamped; never call `time.Now()` elsewhere.
