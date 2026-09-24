# Intake origin: account-totals-row

Source-Ref: jstoup111/ledger-demo#74
Owner: jstoup111

## Desired outcome

- The account page's transaction table ends with a footer row showing total deposits, total withdrawals, and the net amount for the listed transactions.
- Totals are computed exactly in int64 cents per the money-as-int64-cents ADR; no float appears anywhere in the computation or formatting.
- Each total is displayed with the existing dollar formatter, so it reads exactly like the Amount column and the balance.
- The net amount equals the account's displayed balance for the listed transactions.
- An account with no transactions shows no totals row, and no schema, route, or API change is introduced.
