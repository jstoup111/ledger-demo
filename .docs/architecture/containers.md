# C4 Level 2 — Containers

**Status:** Skeleton, generated at bootstrap from the scaffold.

There is exactly one deployable: a single Go binary. No Docker, no reverse proxy, no separate
frontend build — templates and CSS are embedded into the binary via `embed`.

```mermaid
C4Container
  title Containers — ledger-demo

  Person(presenter, "Presenter")

  Container_Boundary(app, "ledger-demo (single Go binary)") {
    Container(server, "HTTP server", "Go stdlib net/http, 1.22+ ServeMux", "Serves the HTML page and the JSON API on a fixed port (8080)")
    Container(assets, "Embedded assets", "html/template + CSS via embed.FS", "One page, one stylesheet, compiled into the binary")
  }

  ContainerDb(db, "ledger.db", "SQLite file", "Accounts and transactions. Dropped and re-seeded by `make reset`.")

  Rel(presenter, server, "HTTP", "localhost:8080")
  Rel(server, assets, "Renders")
  Rel(server, db, "database/sql")
```

## Commands

| Command | Effect |
|---|---|
| `make dev` | Run the server on port 8080 |
| `make seed` | Drop the DB and load deterministic seed data |
| `make reset` | Restore pristine demo state (drop + re-seed + confirm) |
| `make test` | Run the suite (target: under 10s, fully deterministic) |
| `make check` | `gofmt` + `go vet` — what the pre-PR lint gate runs |
