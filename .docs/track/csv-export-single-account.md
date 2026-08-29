# Track: csv-export-single-account

Track: product

Scope boundary: One visible browser control downloads the valid selected account's raw transaction
rows as CSV. Preserve exactly five HTTP endpoints and exclude CLI delivery, JavaScript, filters,
multi-account output, statements, summaries, totals, and every unrelated project non-goal.

This is user-facing demo behavior, so it remains on the product track. The browser-only shape and
narrow scope were approved in PR #24 and reconfirmed by the operator's request to refresh that PR
without changing its behavior.

## Approach decision

- **Selected:** use the existing selected-account page and existing transaction-listing HTTP surface.
  This keeps the action visible during the live demo without adding another endpoint or invocation
  surface.
- **Rejected:** retain the original CLI export. It requires leaving the projected page and duplicates
  the browser invocation surface.
- **Rejected:** add a standalone CSV endpoint. It breaks the governing five-endpoint contract without
  improving the selected-account demo outcome.

## Verify-Claims Ledger — explore — 2026-08-28

### Claims

- [verified] PR #24's approved amendment selects a browser-only download for the selected account.
- [verified] The current router registers exactly five endpoints, and the selected-account page and
  transaction-listing surface already share the selected account's stored transaction order.
- [verified] Issue #14 explicitly authorizes this one narrow exception to the project's export
  non-goal for rehearsal, without authorizing statements, reporting, or broader export behavior.

### Assumptions

- None. The operator requested a format refresh of PR #24, so this pass changes no product behavior.

### Verdict

CLEAR — no unconfirmed load-bearing assumptions remain.

