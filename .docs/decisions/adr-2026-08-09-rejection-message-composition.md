# ADR: One message composer, and the offending value travels as one validated query parameter

**Date:** 2026-08-09
**Status:** APPROVED
**Deciders:** authored and accepted in the DECIDE pass for `rejection-message-detail`, under the
operator's pre-granted routing and Small-tier decisions. The operator was unavailable for the
load-bearing choices below; each is recorded in the assumption ledger of
`.docs/specs/2026-08-09-rejection-message-detail.md` and labelled *assumed, operator unavailable*. No
operator approval is claimed for them.

## Context

`.docs/specs/2026-08-09-rejection-message-detail.md` FR-1 requires a rejection's human-readable message
to name the offending value, and FR-4 requires the page and the programmatic response to produce that
message **character for character identically**. FR-3 forbids any change to the seven machine-readable
identifiers or their statuses.

Two facts about the code as it stands make this more than a copy edit.

**1. The message text already exists twice.** `internal/httpapi/errors.go#codeFor` maps each domain
sentinel to `(status, code, message)`. `internal/httpapi/router.go#pageErrorMessage` holds a second,
independent table of the same seven strings, keyed by code. Today those two tables agree by hand. Once
each message has a variable part, "agree by hand" becomes "drift", and FR-4 is a promise the structure
does not keep.

**2. The offending value does not survive the page's rejection path.** By
`adr-2026-08-08-one-negotiated-posting-endpoint.md` (APPROVED), a form-encoded rejection answers
`303 See Other` with `Location: /?account={id}&error={code}` — the POST-redirect-GET that satisfies
base-ledger FR-7. That ADR names this cost in its own Consequences: *"A rejection's identity travels
through a URL query parameter. The page therefore renders a message looked up from a code supplied by
the client."* The submitted amount, and the submitted description's length, are known only inside the
POST. By the time the page renders, they are gone.

So something must carry the value across the redirect, and whatever carries it is client-supplied —
exactly like `error` and `account` already are.

## Options Considered

### Option A: One `detail` query parameter whose meaning is fixed per identifier, validated on read

The rejection redirect becomes `/?account={id}&error={code}&detail={value}`, and `detail` is added only
for identifiers that define one. A single message composer takes `(code, detail, balance context)` and
is called by both the JSON writer and the page renderer.

- **Pros:** FR-4 holds *structurally* — there is one composer, one input tuple, so the two surfaces
  cannot produce different text. Collapses the duplicated message table (NFR-7). One new query
  parameter, so the address a presenter can read off the projector stays short. Every number in a
  message stays server-derived; only the two genuinely free-text cases echo a submitted string.
- **Cons:** `detail`'s meaning varies by identifier, so reading it requires knowing which identifier it
  arrived with — a small coupling that must be written down (it is, below). One more client-supplied
  value on the page.

### Option B: A separate typed parameter per datum (`amount`, `length`, `cents`)

- **Pros:** Each parameter has one meaning independent of the identifier.
- **Cons:** Three parameters instead of one, three validators instead of one dispatch, and the
  identifier still has to agree with which parameter is present — so the coupling is not removed, only
  spread out. A longer, less readable address for no additional power.

### Option C: Stop redirecting on rejection; re-render the page inline with the submitted values in hand

- **Pros:** Nothing client-supplied is involved at all; the server holds the truth.
- **Cons:** The `?error={code}` rendering path must be retained regardless — FR-6 requires an
  unrecognized identifier to render a generic message, and an existing test pins it. So this adds a
  second rejection render path rather than replacing one, which is worse than the problem it solves. It
  also reverses an APPROVED ADR's decision with no product reason to reopen it, and that ADR rejected
  inline rendering on base-ledger FR-7 grounds.

### Option D: Sign the carried value so it cannot be tampered with

- **Pros:** A crafted address could not put a chosen value into a message.
- **Cons:** Introduces a secret and key handling into a stage prop that has no authentication by
  explicit non-goal. The harm it prevents is already bounded to "a hand-crafted address renders a
  message containing an escaped value the crafter chose" — precisely the exposure `error` and `account`
  already carry and which the project already answers with escaping plus a precedent test. Cost is
  entirely disproportionate.

## Decision

**Option A.**

### The parameter

The form-encoded rejection redirect becomes:

```
/?account={id}&error={code}&detail={value}
```

`detail` is percent-encoded, and is **added only for the identifiers that define one**. Success
redirects are unchanged, and the JSON branch gains no new field — the value appears only inside
`message` (PRD Non-Goals).

### What `detail` carries, per identifier

| Identifier | `detail` carries | Detail sentence appended to the existing message |
|---|---|---|
| `account_not_found` | *nothing* — the id is already the `account` value on the page and the path value on the JSON branch | `Requested: {id}.` |
| `amount_zero` | the submitted amount field, verbatim | `Submitted: {text}.` |
| `amount_malformed` | the submitted amount field, verbatim | `Submitted: {text}.` |
| `description_too_long` | the submitted description's character count, in decimal | `Submitted: {n} characters; the limit is 140.` |
| `balance_would_go_negative` | the attempted amount in **integer cents**, in decimal, optional leading `-` | `Posting {amount} against a balance of {balance}.` |
| `balance_overflow` | the attempted amount in integer cents, as above | `Posting {amount} against a balance of {balance}.` |
| `description_empty` | *nothing* — emptiness has no value to name (PRD FR-2) | *none* |

Two properties of that table are the point of it:

- **Only two identifiers echo free text**, and both are cases where the submitted amount *failed to
  parse* or parsed to zero — so the string itself is the offending value and there is nothing else to
  name. Everywhere else the carried datum is a number.
- **No balance is ever carried.** `{balance}` is derived server-side on both branches, and `{amount}`
  is rendered from integer cents by the same money formatter the page already uses for a balance
  (`internal/httpapi/router.go#formatDollars`), so NFR-2 holds — no float, no float parsing.

### Validating `detail` on read

`detail` is untrusted on every read, including the JSON branch's own composition, so one validator
serves both. A `detail` that fails validation is **ignored**, and the message is the identifier's
existing sentence, whole (PRD FR-5). Never a partial sentence.

| Kind | Accepted iff | Rejected example |
|---|---|---|
| Free text (`amount_zero`, `amount_malformed`) | non-empty; at most **32 characters**; contains no control character | a 500-character amount; a value containing a newline |
| Character count (`description_too_long`) | 1–6 ASCII decimal digits; parses as an integer; **greater than 140** | `abc`; `3`; a 30-digit number |
| Integer cents (`balance_would_go_negative`, `balance_overflow`) | optional leading `-`, then 1–19 ASCII decimal digits; parses as a 64-bit signed integer | `12.50`; `1e9`; a 40-digit number |

Additional rules, each of which is a required negative case:

- A `detail` present for an identifier that defines none is **ignored**, not echoed.
- An **unrecognized identifier** ignores `detail` entirely, renders the static generic message, and
  echoes neither the identifier nor the detail (PRD FR-6; already pinned by an existing test).
- Where a message needs the account's balance and it cannot be derived — the unknown-account page and
  the zero-account page have no selected account — the detail sentence is **omitted** and the existing
  sentence stands alone.
- The 32-character cap applies when **writing** the redirect too: a submitted amount longer than that
  is simply not carried, rather than carried truncated. Suppression is one rule; truncation would be a
  rule plus an ellipsis convention, and the resulting clipped echo is not more useful on a projector
  than the plain sentence (PRD assumption A6).

### Escaping, not filtering

An echoed free-text value is rendered **inert at the point of output**, never sanitised on the way in:

- On the page it goes through the template as ordinary text and is contextually escaped, exactly as
  `RequestedAccount` already is. `router_test.go`'s "script-like unknown account is rendered escaped"
  is the precedent this follows.
- In the programmatic response it is encoded as a JSON string value, so `<`, `>`, and `&` cannot close
  or open anything.

Filtering was rejected on purpose: `amount_malformed`'s entire job is to show the audience the garbage
that was typed, and a filter would silently swallow the interesting cases. PRD FR-7 requires an
explicit negative case on **both** surfaces proving a script-bearing amount creates no element and no
executable content.

### One composer

The seven message strings and the detail sentences live in exactly one function in
`internal/httpapi`. `writeJSONError` and the page renderer both call it with the same inputs.
`router.go#pageErrorMessage`'s duplicate table is deleted, not updated. The sentinel → identifier
mapping stays exactly where it is, in `codeFor`, mapped once at the HTTP boundary
(`adr-2026-08-08-sentinel-errors-for-domain-failures.md`) — this ADR adds a second, separate step
(identifier + detail → message) and does not scatter the first.

## Consequences

### Positive

- FR-4 is structural rather than maintained: one composer, one input tuple, so the page and the JSON
  response cannot report different text for the same rejection.
- The duplicated message table is removed, closing a drift hazard that exists today (NFR-7).
- The seven identifiers, their statuses, and the error body's shape are untouched, so every existing
  consumer and every existing identifier assertion in the suite keeps passing (FR-3).
- Because the enriched message *begins with* today's sentence, existing tests that assert the current
  message text remain valid without being rewritten.
- No new route, no new dependency, no JavaScript, no new colour, no schema change.
- Every number in a message is server-derived from integer cents; there is no float anywhere on the
  path.

### Negative

- `detail`'s meaning depends on the identifier it arrives with. That coupling is real; it is confined
  to the composer and written down in the table above, and it is the price of keeping the projector
  address short.
- The page now carries three client-supplied values rather than two. Each is validated, and each is
  escaped at output.
- The two balance identifiers need the account's balance on the programmatic rejection path, which is
  one derived read that the success path does not perform. It runs only on a rejection.
- A hand-crafted address can put a chosen, escaped value into a message. This is the same exposure
  `error` and `account` already carry, and Option D's remedy was rejected as disproportionate.

### Follow-up Actions

- [ ] Author the single message composer in `internal/httpapi` and delete
      `router.go#pageErrorMessage`'s duplicate table.
- [ ] Add `detail` to the rejection redirect for the five identifiers that define one, percent-encoded,
      suppressed rather than truncated when over-long.
- [ ] Validate `detail` on every read per the table above; fall back to the existing sentence whole.
- [ ] Prove inertness on both surfaces with a script-bearing amount.
- [ ] Amend `.docs/decisions/api-response-contract.md` additively to document the new query parameter
      and the enriched messages, leaving its identifier table untouched.
