---
slug: account-totals-row
spec_hash: ff2f8829e9480b86b66bbc42e3178012f068e06c86ba745b4071326fa48ddd5f
pr: https://github.com/jstoup111/ledger-demo/pull/79
shipped: 2026-09-26
engine_version: 20260926T200710Z-7b7daee60586
---

## Cost
input: 46578
output: 30531
cache_read: 2406559
cache_creation: 220468
cost_usd: 3.5809
dispatches: 29
retries: 0
halts: 4
unmetered: count: 21, duration_ms: 0
cost_unmetered: count: 0
providers:
  claude: input: 112, output: 29071, cache_read: 2298143, cache_creation: 220468, cost_usd: 3.3541, dispatches: 18, cost_unmetered: 0
  codex: input: 46466, output: 1460, cache_read: 108416, cache_creation: 0, cost_usd: 0.2268, dispatches: 11, cost_unmetered: 0

## Time
state: partial
active_ms: 898988
reason: provider-evidence-incomplete

## Build Review
laps_to_pass: 3
skipped: 0
cache_hits: 0
infrastructure_failures: 9
rubrics:
  moneySafety: failures: 0, judged: 2
skip_reasons:
