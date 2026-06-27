# Session Log — Expense Reporter

**Current Session:** 2026-06-26 — Session 41: T-13 implemented — classifier predicts full (type,category,subcategory) path
**Current Layer:** Retire-insertion migration — WS-B (commands → log-append) + T-13
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-26 - Session 41: T-13 implemented — classifier predicts full (type,category,subcategory) path

### Context

Resumed mid-pivot (retire insertion / keep generation). Task: implement T-13 (designed session 40). Mostly-unattended session; warm Ollama before each call (another process was updating personas).

### What Was Done

- Implemented **T-13** end-to-end: the classifier now predicts the full `(type,category,subcategory)` path against `config/taxonomy.json` via an atomic 112-path enum, instead of free-text + post-hoc type lookup. 3 commits on `feat/t13-classifier-full-path`; **PR #36** → master.
- `internal/taxonomy` (new `path.go`): `BuildPathMap`/`PathEnum`, `PathMap.Split` (reverse-map lookup — never parses `/`), `PathFor`, `ResolveLeaf`, `TypesForLeaf`, `CategoryForLeaf`; tested via a committed synthetic fixture, sanity-checked against the real 112-leaf tree.
- `internal/classifier`: `Result.Type` added; `Taxonomy`/`LoadTaxonomy` deleted; `Classify` takes `[]taxonomy.ExpenseType`; 3-level tree prompt; path-enum structured-output schema; off-enum paths dropped; few-shot example paths resolved via `ResolveLeaf`+`PathFor` (dropped if unresolvable); `Example.TypeHint` from training `source`.
- Commands: `auto`/`batch-auto` write `Result.Type` directly (`resolveExpenseType`/`loadTypeIndex` + 2 obsolete tests deleted); `add` uses a `--type`/prompt/non-interactive-error hybrid (made I/O-injectable, fully unit-tested); `correct` uses `CategoryForLeaf`; default model `my-classifier-q3`→`my-classifier-qcoder`.
- Report `.claude/t13-implementation-report.md`; updated taxonomy + classifier QUICK/KNOWLEDGE memories. `go build`/`vet`/`test ./...` green; deterministic type-routing acceptance passes.

### Decisions Made

- `SplitPath` is a **method on `PathMap` (reverse lookup), not a free function** — category and subcategory names contain `/` (`Uber/Taxi`, `Alimentação / Limpeza`), so parsing is unsafe (advisor-driven deviation from plan §4.2).
- Few-shot example paths are built via `ResolveLeaf`+`PathFor`, **not** the training file's category string — building from the training category would re-introduce the exact category-disagreement bug T-13 closes.
- The `§4.6` "type-less==0 on the model path" check is **not** a CI assertion: the only model-path acceptance step is Ollama-gated and deliberately non-deterministic, and enum-honoring is model-dependent. It belongs to the **WS-D real-data gate**; the deterministic apply→generate-workbook cycle already proves full-path routing.

### Next

- **WS-B slice 3** — convert `batch-auto` to log-append only. Its `Classify` call already emits the predicted type (T-13); drop the workbook-insert branch (`workflow.InsertBatchExpensesFromClassified`) + `rollover.csv`, keep classified/review CSV → review → apply.
- **WS-B slice 4** — `apply`: delete the workbook-write half, keep the log-write half.
- Then **WS-D** — retire the bare-name fallback in `taxonomy.scanEntries`, gated on the real-data type-less count reaching ~0 (T-13 is the precondition). A final `advisor()` review of the T-13 PR is recommended (skipped this session per CLAUDE.md rule #5 — unattended).

### Gotchas

- `my-go-qcoder` was **unavailable all session** — `warm_model` returned HTTP 500 ×2 (17.3 GB model, VRAM contention / persona-update). Fell back to `my-go-q25c14`; the `classifier.go` rewrite escalated to direct authoring after two 300 s timeouts on the 14b (per the overlay's interdependent-multi-site rule).
- ollama-bridge `output_file` resolves against the **LLM repo's** REPO_ROOT, not this repo — `path.go` landed in `/mnt/i/workspaces/llm/...` first. Use **absolute** output paths.
- This machine's stdin **is a TTY**, so `add`'s interactive branch fired in tests — made `resolveFullPath(interactive, in io.Reader)` injectable so all branches test deterministically.
- `data/classification/` is **empty** on this machine (training data + feature dict gitignored & absent) → the model path can't run locally; the few-shot `source`→type hint is best-effort (a stale suffix just drops the example). `config/taxonomy.json` `IRFF→IRRF` fix confirmed present (gitignored/local-only).
