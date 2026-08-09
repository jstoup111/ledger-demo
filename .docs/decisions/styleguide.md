# Frontend Styleguide — ledger-demo

This is a **projector styleguide**, not a product styleguide. The single measure of success is
that a room full of people can read the page from a distance while a presenter talks over it.

## Hard rules

| Rule | Value | Why |
|---|---|---|
| Base font size | **20px** (spec floor: 18px) | Legible at 1280×720 from across a room |
| Theme | **Light only** | No dark mode. One code path, no media query to debug on stage |
| Contrast | Near-black `#111` on white `#fff` | Projectors wash out mid-greys badly |
| Animation | **None** | Nothing moves; nothing distracts from the diff being demoed |
| Responsive breakpoints | **None** | The viewport is one known projector |
| JavaScript | **None** | No build step, no framework, no bundler |
| Fonts | `system-ui` stack only | No webfont fetch — the demo runs fully offline |

Dark mode, animation, and breakpoints are not "not yet" — they are excluded. Adding them adds
failure modes to a live demo and buys nothing.

## Palette

Deliberately tiny. Six values, all fixed hex — no theming layer, no CSS custom properties to
indirect through.

| Token | Value | Use |
|---|---|---|
| Background | `#ffffff` | Page |
| Text | `#111111` | Body copy, balances |
| Muted text | `#444444` | Secondary/stub text |
| Rule | `#dddddd` | Table borders |
| Error background | `#fdecea` | Error panel fill |
| Error accent | `#b3261e` | Error panel left border |

## Type scale

| Element | Size | Weight |
|---|---|---|
| Balance | `4rem` | 700 |
| `h1` | `2rem` | bold |
| Body | `20px` (`1rem` base) | 400 |
| Table content | `1rem` | 400 |

The balance is the single largest thing on the page. It is what the audience watches change.

## Error rendering

Errors are **always visible on the page** — never a console log, never a silent no-op, never a
disappearing toast. An error panel uses the error background with a 6px left accent border and
sits directly above the form that produced it.

A validation failure is a thing the presenter will deliberately trigger on stage. It has to be
obvious from the back of the room.

## Layout

- Single column, `max-width: 60rem`.
- `2rem` page padding.
- Order top to bottom: heading → account selector → balance → post form → transaction list.
- Tables use `border-collapse: collapse` with a bottom rule per row; no zebra striping, no
  vertical rules.

  > **Amended 2026-08-09 by amount-column-and-autofocus:** table cells are left-aligned with one
  > exception — the **amount column is right-aligned**, heading included, so a column of money
  > reads with its figures lined up and can be scanned down its right edge at projector distance.
  > The exception is applied by marking the amount cells (an `amount` class), never by column
  > position, so that adding or reordering a column on stage cannot silently align the wrong one.
  > Left alignment remains correct for the description and recorded-at columns: text compares on
  > its left edge, money on its right.

- The post form's amount field holds the input caret on load, requested declaratively in the
  delivered markup (there is no JavaScript to place it). Exactly one element on the page may
  request initial focus, so which field receives the caret is never browser-dependent.

## Current state

`web/style.css` implements the page-level rules above. The `.balance`, `.error`, and table
rules are present but commented out, waiting for the markup that the demo adds. Keep them
commented rather than deleting them — they encode the decisions in this document at the point
of use.
