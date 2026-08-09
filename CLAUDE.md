# ledger-demo

## Harness

This project uses the james-stoup-agents harness. Its skills are installed in the user's
Claude Code configuration; do not copy them into this project.

## What this project is

A toy deposit-account ledger that exists to be **demoed live on a projector while an AI
harness adds a feature to it**. It is a stage prop, not a product. Every constraint below
follows from that: it must be small, legible, deterministic, and instantly resettable.

Despite the name it is **not double-entry** — there are no balancing counter-entries and
transfers are an explicit non-goal. It is a signed-amount transaction log per account,
with balance derived as a fold over that log.

## Tech Stack

- **Framework:** Go (current stable; 1.26.5 locally) — stdlib `net/http` with 1.22+
  `ServeMux` method/pattern routing. **No web framework.**
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, **no CGO**). File-backed for the
  server, in-memory for tests. The only third-party dependency in the project.
- **Test Framework:** stdlib `testing`, table-driven, `net/http/httptest` for requests.
  **No testify, no mocking libraries.**
- **Views:** `html/template`, one page. No frontend framework, no JS build step.
- **Tech-Context:** none — no Go pack exists in the harness (`tech-context/` has only
  `rails-postgres`), so skills use generic behavior.

## Active Skills

| Phase | Skills |
|-------|--------|
| UNDERSTAND | bootstrap, memory |
| DECIDE | explore, prd, stories, conflict-check, plan |
| BUILD | tdd, debugging, code-review, pipeline |
| SHIP | finish, retro |

## Harness Behavioral Rules

All behavioral rules, model selection, communication protocol, and conventions are defined in
the user-scoped `~/.claude/skills/HARNESS.md`.

Claude MUST read and follow `~/.claude/skills/HARNESS.md` at the start of every session.

## Shared Workflow and Claude Invocation

The shared harness workflow and lifecycle gates are defined by `HARNESS.md`; they are the same
for every supported host. Claude invokes a harness skill with native `/skill-name` syntax (for
example, `/tdd`) and must not use Codex `$skill-name` syntax.

## Skill References

Harness references use paths relative to the harness root:
- Skills: `skills/`
- Agents: `agents/`
- Tech-Context: none for this stack

## Project Conventions

These are settled and must not be relitigated. Each has an ADR in `.docs/decisions/`.

1. **Money is `int64` cents throughout. No floats anywhere, ever.**
2. **Time is injected, never read directly.** `clock.Clock` has a single method
   `Now() time.Time`; only `SystemClock` may call `time.Now()`. Tests use `FixedClock`.
3. **Domain errors are sentinel errors wrapped for `errors.Is`** — never bare
   `errors.New` at call sites, never error strings compared by value.
4. **The `Store` interface is declared in `internal/ledger`** and implemented in
   `internal/store`. The domain has no knowledge of SQLite.
5. **Balance is derived from transactions**, never a stored mutable field.
6. **Tests target a 4:1 test-to-implementation line ratio** — table-driven, a negative case
   for every validation rule, full suite under 10 seconds, fully deterministic (`FixedClock`
   everywhere, in-memory SQLite, no `time.Sleep`, no test-ordering dependencies).
7. **`gofmt` clean and `go vet` clean.** Enforced by the pre-PR lint hook in
   `.claude/settings.json`, not by any skill.

## Non-Goals — do NOT build these

Features are added live on stage during a presentation. **If any of these already exist, the
demo is ruined.**

- Duplicate or double-charge detection of any kind. No idempotency keys, no dedup window,
  **no uniqueness constraint beyond the primary key.**
- Overdraft allowance, fees, or percentage calculations.
- Pending transactions or holds; available-vs-posted balance.
- Statements, exports, or reporting.
- Interest or rounding rules.
- Auth, users, sessions, or multi-tenancy.
- Transfers between accounts.
- Docker, CI config, or deployment tooling.
- Metrics, tracing, or structured logging beyond stdlib `log`.

Before adding anything, check it against this list. When in doubt, ask — do not infer that a
missing feature is an oversight.

## Infrastructure Boundaries

This project has **no shared infrastructure**. SQLite is a local file, not a server: there is
no database host, no database port, no Redis, and no `docker compose`. Docker is an explicit
non-goal.

- **`.env.example`** (committed) — reference showing the two variables that exist.
- **`.env`** (gitignored) — working environment for this worktree.
- **`.env.local`** (gitignored) — worktree-specific overrides.

**Boot sequence:**
1. `make reset` — drop the DB and load deterministic seed data
2. `make dev` — serve on `http://localhost:8080`

Worktrees coexist by setting a distinct `PORT` and a distinct `LEDGER_DB_PATH` in
`.env.local`. Nothing is shared between them, so no namespacing is required.

**Adding new infrastructure requires an ADR** — and note that most candidates (Docker,
external services, network calls) are non-goals above.

## Memory

Every session starts with recall.

Note: harness-side `.memory/` is **not** provisioned in this project. The `conduct-ts memory
setup` command the bootstrap skill expects does not exist in the installed `conduct-ts` binary,
so no `.memory` symlink was created. Project memory currently lives in the Claude Code project
memory directory instead, and records this project's purpose, the non-goals rationale, the
locked decisions, the stack constraints, and the repo target.
