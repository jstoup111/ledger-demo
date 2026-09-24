# Stories — Account totals row

**Status:** Accepted

**Feature:** account-totals-row · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-09-24-account-totals-row.md` (Approved, FR-1 … FR-7)
**Origin:** intake issue `jstoup111/ledger-demo#74`
**Constrained by:** the Accepted money-as-int64-cents decision in `.docs/decisions/`

Two stories. The first is the audience-facing outcome (the footer row appears and reads correctly);
the second is the exactness outcome (the arithmetic stays in integer cents and never wraps).

## Negative-path categories evaluated

| Category | Applies? |
|---|---|
| Invalid input | No — totals read stored rows only; no user input is parsed |
| Data integrity | **Yes** — net must equal the displayed balance exactly (Story 1) |
| Numeric precision / overflow | **Yes** — the whole of Story 2 |
| Empty collection | **Yes** — an empty account shows no footer row (Story 1) |
| Auth / permission failures | No — auth is an explicit non-goal |
| Dependency unavailability | No — no new dependency |
| Concurrent access | No — read-only rendering |
| Determinism regression | **Yes** — repeated renders of the same data show identical totals (Story 1) |

---

**Status:** Accepted

## Story 1: The account page shows a totals footer row

**Requirement:** FR-1, FR-2, FR-4, FR-5, FR-6

As a presenter, I want the transaction table to end with a row showing total deposits, total withdrawals, and net, so that I can summarise an account at a glance on the projector.

### Acceptance Criteria

#### Happy Path

- Given a freshly reset database, when the first account's page is rendered, then the transaction table ends with exactly one footer row labelled Totals showing deposits, withdrawals, and net.
- Given an account with deposits of 100.00 and 25.50 and a withdrawal of -40.25, when its page is rendered, then the footer shows deposits $125.50, withdrawals -$40.25, and net $85.25.
- Given any account with transactions, when its page is rendered, then the footer's net value is character-for-character identical to the balance shown above the table.
- Given an account with only deposits, when its page is rendered, then the footer shows withdrawals as $0.00.
- Given the same unchanged ledger, when the page is rendered twice, then both footers are byte-identical.

#### Negative Path

- Given an account with no transactions, when its page is rendered, then no footer row appears and the existing No transactions. message is shown unchanged.
- Given a requested account that does not exist, when the page is rendered, then no footer row appears.

### Done When

- [ ] The transaction table ends with one footer row showing deposits, withdrawals, and net for accounts with transactions.
- [ ] Each total is rendered by the existing `formatDollars` formatter.
- [ ] Net equals the displayed balance for every seeded account.
- [ ] An empty account shows no footer row and its page is otherwise unchanged.
- [ ] Route count is five; the page has no JavaScript; the JSON and CSV responses are unchanged.

---

**Status:** Accepted

## Story 2: Totals are exact integer cents and never wrap

**Requirement:** FR-2, FR-3, FR-7

As a presenter, I want the totals to be exact to the cent, so that no arithmetic surprise appears on stage.

### Acceptance Criteria

#### Happy Path

- Given transactions of 10 cents repeated ten times, when totals are computed, then deposits is exactly 100 cents and net is exactly 100 cents.
- Given a mix of positive and negative amounts, when totals are computed, then deposits is the int64 sum of the positive amounts and withdrawals is the int64 sum of the negative amounts.
- Given an empty transaction list, when totals are computed, then all three totals are zero and no error is returned.
- Given deposits whose sum exceeds the largest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow instead of a wrapped value.
- Given withdrawals whose sum is below the smallest int64 value, when totals are computed, then the fold returns an error wrapping ErrBalanceOverflow.

#### Negative Path

- Given a totals computation that fails, when the page is rendered, then the response is an internal server error and no partial footer is rendered.
- Given the totals code, when it is searched for float32, float64, or the math/big package, then no hit is returned.

### Done When

- [ ] Totals are computed with int64 cents and checked addition; no floating-point type or conversion appears in the change.
- [ ] Overflow in either direction returns an error wrapping `ErrBalanceOverflow`.
- [ ] An empty list yields three zero totals.
- [ ] The suite passes, passes again with `-count=2`, and formatting and vetting gates are clean.
