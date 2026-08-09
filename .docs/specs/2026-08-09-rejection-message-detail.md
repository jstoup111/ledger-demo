# PRD — Rejection Messages Name the Offending Value

**Status:** Approved
**Date:** 2026-08-09
**Track:** product (`.docs/track/rejection-message-detail.md`)
**Tier:** S (`.docs/complexity/rejection-message-detail.md`)
**Stem:** `rejection-message-detail`
**Constrained by:** `.docs/decisions/api-response-contract.md` (Accepted),
`.docs/decisions/styleguide.md`, `.docs/decisions/adr-2026-08-09-rejection-message-composition.md`
(Accepted), and base-ledger FR-12, FR-13, FR-14 in `.docs/specs/2026-08-08-base-ledger.md`

> **FR numbering:** FR-1 … FR-9 below are this feature's requirements. base-ledger's FRs are always
> written with their document named (for example "base-ledger FR-13") so the two sets never collide.

## Problem / Background

`ledger-demo` is a stage prop. Its whole reason to exist is that a presenter stands in front of an
audience and a feature gets added to it live. Deliberately tripping a validation rule is one of the
**scripted** moments: base-ledger FR-13 requires the rejection to be visible on the page, and the
styleguide says outright that "a validation failure is a thing the presenter will deliberately trigger
on stage. It has to be obvious from the back of the room."

Today it is visible but not informative. Each rule renders one fixed sentence — `Amount must not be
zero.`, `Balance would go negative.`, `Description is too long.` The audience learns *which rule*
fired and nothing about *what they just watched the presenter type*. When the presenter types `12.3.4`
into the amount field, the room reads "Amount is malformed." and has to take the presenter's word for
which part was wrong. When the presenter posts a debit larger than the balance, the room reads
"Balance would go negative." and has to hold two numbers in their head that the page never puts in one
sentence.

The gap is small and entirely in the copy. The rejection already reaches the page, already lands in
the right panel, and already carries a stable machine identifier. What is missing is the value.

## Goals

- A person at the back of a room can read one sentence and know both which rule rejected the
  submission **and** which value tripped it.
- A `curl` user reads the same sentence the projector shows.
- Nothing a client can put in a URL can turn that sentence into markup, script, or a broken panel.
- The machine-readable side of the contract does not move at all, so anything already written against
  it keeps working.

## Non-Goals

Excluded **by design**, not deferred. The features added live on stage are drawn from this list; if
any already exists, the demo is ruined.

- Duplicate or double-charge detection of any kind — no idempotency key, no dedup window, no
  uniqueness constraint beyond the primary key.
- Overdraft allowance, fees, or percentage calculations. In particular, a rejection message states the
  attempted amount and the balance; it does **not** compute how far short the account is, because a
  shortfall figure is the first step of an overdraft feature.
- Pending transactions or holds; available-versus-posted balance. A message names *the* balance, of
  which there is exactly one.
- Statements, exports, or reporting.
- Interest or rounding rules.
- Authentication, users, sessions, or multi-tenancy.
- Transfers between accounts, or any balancing counter-entry.
- Containerization, continuous integration, or deployment tooling.
- Metrics, tracing, or structured logging beyond what the standard library provides.

Additionally out of scope for this feature specifically:

- **No change to any machine-readable identifier or status code.** See FR-3.
- **No new field in the error body.** The rejection body stays exactly `{"error":{"code","message"}}`.
  The offending value appears *inside* `message` and nowhere else, so no consumer has to learn a new
  field.
- **No new route.** The HTTP surface stays at exactly five routes.
- **No JavaScript, no new colour, no dark mode, no media query, no animation.** The styleguide's hard
  rules are unchanged and unrelaxed by this feature.
- **No change to which rules exist.** base-ledger FR-12's cases are the cases; this feature adds no
  rule, removes none, and re-classifies none.
- **No echoing of the submitted description.** A description is up to 140 characters of free text and
  putting it on a projector serves nobody; its *length* is the offending value. See FR-1.

## Functional Requirements

### Naming the offending value

- **FR-1** When a submission is rejected and there is an offending value to name, the human-readable
  message names it. The message is the following text, exactly:

  | Rule (base-ledger FR-12) | Message today | Message this feature requires |
  |---|---|---|
  | The account does not exist (FR-12a) | `Account not found.` | `Account not found. Requested: acct-9.` |
  | The amount is zero (FR-12b) | `Amount must not be zero.` | `Amount must not be zero. Submitted: 0.00.` |
  | The description is empty (FR-12c) | `Description must not be empty.` | **unchanged** — see FR-2 |
  | The description exceeds 140 characters (FR-12d) | `Description is too long.` | `Description is too long. Submitted: 187 characters; the limit is 140.` |
  | The amount is not well-formed (FR-12e) | `Amount is malformed.` | `Amount is malformed. Submitted: 12.3.4.` |
  | The transaction would take the balance below zero (FR-12f) | `Balance would go negative.` | `Balance would go negative. Posting -$500.00 against a balance of $128.50.` |
  | The transaction would overflow the balance | `Balance would overflow.` | `Balance would overflow. Posting $500.00 against a balance of $92,233,720,368,547,757.00.` |

  The italicised values are illustrations of shape, not fixed text: `acct-9` is whatever account was
  requested, `0.00` and `12.3.4` are **verbatim what was submitted in the amount field**, `187` is the
  submitted description's actual character count, and the money figures are formatted exactly the way
  the page already formats a balance — a leading sign if negative, a `$`, thousands separated by
  commas, always two decimal places.

- **FR-2** The enriched message is **additive**: it is the message that exists today, followed by one
  further sentence naming the value. Consequences, all of them intentional:

  - The rule is still named first, so the sentence still reads correctly if the room only catches its
    beginning.
  - The existing sentence remains a complete, correct message on its own, which is what FR-5 falls
    back to.
  - `Description must not be empty.` is unchanged, because emptiness has no value to name. This is a
    deliberate exclusion, not an oversight — see the assumption ledger.

- **FR-3** No machine-readable identifier changes. These seven identifiers and their HTTP statuses are
  a stable contract and this feature must leave every one of them byte-for-byte as it is:

  | Identifier | HTTP |
  |---|---|
  | `account_not_found` | `404` |
  | `amount_zero` | `400` |
  | `description_empty` | `400` |
  | `description_too_long` | `400` |
  | `amount_malformed` | `400` |
  | `balance_would_go_negative` | `400` |
  | `balance_overflow` | `400` |

  A change to any identifier, status, or to the error body's shape is a failure of this requirement,
  regardless of how the message reads.

- **FR-4** For one and the same rejection, the message the page displays and the message a
  programmatic client receives are **identical, character for character**. A presenter demonstrating
  the same rejection twice — once in the browser, once with `curl` — reads the same sentence both
  times.

  > **Amended 2026-08-09 by operator decision — product-scope boundary for FR-4.** The assertion
  > above is retained verbatim; this note settles a question it left open rather than changing it.
  >
  > **"One and the same rejection" means a rejected submission** — the six validation rules applied to
  > `POST /api/accounts/{id}/transactions`. It does **not** extend to read errors.
  >
  > Concretely: `internal/httpapi/router.go:276-284` serves an unknown account on
  > `GET /api/accounts/{id}/transactions` through plain `writeJSONError`, unenriched, and that is
  > **correct as written**. A listing 404 is not a rejected submission; nothing was submitted, so
  > there is no offending value to name and no scripted stage moment to support. The page has no
  > rejection-panel equivalent of that error either — an unknown account renders the base-ledger
  > not-found page, whose behavior belongs to that feature's spec and is out of scope here.
  >
  > **Why this boundary and not the broader one.** `prd_audit` raised the ambiguity at ~60% confidence
  > and recorded that the PRD "does not settle" it — the spec never intended to decide it, which is
  > itself evidence the narrow reading was the intent. The broader reading would additionally require
  > the base-ledger not-found page's sentence to match this API error character for character, which
  > amends another feature's shipped behavior from inside this one.
  >
  > **Provenance.** The operator was shown both readings with their consequences and chose "rejected
  > submissions only". Cross-surface message consistency for *read* errors is deliberately left as
  > possible separate work with its own spec, not silently dropped.

### Degrading safely

- **FR-5** When the offending value is unavailable or not plausible for the rule that was reported,
  the message is the existing sentence for that rule, complete and on its own. Never an empty panel,
  never a truncated sentence, never a dangling `Submitted:` with nothing after it. "Not plausible"
  covers, at minimum: absent; longer than a value a person types into the form; containing control
  characters; and, where the value is a number, not a number or not in a range the rule could have
  produced.

- **FR-6** An unrecognized identifier still renders a **generic**, non-empty message —
  `Unable to post transaction.` — and the unrecognized identifier itself is not echoed anywhere in the
  page. This behaviour exists today, is pinned by an existing test, and this feature must not weaken
  it.

- **FR-7** Any submitted value that reaches a rendered output is rendered **inert**. On the page it is
  escaped, so a value containing markup or a script tag appears as visible text and creates no
  element, no attribute, and no executable content. In the programmatic response it is encoded such
  that the body remains well-formed and the value cannot break out of the string it sits in. A
  script-bearing amount is an explicitly required negative case on both surfaces.

### Not disturbing what already works

- **FR-8** Placement is unchanged. A rejection originating from the page is still rendered in the
  panel that sits **directly above the form that produced it** — after the balance, immediately before
  the form — satisfying base-ledger FR-13 exactly as it is satisfied today. The panel is longer; it is
  in the same place.

- **FR-9** Every behaviour of a rejection other than its message text is unchanged: nothing is
  recorded, the response status is the same, the page still responds `200` for a rejection carried in
  its address, a form submission still results in a redirect a reload cannot re-post, and both
  submission paths remain subject to identical validation (base-ledger FR-9).

## Non-Functional Requirements

- **NFR-1 — Projector legibility.** The enriched message must remain readable at 1280×720 from across
  a room, in the existing error panel, at the existing 20px base size and existing six-colour palette.
  No new colour, no dark mode, no responsive breakpoint, no animation, no JavaScript. A value that
  would be too long to read is not shown at all (FR-5) rather than shown and wrapped indefinitely.
- **NFR-2 — Money never becomes a float.** Every amount and balance in every message derives from
  integer cents. No floating-point type and no float parsing appears anywhere on the path.
- **NFR-3 — Determinism.** The same rejection produces the same sentence every time. No timestamps, no
  randomness, and no run-to-run variation in message text.
- **NFR-4 — Fully offline.** No new dependency and no network call.
- **NFR-5 — Test suite.** Full suite stays under 10 seconds with no ordering dependency and no
  sleeping. Roughly a 4:1 test-to-implementation line ratio, table-driven, with a negative case for
  every degradation rule in FR-5 and for both surfaces of FR-7.
- **NFR-6 — Lint clean.** Formatting and vetting gates pass with no findings.
- **NFR-7 — Legibility of the code itself.** The diff is read on a projector by an audience following
  along, so it stays small and direct. The message text for a given rejection must exist in exactly
  one place; two copies of a sentence that now has a variable part in it is a defect waiting to
  happen, and one already exists today.
- **NFR-8 — HTTP surface unchanged.** Exactly five routes, before and after.

## Acceptance Criteria

- Trip each of base-ledger FR-12's cases from the page and read the panel: every case except the empty
  description names the value that was rejected, in the sentence shape FR-1 fixes.
- Trip the same cases with `curl` and diff the `message` strings against what the page rendered: they
  match exactly.
- Every identifier and status in FR-3's table is unchanged, asserted against the identifier string and
  not only against the sentinel.
- Submit an amount of `<script>alert(1)</script>` from the page: the panel shows that text as visible
  characters, the response body contains no runnable script element, and the same submission's
  programmatic response is a well-formed body whose message contains the value inertly.
- Request the page with an identifier that is not one of the seven: the panel is non-empty, generic,
  and does not contain the identifier.
- Request the page with a rejection whose value has been tampered with — absent, over-long, control
  characters, a non-numeric count — and the panel shows the existing sentence for that rule, whole.
- The rejection panel still sits between the balance and the form.
- The suite passes under 10 seconds; the formatting and vetting gates report nothing.

## Scope

**In scope:** the human-readable text of a rejection, on both the page and the programmatic response;
the means by which the offending value reaches the page; validation and escaping of that value;
collapsing the message text to a single source so the two surfaces cannot diverge.

**Out of scope:** everything in Non-Goals above, and specifically: the set of rules, the identifiers,
the statuses, the error body shape, the route surface, the panel's position and styling, and the
behaviour of a successful submission.

## Open Questions

Resolved as recorded assumptions below rather than left open, because the operator was unavailable for
this pass. None of them changes an FR's observable acceptance signal; each changes at most one row of
FR-1's table.

## Assumption Ledger

**Assumed, operator unavailable.** No operator approval was obtained for this pass and none is claimed.
Each entry states its confidence, its impact if wrong, and how to confirm it.

| # | Assumption | Confidence | Impact if wrong | How to confirm |
|---|---|---|---|---|
| A1 | The stage moment is worth carrying one more untrusted value to the page, given the page already carries two under the same escaping rule and an existing precedent test | 85%, inferred from the styleguide's stated purpose for the error panel | The feature should not be built; fixed strings stay | Operator confirms the demo value outweighs the smaller surface |
| A2 | The enriched text may be additive to the existing sentence rather than a rewrite | 97%, verified — it keeps FR-5's fallback trivially correct and leaves the existing message assertions in the suite valid | A rewrite would churn more of the existing suite for no reader benefit | Settled; no confirmation needed |
| A3 | `Description must not be empty.` needs no value named, because the offending condition is absence | 80%, inferred | One of seven cases stays as informative as today | Operator decides whether a `0 characters` clause earns its rule |
| A4 | Naming the account's current balance alongside the attempted amount is wanted, at the cost of deriving the balance on the programmatic rejection path too | 90%, verified from the routed idea's wording, "the attempted amount against the current balance" | Drop the balance clause; the amount alone is named and the balance stays only in the big number above the panel | Operator confirms the balance clause earns its cost |
| A5 | The exact sentences in FR-1 are acceptable copy | 75%, unverified — this is editorial | Copy is edited; every structural requirement, test, and identifier stands unchanged | Operator reads FR-1's table |
| A6 | A value longer than a person types into the form is better suppressed than truncated with an ellipsis | 85%, inferred from NFR-1 and NFR-7 — suppression is one rule, truncation is two plus an ellipsis convention | An over-long garbage amount shows the generic sentence rather than a clipped echo. Only reachable by a hand-crafted address, since the form's own values are bounded | Operator decides whether a clipped echo reads better on stage than the generic sentence |

Consequence recorded for BUILD: A2 keeps almost every existing message assertion valid, because each
enriched message *contains* the sentence the suite already asserts. The exception is base-ledger's
unknown-account page assertion, which checks that the not-found sentence is present — it stays
satisfied for the same reason. No existing test needs its expected identifier changed, because FR-3
changes none.
