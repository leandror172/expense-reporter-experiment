# Feedback System & Corrections

## Overview

The feedback system logs classification decisions to `classifications.jsonl` for training, tuning, and audit purposes. It tracks both automatic classifications and manual entries.

<!-- ref:feedback-entry-structure -->
## Feedback Entry Structure

Each line in `classifications.jsonl` is a JSON entry with:

```json
{
  "id": "abc123def456",           // SHA256 hash prefix of (item|date|value)
  "item": "Dmae",                 // Expense description
  "date": "2023/06",              // DD/MM or YYYY/MM format
  "value": 157.30,                // Numeric value
  "predicted_subcategory": "Transport",  // Model prediction (or empty if manual)
  "predicted_category": "Transport",     // Model prediction (or empty if manual)
  "actual_subcategory": "Utilities",     // User's correction or accepted prediction
  "actual_category": "Utilities",        // User's correction or accepted prediction
  "model": "my-classifier-q3",    // Model used (or empty if manual)
  "status": "confirmed",          // confirmed | corrected | manual
  "timestamp": "2026-04-20T14:07:00Z"
}
```

**Status meanings:**
- `confirmed`: Model prediction was accepted (auto-inserted or user accepted it)
- `corrected`: Model prediction was wrong; user corrected it
- `manual`: User manually entered category (no model involved)

<!-- /ref:feedback-entry-structure -->

<!-- ref:feedback-sources -->
## Feedback Sources (What Creates Entries)

### Command: `add` (Manual Entry)
- User manually specifies category: `add "Dmae" "15/06" "157,30" "Utilities" "Utilities"`
- Creates: `status="manual"` entry
- Has: empty `predicted_subcategory`, `predicted_category`, `model`
- Purpose: Track user-entered expenses (not model predictions)
- Code: `cmd/add.go:logManualFeedback()`

### Command: `auto` (Single Classification + Insert)
- User classifies one expense: `auto "Dmae" "15/06" "157,30"`
- Model predicts category + confidence
- If confidence > threshold (default 85%): auto-inserts + logs
- Creates: `status="confirmed"` entry with `predicted == actual`
- Code: `cmd/auto.go:logConfirmedFeedback()`

### Command: `batch-auto` (Batch Classification + Insert)
- User batch-classifies CSV: `batch-auto expenses.csv`
- For each row where `AutoInserted=true`:
  - Model predictions accepted
  - Creates: `status="confirmed"` entry
- For rows below threshold: no entry (goes to `review.csv` instead)
- Code: `cmd/batch_auto.go:logBatchFeedback()`

### Command: `correct` (Override Prior Auto-Classification)
- User overrides a wrong auto-insert: `correct "Uber Centro;15/04;35,50;Combustível"`
- Looks up the latest entry matching the expense ID via `feedback.FindLatestEntry`
- Copies `predicted_subcategory`, `predicted_category`, `confidence`, `model` from the prior entry
- Sets `actual_subcategory` / `actual_category` from the user's correction (resolved via taxonomy)
- Creates: `status="corrected"` entry
- Fails with a clear error if no prior entry exists (suggests `add` instead) — does NOT touch the workbook
- Code: `cmd/correct.go:runCorrect()`

<!-- /ref:feedback-sources -->

<!-- ref:feedback-training -->
## Feedback as Training Examples

Corrected/confirmed entries become few-shot examples for the next classification run:

```
LoadFeedbackExamples(classifications.jsonl)
  ├─ status="confirmed" → Example { Subcategory: predicted, Category: predicted, Source: SourceConfirmed }
  ├─ status="corrected" → Example { Subcategory: actual, Category: actual, Source: SourceCorrected }
  └─ status="manual" → skipped (no model prediction to learn from)
```

**Code location:** `internal/classifier/loader.go:LoadFeedbackExamples()`

These examples are injected into the classifier prompt as few-shot learning data.

<!-- /ref:feedback-training -->

<!-- ref:feedback-correction-workflow -->
## Correction Workflow (status="corrected")

**Closed in Layer 5.9.** The feedback loop now writes corrected entries via the `correct` command.

**Flow when a confirmed auto-insert was wrong:**
1. User runs `correct "item;DD/MM;value;right_subcategory"`
2. `feedback.FindLatestEntry` scans `classifications.jsonl` for the most recent entry with the matching ID (last occurrence wins — handles re-classifications across model upgrades)
3. `feedback.NewCorrectedEntry` builds a new entry: `predicted_*` and `model` carried over from the prior entry, `actual_*` from the user's correction (category resolved via `feature_dictionary_enhanced.json`)
4. Entry appended to `classifications.jsonl`; workbook is **not** touched

**Constraint:** `correct` requires a prior entry. If none exists, it fails with a hint to use `add` instead. This matches the Telegram-flow design intent — corrections always override a model prediction; entries with no prediction history are manual entries.

**What's still future work:**
- The Telegram bot equivalent: writing `corrected` entries at insert time when the user picks a non-top candidate from a model's proposals (currently `add` only writes `manual` — needs an MCP/bot-layer extension that accepts a predicted-subcategory parameter)
- Workbook cell relocation (`correct` does not move the value to the new subcategory cell — user fixes the workbook manually)

<!-- /ref:feedback-correction-workflow -->

<!-- ref:feedback-file-path -->
## Feedback File Location

Configured in `config/config.json`:
```json
{
  "classifications_path": "classifications.jsonl",
  "expenses_log_path": "expenses_log.jsonl"
}
```

**Resolution:**
- Absolute path: used as-is
- Relative path: resolved relative to binary directory

**Code:** `internal/config/config.go:ClassificationsFilePath()`

<!-- /ref:feedback-file-path -->

<!-- ref:feedback-cold-start -->
## Cold Start Behavior

If `classifications.jsonl` doesn't exist:
- First run: no feedback examples loaded
- Classifier uses training data only (`training_data_complete.json`)
- After first batch-auto run: file is created
- Subsequent runs: use confirmed/corrected examples + training data

**Code:** `internal/classifier/loader.go:LoadFeedbackExamples()` (returns nil, nil if file missing)

<!-- /ref:feedback-cold-start -->
