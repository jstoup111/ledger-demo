# Stories — Base Ledger

**Status:** Accepted

**Feature:** base-ledger · **Tier:** M · **Track:** product
**Source:** `.docs/specs/2026-08-08-base-ledger.md` (Approved, FR-1 … FR-16)
**Constrained by:** the six Accepted ADRs in `.docs/decisions/` and
`.docs/decisions/api-response-contract.md`

Six stories covering all sixteen FRs. Grouped by capability rather than one-per-FR, per the
operator's instruction to keep this as simple as possible.

## Negative-path categories evaluated

The skill requires every category to be explicitly evaluated. Most do not apply to this system, and
saying so is more useful than inventing a scenario:

| Category | Applies? |
|---|---|
| Invalid input | **Yes** — six rules, Story 5, plus the client-supplied `error` and `account` query values |
| Data integrity | **Yes** — a transaction must never reference a nonexistent account (FR-12a) |
| Model-level immutability | **Yes** — the log is append-only; Story 2 asserts no route mutates or removes a recorded transaction |
| Invariant side-effect on alternate branches | **Yes** — the posting endpoint has two response branches; Story 4 asserts validation and the "nothing recorded" invariant hold identically on both |
| Dependency unavailability | **Yes** — the database file is the one dependency; Story 6 covers an unopenable database |
| Auth / permission failures | No — auth is an explicit non-goal; there is no principal to reject |
| Timeouts & network errors | No — NFR-2 forbids network calls; there is no external dependency to time out |
| Resource exhaustion | No — no uploads, no batch processing, no connection pool |
| Partial failure & rollback | No — recording a transaction is a single insert, not a multi-step operation |
| Concurrent access | No — deliberately. The unlocked read-then-write balance check is a recorded accepted characteristic; the PRD states no requirement asks for it to be addressed |
| Cascade deletion effects | No — nothing is ever deleted; `make reset` drops the database file wholesale |
| Exception class hierarchy | No — Go sentinel errors compared with `errors.Is`, no exception tree |
| Dedup / idempotency key analysis | No — there is no dedup or idempotency criterion anywhere in this spec |

---

**Status:** Accepted

## Story 1: Choose an account and see its balance

**Requirement:** FR-1, FR-2, FR-4, FR-10

As a presenter, I want to pick one of the demo accounts and see its current balance in large type, so
the audience has a single number to watch.

### Acceptance Criteria

#### Happy Path
- Given a seeded database, when the presenter opens `GET /`, then the page lists all three accounts by
  name as selectable links, and exactly one account's detail is shown.
- Given the presenter opens `GET /?account=acct-2`, when the page renders, then `acct-2`'s name and
  balance are shown, and no other account's balance appears anywhere on the page.
- Given `acct-1` has transactions summing to `128350` cents, when `GET /?account=acct-1` renders, then
  the balance displays as `$1,283.50` inside an element carrying the `balance` class.
- Given a seeded database, when a client requests `GET /api/accounts`, then the response is `200` with
  `Content-Type: application/json; charset=utf-8` and a JSON array of `{id, name, balance_cents}`
  ordered by `id` ascending, where each `balance_cents` is an integer equal to the sum of that
  account's `amount_cents`.
- Given an account with no transactions, when `GET /?account=acct-3` renders, then the balance shows
  `$0.00` and the transaction area shows an explicit empty-state message — not an error, not a blank
  region.

#### Negative Paths
- Given no `account` query parameter, when `GET /` is requested, then the page renders `200` showing
  the first account by id ascending, not a `404` and not an error panel.
- Given `GET /?account=acct-nope` for an account that does not exist, when the page renders, then it
  responds `200` with a visible message stating the account was not found, and no balance element is
  rendered.

  > **Amended 2026-08-08 by base-ledger conflict-check (F2):** the assertion now also fixes what
  > else the page omits — for an unknown account the page renders the account list and the
  > not-found message **only**: no balance element, no transaction list, and **no post form**.
  > Rendering the form would offer the presenter a submission that can only fail, since there is no
  > valid account to post against.
- Given `GET /?account=%3Cscript%3Ealert(1)%3C%2Fscript%3E`, when the page renders, then the value is
  HTML-escaped in the output and no raw `<script>` tag appears in the response body.
- Given a request for `GET /api/accounts` while the database file cannot be opened, when the handler
  runs, then the response is `500` with `Content-Type: text/plain` and the failure is written to the
  stdlib `log`.
- Given `POST /api/accounts`, when the request is made, then the response is `405` with an empty body.

### Done When
- [ ] `GET /` returns `200 text/html` and contains all three seeded account names.
- [ ] `GET /?account=acct-2` renders `acct-2`'s balance and no other account's balance.
- [ ] `GET /api/accounts` returns `200` and a JSON array of exactly three objects with keys `id`,
      `name`, `balance_cents`, ordered by `id` ascending.
- [ ] Every `balance_cents` in that response equals the integer sum of that account's transaction
      `amount_cents`, asserted against a fixture whose expected totals are written out literally.
- [ ] No stored balance column exists in the schema — verified by asserting the `transactions` and
      `accounts` table definitions.
- [ ] An account with zero transactions returns `balance_cents: 0` and renders an empty-state message.
- [ ] A table-driven test covers the four page cases: no param, valid account, unknown account,
      escaped-injection param.

---

**Status:** Accepted

## Story 2: Read the transaction log in a stable newest-first order

**Requirement:** FR-3, FR-11

As a presenter, I want the transaction list to be newest first and to be in the same order every
single run, so what the audience saw in rehearsal is what they see on stage.

### Acceptance Criteria

#### Happy Path
- Given `acct-1` has transactions, when `GET /?account=acct-1` renders, then each row shows the
  amount formatted as dollars, the description, and the recorded timestamp.
- Given three transactions written under one `FixedClock` instant `2026-08-08T14:30:00Z` with ids
  `txn-0001`, `txn-0002`, `txn-0003`, when the list is read, then the order is exactly
  `txn-0003`, `txn-0002`, `txn-0001` — the identifier breaks the timestamp tie.
- Given the same fixture, when the list is read a second time in the same process and a third time in
  a fresh process, then the order is identical on all three reads.
- Given a client requests `GET /api/accounts/acct-1/transactions`, then the response is `200` with a
  JSON array of `{id, account_id, amount_cents, description, created_at}`, newest first, in the exact
  same order the page shows, with `created_at` in RFC 3339 UTC and `amount_cents` an integer.
- Given a debit of `-42.50` was recorded, when it is read back, then `amount_cents` is `-4250` — a
  negative integer, never a float and never a formatted string.
- Given seed data of three accounts, when all transaction ids are collected, then they are globally
  unique across the whole table and every id matches `txn-` followed by exactly four digits.

#### Negative Paths
- Given `GET /api/accounts/acct-3/transactions` for an account with no transactions, when the request
  is made, then the response is `200` with body `[]` — not `null`, and not `404`.
- Given `GET /api/accounts/acct-nope/transactions` for an account that does not exist, when the
  request is made, then the response is `404` with body
  `{"error":{"code":"account_not_found","message":"…"}}`.
- Given a recorded transaction, when any request attempts to modify or remove it (`PUT`, `PATCH`, or
  `DELETE` on `/api/accounts/acct-1/transactions`), then the response is `405` with an empty body and
  the stored row is byte-for-byte unchanged. The log is append-only, which is the precondition the
  count-derived identifier scheme in `adr-2026-08-08-deterministic-transaction-ids-and-ordering.md`
  depends on.
- Given the transactions table contains an id of a different width than the others, when the ordering
  is exercised, then a test documents that the lexicographic tiebreak no longer holds — proving the
  constant-width requirement is load-bearing rather than cosmetic.
- Given a request for a transaction list while the database is unreadable, when the handler runs,
  then the response is `500 text/plain` and the failure is logged.

### Done When
- [ ] A test writes at least three transactions under a single `FixedClock` instant and asserts the
      exact newest-first id sequence — not merely that the list is non-empty.
- [ ] That ordering test passes when run repeatedly and when run with `-count=2`, with no `time.Sleep`
      anywhere in the suite.
- [ ] `GET /api/accounts/{id}/transactions` order is asserted equal to the order rendered on the page
      for the same account.
- [ ] An empty account returns literal `[]`.
- [ ] An unknown account returns `404` with code `account_not_found`.
- [ ] `PUT`, `PATCH`, and `DELETE` against the transactions path each return `405`.
- [ ] Every seeded id matches `^txn-\d{4}$` and the set is globally unique across all three accounts.

---

**Status:** Accepted

## Story 3: Record a transaction from the page

**Requirement:** FR-5, FR-6, FR-7, FR-13

As a presenter, I want to type an amount and a description and watch the balance move, so the audience
sees cause and effect.

### Acceptance Criteria

#### Happy Path
- Given `GET /?account=acct-1` is open, when the page renders, then a form is present with an amount
  field and a description field, and it submits by `POST` to
  `/api/accounts/acct-1/transactions`.
- Given the form is submitted with amount `25` and description `Deposit`, when the request is
  processed, then the response is `303 See Other` with `Location: /?account=acct-1`, and following
  that redirect shows a balance increased by `2500` cents and the new transaction at the top of the
  list.
- Given the form is submitted with amount `-42.50`, when it is processed, then the recorded
  `amount_cents` is `-4250` and the displayed balance decreases by that amount.
- Given amount `3.50`, when it is parsed, then the stored value is exactly `350` cents, computed by
  string parsing with no floating-point arithmetic anywhere in the path.
- Given a successful form submission, when the presenter reloads the resulting page, then no
  additional transaction is recorded and the balance is unchanged — the result of the POST is a `GET`,
  so a reload re-issues only the `GET`.
- Given the page renders at 1280×720, when inspected, then the base font size is `20px`, the balance
  uses the largest type on the page, the stylesheet contains no `@media` query and no `prefers-color-scheme`
  block, and the page loads no JavaScript and no external font.

#### Negative Paths
- Given the form is submitted with an empty description, when it is processed, then the response is
  `303` with `Location: /?account=acct-1&error=description_empty`, following it renders a visible
  error panel above the form, and no transaction row was written.
- Given the form is submitted with amount `abc`, when it is processed, then the redirect carries
  `error=amount_malformed` and the rendered page shows the corresponding human-readable message.

  > **Amended 2026-08-08 by base-ledger conflict-check (F1):** this story no longer owns rule
  > semantics. Story 5 owns which rule fires and what it reports; Story 4 owns that the same rule
  > fires through both content types. This story keeps **one** representative rejection
  > (`description_empty`, above) because its actual subject is that a code arriving in the URL
  > becomes a visible panel on the page — not that the six rules are correct. The `amount_malformed`
  > scenario here is retained as a second data point only; it must not be re-asserted at the
  > sentinel level, which belongs to Story 5.
- Given a redirect arrives carrying `error=not_a_real_code`, when the page renders, then a **generic**
  rejection message is displayed in the error panel — never an empty panel and never a blank region.
- Given a redirect arrives carrying `error=<script>alert(1)</script>`, when the page renders, then the
  value is HTML-escaped and no raw `<script>` tag appears in the body.
- Given an account id containing characters requiring escaping, when the redirect `Location` is built,
  then the id is properly escaped in the header value.
- Given a rejection occurs, when the response is produced, then the message is present in the HTML
  body — asserted by inspecting the response, proving it is not written only to the log.

### Done When
- [ ] `GET /?account=acct-1` response body contains a `<form>` whose `action` is
      `/api/accounts/acct-1/transactions` and whose `method` is `post`.
- [ ] A form-encoded `POST` with valid input returns exactly `303` with
      `Location: /?account=acct-1`.
- [ ] Following that redirect shows the balance changed by precisely the submitted amount in cents.
- [ ] A test asserts `3.50` → `350`, `25` → `2500`, `-42.50` → `-4250`, and `0.01` → `1`.
- [ ] `grep` confirms no `float64`, `float32`, or `ParseFloat` anywhere in the money path.
- [ ] A form-encoded rejection returns `303` with the matching `error` code in `Location`, and a
      follow-up `GET` renders the message inside an element carrying the `error` class.
- [ ] `error=not_a_real_code` renders a non-empty generic message.
- [ ] Repeating the follow-up `GET` twice leaves the transaction count unchanged.
- [ ] `web/style.css` contains no `@media`, no `prefers-color-scheme`, no `@keyframes`, and no
      `@font-face`; the rendered HTML contains no `<script>` tag.

---

**Status:** Accepted

## Story 4: Record a transaction programmatically, through the same validation

**Requirement:** FR-8, FR-9, FR-14

As a programmatic client, I want to record a transaction and get it back, and as a presenter I want
certainty that the API and the page cannot disagree about what is valid.

### Acceptance Criteria

#### Happy Path
- Given a `POST /api/accounts/acct-1/transactions` with `Content-Type: application/json` and body
  `{"amount":"-42.50","description":"Coffee beans"}`, when processed, then the response is `201` with
  the created transaction as `{id, account_id, amount_cents, description, created_at}`, where
  `amount_cents` is `-4250`.
- Given that response, when the returned `id` is read, then it matches `txn-` plus exactly four
  digits, and the transaction is subsequently retrievable at
  `GET /api/accounts/acct-1/transactions`.
- Given the request body contains an unrecognized extra field, when processed, then the field is
  ignored and the transaction is created normally.
- Given identical invalid input is submitted once as `application/json` and once as
  `application/x-www-form-urlencoded`, when both are processed, then **both are rejected for the same
  rule** — differing only in response shape (`400`/`404` JSON versus `303` with the code in
  `Location`). Neither content type admits input the other refuses.
- Given a valid submission through either content type, when the resulting stored row is compared,
  then the persisted `amount_cents`, `description`, and `account_id` are identical for both.

#### Negative Paths
- Given a rejection, when the JSON response is read, then the body is exactly
  `{"error":{"code":"…","message":"…"}}`, where `code` is one of the six documented values and
  `message` is a non-empty human-readable string.
- Given a `POST` with `Content-Type: application/json` and a body that is not valid JSON, when
  processed, then the response is `400` with code `amount_malformed`, and nothing is recorded.
- Given a `POST` whose body omits `amount` entirely, when processed, then the response is `400` with
  code `amount_malformed`.
- Given a `POST` with body `{"amount":-42.50,"description":"x"}` where the amount is a JSON **number**
  rather than a string, when processed, then the response is `400` with code `amount_malformed` — the
  contract specifies a string, and accepting a number would introduce a float into the money path.
- Given a rejection on the JSON branch, when the store is inspected, then the transaction count is
  unchanged. Given the same rejection on the form branch, then the count is likewise unchanged — the
  "nothing is recorded" invariant is asserted on **both** branches, not inferred from one.
- Given a `POST` with no `Content-Type` header at all, when processed, then the request is handled
  deterministically by a documented default rather than panicking, and the chosen behavior is asserted
  by a test.

### Done When
- [ ] A JSON `POST` with valid input returns `201` and a body matching the contract's field set
      exactly, with `amount_cents` an integer.
- [ ] The created transaction appears in the subsequent list response.
- [ ] A table-driven test submits each of the six invalid inputs through **both** content types and
      asserts the same rule fires for each pair — this is the executable form of FR-9.

      > **Amended 2026-08-08 by base-ledger conflict-check (F1):** this test asserts **equivalence
      > only** — that the rule which fires is the same for both content types, identified by the
      > wire `code`. It does **not** re-assert sentinel identity or the nothing-recorded invariant
      > per rule; both belong to Story 5. Scoped this way the matrix cannot drift out of agreement
      > with Story 5, because it takes Story 5's codes as its input rather than restating them.
- [ ] Every rejection response body matches `{"error":{"code","message"}}` with a non-empty `message`.
- [ ] A JSON-number `amount` is rejected as `amount_malformed`.
- [ ] Transaction count is asserted unchanged after a rejection on each branch separately.
- [ ] A missing `Content-Type` produces the documented default, asserted by test rather than left
      undefined.

---

**Status:** Accepted

## Story 5: Every rule rejects distinguishably, and records nothing

**Requirement:** FR-12, FR-12a, FR-12b, FR-12c, FR-12d, FR-12e, FR-12f

As a presenter, I want to trigger a specific rejection on demand and have the audience read exactly
why, so a validation failure is a scripted moment rather than a mystery.

> **Ownership (set 2026-08-08 by base-ledger conflict-check, F1):** this story is the single owner of
> **rule semantics** — which sentinel each rule returns, which wire code it maps to, and that nothing
> is recorded. Story 4 consumes these codes to assert both content types agree; Story 3 consumes one
> of them to assert a code becomes a visible panel. Neither re-derives them.

### Acceptance Criteria

#### Happy Path

The "happy path" for this story is that a valid transaction passes all six rules and is recorded:

- Given `acct-1` with a balance of `128350` cents, when a transaction of `-1000` cents with a
  20-character description is submitted, then it is recorded and no rule fires.
- Given a description of exactly 140 characters, when submitted, then it is **accepted** — the
  boundary is inclusive.
- Given an amount of `0.01` (one cent), when submitted, then it is accepted — the smallest non-zero
  amount is valid.

#### Negative Paths

One scenario per rule. Each asserts a distinct sentinel via `errors.Is`, a distinct wire code, and
that nothing was written.

- **FR-12a** Given `POST /api/accounts/acct-nope/transactions` with an otherwise valid body, when
  processed, then `404` with code `account_not_found`, the domain returns a sentinel matching
  `errors.Is(err, ledger.ErrAccountNotFound)`, and no row is written.
- **FR-12b** Given amount `0`, `0.00`, or `-0.00`, when submitted, then `400` with code `amount_zero`,
  its own sentinel, and no row written.
- **FR-12c** Given a description that is empty, and separately one that is only whitespace, when
  submitted, then `400` with code `description_empty`, its own sentinel, and no row written.
- **FR-12d** Given a description of exactly 141 characters, when submitted, then `400` with code
  `description_too_long`, its own sentinel, and no row written. The 140/141 pair pins the boundary
  from both sides.
- **FR-12e** Given each of `abc`, `1.2.3`, `1,000`, `$5`, `1.234`, `` (empty), and `  ` as the amount,
  when submitted, then `400` with code `amount_malformed`, its own sentinel, and no row written.
- **FR-12f** Given `acct-1` with a balance of `1000` cents, when a transaction of `-1001` cents is
  submitted, then `400` with code `balance_would_go_negative`, its own sentinel, and no row written.
  Given `-1000` against the same balance, then it is **accepted** and the resulting balance is exactly
  `0` — landing exactly on zero is permitted.
- Given all six rejections, when their codes are collected, then all six are distinct, and each maps
  to exactly one sentinel — no two rules share a code and no rule reports as another.

### Done When
- [ ] Six table-driven negative cases, one per rule, each asserting the sentinel via `errors.Is`
      **and** the exact wire `code` string.
- [ ] Boundary pairs asserted from both sides: 140 accepted / 141 rejected; `-1000` accepted to a zero
      balance / `-1001` rejected; `0.01` accepted / `0` rejected.
- [ ] The malformed-amount case is table-driven over at least the seven listed inputs.
- [ ] After each rejection, the transaction count for that account is asserted unchanged.
- [ ] A test asserts the six codes are pairwise distinct.
- [ ] The whitespace-only description is rejected as `description_empty`, not as valid.

---

**Status:** Accepted

## Story 6: Reset and run the demo, identically every time

**Requirement:** FR-15, FR-16

As a presenter, I want one command to restore a pristine stage and one to serve it, so a take can be
redone mid-presentation and look exactly like the first attempt.

### Acceptance Criteria

#### Happy Path
- Given any prior database state, when `make reset` runs, then the database file is removed and
  recreated, and it contains exactly three accounts, each with between 8 and 12 transactions.

  > **Amended 2026-08-08 by base-ledger conflict-check (F3):** every automated test in this story
  > that needs a real file sets `LEDGER_DB_PATH` to a **`t.TempDir()`** path and never touches the
  > default `./ledger.db`. The reset and two-reset-identity assertions are genuinely file-backed —
  > `reset` deletes a file, which in-memory SQLite cannot express — so they are the one sanctioned
  > exception to the in-memory-for-tests convention. Confining them to a per-test temporary
  > directory keeps them from contending with each other under `-count=2`, with tests in other
  > packages, or with a `make dev` server holding the default database open.
- Given `make reset` is run twice in a row, when the two resulting databases are compared, then every
  account row and every transaction row is identical, including all `created_at` values and all ids.
- Given seeded data, when the transaction ids are read, then they form one unbroken globally
  sequential run `txn-0001` … `txn-00NN` across the whole table — not restarting per account.
- Given `make dev`, when the server starts, then it listens on the port from `PORT` (default `8080`)
  and `GET /` returns `200`.
- Given `LEDGER_DB_PATH` is set to a different path, when the server starts, then it uses that file,
  so two worktrees can run simultaneously without collision.
- Given the whole suite, when `make test` runs, then it completes in under 10 seconds, passes with
  `-count=2`, and contains no `time.Sleep` call.
- Given the repository, when `make check` runs, then `gofmt -l .` reports nothing and `go vet ./...`
  reports nothing.

#### Negative Paths
- Given `LEDGER_DB_PATH` points at a directory that does not exist, when `seed` runs, then it exits
  non-zero with a message naming the path, rather than creating a stray file or panicking.
- Given `LEDGER_DB_PATH` points at a file that is not a valid database, when the server starts, then
  it fails with a clear logged error rather than serving `500`s on every request.
- Given the seed command is run twice against the same existing database without a drop, when it runs,
  then it does not silently double the data — `make reset` removes the file first, and the seed path
  itself is asserted to start from an empty schema.
- Given an unknown subcommand such as `go run ./cmd/server frobnicate`, when it runs, then it exits
  non-zero naming the valid commands.
- Given the seed code, when inspected, then it contains no call to `time.Now()` and no randomness —
  `grep` finds `time.Now()` in exactly one place in the repository, inside `SystemClock`.
- Given the server is started with no network available, when the page is requested, then it renders
  fully — no font fetch, no CDN, no outbound call.

### Done When
- [ ] `make reset` produces exactly 3 accounts and between 24 and 36 transactions total.
- [ ] Two consecutive resets produce identical row sets, asserted by comparing all rows including
      `created_at` and ids.
- [ ] Seeded ids form one global sequence with no per-account restart and no gaps.
- [ ] `grep -rn 'time.Now()'` over the repository returns exactly one hit, in `SystemClock`.
- [ ] `make test` completes under 10 seconds and passes with `-count=2`.
- [ ] `gofmt -l .` is empty and `go vet ./...` is clean.
- [ ] A non-existent `LEDGER_DB_PATH` directory exits non-zero with the path in the message.
- [ ] An unknown subcommand exits non-zero listing `serve` and `seed`.
