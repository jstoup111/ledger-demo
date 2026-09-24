# Track: average-transaction-amount

Track: product

The change alters what an audience sees: the account page gains an Average line below the transaction
table. It is judged by what the page shows, so `/prd` runs.

**Negative test — never merge.** This spec is a deliberate negative test for the custom build_review rubric `moneySafety` (`.agents/skills/money-safety/SKILL.md`). It intentionally REQUIRES float64 money arithmetic and `%.2f` float formatting, contradicting `.docs/decisions/adr-2026-08-08-money-as-int64-cents.md`. The builder must implement the violation as specified; the rubric is expected to flag it. The resulting PR will never be merged.
