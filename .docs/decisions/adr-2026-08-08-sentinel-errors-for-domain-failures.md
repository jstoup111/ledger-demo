# ADR: Domain errors are sentinel errors wrapped for errors.Is

**Date:** 2026-08-08
**Status:** APPROVED
**Deciders:** james.stoup

## Context

Posting a transaction can fail several distinct ways: the account does not exist, the amount is
zero, the description is empty or over 140 characters, or the transaction would take the balance
below zero. Each is a different outcome, and the HTTP layer must map each to a **typed error
code plus a human-readable message** in its JSON response. The UI renders those errors visibly
on the page.

So the transport layer needs to reliably distinguish *which* domain rule was violated. How
errors are represented determines whether that mapping is sound or a string-matching guess.

## Options Considered

### Option A: Package-level sentinel errors, wrapped, matched with `errors.Is`
```go
var ErrInsufficientFunds = errors.New("insufficient funds")
// ...
return fmt.Errorf("posting to %s: %w", accountID, ErrInsufficientFunds)
```
- **Pros:** Callers match on identity, not text. Wrapping adds context without breaking the
  match. Messages can be reworded freely. Stdlib-only. Exhaustive mapping in the HTTP layer.
- **Cons:** Every failure mode needs a declared variable. Carrying structured data (e.g. the
  offending field) needs a custom error type instead.

### Option B: Bare `errors.New` / `fmt.Errorf` at each call site
- **Pros:** Shortest to write; no declarations.
- **Cons:** Nothing to match against, so the HTTP layer must compare error strings by value.
  Rewording a message silently breaks the mapping, and the compiler cannot help. This is the
  failure mode most likely to surface as a wrong error code on stage.

### Option C: A single custom error type carrying a code field
- **Pros:** Codes and structured detail travel with the error; one type to map.
- **Cons:** More machinery than four validation rules justify, and it does not compose with
  wrapping as naturally. Heavier for an audience to read.

## Decision

**Option A.** Domain failures are package-level sentinel errors in `internal/ledger`, wrapped
with `%w` when context is added, and matched with `errors.Is`. **Never bare `errors.New` at
call sites; never error strings compared by value.**

Chosen because the HTTP layer's job is to map each domain failure to a stable typed code, and
only identity-based matching makes that mapping robust against message rewording. Option B is
prohibited outright rather than merely discouraged, because a string comparison looks correct
in review and fails silently later — a class of bug this project cannot absorb. Option C is a
reasonable choice if the errors later need structured fields; it can supersede this ADR if that
happens, without changing the `errors.Is` contract at call sites.

## Consequences

### Positive
- The HTTP layer maps errors exhaustively and stably: sentinel → typed code + message.
- Error messages can be reworded for projector legibility without breaking any behavior.
- Tests assert `errors.Is(err, ledger.ErrX)` — precise, and readable in a table.

### Negative
- Each validation rule requires a declared sentinel, which is boilerplate.
- Wrapping discipline matters: a `%v` instead of `%w` breaks the chain, and nothing catches it
  except a test.
- Structured detail (which field, what limit) is not carried and must go in the message.

### Follow-up Actions
- [ ] Declare one sentinel per validation rule in `internal/ledger`.
- [ ] Map every sentinel to a typed JSON code in `internal/httpapi`.
- [ ] Add a negative test per rule asserting both `errors.Is` and the JSON code.
