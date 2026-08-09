# PRD — Base Ledger

**Status:** Approved
**Date:** 2026-08-08
**Track:** product (`.docs/track/base-ledger.md`)
**Tier:** M (`.docs/complexity/base-ledger.md`)
**Stem:** `base-ledger`

## Problem / Background

`ledger-demo` exists to be demoed live on a projector while an AI harness adds a feature to it in
front of an audience. It is a stage prop, not a product.

The stage is not ready. The repository is scaffold only: the domain, persistence, and time packages
contain nothing but doc comments describing what they will hold; no account or transaction data can
be stored; the page is a placeholder that says "Scaffold only. The ledger is built live."; and the
reset command reports that it has nothing to load. A presenter cannot currently show an account, a
balance, or a transaction, which means there is nothing for a live feature to be added *to*.

This document specifies the base ledger: the smallest complete deposit-account ledger a presenter
can stand in front of and operate. It is deliberately **not** double-entry — there are no balancing
counter-entries. Each account owns a log of signed amounts, and its balance is the sum of that log.

## Goals

- A presenter can open one page and, with no setup beyond two commands, show an account, its
  balance, and its history.
- A presenter can record a transaction on stage and the audience can watch the balance change.
- A presenter can deliberately trigger a rejection and the audience can read why from the back of
  the room.
- Every run starts from an identical state, so a take can be redone mid-presentation.

## Non-Goals

Everything in this list is excluded **by design**, not deferred. The features added live on stage are
drawn from here; if any already exists, the demo is ruined.

- Duplicate or double-charge detection of any kind.
- Overdraft allowance, fees, or percentage calculations.
- Pending transactions or holds; available-versus-posted balance.
- Statements, exports, or reporting.
- Interest or rounding rules.
- Authentication, users, sessions, or multi-tenancy.
- Transfers between accounts, or any balancing counter-entry.
- Containerization, continuous integration, or deployment tooling.
- Metrics, tracing, or structured logging beyond what the standard library provides.

## Functional Requirements

### Viewing an account

- **FR-1** A presenter can see which demo accounts exist and choose one to view. Exactly one account
  is displayed at a time, and the choice is carried in the page's address so the same view can be
  reopened directly.
- **FR-2** The selected account's current balance is displayed as the most prominent element on the
  page, and is computed from that account's recorded transactions rather than read from a stored
  figure.
- **FR-3** The selected account's transactions are listed newest first, each showing its amount, its
  description, and when it was recorded. The ordering is a stable total order: two transactions
  recorded at the same instant still have a defined, repeatable position relative to each other.
- **FR-4** When the selected account has no transactions, the page shows a zero balance and an empty
  list — not an error and not a blank region.

### Recording a transaction

- **FR-5** A presenter can record a transaction against the selected account by supplying an amount
  and a description.
- **FR-6** Amounts are entered the way a person writes money — whole dollars or dollars and cents —
  and the sign of the amount determines whether the transaction adds to or subtracts from the
  balance.
- **FR-7** After a successful submission from the page, the page re-displays showing the new
  transaction in the list and the updated balance, and reloading that result does not record the
  transaction a second time.
- **FR-8** A programmatic client can record a transaction and receives the recorded transaction back
  on success.
- **FR-9** The page's submission and the programmatic submission are served by one and the same
  capability, so both are subject to identical validation and neither can bypass a rule the other
  enforces.

### Retrieving data programmatically

- **FR-10** A programmatic client can retrieve every account together with its current balance.
- **FR-11** A programmatic client can retrieve one account's transactions, newest first, in the same
  order the page shows.

### Rejecting invalid input

- **FR-12** A transaction is rejected in each of the following six cases. Every case is
  independently identifiable — a caller can tell which one failed, and no two report as each other.
  Nothing is recorded when a transaction is rejected.

  | | Case |
  |---|---|
  | **FR-12a** | The account does not exist. |
  | **FR-12b** | The amount is zero. |
  | **FR-12c** | The description is empty. |
  | **FR-12d** | The description exceeds 140 characters. |
  | **FR-12e** | The amount is not a well-formed money value. |
  | **FR-12f** | The transaction would take the account's balance below zero. |

- **FR-13** A rejection originating from the page is rendered visibly on the page, directly above the
  form that produced it. It is never written only to a log, never silently discarded, and never
  shown in a form that disappears on its own.
- **FR-14** A rejection returned to a programmatic client carries a stable machine-readable
  identifier for which rule failed, alongside a human-readable message.

### Operating the demo

- **FR-15** A single reset action restores a pristine starting state: three accounts, each with
  between eight and twelve plausible transactions, identical on every reset with no run-to-run
  variation of any kind.
- **FR-16** A single run action serves the page locally on a fixed, configurable port.

## Non-Functional Requirements

- **NFR-1 — Projector legibility.** The page must be readable at 1280×720 from across a room: 20px
  base text, near-black on white. Light theme only. No dark mode, no responsive breakpoints, no
  animation. Fixed by `.docs/decisions/styleguide.md`.
- **NFR-2 — Fully offline.** Zero network calls at build time or run time, including fonts. The demo
  must work with no connectivity.
- **NFR-3 — Determinism.** No randomness anywhere. Identifiers, ordering, timestamps, and seeded
  data are reproducible; two resets produce indistinguishable state.
- **NFR-4 — Test suite.** Full suite under 10 seconds, no ordering dependencies between tests, no
  sleeping. Roughly a 4:1 test-to-implementation line ratio, table-driven, with a negative case for
  every case in FR-12.
- **NFR-5 — Lint clean.** Formatting and vetting gates pass with no findings.
- **NFR-6 — Legibility of the code itself.** The implementation is read on a projector by an
  audience following a live diff, so it stays small and direct. This is a real constraint on the
  solution, not a preference.

## Acceptance Criteria

- Reset, then run, then open the page: an account, a balance, and a transaction list are visible
  without further action.
- Recording a valid transaction from the page changes the displayed balance by exactly the amount
  entered.
- Each of the six cases in FR-12 can be triggered on demand and produces a visible,
  distinguishable rejection on the page and a distinguishable machine-readable identifier
  programmatically.
- The balance shown on the page for a given account equals the balance the programmatic listing
  reports for that same account, always.
- Two consecutive resets produce identical data.
- The suite passes under 10 seconds; the formatting and vetting gates report nothing.

## Scope

**In scope:** accounts and their derived balances; a signed transaction log per account; the six
validation rules; the single page with selector, balance, form, list, and visible errors; the
programmatic read and write capabilities; deterministic seed data; reset and run actions.

**Out of scope:** everything under Non-Goals, without exception.

## Key Decisions & Rationale

| Decision | Rationale |
|---|---|
| One account displayed at a time, chosen by the presenter | Keeps the balance the single largest thing on screen. Three balances side by side gives the audience nothing to focus on. Follows the layout order already fixed in the styleguide. |
| Amounts entered as people write money, with the sign carrying the direction | One input field instead of an amount plus a direction control. Less to explain and less to mis-click while talking. |
| Balance derived from the log rather than stored | The audience must be able to trust that the number moved *because* a transaction was recorded. A stored balance invites the question of whether the two agree. |
| Rejections rendered above the form, permanently | Triggering a rejection is a scripted stage moment. A message that fades, or that only a developer console shows, wastes it. |
| Reset restores identical state rather than merely clearing | A take can be redone mid-presentation and look exactly like the first attempt. |
| Empty account shows zero rather than an error | The presenter may want to build an account up from nothing on stage. |

## Known Characteristics (accepted, not to be fixed)

The balance check reads the current balance and then records the transaction, without holding a lock
across the two. Two submissions arriving at genuinely the same moment could both observe a
sufficient balance and both be recorded. This is documented in `.docs/architecture/sequences.md` and
is deliberately left alone: it cannot occur with one presenter driving one browser, and serialising
it would add machinery that makes the code harder to read on a projector. It is recorded here so it
is a known property rather than a discovered surprise. **No requirement in this document asks for it
to be addressed.**

## Dependencies

These are pre-existing constraints this feature must live within. None is chosen here; each is
already committed and settled.

| Constraint | Recorded in |
|---|---|
| Money is handled as whole cents in a 64-bit integer. No floating point anywhere, ever. | `adr-2026-08-08-money-as-int64-cents.md` (Accepted) |
| The current time is injected rather than read directly, so tests are deterministic. | `adr-2026-08-08-injected-clock.md` (Accepted) |
| Domain failures are distinguishable sentinel values, not strings compared by value. | `adr-2026-08-08-sentinel-errors-for-domain-failures.md` (Accepted) |
| The persistence contract is declared by the domain and satisfied by the storage layer, so the domain has no knowledge of the database. | `adr-2026-08-08-store-interface-in-domain-package.md` (Accepted) |
| Data shape: accounts and transactions as specified, with no stored balance, amounts as integers, and no uniqueness constraint beyond the primary key. | `.docs/architecture/erd.md` |
| Page layout order, palette, type scale, and error presentation. | `.docs/decisions/styleguide.md` |
| The address surface the page and programmatic clients use is already published in the routing documentation and the posting sequence diagram. | `internal/httpapi/router.go:17-19`, `.docs/architecture/sequences.md` |
| Mandated platform: Go with only the standard library for serving and testing, one pinned pure-Go database dependency, standard-library templating, and no front-end build step or client-side scripting. | `CLAUDE.md` (Tech Stack) |
| Seed shape: three accounts, eight to twelve transactions each, fixed timestamps. | `cmd/server/main.go:59-61` |

## Open Questions

Both entries below are load-bearing technical choices with operator-confirmed direction. They are
recorded here rather than decided in this document; `/architecture-review` weighs them and captures
each as an ADR.

1. **One posting capability serving both the page form and programmatic clients, versus two separate
   ones.** FR-9 requires that neither can bypass the other's validation, which either shape can
   satisfy. Operator-confirmed direction: a single capability that adapts its response to the caller,
   keeping the published address surface unchanged. Trade-off to weigh: one entry point with two
   response modes versus two single-purpose entry points and a larger published surface.
2. **How stable identity and stable ordering are achieved under an injected clock.** FR-3 requires a
   repeatable total order even for transactions recorded at the same instant, and NFR-3 forbids
   randomness in identifiers. Operator-approved direction: sequential, zero-padded identifiers, with
   ordering falling back to identifier order when timestamps tie — which needs no change to the
   committed data shape. Trade-off to weigh: that, versus adding a dedicated monotonic ordering
   field, which would modify `.docs/architecture/erd.md`.
