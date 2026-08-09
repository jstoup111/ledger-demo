# C4 Level 3 — Components

**Status:** Skeleton. `internal/ledger`, `internal/store`, and `internal/clock` are currently
empty placeholders — the domain is built during the demo. The dependency arrows below are the
*intended* structure and are already enforced by the package layout.

```mermaid
C4Component
  title Components — ledger-demo

  Container_Boundary(app, "ledger-demo binary") {
    Component(main, "cmd/server", "main", "Entry point. Dispatches `serve` and `seed`. Reads PORT and LEDGER_DB_PATH.")
    Component(httpapi, "internal/httpapi", "handlers + routing", "ServeMux routes, JSON encoding, HTML rendering. Maps domain sentinels to typed error codes.")
    Component(ledger, "internal/ledger", "domain", "Account, Transaction, derived balance, validation rules, sentinel errors. DECLARES the Store interface.")
    Component(store, "internal/store", "persistence", "IMPLEMENTS ledger.Store over SQLite.")
    Component(clock, "internal/clock", "time", "Clock interface, SystemClock, FixedClock.")
    Component(web, "web", "embedded assets", "index.html.tmpl + style.css via embed.FS")
  }

  ContainerDb(db, "ledger.db", "SQLite")

  Rel(main, httpapi, "Builds router")
  Rel(main, store, "Opens DB, injects store")
  Rel(httpapi, ledger, "Calls domain operations")
  Rel(httpapi, web, "Renders template")
  Rel(store, ledger, "Satisfies Store interface")
  Rel(ledger, clock, "Reads current time via Clock")
  Rel(store, db, "database/sql")
```

## The dependency rule that matters

`internal/store` depends on `internal/ledger`, **not the reverse**. The `Store` interface is
declared in the domain package and implemented in the store package, so the domain has no
knowledge of SQLite and unit-tests against a trivial in-memory fake.

See `.docs/decisions/adr-2026-08-08-store-interface-in-domain-package.md`.

## Current state

| Package | State |
|---|---|
| `cmd/server` | Serves; `seed` is a stub that reports it has nothing to load |
| `internal/httpapi` | `NewRouter()` wires `GET /` and `GET /style.css` only |
| `internal/ledger` | Empty — doc comment describing intended contents |
| `internal/store` | Empty — blank driver import pins `modernc.org/sqlite` for offline builds |
| `internal/clock` | Empty — doc comment only |
| `web` | Complete for the scaffold: stub page + projector-legible CSS |
