# Intake origin: seed-recorded-times

Source-Ref: jstoup111/ledger-demo#3
Owner: jstoup111

## Desired outcome

- Seeded transactions carry distinct recorded times, so the page's time column varies row to row.
- Two consecutive `make reset` runs still produce byte-identical data, and the suite still passes with `-count=2`.
- Newest-first ordering still holds, and the id tiebreak still has at least one pair of equal-timestamp rows somewhere in the seed so the rule stays covered by a test rather than becoming dead code.
- No wall-clock read is introduced: `time.Now()` still appears exactly once in the repository, inside `SystemClock`.
