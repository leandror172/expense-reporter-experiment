# Session Log — Expense Reporter

**Current Session:** 2026-07-03 — Session 47: T-22 type descriptions — no-think A/B, English adopted via sidecar (PR #42)
**Current Layer:** Classifier accuracy & calibration (post-T-22) → T-23 gate rethink → WS-D
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-03 - Session 47: T-22 type descriptions — no-think A/B, English adopted via sidecar (PR #42)

### Context

Resumed after the T-14 PR (and the stacked T-19 #40) merged to master. Picked T-22
(taxonomy type-descriptions A/B) as the next item, per the reading guide. Ran on branch
`feat/t22-type-descriptions`; work offloaded to sonnet subagents with Ollama-delegated codegen.

### What Was Done

- Implemented T-22: optional type-level `Description` on `taxonomy.ExpenseType`, rendered parenthetically on the classifier prompt's type header line (`writeTaxonomyTree`). Strictly additive — no enum/grammar/routing change.
- Ran a no-think A/B/C benchmark (300 items + 20 OOD each) via `run_t22.sh`: nodesc / English / Portuguese. Result: **English +6.0 pp TYPE accuracy** (74.3→80.3), +2.6 pp full-path; Portuguese weaker (dropped). Both languages lift type ~5–6 pp → real effect, not q3 noise. Calibration unchanged (HC-wrong ~91–95%); OOD 0/20 decline at no-think.
- Adopted English via a tracked, non-sensitive sidecar `config/type-descriptions.json`, merged at classify-time (`taxonomy.LoadTypeDescriptions`/`ApplyDescriptions`, applied in the classifier-scoped cmd `loadTaxonomyTree`). Gitignored `config/taxonomy.json` left untouched; `generate-workbook` unaffected.
- Verified end-to-end: 6 new taxonomy tests + prompt-render tests green; live `classify` works with the real sidecar; a malformed sidecar errors pre-Ollama (overlay proven live, not dead code).
- Docs: index.md, benchmark-report "T-22 Addendum", both packages' QUICK+KNOWLEDGE, durable memory `project_t22_type_descriptions`; gitignore hardening for personal-derived scratch `taxonomy-*.json`.
- Committed the 20-file explicit set on `feat/t22-type-descriptions`, opened **PR #42**; follow-up doc commit flags the runner's now-stale sidecar-toggle mechanism for the deferred think-on run.

### Decisions Made

- **English adopted; Portuguese dropped** — English won the language A/B on every metric.
- **Authoring home = tracked sidecar**, not injection into the gitignored `taxonomy.json` — so descriptions survive any future taxonomy regeneration; overlay is classifier-scoped so `generate` is untouched.
- **Type-level descriptions help TYPE accuracy (~6 pp, robust) but NOT calibration and NOT leaf disambiguation** — leaf-level descriptions are the future lever (T-27); calibration stays T-23's job.
- **Deferred the think-on confirmation run** (long pole, ~72 min) — captured as T-26; couples to T-23/T-24.

### Next

- **T-23 — calibration / gate rethink** (the real WS-D blocker). T-22 confirmed descriptions do NOT fix the auto-insert gate, so it still needs replacing before WS-D can lean on the model path.
- Merge PR #42.
- Later: T-26 (T-22 think-on confirmation), T-27 (leaf-level descriptions), T-28 (alternative classification strategies).

### Gotchas

- `run_t22.sh` argparse bug: `--extra-args "--think=false"` (two tokens) fails ("expected one argument" — argparse reads the dash-prefixed value as a flag); needs the `=` form `--extra-args="--think=false"`. Fixed.
- `go build ./...` does NOT emit the cmd binary — must `go build -o expense-reporter ./cmd/expense-reporter` before benchmarking or the run drives stale code.
- Personal-derived scratch `taxonomy-descen/descpt.json` are NOT covered by the `*.jsonl` ignore — added `taxonomy-*.json` to the local `.gitignore`. Always commit with explicit file lists, never `-A` (a `git diff --cached | grep ^+ | grep <personal-leaf>` check caught nothing, but the variants would leak under `-A`).
- Post-adoption, the deferred think-on run must toggle the SIDECAR (`config/type-descriptions.json` present=descen, absent=nodesc), NOT swap `taxonomy.json` — `run_t22.sh`'s mechanism is now stale (⚠ warning in its header).
