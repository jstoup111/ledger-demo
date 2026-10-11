# Track: open-a-new-account

Track: product

Scope boundary: open a new, empty account by name — domain capability first, then the JSON/form
endpoint and the page form. Excluded: renaming, closing or deleting accounts, an opening balance
(a first deposit is an ordinary transaction), and any uniqueness check on names (duplicate
detection is a non-goal; two accounts may share a name and differ by id).

The change alters what an audience sees and does: the presenter opens an account on stage and it
appears in the account list, selectable and ready to post to. It is judged by observable behavior,
so `/prd` runs.

Confirmed by the operator in advance (pre-authorized DECIDE outputs for this spec; recorded as a
pre-granted decision).
