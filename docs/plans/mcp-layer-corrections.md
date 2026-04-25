Big finding — the schema, constructors, and even confirmation logging (via `auto`) already exist:

- `auto.go:182` writes `NewConfirmedEntry` on HIGH-confidence auto-accept
- `correct.go:69` writes `NewCorrectedEntry` for post-hoc overrides
- `add.go:163` writes `NewManualEntry` — **no prediction context, even when caller has one**
- `batch_auto.go:277` also writes confirmed

The gap is exactly one-shaped: `add` doesn't participate in the predicted/actual schema. Everything else is plumbing.

One small wrinkle this raises for #2: you picked (C) "defer confirmations," but `auto` already logs `confirmed`. So "defer" really means *don't newly confirmation-log in `add`* — the auto path keeps doing what it does. That leaves a choice when `add --predicted-subcategory X` is called with `chosen == X`: write `NewConfirmedEntry` (consistency with `auto`) or skip the feedback write (smaller diff, user's "defer" intent). I'll flag this in the plan as a small call, leaning toward **writing confirmed** for consistency — the Telegram "user clicked the top candidate" case is semantically identical to `auto` high-confidence auto-accept.

---

# Plan: MCP-Layer Corrections

## Goal
Close the feedback loop for the Telegram flow: when the MCP bot shows classify candidates and the user picks one, `add_expense` propagates enough context that the Go CLI can log a `corrected` (or `confirmed`) feedback entry linked by sha256[:12] ID.

## Scope
- Go CLI: `add` command gains `--predicted-subcategory`, `--predicted-category`, `--classification-id`, `--confidence`, `--model` flags
- Feedback writer: `add` writes to both `expenses_log.jsonl` (existing) and `classifications.jsonl` (new, when prediction flags present)
- MCP wrapper (`mcp-server/`): `classify_expense` returns the ID; `add_expense` accepts and forwards the new fields
- Acceptance tests: three new scenarios covering confirmation, correction, backwards compat
- Out of scope: status lifecycle, SQLite, Telegram bot itself, 5.R1 retrieval work

## Step 1 — Verify no double-count risk
Quick grep to confirm nothing in `internal/classifier/` reads `expenses_log.jsonl`. If clean, both-log strategy is safe. (~5 min)

## Step 2 — Acceptance tests first
Three scenarios in `expense-reporter/test/` mirroring the `correct` command's pattern:

- **`add_with_predicted_match`** — `add` with `--predicted-subcategory == chosen` → insert log entry + `confirmed` feedback entry, same ID across both
- **`add_with_predicted_mismatch`** — `add` with `--predicted-subcategory != chosen` → insert log entry + `corrected` feedback entry, predicted/actual fields both set correctly
- **`add_without_prediction_flags`** — existing behavior unchanged: insert log entry + `manual` feedback entry, no correction log

Helpers already exist (`verify.CorrectionLogged`, `harness.SeedFileFromFixture`, `actions.RunAdd`). May need `actions.RunAdd` extension for new flags and a `verify.ConfirmationLogged`. Follow PATTERNS.md (Given past-tense, composable Then).

## Step 3 — Go CLI changes (TDD inner loop, delegate to Ollama)

**Files touched:**
- `cmd/expense-reporter/cmd/add.go` — add cobra flags, branch feedback-writing logic
- Possibly `internal/feedback/feedback.go` — if `FindLatestEntry` lookup-by-ID is needed for `--classification-id` validation (likely yes, already exists per earlier grep)

**Logic in `add.go`:**
1. Parse new flags (all optional; keep backwards compat default)
2. Write insert entry to `expenses_log.jsonl` (unchanged)
3. If `--predicted-subcategory` present:
   - If `--classification-id` given, validate it resolves via `FindLatestEntry` (warn-and-continue on miss — don't block insert)
   - If `chosen == predicted`: `NewConfirmedEntry` → append to `classifications.jsonl`
   - If `chosen != predicted`: `NewCorrectedEntry` → append to `classifications.jsonl`
4. Else: existing `NewManualEntry` path

Delegate to `my-go-qcoder` per tier list; expected ~60 LOC diff, single-file, well-bounded.

## Step 4 — MCP wrapper changes (`mcp-server/`)

Need to peek at the wrapper first to know the exact shape, but likely:
- `classify_expense` already calls `auto --json`; ensure response surfaces the ID (may require small Go-side `auto --json` addition if not already there — flag for investigation)
- `add_expense` accepts optional `predicted_subcategory`, `predicted_category`, `classification_id`, `confidence`, `model` args and forwards as CLI flags

Python is outside the Go tier list; still usable for Ollama delegation on the bridging logic.

## Step 5 — Handoff notes
- Vision doc doesn't need editing (this is within Phase 3 scope)
- Session-context line 121 updated: "Telegram-flow corrections" moves from deferred → shipped
- README/FEEDBACK_SYSTEM: document the new `add` flags and the predicted/actual flow through MCP

## Open micro-decisions (pick before I start, or I'll pick)
1. `chosen == predicted` case: write `confirmed` (my lean, consistency with `auto`) or skip (honors your 2.C more strictly)?
2. `--classification-id` missing-lookup: warn-and-continue or hard error? (My lean: warn — don't block the insert because a log was rotated)
3. Does `auto --json` already emit the ID? I'll check during Step 4 scoping; if not, tiny add.

Ready to proceed, or want to refine anything?