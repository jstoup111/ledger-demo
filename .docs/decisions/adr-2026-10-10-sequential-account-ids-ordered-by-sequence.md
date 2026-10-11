# ADR: Sequential account ids, listed in sequence order

**Date:** 2026-10-10
**Status:** APPROVED
**Deciders:** james.stoup (pre-authorized for the open-a-new-account spec)

## Context

Until now every account came from seed data with a hand-written id: `acct-1`, `acct-2`, `acct-3`.
Opening accounts at runtime (`.docs/specs/2026-10-10-open-a-new-account.md`, FR-1, FR-2, FR-9) needs
an id rule, and that rule interacts with how accounts are listed.

Two facts constrain the choice:

- **Determinism (NFR, CLAUDE.md convention).** No randomness and no time in identifiers; the same
  sequence of openings after `make reset` must produce the same ids.
- **Listing order is currently id text order.** `GET /api/accounts` and the page navigation both
  sort accounts with `sort.Slice` comparing `ID` strings, and `store.Accounts()` orders by
  `id ASC` (verified: `internal/httpapi/router.go` `handleAccounts` and `handlePage`;
  `internal/store/sqlite.go` `Accounts`). As text, `acct-10` sorts between `acct-1` and `acct-2`.

The transaction-id ADR (`adr-2026-08-08-deterministic-transaction-ids-and-ordering.md`) solved the
same problem with zero padding, but accepted a bound of 9,999 rows. Accounts cannot reuse the
padding trick: the seeded ids are already unpadded, and `acct-0004` sorts *before* `acct-1` as text.

## Options Considered

### Option A: Continue the unpadded sequence; list accounts by sequence, not text
New ids are `acct-` followed by the decimal count of all accounts plus one. Listing compares ids by
length first, then by text — for one shared prefix that is exactly numeric order.
- **Pros:** Seeded ids, seed data, and every existing test fixture stay byte-identical. One small,
  legible comparator fixes ordering for any number of accounts. No schema change.
- **Cons:** Ordering now depends on a comparator, not plain string order; every place that lists
  accounts must use it.

### Option B: Zero-padded ids for new accounts (`acct-0004`)
- **Pros:** Plain string order would work among new accounts.
- **Cons:** Breaks order against the seeded `acct-1`..`acct-3` (`acct-0004` < `acct-1`). Fixing that
  means re-keying seed accounts, which changes seed data and every fixture that names them.

### Option C: A sequence column, ordered by it
- **Pros:** Order is explicit and typed.
- **Cons:** A schema change and an ERD change to buy what a comparator gives; the ERD is deliberately
  minimal.

## Decision

1. **New account ids continue the seeded sequence.** An opened account's id is `acct-` followed by
   the unpadded decimal of (number of existing accounts + 1). Like transaction ids, deriving the next
   id from a count is sound only because nothing deletes an account; if deletion is ever added, this
   must move to max-derived numbering.
2. **Accounts are listed in sequence order, defined once in the domain.** `internal/ledger` owns one
   ordering rule — shorter id first, then id text — and every account listing (the page navigation
   and `GET /api/accounts`) sorts with it instead of comparing id strings directly. The store's SQL
   order is left as it is; callers already re-sort.
3. **Opening validation is a domain rule with sentinel errors.** Blank names (empty or whitespace
   only) return `ErrAccountNameEmpty`; names over 40 characters (runes, as descriptions are counted)
   return `ErrAccountNameTooLong`. Both wrap per
   `adr-2026-08-08-sentinel-errors-for-domain-failures.md` and map at the HTTP boundary to
   `account_name_empty` and `account_name_too_long` (`400`). Nothing is recorded on rejection.
4. **`POST /api/accounts` follows the existing negotiation rule.** Per
   `adr-2026-08-08-one-negotiated-posting-endpoint.md`, a JSON body gets `201` with the created
   account (`id`, `name`, `balance_cents: 0`, the account-list shape); any other body is the page
   form and gets `303 See Other` to `/?account=«new id»`. A JSON body that does not parse, or whose `name` is
   missing or not a string, submitted no usable name and is rejected as `account_name_empty` — the
   frozen code set gains exactly two codes, not a third for malformed bodies. A form rejection redirects to
   `/?error=«code»` with no `account` parameter; for `account_name_too_long` it adds
   `detail=«character count»`, validated on read like `description_too_long`'s detail (1–6 ASCII
   digits, greater than 40) per `adr-2026-08-09-rejection-message-composition.md`. The open-account
   form is rendered only on the selected-account page, after the transactions section, so the posting
   form stays the page's first form and the unknown-account and zero-account pages keep rendering no
   form at all. The page renders `account_name_*` messages directly above the open-account form and
   every other code where it renders today; its name input carries no `autofocus`.

## Consequences

### Positive
- No schema, seed, or fixture change; the ERD is untouched.
- The tenth account lists after the ninth on both surfaces, so the projector never shows a jumbled
  navigation bar.
- The open route behaves exactly like the posting route for both callers.

### Negative
- Two listing call sites must remember the comparator; a third listing added later could forget it.
- Opening an account is read-then-write (count, then insert) with no lock. Two concurrent opens could
  compute the same id; the primary key rejects the second as a storage error (`500`). Irrelevant at
  demo scale, documented as a known property like the posting balance check.

### Follow-up Actions
- [ ] `ledger.OpenAccount` validates, derives the id from the account count, and records via
      `Store.InsertAccount` (added to the interface; already implemented by the SQLite store).
- [ ] One domain ordering helper used by both account listings.
- [ ] Map the two sentinels to their codes and messages; wire the negotiated route and page form.
