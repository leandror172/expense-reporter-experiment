# Bug: default workbook path resolves against build cache under `go run`

**Date:** 2026-04-24
**Status:** Reported (latent — surfaced while investigating batch-auto output-loss bug)
**Severity:** Low (affects UX, not data) — but makes error messages very confusing

## Issue
`GetWorkbookPath` (cmd/expense-reporter/cmd/root.go:52-74) computes its default path
relative to `os.Executable()`. Under `go run ./cmd/expense-reporter`, the executable
lives in Go's build cache (`~/.cache/go-build/<hash>/`), so the resolved default is
nonsensical — e.g. `/home/leandror/.cache/go-build/68/Planilha_..._2025.xlsx`.

Users see the cache path in "workbook not found" errors with no clue where it came from.

## Reproduction
```bash
cd expense-reporter
unset EXPENSE_WORKBOOK_PATH
go run ./cmd/expense-reporter batch-auto some.csv --data-dir ./data/classification
```

## Error surface
```
Error: workbook not found: /home/leandror/.cache/go-build/68/Planilha_..._2025.xlsx
```

The path includes a content-addressed build-cache hash, so it changes across builds —
doubly confusing.

## Root cause
`root.go:64-73`:
```go
execPath, err := os.Executable()
...
execDir := filepath.Dir(execPath)
defaultPath := filepath.Join(execDir, "..", "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx")
```

`os.Executable()` returns the path of the currently running binary. For `go run`, that's
a throwaway file in the build cache, not the project root. The `../` does nothing useful
from there.

There's also a Windows-style hardcoded fallback
(`Z:\Meu Drive\controle\...`) that is dead code on WSL/Linux.

## Impact
- Confusing errors under `go run` workflows (common for dev and ad-hoc invocations)
- First-time users without `EXPENSE_WORKBOOK_PATH` set get a path that never existed
- Hardcoded Windows fallback suggests the default resolution has never really worked
  reliably across environments

## Suggested fixes (pick one)

1. **Remove the executable-relative default entirely.** If neither `--workbook` nor
   `EXPENSE_WORKBOOK_PATH` is set, fail with a clear message:
   > Workbook path not configured. Use `--workbook <path>` or set `EXPENSE_WORKBOOK_PATH`.

   Cleanest. No magic. Matches Unix CLI conventions.

2. **Resolve relative to CWD, not executable.** e.g., look for
   `./Planilha_...2025.xlsx` in the working directory. Predictable, but still magical
   and still fails silently if the user is in the wrong directory.

3. **Read default from `config/config.json`.** The `config.Config` loader already
   exists (see `config.Load`). Add a `default_workbook_path` field; fail clearly if
   unset. Most flexible, but adds a config surface.

**Recommendation:** Option 1. Drops dead code (the Windows fallback) and eliminates
an entire class of confusing error messages. Users who want a default can set
`EXPENSE_WORKBOOK_PATH` in their shell profile.

## Related
- Primary bug: `BUG_REPORT.md` (batch-auto loses classification output when workbook
  insertion fails). The path-resolution defect made the primary bug more likely to
  surface — a user who never set `EXPENSE_WORKBOOK_PATH` will hit the "workbook not
  found" error every run under `go run`.

## Out-of-scope notes
- `GetWorkbookPath` is called from `add`, `auto`, `batch`, `batch-auto` (`cmd/*.go`).
  Any fix applies uniformly — they all share this entry point.
- Does not affect the production binary path (`./expense-reporter`) if the binary is
  placed next to the workbook. But that's not how most invocations happen.
