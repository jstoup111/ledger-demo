---
name: money-safety
description: Review policy for build_review. Flags monetary amounts represented or computed with floating point, or precision lost in parsing, rounding, or conversion.
---

# Money safety review policy

Monetary amounts in this ledger are integer cents (`int64`). Judge only the changed code.

## Findings (report each occurrence)

1. **Float money.** A monetary amount stored, passed, returned, or computed as `float32`/`float64`
   (including `strconv.ParseFloat`, `math.Round` on dollars, or `json` fields typed as float).
2. **Lossy conversion.** Converting between dollars and cents via floating-point multiplication or
   division (e.g. `int64(x * 100)`), or truncating instead of rejecting sub-cent input.
3. **Overflow blind spot.** Summing or multiplying cent values with no bound or overflow check where
   inputs are user-controlled.
4. **Formatting drift.** Rendering cents to a display string via float formatting (`%f`, `%.2f` on a float).

## Not findings

- Floats used for non-monetary values (percent display, timings, metrics).
- Integer-cent arithmetic that is bounded or already validated upstream.
- Test fixtures that assert a float path is *rejected*.

## Evidence

Cite the file and line, quote the expression, and state which rule it breaks.
