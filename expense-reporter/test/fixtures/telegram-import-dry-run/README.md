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
| 5 | comma where the first `;` should be → 2 fields | rejected: 2 fields |
| 6 | conversation, no `;` and no money token | ignored |
| 7 | value is a clock time `16:20` | rejected: bad value |
| 8 | `31/02` — not a calendar date | rejected: bad date |
| 9 | stray `;` inside the item → 4 fields (the value `405,25 x4` is fine) | rejected: 4 fields |
| 10 | text as an ENTITY ARRAY (`["…", {"type":"bold","text":"08/07"}, "…"]`) | converted — the adapter flattens it |
| 11 | explicit `DD/MM/YYYY` | converted |
| 12 | one field but mentions money (`450,00`) — an attempted expense a human should see | rejected: 1 field |

All dates are in 2025 so the year resolution (relative to each message's own
timestamp, D1) is stable whatever year the suite runs in.
