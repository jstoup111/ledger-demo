---
slug: account-totals-row
spec_hash: ff2f8829e9480b86b66bbc42e3178012f068e06c86ba745b4071326fa48ddd5f
pr: https://github.com/jstoup111/ledger-demo/pull/79
shipped: 2026-09-26
engine_version: 20260926T202633Z-13eb9339e4f3
---

## Cost
input: 197274
output: 56446
cache_read: 4422738
cache_creation: 231992
cost_usd: 5.5966
dispatches: 32
retries: 0
halts: 4
unmetered: count: 21, duration_ms: 0
cost_unmetered: count: 0
providers:
  claude: input: 118, output: 29501, cache_read: 2318034, cache_creation: 231992, cost_usd: 3.4589, dispatches: 19, cost_unmetered: 0
  codex: input: 197156, output: 26945, cache_read: 2104704, cache_creation: 0, cost_usd: 2.1378, dispatches: 13, cost_unmetered: 0

## Time
state: partial
active_ms: 1259483
reason: provider-evidence-incomplete

## Build Review
laps_to_pass: 3
skipped: 0
cache_hits: 0
infrastructure_failures: 9
rubrics:
  moneySafety: failures: 0, judged: 3
skip_reasons:
