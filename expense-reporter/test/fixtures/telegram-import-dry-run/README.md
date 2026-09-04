# telegram-import-dry-run

Synthetic Telegram Desktop export (`result.json`) for the `telegram-import --dry-run`
bucket report. **No real messages** — the real export is personal data and lives outside
the repo; its per-message-id gate is run by hand (T-75 plan, tracker step 1).

One message per bucket, so a classification drift moves an id between lines:

| id | shape | bucket |
|---:|-------|--------|
| 13 | `type: service` (group created) | not a message — dropped by the adapter, never counted |
| 1 | `item; DD/MM; value`, same-day | converted |
| 2 | reported 4 days later, thousands separator `1.234,56` | converted (date resolves to 30/04/2025) |
| 3 | photo, empty text | receipts |
| 4 | PDF file, empty text | receipts |
| 5 | comma where the first `;` should be → 2 fields as typed | **repaired** (step 2, D4: that one comma → `;` is the only single edit that parses) |
| 6 | conversation, no `;` and no money token | ignored |
| 7 | value is a clock time `16:20` | rejected: bad value (no single edit parses either; the as-typed reason is kept) |
| 8 | `31/02` — not a calendar date | rejected: bad date |
| 9 | stray `;` inside the item → 4 fields as typed (the value `405,25 x4` is fine) | **repaired** (D4's four-field rule: first two fields merged with one space) |
| 10 | text as an ENTITY ARRAY (`["…", {"type":"bold","text":"08/07"}, "…"]`) | converted — the adapter flattens it |
| 11 | explicit `DD/MM/YYYY` | converted |
| 12 | one field but mentions money (`450,00`) — an attempted expense a human should see | rejected: 1 field (the comma edits give 2 fields, never 3) |
| 14 | `Bar; 15/05/ 2025,50` — TWO single edits parse with an in-window date: `/`→`;` gives 15/05/2025 + 2025,50, and `,`→`;` gives the date `15/05/ 2025` + value 50 | **rejected: ambiguous** — D4's exactly-one rule; guessing would write a row wrong by R$1.975,50 and indistinguishable from a right one. The value has to spell the current year for this to happen, which is why the real export has zero of these |
| 15 | `Tela celular; 25/07/ 90,00` — the s75 slash-typo shape | **repaired** (`/`→`;`); the comma edit reads the year as 2090, outside the window (and beyond the current year), so exactly one edit survives. The 2025 export's `21/08/ 1200,00` is the same shape with a PAST year: year 1200 is inside the boundary's 1..9999 and behind the current year, so only the window rule rejects that reading — the reason D4 gained it |

All dates are in 2025 so the year resolution (relative to each message's own
timestamp, D1) is stable whatever year the suite runs in.

**Step 1 → step 2 (s75).** Before repairs existed, ids 5 and 9 sat in `rejected: 2 fields`
and `rejected: 4 fields`; step 2 moves them to `repaired`, and ids 14 and 15 were added so
the two sides of the exactly-one rule are both pinned end to end.
