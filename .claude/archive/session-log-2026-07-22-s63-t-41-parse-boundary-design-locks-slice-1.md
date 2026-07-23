## 2026-07-22 - Session 63: T-41 parse boundary — design locks + slice 1 (internal/parse, add/correct; PR #54); Fable verdict transcript gap

### Context

Session started from resume with all prior PRs merged and local master updated; user asked for a next-steps discussion, chose the T-41 design session, locked all four §10 questions interactively, then green-lit implementation of slice 1.

### What Was Done

- T-41 design session: all §10 opens locked; `.claude/plans/parse-boundary.md` DRAFT → FINAL (commit `784eb03`)
- `internal/parse` boundary package built TDD via my-go-qcoder: `ParsedExpense` (single `time.Time` + `DateString()`), year ladder, BR thousands normalization (`358c4fd`)
- `add`/`correct` repointed to the boundary; `parseExpenseForFeedback` deleted; `--year` on both; `date_year` live as rung 3 (`6b54428`)
- Acceptance: 4 year-precedence tests (`add_year_test.go` + `expect.OutputJSONHasDate`); `correct --year` join test (fixture `correct-year-flag`, seed id `26138364fcc9`); BR thousands CLI pin; `RunCorrect` gained variadic flags (`5021e0b`)
- Index + memories synced: new `internal/parse` QUICK, root/app/test QUICK + app KNOWLEDGE updates (`0f787b2`, `3481274`)
- Pattern-conformance pass vs llm-repo code-design/refactoring conventions: single-home field validation, contract-not-mechanism doc comments (`2b30dc3`)
- Pushed master (fast-forward) + branch; opened PR #54
- Verdict pipeline investigation: Fable transcripts carry NO assistant text → Stop-hook capture inert; 4 verdicts (2/1/1/1) appended directly to `calls.jsonl` (`source: manual-append-2026-07-22-transcript-gap`); memory `feedback_verdict_transcript_gap` written; llm-repo T-109 note added (uncommitted there)
- Verification: unit suite green, deterministic acceptance 36 pass, `-full` 114s green, gofmt/vet clean

### Decisions Made

- §10 locks: home = new `internal/parse` (NOT rehabilitated `internal/parser` — that dies with WS-E); struct = single stored `time.Time` + `DateString()` method (user proposal — inconsistent date pair unrepresentable; formatting can't fail; supersedes the "both fields" lean); API = field-wise core + semicolon wrapper returning subcategory ALONGSIDE (classification ≠ parsing); migration order add/correct → auto → batch-auto → apply, one PR per slice; boundary owns ALL input normalization (Q3); internal-first — MCP `parse_expense` deferred to Layer 6 (T-15 slice-2 lesson)
- Grace window = 0 (user never enters future-dated transactions); bare date strictly after Now → LAST year; same-day is not future
- `correct` also gets `--year` (beyond the plan's list) — T-16 parity; its GenerateID lookup must resolve years identically to what `add` wrote
- `parseOptions(yearFlag, cfg)` cmd-level extraction deferred to slice 2 (no seam below 3 callers — extract-keep-divergence rule 3); recorded in plan §5
- Rung 4 stays unit-only (injected clock): runtime-dependent acceptance expectations would be their own time bomb

### Next

- Merge PR #54, then slice 2: repoint `auto` (auto.go:51/:58) to `parse.Fields` + extract cmd-level `parseOptions` (third caller)
- Then slice 3 (`batch-auto` parse-once-per-row; T-40 becomes a boundary unit test) → slice 4 (`apply`, retires `canonicalDate`) → T-21 threading → T-42 monthly close
- T-43 (correct fixtures explicit year) is a 2-line quickie any session can absorb

### Gotchas

- `generate_code` `output_file` RELATIVE paths resolve against the LLM repo's REPO_ROOT even from this repo's session — the first file landed under `/mnt/i/workspaces/llm/expense-reporter/`; always pass absolute paths
- Fable (claude.ai/code) transcripts store NO assistant text blocks → `verdict-capture.py` can never see `[VERDICT]` blocks written here; append records directly to `calls.jsonl` in the hook schema (memory `feedback_verdict_transcript_gap`; llm T-109 lists all 5 findings incl. background-call template bypass and warm_model not covering large-context first calls — 120s default still times out, retry with timeout=300)
- `utils.ParseCurrency` NEVER parsed `1.234,56` (naive comma→dot swap) — the documented BR thousands format was unsupported everywhere until the boundary's normalizeThousands (both separators present → strip dots; dot-only keeps legacy decimal meaning)
- Existing `correct` acceptance tests send bare `15/04` against `15/04/2026` seeds — pass today via rung 4 (past → current year) but go RED in January 2027 (→ T-43)
