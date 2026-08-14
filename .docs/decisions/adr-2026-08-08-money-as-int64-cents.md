# ADR: Money is int64 cents throughout

**Date:** 2026-08-08
**Status:** APPROVED
**Deciders:** james.stoup

## Context

The ledger stores and sums monetary amounts, and derives account balances by folding over a
transaction list. It also rejects any transaction that would take a balance below zero — a
comparison that must be exact.

This project is demoed live. An arithmetic surprise on stage (a balance displaying as
`10.199999999999999`, or a below-zero check that passes when it should fail) is
unrecoverable in front of an audience.

## Options Considered

### Option A: `int64` minor units (cents)
- **Pros:** Exact. Addition and comparison are trivially correct. No library needed — it is
  a builtin. Serializes to JSON as a plain integer. Fast.
- **Cons:** Every display site must divide by 100 and format. Every input site must parse to
  cents. Sub-cent amounts are unrepresentable.

### Option B: `float64`
- **Pros:** Direct arithmetic with decimal input; no conversion at boundaries.
- **Cons:** Binary floating point cannot represent 0.10 exactly. Sums drift. Equality and
  below-zero comparisons become unreliable in exactly the way a ledger cannot tolerate.

### Option C: a decimal library (e.g. `shopspring/decimal`)
- **Pros:** Exact decimal arithmetic with an ergonomic API and sub-cent precision.
- **Cons:** Adds a dependency, and this project deliberately has exactly one. Amounts become
  a struct rather than a comparable builtin, which complicates JSON, SQLite storage, and test
  table literals. Substantially more code on screen for an audience to read.

## Decision

**Option A: `int64` cents everywhere.** No floats anywhere, ever — not in the domain, not in
the store, not in JSON, not in test fixtures.

Chosen because exactness is non-negotiable for a ledger and `int64` gets it for free, with
zero dependencies and zero cognitive overhead for an audience reading the code on a projector.
Option C is the better choice for a real financial system that needs sub-cent precision or
multi-currency, but it buys precision this demo does not need at the cost of a dependency and
extra surface area. Option B is simply incorrect for money and is prohibited outright.

The prohibition is absolute rather than a guideline because a single `float64` leaking into an
intermediate calculation reintroduces the whole class of bug, and it would be invisible until
the exact moment it matters.

## Consequences

### Positive
- Balance sums and the below-zero check are exact by construction.
- Amounts are directly comparable, hashable, and storable as SQLite `INTEGER`.
- Test tables read plainly: `{amount: -350, want: ErrInsufficientFunds}`.

### Negative
- Formatting for display is a manual concern at every render site.
- Parsing user input ("3.50" → `350`) needs care, including a negative case for malformed
  input.
- Sub-cent amounts and percentage math are unrepresentable — acceptable here, since interest,
  fees, and percentage calculations are all explicit non-goals.

### Follow-up Actions
- [ ] Define the amount type and its parse/format helpers in `internal/ledger`.
- [ ] Add a negative test asserting malformed amount input is rejected.
