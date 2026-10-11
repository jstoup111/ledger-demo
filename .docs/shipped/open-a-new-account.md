---
slug: open-a-new-account
spec_hash: 18968d4d2728c22bc58a0313600bb1670832ea75c6fe01ba50a9f81544eb4284
pr: https://github.com/jstoup111/ledger-demo/pull/87
shipped: 2026-10-11
engine_version: 20261011T113509Z-bd1488cfe309
findings:
  - gate: prd_audit
    grade: OVER_SCOPE
    criterion: NC-1
    summary: "Verified with 98% confidence: this committed runner is additional build-harness functionality beyond the account-opening plan. It does not alter ledger or user-facing behavior, so the widening is harmless but unowned."
    accepted: false
    authority: engine
---

## Cost
input: 1505668
output: 132525
cache_read: 27013498
cache_creation: 18046
cost_usd: 13.6577
dispatches: 20
retries: 2
halts: 4
unmetered: count: 0, duration_ms: 0
cost_unmetered: count: 0
providers:
  codex: input: 1505652, output: 131219, cache_read: 26945664, cache_creation: 0, cost_usd: 13.4736, dispatches: 18, cost_unmetered: 0
  claude: input: 16, output: 1306, cache_read: 67834, cache_creation: 18046, cost_usd: 0.1841, dispatches: 2, cost_unmetered: 0

## Time
state: partial
reason: open-executions:step:execution\u0000["timing-rollup","persisted-ledger","8f4a2f55-cc49-4e3c-bc01-153bf1ed7a34","lifecycle-step","finish"]

## Build Review
laps_to_pass: 1
skipped: 0
cache_hits: 0
infrastructure_failures: 0
rubrics:
  moneySafety: failures: 0, judged: 2
skip_reasons:
