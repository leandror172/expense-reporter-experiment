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
writes explicit `DD/MM/YYYY`. The per-year split is retire-capable — promoting the merged
log to canonical is still the user's call.
