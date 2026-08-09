# ledger-demo — Codex Instructions

## Harness

This project uses the james-stoup-agents harness. Its skills are installed in the user's
Codex configuration at `~/.agents/skills/`; do not copy them into this project.

Codex MUST read and follow `~/.agents/skills/HARNESS.md` at the start of every session. A
project-local skill is only appropriate when this project deliberately overrides a harness skill.

## Shared Workflow and Codex Invocation

The shared harness workflow and lifecycle gates are defined by `HARNESS.md`; they are the same
for every supported host. Codex invokes a harness skill with native `$skill-name` syntax (for
example, `$tdd`) and must not use Claude slash-command syntax.

## What this project is

A toy deposit-account ledger built to be **demoed live on a projector while an AI harness adds
a feature to it**. It is a stage prop, not a product: small, legible, deterministic, instantly
resettable. It is **not** double-entry.

The full project conventions, tech stack, and — critically — the **non-goals list** are in
`CLAUDE.md`. Both host files describe the same project and the same harness contract; read
`CLAUDE.md` for the authoritative project detail rather than duplicating it here.

**Before adding any feature, read the non-goals list in `CLAUDE.md`.** Features are added live
on stage; if they already exist, the demo is ruined.

## Agent Personas

Use the harness agent personas when their specialized review or implementation roles fit the task.

### Generator
- **Prompt:** `agents/generator.md`
- **Role:** Implements code via TDD
- **Context:** Receives only files relevant to current task
- **Statuses:** DONE, DONE_WITH_CONCERNS, NEEDS_CONTEXT, NEEDS_DRILL_DOWN, BLOCKED

### Evaluator
- **Prompt:** `agents/evaluator.md`
- **Role:** Reviews code with calibrated skepticism
- **Context:** Fresh context reset — no shared state with generator
- **Stages:** Spec compliance → Code quality → Domain integrity

### Domain Reviewer
- **Prompt:** `agents/domain-reviewer.md`
- **Role:** Checks domain integrity during TDD DOMAIN phases
- **Authority:** Veto — can reject and send back to RED/GREEN

### Planner
- **Prompt:** `agents/planner.md`
- **Role:** Expands requirements into specifications
- **Output:** Structured specs with scope, edge cases, and story suggestions

## Dispatch Patterns

### TDD Cycle (Structural Enforcement)
```
Generator (RED, test-only context)
  → Domain Reviewer (test review)
  → Generator (GREEN, source-only context)
  → Domain Reviewer (implementation review)
  → COMMIT
```

### Code Review (Generator/Evaluator Separation)
```
Generator produces code
  → Evaluator reviews with fresh context
  → Generator fixes (if needed)
  → Evaluator re-reviews
```

### Pipeline (Autonomy-Based)
```
Conservative: Human approves → Generator → Evaluator → Human reviews
Standard:     Generator → Evaluator → auto-continue (escalate on 3 rework failures)
Full:         Parallel Generators → Evaluators → auto-merge on green
```
