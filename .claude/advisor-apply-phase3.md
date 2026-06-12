# Advisor Review — apply command Phase 3
**Date:** 2026-05-29
**Context:** Pre-implementation review of cmd/apply.go before Ollama codegen call.

---

## Summary verdict

Plan is sound. Lazy-workbook delta is the right and only material correction. Task list is appropriately scoped.

---

## 1. Most likely to fail the test: model name on corrected-already-inserted line

`FeedbackMatchesExpected` checks every field present in expected **except `id` and `timestamp`** — and `model` IS checked. Expected line 3 has `"model":"my-classifier-q3"`.

That value can ONLY come from the prior entry via `FindLatestEntry` (reviewed.json has no `model` field). The `"review"` sentinel is for **new rows only** (no prior entry). For Diarista (corrected + found), use `prior.Model`, `prior.PredictedSubcategory`, `prior.PredictedCategory` — not the sentinel.

**Ollama prompt constraint:** "for corrected rows that already have a prior classifications entry, the corrected feedback entry's model and predicted_* come from the prior entry, not the sentinel."

---

## 2. Exactly 3 lines, in order

`FeedbackMatchesExpected` asserts `len(expected) == len(actual)` first, then compares line-by-line in file order. Actual must be: seed line 1 (Uber confirmed), seed line 2 (Diarista confirmed), appended line 3 (Diarista corrected).

Uber Centro (confirmed + already-inserted → no-op) must write NOTHING. If the code appends a confirmed entry for Uber, or anything for pending Academia, you get 4 lines and fail the length check.

---

## 3. Brittle string match

`OutputContains("workbook not updated")` is a literal substring. Feed the exact phrase to Ollama as a hard constraint on `printSummary`. The ⚠ section emits when `alreadyInsertedCorrections` is non-empty (Diarista populates it).

---

## 4. Lazy validation — correct, with one edge the plan omits

Open/validate workbook only when `len(newRows) > 0`. But also: **newRows non-empty AND workbookPath == "" → return a clear error**, not a panic or silent skip.

---

## 5. Real blind spot: insertNewRows is entirely uncovered by this test

All three fixture entries are no-op / feedback-only / pending. The insertion path never runs. A green acceptance test tells you nothing about whether `insertNewRows` is correct.

Batch API shapes to get right:
- `FindSubcategoryRowBatch` → `map[sheetName]map[subcategory]int` (two-level)
- `AllocateEmptyRows` → `map[int]int` keyed by `ExpenseIndex` (not sheet/subcategory)
- Both open the workbook once over all requests — call them batched, not per-entry

Phase 4 smoke against a real `reviewed.json` is the only behavioral check on insertion. Do not skip it. If no real `reviewed.json` is available, name that gap in the handoff.

`ParseDateWithYear` is only on the insertion path — the feedback path stores the date string as-is. Won't be exercised by the acceptance test.

---

## 6. Simplification: no taxonomy loading needed

`apply` gets full `category` directly from `reviewed.json`, so it does NOT need `resolveCategoryFromTaxonomy`, `--data-dir`, or taxonomy loading (unlike `correct.go`). It still imports `classifier` for the `classifier.Result` struct. Do not let Ollama drag in correct.go's taxonomy-resolution machinery by pattern-matching.

---

## Net

- Lazy-validation is the only material correction to the plan.
- Model-name-from-prior-entry is the detail most likely to silently fail the test.
- Insertion path silently ships a bug if wrong (no test coverage); verify by reading after codegen.
