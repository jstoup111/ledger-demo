# Track: rejection-message-detail

Track: product

The only thing this feature changes is a sentence a human reads. A presenter deliberately triggers a
validation rejection on stage and the audience must be able to tell, from the back of the room, which
value was rejected — not merely which rule fired. That is user-facing copy with observable acceptance
signals, so `/prd` runs and enumerates FRs.

The machine-readable side of the same response is explicitly **not** changing: the seven `code` values
are a published contract in `.docs/decisions/api-response-contract.md`. A feature that only enriches
human-readable text while holding a machine contract fixed is a product change with a hard technical
boundary, not a technical refactor.

## Discovery notes (ephemeral — no design doc authored here)

**The framing this feature was routed on** is the problem plus the desired outcome, not a solution
sketch: *rejection messages are fixed strings per rule, so a rejection tells the audience which rule
fired but never which value tripped it.* No hypothesis was attached to the idea, so nothing here is
carried as a candidate implementation.

**What the code does today** (read directly, 2026-08-09):

| Fact | Where | Consequence |
|---|---|---|
| Sentinel → `(status, code, message)` mapping happens once | `internal/httpapi/errors.go#codeFor` | The right seam already exists; nothing needs scattering |
| The same seven message strings are duplicated a second time | `internal/httpapi/router.go#pageErrorMessage` | Page and JSON messages can already drift. Two copies is the defect this feature must collapse to one, since the enriched text is longer and harder to keep in sync by hand |
| A page rejection arrives as `303` → `/?account={id}&error={code}` | `internal/httpapi/router.go#writePostError` | **The offending value does not survive the redirect.** This is the whole difficulty of the feature |
| The `error` value is therefore client-supplied | `adr-2026-08-08-one-negotiated-posting-endpoint.md` (Consequences) | Anything else that travels the same way is equally client-supplied and equally untrusted |
| The page already echoes one untrusted value under escaping | `web/index.html.tmpl` (`RequestedAccount`), asserted by `router_test.go` "script-like unknown account is rendered escaped" | Escaping — not filtering — is this project's established answer, with a precedent test |
| Balance is derived on the page render path | `router.go#handlePage` → `ledger.Balance` | A balance named in a message can be server-derived on both branches; it never has to travel in a URL |

**Approaches considered.**

1. **Carry the offending datum in the rejection redirect, and compose every message in one place from
   `(code, datum)`.** Keeps the accepted `303 …&error={code}` contract, keeps the one-handler
   structure that makes base-ledger FR-9 structural, and collapses the duplicated message tables.
   Cost: one more client-supplied value on the page, so it needs validation and escaping.
2. **Stop redirecting on rejection; re-render the page inline with the submitted values in hand.**
   Zero client-supplied message inputs — the server holds the truth. Rejected: the accepted
   `?error={code}` rendering path must be retained regardless (an existing test pins the generic
   fallback for an unrecognized code), so this would leave the project with *two* rejection render
   paths instead of one, which is worse than the problem it solves. It also contradicts an Accepted
   ADR's decision without a product reason to reopen it.
3. **Enrich only the page, leave the JSON `message` generic.** Rejected: the two messages come from
   one field of one contract and the demo shows `curl` alongside the page. Deliberately introducing a
   page/JSON divergence re-creates, by design, exactly the drift that the duplicated message table
   already threatens by accident.

**Approach 1 is carried into the PRD.** The mechanism is recorded in
`.docs/decisions/adr-2026-08-09-rejection-message-composition.md`, not in the PRD.

## Assumptions surfaced (assumed, operator unavailable)

No operator approval was obtainable for this pass. Each assumption below is recorded with its
confidence and its impact if wrong, and is restated at the point of use in the PRD or the ADR.

| # | Assumption | Confidence | Impact if wrong | How to confirm |
|---|---|---|---|---|
| A1 | Enriching the message is worth one additional untrusted value on the page, given escaping is already this project's answer for the two it carries | 85% — inferred from the styleguide's "a validation failure is a thing the presenter will deliberately trigger on stage" | Feature is not worth building; revert to fixed strings | Operator confirms the stage moment matters more than the smaller surface |
| A2 | Naming the value must not change any `code`, and the enriched text may be additive to the existing sentence | 97% — verified: the task framing states the codes are a stable contract, and `api-response-contract.md` publishes them | If codes could change, a simpler redesign is available | Already settled by the contract document |
| A3 | `description_empty` has no offending value worth naming, so its message is unchanged | 80% — inferred; the offending condition is absence, and echoing "0 characters" adds a rule for no stage value | A presenter triggering the empty case gets a message no better than today | Operator decides whether emptiness deserves a count |
| A4 | Naming the account's current balance alongside the attempted amount is wanted, at the cost of deriving the balance on the JSON rejection path | 90% — verified from the routed idea's own wording, "the attempted amount against the current balance" | One extra derived read on a rejection path, for no gain; drop the balance clause | Operator confirms the balance clause earns its cost |
