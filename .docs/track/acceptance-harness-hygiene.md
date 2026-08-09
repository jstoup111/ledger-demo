# Track: acceptance-harness-hygiene

Track: technical

Three verified defects in the acceptance suite's own scaffolding
(`test/acceptance/harness_test.go`): a yield-free busy-wait readiness probe, a child server process
that outlives an interrupted `go test`, and a check-then-bind race on the port it reserves. Nothing
user-facing changes — the five HTTP routes, the domain, the schema, the seed data, and the page are
all untouched — so there are no product requirements to enumerate and `/prd` is skipped. Acceptance
criteria live directly in `.docs/stories/acceptance-harness-hygiene.md`.

Confirmed by the operator in advance (operator unavailable during authoring; recorded as a
pre-granted decision, not as fabricated approval).
