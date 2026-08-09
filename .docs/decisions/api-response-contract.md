# API Response Contract — ledger-demo

**Status:** Accepted
**Date:** 2026-08-08

The contract stories assert against. Deliberately tiny: three JSON endpoints, one error shape, six
error codes. No envelope, no pagination, no versioning — this API is read by `curl` on a projector.

> **Amended 2026-08-09 by `spec/document-the-balance-overflow-error-code-in-the-ac`:** the shipped API
> emits **seven** error codes, not six. `balance_overflow` was added during implementation — after this
> contract was accepted on 2026-08-08 — and is documented in the error table below. The count above is
> left as originally written; "six error codes" is accurate as of the accepted 2026-08-08 contract and
> stale as of the shipped API.

## Conventions

| Rule | Value |
|---|---|
| Content type | `application/json; charset=utf-8` on every JSON response |
| Money | Integer cents, field name `amount_cents`. **Never** a float, never a formatted string. |
| Timestamps | RFC 3339 in UTC, e.g. `2026-08-08T14:30:00Z` |
| Field naming | `snake_case` |
| Success envelope | None. The resource is the whole body. |
| Unknown fields in a request body | Ignored |

## Success responses

### `GET /api/accounts` → `200`

Balances are derived, never stored. Order is by account id ascending, so output is stable.

```json
[
  { "id": "acct-1", "name": "Checking", "balance_cents": 128350 },
  { "id": "acct-2", "name": "Savings",  "balance_cents": 2500000 }
]
```

### `GET /api/accounts/{id}/transactions` → `200`

Newest first. The order is a stable total order — see `ORDER BY created_at DESC, id DESC` in the
plan; transactions sharing a timestamp still order deterministically.

```json
[
  {
    "id": "txn-0009",
    "account_id": "acct-1",
    "amount_cents": -4250,
    "description": "Coffee beans",
    "created_at": "2026-08-08T14:30:00Z"
  }
]
```

An account with no transactions returns `[]`, not `null` and not `404`.

### `POST /api/accounts/{id}/transactions` → `201`

Request:

```json
{ "amount": "-42.50", "description": "Coffee beans" }
```

`amount` is a **string** of dollars-and-cents as a person writes it, parsed to integer cents without
floating point. A leading `-` makes it a debit. Response is the created transaction, identical in
shape to a list element above.

### `POST` from the browser form → `303 See Other`

When the request body is `application/x-www-form-urlencoded`, the same endpoint responds
`303 See Other` with `Location: /?account={id}`, so a reload does not re-record the transaction.
On rejection it redirects to `/?account={id}&error={code}` and the page renders the message.

**Negotiation rule, stated exhaustively so there is no undefined case:** the JSON branch is taken
when the request's `Content-Type` is `application/json`. **Every other value, including a missing
`Content-Type`, takes the form branch.** Form-encoded is the default because a browser form post is
the case that cannot send anything else, and defaulting the other way would make the page's own
submission depend on a header it does not control. Negotiation reads the request's `Content-Type`,
never `Accept` — a browser form post's `Accept` header lists HTML alongside other types and is not a
reliable signal.

**Unrecognized `error` code on the page.** The `error` value arrives in the URL and is therefore
client-supplied. A value that is not one of the six codes below renders a **generic** rejection
message. It never renders an empty panel, and it is escaped on output like any other untrusted
value.

> **Amended 2026-08-09 by `spec/document-the-balance-overflow-error-code-in-the-ac`:** read "the six
> codes below" as **the seven codes below** — `balance_overflow` renders its own message, not the
> generic one. The rule itself is unchanged: any code outside the documented set renders the generic
> message, escaped, never an empty panel.

## Error responses

One shape, always:

```json
{ "error": { "code": "amount_zero", "message": "Amount must not be zero." } }
```

`code` is stable and machine-readable; `message` is human-readable and is what the page displays.

| `code` | HTTP | Rule | FR |
|---|---|---|---|
| `account_not_found` | `404` | The account does not exist | FR-12a |
| `amount_zero` | `400` | Amount is zero | FR-12b |
| `description_empty` | `400` | Description is empty | FR-12c |
| `description_too_long` | `400` | Description exceeds 140 characters | FR-12d |
| `amount_malformed` | `400` | Amount is not a well-formed money value | FR-12e |
| `balance_would_go_negative` | `400` | The transaction would take the balance below zero; nothing is recorded | FR-12f |
| `balance_overflow` | `400` | Folding the account's signed `int64` cents would overflow `int64`; nothing is recorded | — (added during implementation) |

> **Amended 2026-08-09 by `spec/document-the-balance-overflow-error-code-in-the-ac`:** the
> `balance_overflow` row is **added**, not changed — no existing code string, HTTP status, or rule was
> touched. It has no FR because it was not foreseen when this contract was accepted on 2026-08-08. It
> was introduced during implementation by the `checkedAdd` guard in `internal/ledger/balance.go`, which
> refuses a fold that would overflow `int64`, and mapped at the boundary in `internal/httpapi/errors.go`
> (commit `85df875`). The guard is load-bearing and stays: summing signed `int64` cents genuinely can
> overflow, and an unguarded fold would wrap silently and report a wrong balance. Documenting it here
> makes the shipped set of seven codes the contract, closing a gap in the document — not in the code.

Each code corresponds 1:1 to a domain sentinel error, mapped once at the HTTP boundary
(`adr-2026-08-08-sentinel-errors-for-domain-failures.md`). Tests assert both `errors.Is` on the
sentinel and the `code` string, so the mapping cannot drift silently.

`balance_would_go_negative` is named for what it is — a rejection — and not for a banking feature.
Overdraft allowance and fees are non-goals.

## Non-JSON responses

| Situation | Response |
|---|---|
| `GET /` | `200 text/html` |
| `GET /style.css` | `200 text/css` |
| Wrong method on a known path | `405`, empty body |
| Unknown path | `404`, empty body |
| Template or database failure | `500`, `text/plain`, logged via stdlib `log` |
