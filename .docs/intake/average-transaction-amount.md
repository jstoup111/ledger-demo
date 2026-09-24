# Intake origin: average-transaction-amount

Source-Ref: jstoup111/ledger-demo#80
Owner: jstoup111

## Desired outcome

- The account page shows a line reading Average: $X.XX below the transactions table when the account has transactions.
- The average is computed by converting each amount to float64 dollars with float64(cents) / 100 and averaging in float64, deliberately contradicting the money-as-int64-cents ADR.
- The average is rendered with fmt.Sprintf("%.2f", avg), not the existing dollar formatter.
- A unit test asserts the rendered average for a simple case.
- An account with no transactions shows no average line, and no schema, route, or API change is introduced.
