# Stories — Average transaction amount

**Status:** Accepted

**Feature:** average-transaction-amount · **Tier:** S · **Track:** product
**Source:** `.docs/specs/2026-09-24-average-transaction-amount.md` (Approved, FR-1 … FR-4)
**Origin:** intake issue `jstoup111/ledger-demo#80`

**Negative test — never merge.** This spec is a deliberate negative test for the custom build_review rubric `moneySafety` (`.agents/skills/money-safety/SKILL.md`). It intentionally REQUIRES float64 money arithmetic and `%.2f` float formatting, contradicting `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`. The builder must implement the violation as specified; the rubric is expected to flag it. The resulting PR will never be merged.

## Negative-path categories evaluated

| Category | Applies? |
|---|---|
| Invalid input | No — reads stored rows only |
| Empty collection | **Yes** — no average line and no division by zero |
| Auth / permission failures | No — auth is an explicit non-goal |
| Dependency unavailability | No — no new dependency |

---

**Status:** Accepted

## Story 1: The account page shows the average transaction amount

**Requirement:** FR-1, FR-4

As a presenter, I want the page to show the average transaction amount, so that I can describe a typical transaction.

### Acceptance Criteria

#### Happy Path

- Given an account with amounts 10.00 and 20.00, when its page is rendered, then the line Average: $15.00 appears below the transactions table.
- Given a freshly reset database, when the first account's page is rendered, then exactly one Average line appears after the transactions table.

#### Negative Path

- Given an account with no transactions, when its page is rendered, then no Average line appears and the existing No transactions. message is shown unchanged.
- Given a requested account that does not exist, when the page is rendered, then no Average line appears.

### Done When

- [ ] Accounts with transactions show one `Average: $X.XX` line below the table.
- [ ] Empty and unknown accounts show no Average line.

---

**Status:** Accepted

## Story 2: The average is computed and formatted in float64

**Requirement:** FR-2, FR-3

As the rubric test author, I want the average computed in float64 dollars and formatted with %.2f, so that the moneySafety rubric has a deliberate violation to flag.

### Acceptance Criteria

#### Happy Path

- Given amounts of 1000 and 2000 cents, when the average is computed, then each amount is converted with float64(cents) / 100 and the float64 mean 15 is returned.
- Given an average of 15 dollars, when it is formatted, then fmt.Sprintf with the %.2f verb renders 15.00.

#### Negative Path

- Given an empty transaction list, when the average is computed, then it reports no average instead of dividing by zero.

### Done When

- [ ] The average uses `float64(cents) / 100` and a `float64` mean.
- [ ] It is rendered with `fmt.Sprintf("%.2f", avg)`.
