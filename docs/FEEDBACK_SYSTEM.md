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

<!-- ref:feedback-missing-feature -->
## Missing: Correction Workflow (status="corrected")

**The problem:**
- System *can read* corrected entries (loader supports them)
- System *can use* them for training
- But **nothing writes them**

**What's missing:**
- No `NewCorrectedEntry()` function
- No `correct` command to accept user corrections from `review.csv`
- User has no way to log: "Model predicted X, but actual is Y"

**Current state:**
- `add` command creates `status="manual"` entries
- `auto`/`batch-auto` create `status="confirmed"` entries
- No way to create `status="corrected"` entries

**Impact:**
- If model mispredicts but user doesn't catch it (auto-inserts), it becomes training data poison
- Users can't leverage their corrections to improve the model
- Feedback loop is broken

<!-- /ref:feedback-missing-feature -->

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
