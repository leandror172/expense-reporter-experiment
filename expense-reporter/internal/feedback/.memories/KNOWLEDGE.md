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
It is set **post-construction** (constructor signatures unchanged) and **only on the
apply path** — `apply.go` assigns `.Type = entry.Reviewed.Type`. `omitempty` means
type-less entries (auto/batch-auto/add/correct, which build from the classifier — no
type yet) serialize byte-identically to before. The type is the join field the generator
uses for full-path routing ([[taxonomy]] two-tier). Closing the type-less producer gap =
classifier full-path label (5.R4/RUI-4).

**Time:** timestamps are RFC3339 UTC.
