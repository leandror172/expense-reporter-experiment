# PR #22 Code Review — `review` command

**Branch:** `worktree-feat+review-command` → `master`
**Reviewer:** Claude Sonnet 4.6 (session 23, 2026-05-18)
**Advisor:** called and incorporated.
**Scope:** Autonomous review, no changes made. Findings only.

---

## Overview

Implements `expense-reporter review <classified.csv>` end-to-end: CSV reader, taxonomy
builder, HTML renderer, cobra command wiring, unit tests (17), acceptance test, and
embedded HTML template. All tests green. Smoke confirmed (349 rows, 23 need review).

**Blocking issues: none. PR is mergeable.**

Recommended before merge: finding #1 (one comment line).
Post-merge backlog: #5, #6, #7, and items A–C below.

---

## Defect found and fixed during review

**`taxonomy.go` — alphabetical tiebreak missing for unknown sheets** (fixed in commit `6911bdd`, pushed to PR branch after opening).

`sort.Slice` with only rank comparison is unstable for equal-rank (all-unknown) sheets — order depends on map iteration, which is random in Go. Plan §5.2 says "any others alphabetically" but the original comparator had no secondary key.

Found during the q25c14 benchmark comparison: the model generated a test with two unknown sheets in one case, which would have been flaky. Fix: added `sheetNames[i] < sheetNames[j]` tiebreak.

**Gap that remains:** `taxonomy_test.go` has no test case with two unknown sheets in the same taxonomy. The bug was caught by external pressure (benchmark), not by existing tests. Should be added.

---

## Findings

### 1. `queue.go` — `GenerateID` date constraint not documented [MEDIUM — fix before merge]

Plan §5.1 says: `date` is `DD/MM`, no year — acceptable, **document this**.

The code has no comment about this invariant. The `apply` command (RUI-3) will need to
know that IDs from `review` output use year-free dates and cannot be correlated against
entries that used full `YYYY-MM-DD` dates. This is a future footgun.

**Fix:** add one comment above the `GenerateID` call:
```go
// ID uses DD/MM date (no year) — stable within a review-to-apply cycle only
```

---

### 2. `types.go` — `RawValue` field not in original plan spec [LOW]

Plan §4 `QueueEntry` had no `RawValue`. It was added (correctly) per O1 decision to
carry installment notation (`250,00/2`). The JSON output now includes `"rawValue"` —
the hand-maintained HTML template must handle this field. This is a contract the next
template drop needs to be aware of.

---

### 3. `queue.go` — nil vs empty slice for header-only input [LOW]

When the CSV has only a header row, `entries` is never appended to and remains `nil`.
`return entries, nil` returns a nil slice. The test "header only returns empty slice"
asserts `Len(t, entries, 0)` — which passes for both nil and empty, masking the
distinction. The cobra command guards this correctly (`len(queue) == 0 → error`), but
the function's contract is ambiguous for future callers.

**Suggestion:** `return []QueueEntry{}, nil` at the end, or document nil explicitly.

---

### 4. PR includes `.claude/tasks.md` tracking file [SCOPE — trivial]

Feature PRs should contain only code/test/doc changes. The tasks.md change was a
session tracking update that belongs as a direct commit to `master`, not in a feature
branch. Already committed to master separately; the diff in the PR is vestigial but
misleading.

---

### 5. Test gap — `confidence` parse error not tested [LOW]

Plan §5.1 explicitly calls out `confidence` parsing. `queue_test.go` covers bad
`auto_inserted` but not malformed `confidence` (e.g., `"abc"`). The error path exists
in `queue.go`; it just has no test.

---

### 6. Acceptance test coupled to production workbook [ARCHITECTURE — LOW]

`TestReview_ProducesHTMLWithQueueAndTaxonomy` uses `Planilha_Normalized_Final.xlsx`
via `EXPENSE_WORKBOOK_PATH`. No synthetic fixture workbook exists. The test silently
skips (not fails) without the env var, hiding regressions in CI-like environments.
A minimal synthetic `.xlsx` with the "Referência de Categorias" sheet would make it
fully hermetic.

---

### A. `os.WriteFile` silently clobbers existing output [FUTURE TASK]

If the user has a partially-reviewed `review.html` open or saved, it gets wiped without
warning. No `--force` flag, no stat-check, no confirmation. Should be documented in
the command's `Long` help at minimum; a `--force` flag or existence check is a future
quality-of-life improvement.

---

### B. No stdout (`-`) output option [FUTURE TASK]

Always writes to a file. For piping, shell redirection, or hermetic testing without
filesystem state, `-` as output path (stdout) is conventional in Unix CLI tools.
Not a blocker but useful for scripting.

---

### C. `taxonomy_test.go` — no multi-unknown-sheet test case [COVERAGE GAP]

See "Defect found and fixed during review" above. The fix is in, but the test coverage
that would catch a regression is missing. Add a case: two mappings with unknown sheet
names, assert alphabetical ordering.

---

## Dropped from initial findings (advisor review)

The following were initially flagged but dropped as non-issues:

- **`template`/`html` variable names shadowing built-in packages** — Go has no global
  package namespace; these only collide if the package is imported, and it isn't.
- **Mixed `slices.Index` + `sort.Slice`** — both are stdlib; mixed use during stdlib
  evolution is not a style violation.
- **`TemplateHTML` exported from `internal/`** — required: `cmd` package imports
  `review` and must pass the template to `Render`. Unexported + getter is identical in
  practice and introduces churn for no gain.

---

## Summary Table

| # | File | Severity | Category |
|---|------|----------|----------|
| Fix in PR | `taxonomy.go` | Bug (fixed) | Determinism — tiebreak added in 6911bdd |
| 1 | `queue.go` | Medium | Spec miss — GenerateID date invariant undocumented |
| 2 | `types.go` | Low | RawValue not in plan spec — template contract |
| 3 | `queue.go` | Low | Nil vs empty slice contract |
| 4 | `.claude/tasks.md` | Trivial | PR scope — tracking file in feature diff |
| 5 | `queue_test.go` | Low | Coverage — confidence parse error untested |
| 6 | acceptance test | Low | Architecture — coupled to production workbook |
| A | `cmd/review.go` | Future | UX — silent clobber of existing output file |
| B | `cmd/review.go` | Future | UX — no stdout (`-`) output option |
| C | `taxonomy_test.go` | Low | Coverage — multi-unknown-sheet case missing |
