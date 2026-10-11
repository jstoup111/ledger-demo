# Coherence: Open a new account

Date: 2026-10-10
Track: product
Tier: M
Verdict: covered

## Mapping

Maps [the PRD](../specs/2026-10-10-open-a-new-account.md) (FR-1 … FR-9) →
[the stories](../stories/open-a-new-account.md) (7 stories, 35 criteria) →
[the plan](../plans/open-a-new-account.md) (12 tasks in 2 stacked slices).

- **Outcome rows** are omitted: this is a chat-origin idea with no intake marker or staged outcomes.
- **ADR rows:** one — the only `adr-*` file in this spec's change set. `api-response-contract.md`
  carries an additive amendment but is not an `adr-*` file.
- **Criterion rows** are identical to the plan's `## Coverage Check` rows; each quote is exact text
  from a cited task's `Done when` block.
- **Slices:** Stories 1–3 are owned wholly by slice 1 (Tasks 1–5), Stories 4–7 wholly by slice 2
  (Tasks 6–12); no slice 1 task depends on a slice 2 task.

**Independent judgements** (fresh subagents, plan §7a/§7b): coverage — all 35 criteria assert in
round 1; contradiction — no conflicts across all 12 tasks, 35 criteria, and the decisions of `adr-2026-10-10-sequential-account-ids-ordered-by-sequence` and `adr-2026-08-08-one-negotiated-posting-endpoint` (the final plan is unchanged since that sweep). The `PUT /api/accounts` empty-body `405` comes from the existing `emptyDefaultErrors` wrapper in `internal/httpapi/router.go`.

**Achievability (§4g):** every criterion is reached through a named entry point — the domain
`OpenAccount`/`SortAccounts` functions for Stories 1–3 and the ordering rule, and `NewRouter` request
handling for Stories 4–7. No criterion relies on a value outside the closed sets the ADR defines (two
codes, two sentinels).

## Table

| Row class | Cited id(s) | Counterpart id(s) | Verdict | Quote / Notes |
| --- | --- | --- | --- | --- |
| fr | fr-1 | story-1 | covered | Opening records an empty named account (Story 1 happy 1). |
| fr | fr-2 | story-1 | covered | Deterministic acct-N continuation; same openings yield same ids (Story 1, Task 3 two-database check). |
| fr | fr-3 | story-2 | covered | Blank and whitespace-only names rejected, nothing recorded. |
| fr | fr-4 | story-3 | covered | 40/41 boundary in characters, single- and multi-byte. |
| fr | fr-5 | story-4 | covered | 201 with the account-list shape and zero balance. |
| fr | fr-6 | story-5 | covered | 400 with distinct stable codes; too-long message names count and limit. |
| fr | fr-7 | story-6 | covered | Form on the selected-account page; success selects the new account. Unknown/zero-account pages unchanged (conflict resolution). |
| fr | fr-8 | story-6 | covered | Same message as JSON, above the open-account form; nothing opened. |
| fr | fr-9 | story-7 | covered | Numeric sequence order on both listings past nine accounts. |
| story | story-1 | task-1, task-2, task-3 | covered | Open an account in the domain |
| story | story-2 | task-4 | covered | Reject a blank account name |
| story | story-3 | task-5 | covered | Limit account names to 40 characters |
| story | story-4 | task-6, task-7 | covered | Open an account with a JSON request |
| story | story-5 | task-8 | covered | JSON rejections name the broken rule |
| story | story-6 | task-9, task-10 | covered | Open an account from the page |
| story | story-7 | task-11, task-12 | covered | Accounts are listed in the order they were opened |
| task | task-1 | story-1 | covered | Open an account in the domain with the next sequential id (happy-path) |
| task | task-2 | story-1 | covered | Storage failures while opening return a wrapped error and record nothing (negative-path) |
| task | task-3 | story-1 | covered | Opening through the real SQLite store (infrastructure) |
| task | task-4 | story-2 | covered | Reject a blank account name (negative-path) |
| task | task-5 | story-3 | covered | Limit account names to 40 characters (negative-path) |
| task | task-6 | story-4 | covered | `POST /api/accounts` opens an account for a JSON client (happy-path) |
| task | task-7 | story-4 | covered | Malformed JSON bodies and unsupported methods (negative-path) |
| task | task-8 | story-5 | covered | Codes and messages for the two name rules (negative-path) |
| task | task-9 | story-6 | covered | The page's open-account form opens and selects the account (happy-path) |
| task | task-10 | story-6 | covered | Page rejections render above the open-account form (negative-path) |
| task | task-11 | story-7 | covered | A domain ordering rule for accounts (infrastructure) |
| task | task-12 | story-7 | covered | Both account listings use sequence order (happy-path) |
| adr | adr-2026-10-10-sequential-account-ids-ordered-by-sequence | story-1, story-2, story-3, story-4, story-5, story-6, story-7 | covered | New ADR. D1 → task 1 (acct-4 from count+1); D2 → tasks 11, 12 (sequence order on both listings); D3 → tasks 4, 5, 8 (sentinels and their 400 codes); D4 → tasks 6, 7, 9, 10 (negotiated route, malformed-body rule, form redirects, message placement). Each mapping judged against the cited Done-when checks. |
| criterion | Story 1 happy: Given the seeded ledger, when an account named "Holiday Fund" is opened, then the returned account has id `acct-4` and name "Holiday Fund", the store lists four accounts, its derived balance is 0, and it has no transactions. | 1 | covered | "returns id `acct-4` and name "Holiday Fund", the fake then lists four accounts" | diff-local |
| criterion | Story 1 happy: Given the seeded ledger after opening "Holiday Fund", when a second account named "Holiday Fund" is opened, then it is returned with id `acct-5`, both accounts exist, and neither opening is rejected for sharing a name. | 1 | covered | "returns `acct-5` with no error, and the fake lists both acct-4 and acct-5" | diff-local |
| criterion | Story 1 happy: Given a ledger with no accounts, when an account named "First" is opened, then it is returned with id `acct-1`. | 1 | covered | "over an empty fake store returns id `acct-1`" | diff-local |
| criterion | Story 1 negative: Given a store whose account listing fails, when an account is opened, then an error wrapping the storage failure is returned, no account is returned, and nothing is inserted. | 2 | covered | "When the fake's `Accounts` fails, `OpenAccount` returns a zero `Account`" | diff-local |
| criterion | Story 1 negative: Given a store whose insert fails, when an account is opened, then an error wrapping the storage failure is returned and no account is returned. | 2 | covered | "When the fake's `InsertAccount` fails, `OpenAccount` returns a zero `Account`" | diff-local |
| criterion | Story 2 happy: Given the seeded ledger, when an account named "  Rainy Day  " (surrounding spaces) is opened, then it is accepted and stored with the name exactly as submitted, spaces included. | 4 | covered | "the stored account's name equals `"  Rainy Day  "` byte for byte" | diff-local |
| criterion | Story 2 negative: Given the seeded ledger, when an account is opened with an empty name, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts. | 4 | covered | "returns an error satisfying `errors.Is(err, ledger.ErrAccountNameEmpty)`, the fake still lists three accounts" | diff-local |
| criterion | Story 2 negative: Given the seeded ledger, when an account is opened with a name of only spaces, tabs, and a newline, then the error satisfies `errors.Is(err, ledger.ErrAccountNameEmpty)` and the store still lists three accounts. | 4 | covered | "For names `""`, `"   "`, and `" \t\n"`" | diff-local |
| criterion | Story 3 happy: Given the seeded ledger, when an account is opened with a name of exactly 40 ASCII characters, then it is accepted as `acct-4`. | 5 | covered | "A 40-ASCII-character name is accepted as `acct-4`" | diff-local |
| criterion | Story 3 happy: Given the seeded ledger, when an account is opened with a name of exactly 40 "é" characters (80 bytes), then it is accepted, because length counts characters, not bytes. | 5 | covered | "a name of 40 `é` characters (80 bytes) is accepted" | diff-local |
| criterion | Story 3 negative: Given the seeded ledger, when an account is opened with a name of 41 ASCII characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and the store still lists three accounts. | 5 | covered | "A 41-ASCII-character name and a 41-`é` name each return an error satisfying `errors.Is(err, ledger.ErrAccountNameTooLong)`" | diff-local |
| criterion | Story 3 negative: Given the seeded ledger, when an account is opened with a name of 41 "é" characters, then the error satisfies `errors.Is(err, ledger.ErrAccountNameTooLong)` and nothing is recorded. | 5 | covered | "after each the fake still lists three accounts with zero `InsertAccount` calls" | diff-local |
| criterion | Story 4 happy: Given the seeded ledger, when `POST /api/accounts` is sent with `Content-Type: application/json` and body `{"name":"Holiday Fund"}`, then the response is `201` with `Content-Type: application/json; charset=utf-8` and body `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`. | 6 | covered | "returns `201`, `Content-Type: application/json; charset=utf-8`, and body exactly `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}`" | diff-local |
| criterion | Story 4 happy: Given that account was just opened, when `GET /api/accounts` is requested, then its fourth element is `{"id":"acct-4","name":"Holiday Fund","balance_cents":0}` and `GET /api/accounts/acct-4/transactions` returns `[]`. | 6 | covered | "`GET /api/accounts/acct-4/transactions` returns `[]`" | diff-local |
| criterion | Story 4 happy: Given that account was just opened, when a JSON transaction of amount "25.00" is posted to `/api/accounts/acct-4/transactions`, then it is accepted with `201` and `acct-4` lists a balance of 2500 cents. | 6 | covered | "returns `201` and `GET /api/accounts` then reports acct-4 with `balance_cents` 2500" | diff-local |
| criterion | Story 4 negative: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `not json`, then the response is `400` with code `account_name_empty` and `GET /api/accounts` still lists three accounts. | 7 | covered | "with body `not json`, with `{"name":42}`, and with `{}` each returns `400` with error code `account_name_empty`" | diff-local |
| criterion | Story 4 negative: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with body `{"name":42}` or `{}`, then the response is `400` with code `account_name_empty` and nothing is recorded. | 7 | covered | "`GET /api/accounts` afterwards still returns exactly three accounts" | diff-local |
| criterion | Story 4 negative: Given the running server, when `PUT /api/accounts` is sent, then the response is `405` with an empty body. | 7 | covered | "`PUT /api/accounts` through `NewRouter` returns `405` with an empty body." | diff-local |
| criterion | Story 5 happy: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with name "   ", then the response is `400` with body `{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}`. | 8 | covered | "{"error":{"code":"account_name_empty","message":"Account name must not be empty."}}" | diff-local |
| criterion | Story 5 happy: Given the seeded ledger, when `POST /api/accounts` is sent as JSON with a 41-character name, then the response is `400` with body `{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}`. | 8 | covered | "{"error":{"code":"account_name_too_long","message":"Account name is too long. Submitted: 41 characters; the limit is 40."}}" | diff-local |
| criterion | Story 5 negative: Given either rejection above, when `GET /api/accounts` is requested afterwards, then it still lists exactly three accounts and the next successful opening is `acct-4` (no id was consumed). | 8 | covered | "After both rejections `GET /api/accounts` returns exactly three accounts, and the next successful JSON opening returns id `acct-4`." | diff-local |
| criterion | Story 5 negative: Given the existing seven error codes, when this feature ships, then each still returns its original HTTP status and code string for its original rule. | 8 | covered | "the existing table for the original seven sentinels passes with every status and code string unchanged" | diff-local |
| criterion | Story 6 happy: Given the seeded ledger, when `GET /` is requested, then the page contains an "Open account" form that posts a `name` field to `/api/accounts`, placed after the transactions section so the existing posting form is still the first form on the page, and its name input carries no `autofocus`. | 9 | covered | "positioned after the Transactions section so that the first `<form` in the body is still the posting form, and the name input carries no `autofocus` attribute" | diff-local |
| criterion | Story 6 happy: Given the seeded ledger, when the open-account form is submitted with name "Holiday Fund", then the response is `303 See Other` with `Location: /?account=acct-4`, and following it shows "Holiday Fund" in the account navigation, "Selected account: Holiday Fund", balance "$0.00", and "No transactions.". | 9 | covered | "returns `303` with `Location: /?account=acct-4`" | diff-local |
| criterion | Story 6 happy: Given an account was opened from the page, when a transaction is then posted to it from the posting form, then the page shows the new account selected with the posted amount as its balance. | 9 | covered | "which shows "Selected account: Holiday Fund" and `class="balance">$25.00`" | diff-local |
| criterion | Story 6 happy: Given an account was just opened from the page, when the resulting page is reloaded, then the ledger still holds exactly one new account. | 9 | covered | "Requesting `GET /?account=acct-4` a second time leaves `GET /api/accounts` at exactly four accounts." | diff-local |
| criterion | Story 6 negative: Given the seeded ledger, when the open-account form is submitted with a blank name, then the response is `303` to `/?error=account_name_empty`, the page shows "Account name must not be empty." above the open-account form and not above the posting form, and three accounts remain. | 10 | covered | "A form-encoded blank name returns `303` to `/?error=account_name_empty`" | diff-local |
| criterion | Story 6 negative: Given the seeded ledger, when the open-account form is submitted with a 41-character name, then the redirect is `/?error=account_name_too_long&detail=41` and the page shows "Account name is too long. Submitted: 41 characters; the limit is 40." above the open-account form. | 10 | covered | "A form-encoded 41-character name returns `303` to `/?error=account_name_too_long&detail=41`" | diff-local |
| criterion | Story 6 negative: Given the page is requested with `/?error=account_name_too_long&detail=abc` or `detail=12`, when it renders, then the message is exactly "Account name is too long." with no detail sentence. | 10 | covered | "each render the message exactly "Account name is too long." with no "Submitted:" sentence" | diff-local |
| criterion | Story 6 negative: Given `GET /?account=acct-nope` or a ledger with no accounts, when the page renders, then it contains no `<form` at all, exactly as before this feature. | 10 | covered | "`GET /?account=acct-nope` and the zero-account page each contain no `<form` substring." | diff-local |
| criterion | Story 6 negative: Given the open-account form is submitted with name `<script>alert(1)</script>`, when the resulting page renders, then the account is opened and its name appears escaped as text, never as markup. | 10 | covered | "contains `&lt;script&gt;alert(1)&lt;/script&gt;` and no raw `<script>` tag" | diff-local |
| criterion | Story 7 happy: Given the seeded ledger plus nine opened accounts (`acct-4` … `acct-12`), when `GET /api/accounts` is requested, then the ids are exactly `acct-1`, `acct-2`, … `acct-12` in numeric order. | 12 | covered | "returns ids exactly `acct-1`, `acct-2`, … `acct-12` in numeric order" | diff-local |
| criterion | Story 7 happy: Given the same twelve accounts, when `GET /` is requested, then the account navigation links appear in the same numeric order, with `acct-10` after `acct-9`. | 12 | covered | "with the `acct-10` link after the `acct-9` link" | diff-local |
| criterion | Story 7 negative: Given only the three seeded accounts, when `GET /api/accounts` and `GET /` are requested, then the order is `acct-1`, `acct-2`, `acct-3`, identical to today. | 12 | covered | "With only the three seeded accounts, `GET /api/accounts` and the `GET /` navigation both list `acct-1`, `acct-2`, `acct-3` in that order." | diff-local |
| criterion | Story 7 negative: Given twelve accounts, when `GET /` is requested with no `account` parameter, then the selected account is still `acct-1`, not whichever id sorts first as text. | 12 | covered | "with no `account` parameter the selected account is acct-1" | diff-local |
