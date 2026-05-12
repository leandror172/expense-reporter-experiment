# Plan: Fix Default Workbook Path Resolution

**Status:** Ready to implement
**Scope:** Bug fix — remove `os.Executable()`-relative default from `GetWorkbookPath`; use config as fallback; fail clearly when unconfigured
**Follow-up (deferred):** Multi-workbook config keyed by year — tracked in tasks.md `5.D*`

---

## Context Files — Read on Session Start

| File | When | Why |
|------|------|-----|
| `BUG_REPORT_DEFAULT_WORKBOOK_PATH.md` | Immediately | Bug description, root cause, reproduction |
| `.claude/memory/MEMORY.md` | Immediately | Active feedback rules (TDD cadence, testify, advisor protocol) |
| `.claude/memory/feedback_tdd_commit_cadence.md` | Immediately | TDD-first ordering rule — governs step sequence |
| `.claude/memory/feedback_testify.md` | Immediately | Use testify (assert/require), not stdlib-only |
| `.claude/memory/feedback_advisor_protocol.md` | Immediately | When to call advisor during implementation |
| `expense-reporter/cmd/expense-reporter/cmd/root.go` | Before Step 1 | `GetWorkbookPath` — the function being replaced |
| `expense-reporter/internal/config/config.go` | Before Step 1 | `Config` struct, `ClassificationsFilePath()` pattern to mirror |
| `expense-reporter/config/config.json` | Before Step 1 | `workbook_path` field value |
| `expense-reporter/cmd/expense-reporter/main_test.go` | Before Step 1 | `TestGetWorkbookPath` — test cases that need updating (write tests first) |
| `expense-reporter/cmd/expense-reporter/cmd/add.go` (line 71) | Before Step 3 | Verify error surfaced, not swallowed |
| `expense-reporter/cmd/expense-reporter/cmd/auto.go` (line 144) | Before Step 3 | Same |
| `expense-reporter/cmd/expense-reporter/cmd/batch.go` (line 64) | Before Step 3 | Same |
| `expense-reporter/cmd/expense-reporter/cmd/batch_auto.go` (lines 102, 218) | Before Step 3 | Same |

---

## Implementation Steps (TDD order — tests before implementation)

### Step 1 — Write failing tests FIRST

**`TestGetWorkbookPath` (`main_test.go`)**

Update existing cases:
- `empty environment variable`: flip `wantErr: false` → `wantErr: true`

**`TestConfig_WorkbookFilePath` (new file: `internal/config/config_test.go`)**

Use testify (`assert`/`require`). Mirror the pattern of any existing `ClassificationsFilePath` tests in that package.
Three cases:
- empty `WorkbookPath` → returns `""`
- absolute path → returned as-is
- relative path → resolved relative to `os.Executable()` dir (assert it is absolute and ends with the relative segment)

Note: `SetupBinaryConfig` is `//go:build acceptance` — do NOT import it. Write the temp config inline with `t.Cleanup` in `config_test.go`.

Commit after both test files compile (even while failing).

**→ Call `advisor()` before proceeding to Step 2.**

---

### Step 2 — Add `WorkbookFilePath()` to `Config` (`internal/config/config.go`)

Mirror the `ClassificationsFilePath()` pattern exactly:
- If `WorkbookPath` is empty → return `""`
- If absolute → return as-is
- If relative → resolve relative to `os.Executable()` directory

```go
func (c *Config) WorkbookFilePath() string {
    if c.WorkbookPath == "" {
        return ""
    }
    if filepath.IsAbs(c.WorkbookPath) {
        return c.WorkbookPath
    }
    exe, err := os.Executable()
    if err != nil {
        return c.WorkbookPath
    }
    return filepath.Join(filepath.Dir(exe), c.WorkbookPath)
}
```

Commit once `TestConfig_WorkbookFilePath` goes green.

### Step 3 — Rewrite `GetWorkbookPath` (`cmd/root.go`)

New resolution order:
1. `--workbook` flag
2. `EXPENSE_WORKBOOK_PATH` env var
3. `config.Load()` → `cfg.WorkbookFilePath()` — only if non-empty
4. Return typed error

Rules:
- `config.Load()` error (malformed JSON) → surface it, do NOT silently fall back
- `config.Load()` "file not found" (IsNotExist) → continue to step 4 (already handled by `Load()` returning `&Config{}`, nil)
- Remove: the `os.Executable()` relative-path block
- Remove: the Windows hardcoded fallback (`Z:\Meu Drive\...`)

Error message when nothing is configured:
```
workbook path not configured: use --workbook <path> or set EXPENSE_WORKBOOK_PATH
```

Commit once `TestGetWorkbookPath` goes green.

### Step 4 — Verify call sites (read-only check)

Confirm `add.go`, `auto.go`, `batch.go`, `batch_auto.go` all handle the error return from `GetWorkbookPath()` by surfacing it (not swallowing). No changes expected — just confirm.

---

## Acceptance Tests

Acceptance tests pass `--workbook` explicitly via `ctx.WorkbookPath` (see `test/actions/commands.go`).
**No acceptance test changes needed.**

---

## Verification

```bash
cd expense-reporter && go vet ./...
cd expense-reporter && go test ./...
cd expense-reporter && go build ./...
```

---

## Session-End Housekeeping

- `tasks.md`: add this bug under "Deferred Technical Debt" if not already there; mark complete when done
- `session-log.md`: append session entry
- `session-context.md`: update active state

---

## Out of Scope (Deferred)

- Multi-workbook config keyed by year (`"workbooks": {"2025": "...", "2026": "..."}`)
- Year extraction from expense date to drive workbook selection
- Tracked in tasks.md under deferred items
