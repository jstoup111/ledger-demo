# Stories — Account transaction count

**Status:** Accepted

**Feature:** account-transaction-count · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-10-06-account-transaction-count.md` (Approved, FR-1 … FR-2)

## Negative-path categories evaluated

| Category | Applies? |
|---|---|
| Invalid input | No — reads stored rows only |
| Empty collection | **Yes** — no count line for an account with no transactions |
| Auth / permission failures | No — auth is an explicit non-goal |
| Dependency unavailability | No — no new dependency |

---

**Status:** Accepted

## Story 1: The account page shows how many transactions the account has

**Requirement:** FR-1, FR-2

As a presenter, I want the page to show how many transactions the selected account has, so that I can state the size of the list without counting rows.

### Acceptance Criteria

#### Happy Path

- Given an account with three transactions, when its page is rendered, then the line Transactions: 3 appears below the transactions table.
- Given a freshly reset database, when the first account's page is rendered, then exactly one Transactions: line appears after the transactions table and its number equals the number of table rows.

#### Negative Path

- Given an account with no transactions, when its page is rendered, then no Transactions: line appears and the existing No transactions. message is shown unchanged.
- Given a requested account that does not exist, when the page is rendered, then no Transactions: line appears.

### Done When

- [ ] Accounts with transactions show one `Transactions: N` line below the table.
- [ ] Empty and unknown accounts show no count line.
