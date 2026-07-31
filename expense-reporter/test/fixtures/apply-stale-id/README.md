# apply-stale-id fixture (T-41 slice 4)

`reviewed.json` carries an id **that does not describe its own row**: `f0c3bf1293f3`, hashed
from the BARE `15/04`, while the seeded classification carries `8c434a0b64c0`, hashed from
the canonical `15/04/2026` that apply itself writes.

That mismatch is the fixture's whole point. `apply` rewrites a bare date to canonical
form before doing anything else, so an id supplied by whoever produced the file was
derived from a date apply is about to change. Trusting it means looking up an id apply
will never write — and a lookup miss is SILENT: the entry is treated as new and
appended, so the symptom is a duplicate expense rather than an error.

The row must therefore be recognised as ALREADY APPLIED: no expense-log append.

Do NOT "fix" the id to the canonical value. The sibling `apply-basic` fixture is
self-consistent, which is correct for it but means it cannot catch this — with the id
recompute reverted, `apply-basic` still passes. This is the only fixture that
discriminates.

The date stays BARE for the same reason as `apply-join-id`: a full date makes raw and
canonical identical and disables the test rather than breaking it.
