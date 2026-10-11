# PRD: Open a new account

**Date:** 2026-10-10
**Status:** Approved

## Problem / Background

The ledger has exactly the accounts the seed data creates. A presenter who wants to show the ledger
starting from nothing — a fresh account, a first deposit, a balance appearing — has no way to do it
on stage: accounts exist only by editing seed data and resetting. Opening an account is the most
basic thing a deposit ledger does, and its absence is visible the moment someone asks "can I make a
new one?".

## Goals & Non-Goals

**Goals**

- A presenter can open a new, empty account by giving it a name, from the page or from a programmatic
  client, and immediately select it and post to it.
- Every rejection names the rule that was broken, in the same words on the page and in the
  programmatic response.
- The account list stays in a predictable order however many accounts are opened.

**Non-Goals**

- Renaming, closing, or deleting an account.
- An opening balance or initial deposit on creation — a first deposit is an ordinary transaction.
- Any uniqueness check on account names. Two accounts may share a name; they differ by identifier.
  Duplicate detection of any kind is a project non-goal.
- Account types, owners, users, or any notion of who opened the account.

## Users / Personas

- **Presenter:** drives the single page on a projector and wants a new account to appear on cue.
- **Programmatic client:** a `curl` on stage, exercising the same capability as the page.

## Functional Requirements

Domain capability (usable without any web surface):

- **FR-1:** Opening an account with a valid name records a new account with that exact name and a
  new identifier, and returns it. The new account has a balance of zero and no transactions.
- **FR-2:** Identifiers are assigned deterministically in the order accounts are opened, continuing
  the sequence the seeded accounts already use, so the same sequence of openings after a reset always
  yields the same identifiers.
- **FR-3:** A name that is empty or consists only of whitespace is rejected; nothing is recorded.
- **FR-4:** A name longer than 40 characters is rejected; nothing is recorded. A name of exactly 40
  characters is accepted. Length counts characters, not bytes.

Web surface (builds on FR-1 – FR-4):

- **FR-5:** A programmatic client can open an account and receives the created account — identifier,
  name, and zero balance — with a "created" status, in the same shape the account list already uses.
- **FR-6:** A programmatic rejection (FR-3, FR-4) returns the project's standard error shape with a
  stable code distinct per rule and a "bad request" status. The too-long message names the submitted
  character count and the 40-character limit.
- **FR-7:** Whenever the page shows a selected account, it also offers an "Open account" form, below
  the account's transactions. Submitting a valid name shows the page with the new account listed and
  selected. The unknown-account and no-accounts pages are unchanged and offer no form.
- **FR-8:** Submitting an invalid name from the page shows the same message the programmatic response
  carries for that rule, adjacent to the open-account form, and opens no account.
- **FR-9:** Accounts are listed — on the page and in the programmatic account list — in the order
  they were opened. The tenth account and later appear after the ninth, never between earlier ones.

## Non-Functional Requirements

- No floating point anywhere; a new account's balance is integer cents (zero).
- Deterministic: identifiers depend only on how many accounts exist, never on time or randomness.
- The page carries no JavaScript; both forms are plain HTML posts.
- Reloading the page after a successful open does not open a second account.

## Acceptance Criteria / Success Metrics

- Every FR has a passing test at the lowest sufficient layer; FR-7/FR-8 are covered by an
  end-to-end flow against the running server.
- On stage: `make reset`, type a name, submit — the account appears selected with `$0.00`, and a
  deposit posted to it updates its balance.

Stories: `.docs/stories/open-a-new-account.md`.

## Scope

### In Scope

- Domain capability to open a named account (FR-1 – FR-4), delivered and tested first.
- Programmatic and page surfaces for it (FR-5 – FR-8).
- Account list ordering that survives more than nine accounts (FR-9).

### Out of Scope

- Everything under Non-Goals, and every item on the project's non-goals list.
- Any change to transaction posting, balances, or seed data contents.

## Key Decisions & Rationale

- **No name uniqueness.** A uniqueness rule is duplicate detection, which is reserved as a live-demo
  feature; identifiers alone distinguish accounts.
- **40-character limit.** A name must fit the account navigation on a projector; descriptions keep
  their existing 140-character limit.
- **Name kept exactly as submitted.** Matches how transaction descriptions are stored; only the
  blank-check looks past whitespace.
- **Domain first, surfaces second.** The capability is useful and testable alone, so it ships as its
  own reviewable step before any route or form depends on it.

## Dependencies

- Existing accepted contracts: the API response contract (error shape, negotiation between JSON and
  form posts, redirect-after-post) and the rejection-message composition decision.
- The existing domain storage boundary, which must gain the ability to record an account.

## Open Questions

- Identifier scheme versus listing order: the seeded identifiers are a prefix plus an unpadded
  number, which sorts wrongly as text from the tenth account. Pad new identifiers, change the
  scheme, or order by sequence? — for `/architecture-review`.
- Where the page shows an open-account rejection, given the page already shows posting rejections
  above the posting form — for `/architecture-review`. (Resolved: above the open-account form; ADR
  `adr-2026-10-10-sequential-account-ids-ordered-by-sequence` D4.)
