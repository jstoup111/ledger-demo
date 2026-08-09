# ledger-demo

A toy deposit-account ledger. Small, legible, deterministic, instantly resettable —
built to be demoed live on a projector while an AI harness adds a feature to it.

## Quick Start

```
make reset   # drop the DB and load deterministic seed data
make dev     # serve on http://localhost:8080
make test    # run the suite (under 10s, fully deterministic)
```

## Stack

Go (stdlib `net/http` with 1.22+ `ServeMux` routing), SQLite via
`modernc.org/sqlite` (pure Go, no CGO), stdlib `testing`, `html/template`.
One dependency total. No framework, no JS build step, no Docker. Runs fully offline.

## Layout

```
cmd/server/       entry point: serve + seed commands
internal/ledger/  domain: accounts, transactions, balances, validation
internal/store/   SQLite persistence behind an interface declared in internal/ledger
internal/httpapi/ handlers, routing, JSON + HTML rendering
internal/clock/   Clock interface, SystemClock, FixedClock
web/              the single page and its stylesheet
```

## Conventions

These are settled. See `.docs/decisions/` for the reasoning.

- Money is `int64` cents throughout. No floats anywhere, ever.
- Time is injected via `clock.Clock`. Only `SystemClock` may call `time.Now()`.
- Domain errors are sentinel errors wrapped for `errors.Is`.
- The `Store` interface lives in `internal/ledger`; SQLite implements it in `internal/store`.
- Tests target a 4:1 test-to-implementation line ratio, table-driven, with a negative
  case for every validation rule.

## Status

Scaffold only. `internal/ledger`, `internal/store`, and `internal/clock` are
deliberately empty — the ledger is built during the demo.

**Do not add features speculatively.** Duplicate/idempotency detection, overdraft or
fees, pending holds, statements, interest, auth, transfers, Docker, and CI are all
deliberate non-goals: they are the features added live on stage, and building any of
them ahead of time ruins the demo.
