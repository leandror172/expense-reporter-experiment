# feedback — KNOWLEDGE

**Two-file split rationale.** `classifications.jsonl` is the rich audit/training log
(predicted vs actual, confidence, model, status). `expenses_log.jsonl` is a slim
"what got inserted" log that the **workbook generator reads** as its entry source
(internal/taxonomy `scanEntries`). So `ExpenseEntry`'s shape is also the generator's
input contract — changing it ripples into taxonomy routing + generate fixtures.

**ID is the contract glue.** `GenerateID` normalizes item (lowercase+trim) then hashes
`item|date|value` (`%.2f`). The same ID appears in `classifications.jsonl`,
`expenses_log.jsonl`, and the UI's `reviewed.json`, which is what lets a backfill match
across them (Plan A Phase B-fill).

**Entry field semantics.** `Entry` keeps both predicted (`PredictedSubcategory/Category`)
and actual (`ActualSubcategory/Category`); confirmed sets predicted==actual, manual
leaves predicted empty, corrected differs. `Now` is a `var` so tests inject a fixed
timestamp.

**Type field (Plan A / T-05).** Both `Entry` and `ExpenseEntry` carry
`Type string json:"type,omitempty"` (expense type: Fixas/Variáveis/Extras/Adicionais).
Set **post-construction** (constructor signatures unchanged). **5.R4 landed:** the
expenses_log producers all assign it — `apply.go` (`.Type = entry.Reviewed.Type`),
`auto.go:200`, `batch_auto.go:215` (`resolveExpenseType`) — so **expenses_log is fully
typed going forward**; `omitempty` keeps any residual type-less line byte-identical. The
type is the join field the generator uses for full-path routing ([[taxonomy]] two-tier).
Remaining type-less writers (`add`, `correct`) target only `classifications.jsonl`, not
the generator input. **Income is structurally type-less** → still uses the bare-name
fallback, which is why retiring it (T-09) needs a dedicated income route, not just the
classifier. See [[project_workbook_extraction_5r4]] for the per-year/year-implicit log
constraint and the pending "year adaptation".

**Time:** timestamps are RFC3339 UTC.

**Year handling (updated 2026-07-01).** expenses_log was historically year-implicit
(`DD/MM`), which forced the 5.R4 multi-year history into per-year files
`expenses_log-{year}.jsonl` fed to generate via `--year`. WS-A/T-11 (session 37) landed
"year adaptation": `taxonomy.parseDate` now accepts `DD/MM/YYYY` too and `scanEntries`
filters by target year, so one merged multi-year log suffices; the append path (WS-B)
writes explicit `DD/MM/YYYY`.

**PROMOTION DONE (session 70).** `expenses_log-allyears.jsonl` is now canonical
`expenses_log.jsonl`: 2073 rows, **every date `DD/MM/YYYY`**, spread 2022:138 / 2023:853 /
2024:349 / 2025:733. The per-year files are kept on disk as backup, not deleted.
**Why it had to happen before the first 2026 close:** `scanEntries` keeps an entry iff
`entryYear==0 || targetYear==0 || entryYear==targetYear` — year-0 legacy is ALWAYS kept, by
design, so the old per-year files still work with `--year N`. But that made the year-less
2025 log leak wholesale into any other year. Measured: `generate-workbook --year 2026`
against the pre-promotion log produced a **107K** workbook — the same size as the 2025 one,
because it contained all 733 of 2025's expenses labelled 2026. Post-promotion: 83K, the
empty skeleton. Verify a rollover this way (a check that can FAIL), never by reading a
success message.
⚠️ **The merge rewrote `date` and deliberately did NOT rewrite `id` — do not "fix" this.**
The ids are sha256 of the PRE-merge year-less bytes, so they no longer equal
`GenerateID(item, date, value)` of the row's own stored date. That looks like a breach of
"identity is DERIVED, never trusted", but `classifications.jsonl` still stores `DD/MM`, and
these ids are the ONLY thing that still joins the two logs: measured 347 joined ids before
the promotion and 347 after. Recomputing them from the new dates would silently drop that
join to zero. The `date` field serves the generator's year filter; `id` serves the cross-log
join; they are pinned to different bytes on purpose.
**Also fixed during the promotion:** the merged log predated the s68 `IRFF`→`IRRF` backfill
and still carried the drift-corrupted spelling on one 2025 row — promoting it unchecked
would have reintroduced exactly the S2 bug (unroutable leaf → `generate-workbook`
warn-skips → row vanishes at exit 0). Caught by diffing the merged file's 2025 subset
against the live log before copying; validate every typed path against
`config/taxonomy.json` before promoting anything.

**ID as ledger key (T-20, session 60).** `LoadExpenseIDCounts(path)` (`id_counts.go`)
returns per-`id` multiplicity from `expenses_log.jsonl`; it is the source-of-truth ledger
for `batch-auto --resume` (skip already-logged rows) and the always-on duplicate-append
warning. Missing file → empty ledger; a malformed line is an ERROR, not skipped (the log
is authoritative, corruption must surface). `appender.PredictEntryIDs` computes the same
ids `ExpandAndAppend` writes (shared `expandEntries`), so resume matches installment
series `(i/N)` + per-month dates exactly. Counts (not a set) because identical
(item,date,value) triples can be legitimate distinct expenses.
