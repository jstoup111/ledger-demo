---
slug: account-transaction-count
spec_hash: 3c5c604c1d96cb247af4f77418491c44d14ebc81e71da2ebd5b96abb97f7c0c2
pr: https://github.com/jstoup111/ledger-demo/pull/84
shipped: 2026-10-06
engine_version: 20261006T003806Z-c9d1a5549868
---

## Cost
input: 253859
output: 24487
cache_read: 3668071
cache_creation: 13020
cost_usd: 2.2091
dispatches: 5
retries: 0
halts: 0
unmetered: count: 0, duration_ms: 0
cost_unmetered: count: 0
providers:
  codex: input: 253853, output: 24218, cache_read: 3649152, cache_creation: 0, cost_usd: 2.0958, dispatches: 4, cost_unmetered: 0
  claude: input: 6, output: 269, cache_read: 18919, cache_creation: 13020, cost_usd: 0.1133, dispatches: 1, cost_unmetered: 0

## Time
state: partial
reason: open-executions:step:execution\u0000["timing-rollup","persisted-ledger","524194ea-cd33-4c00-8a0a-a038b386cb3b","lifecycle-step","finish"]

## Build Review
laps_to_pass: 1
skipped: 0
cache_hits: 0
infrastructure_failures: 0
rubrics:
  moneySafety: failures: 0, judged: 1
skip_reasons:
