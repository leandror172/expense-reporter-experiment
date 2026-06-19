# Runbook — Plan A→B Real-Data Verification (Bf1–Bf3 + routing proof)

**Purpose:** the only end-to-end proof that the Plan A (persist type) → Plan B (full-path
routing) chain works on **real data**, not just synthetic test literals. All automated
tests use entries written as string literals alongside the taxonomy, so both sides of the
`byPath` lookup match *by construction* — that proves the mechanism, not the integration.
Real inputs flow review UI (reads workbook) → `reviewed.json` → `apply` → `expenses_log.jsonl`,
looked up against the **separately authored** `config/taxonomy.json`. This runbook checks
those strings actually match.

**Status when written:** Plan A (T-05, PR #29) + Plan B (T-04, PR #30) implemented &
pushed. NFC-normalization safeguard in place. Bf2 tool (`backfill-type.py`) built. Bf1/Bf3
are manual and PENDING — execute next session.

**Prereqs / file locations** (repo root = `/mnt/i/workspaces/expenses/code`):
- Saved review page: `review-2026-05-25.html` (repo root) — holds corrections in browser
  `localStorage` key `expense-review:v1:rows:<source>:<generatedAt>`.
- Real logs (gitignored): `expense-reporter/{classifications,expenses_log}.jsonl`.
- Real taxonomy (gitignored): `expense-reporter/config/taxonomy.json`.
- Pre-existing backup of logs: `~/expense-data-backup-20260619-121709/`.
- Backfill tool: `.claude/tools/backfill-type.py` (stdlib Python; backs up `*.bak` first).

> ⚠️ These logs are personal financial data — already gitignored (session 33). Never
> `git add -A`. Keep data out of commits.

---

## Step 1 — Export `reviewed.json` from the browser (Bf1)

1. Open `review-2026-05-25.html` in the browser whose `localStorage` holds the corrections.
2. Confirm past corrections are still displayed.
3. Click **"Export reviewed file"** (⇧E) → save as `~/reviewed.json`.

Note: the export carries the type. An old export may use the legacy `"sheet"` key — both
the backfill tool and `apply`'s `ReviewedLocation.UnmarshalJSON` read that as a fallback,
so an old export still works.

## Step 2 — Run the backfill tool (Bf2 — already built)

```bash
cd /mnt/i/workspaces/expenses/code
python3 .claude/tools/backfill-type.py ~/reviewed.json \
  --expenses-log     expense-reporter/expenses_log.jsonl \
  --classifications  expense-reporter/classifications.jsonl
```
Prints `matched / filled / unmatched`; writes `*.bak` backups first. Backfill is **partial
by design** — only reviewed entries get a type; auto/batch-auto/add entries stay type-less
(handled by the bare-name fallback). Do not try to drive `unmatched` to zero.

## Step 3 — Verify the backfill (Bf3)

```bash
cd /mnt/i/workspaces/expenses/code/expense-reporter
# (a) line counts unchanged — no lines added/lost
wc -l expenses_log.jsonl expenses_log.jsonl.bak
wc -l classifications.jsonl classifications.jsonl.bak

# (b) ambiguous leaves now carry a type (adjust names to real data)
grep -E 'Orion|Gás|Dentista' expenses_log.jsonl | grep '"type"'
```

## Step 4 — Real-data routing proof (the advisor's key check)

```bash
cd /mnt/i/workspaces/expenses/code/expense-reporter
go run ./cmd/expense-reporter generate-workbook \
  --taxonomy config/taxonomy.json \
  --entries  expenses_log.jsonl \
  --year 2026 \
  -o /tmp/real-check.xlsx
```

Read stderr carefully:
- `note: N entries routed via the type-less bare-name fallback` — expected; N = not-yet-typed entries.
- `skipping entry "X": subcategory "Y" ... ambiguous` — a **type-less** ambiguous entry; expected to skip until typed.
- `skipping entry "X": subcategory "Y" not in taxonomy` **on an entry that carries a type**
  → ⚠️ THE FAILURE MODE: the entry's `type`/`category`/`subcategory` don't byte-match
  `config/taxonomy.json` (spelling/accent/whitespace divergence beyond the NFC pass).

Confirm a known ambiguous item landed in the right sheet only:
```bash
go run ./cmd/workbook-inspect /tmp/real-check.xlsx /tmp/real-dump
grep -l 'YourOrionItem' /tmp/real-dump/*.json   # expect exactly the correct sheet
```

---

## Pass / fail criteria

- **PASS:** no `not in taxonomy` warnings on typed entries; backfilled ambiguous items
  appear in exactly the right sheet; line counts unchanged.
- **FAIL (typed entry warn+skips):** capture the entry's `type`/`category`/`subcategory`
  and the matching `config/taxonomy.json` line. Diagnose divergence:
  - accent NFC/NFD → already handled; if it still fails, the diff is elsewhere
  - trailing whitespace / casing → may justify extending `normalizeKey` (trim/casefold)
  - genuine spelling difference → data fix in taxonomy or review-derived strings
  Decide: data fix vs broaden `normalizeKey` (loader.go) — then add a regression test.

## Follow-on (separate work, not this runbook)
- Classifier full-path label (5.R4/RUI-4) — makes auto/batch-auto emit a taxonomy-exact
  type, shrinking the type-less fallback toward zero (then the bridge can be retired).
- The backfilled typed lines become gold few-shot labels for that work.
