# Session Log — Expense Reporter

**Current Session:** 2026-07-20 — Session 62: T-39 suite split + T-24 think-off default (PR #52); vision reframe → parse-boundary plan (T-41)
**Current Layer:** Layer 5 — Expense Classifier (T-39/T-24 shipped, PR #52 open; next: parse boundary T-41 → monthly close T-42)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-20 - Session 62: T-39 suite split + T-24 think-off default (PR #52); vision reframe → parse-boundary plan (T-41)

### Context

Session started with PRs #50/#51 (T-35/T-36) freshly merged to master. Backlog discussion picked Track A (suite health); mid-session the user redirected to the grand vision, which reframed the roadmap and produced the parse-boundary plan. All code work on branch `feat/t39-suite-split-t24-think-default` (PR #52) — master has no session-62 commits.

### What Was Done

- T-39: acceptance suite split — `testing.Short()` skip added to `extern.RequireOllama` (per-test seam, not per-file); `run-acceptance.sh` now two modes: deterministic default (`-short`, ~8s, 30 pass/26 skip, no Ollama pre-flight, 600s) and `-full` (whole suite, 3600s ceiling)
- T-24: `--think=false` made the default on `classify`/`auto`/`batch-auto`; full suite verified 56/56 green in 128s under the new default (was 840–1989s think-on, ~7–15× faster)
- PR #52 opened with the code + runner docs (READMEs, `ref:acceptance-run` block, CLAUDE.md, index.md)
- Vision re-read (`docs/expense-classifier-vision.md` + `.claude/t23-strategic-implications.md`) → project reframed as PRE-FIRST-USE; organizing milestone = first real monthly close on 2026 data (T-42)
- Parse-boundary plan drafted + indexed (`.claude/plans/parse-boundary.md`): structured input, parse-once; the single-string input format identified as the common ancestor of the date/data debt (T-35 class, T-18, T-21, six parsers, GenerateID format-sensitivity)
- Memories/README staleness sweep (committed to PR #52): dead-`date_year` false documentation corrected in app KNOWLEDGE + README; test/classifier memories updated for the suite split + think default; root QUICK compressed under its line budget with Milestone Log entries for sessions 60–62

### Decisions Made

- T-39 fixed by SPLIT (not a ceiling raise); script default = deterministic-fast, `-full` deliberate
- T-24: think-off uniform across all three commands — gate band −2.3pp WITH think, 5.R2 miss path +2.5pp at 5.5× latency, sentinel loss covered by the T-32 agreement gate (T-16 defaults-parity lesson applied)
- Chat-era input architecture (user): one parse tool converts input to structured fields; downstream tools take fields, not strings — parse boundary built BEFORE T-21, which becomes its first consumer
- Year precedence locked: explicit year in string > `--year` arg > config `date_year` > most-recent-non-future occurrence (grace window ~days TBD in design session)
- T-21 UX locked: 1 row + ×N badge in the review UI; `apply` expands to N log entries
- `--year` flags to be added uniformly to `auto`/`batch-auto`/`add`
- T-38 RESOLVED by decision: plain `batch` is a pre-pivot artifact (writes the workbook directly; no vision path uses it) — delete `utils.ParseDate` with its caller under WS-E, do NOT repair. Widens WS-E's previously-narrow scope.

### Next

- Merge PR #52 (T-39 + T-24 + doc sync)
- T-41 parse-boundary design session — plan §10 open questions: package home/struct shape, migration order (candidate: add/correct → auto → batch-auto → apply), currency-normalization ownership, MCP `parse_expense` timing
- Then T-21 threading (first consumer slice), then T-42 (first real monthly close on 2026 data)
- Standing queue: T-25, T-27, T-28, T-29, Apoia-se rename, promote all-years log

### Gotchas

- Piping a background full-suite run through `tail -80` truncated the captured log — the roster was unverifiable from it. Capture full output to a file; the re-run was free because `go test` served cached results.
- `go test` result caching replays the recorded output of an unchanged run (`ok ... (cached)`) — legitimate for roster verification, but TIMING numbers must come from uncached runs.
- GPU contention invalidates suite timing — the first `-full` run was stopped because the GPU was in use; verify the GPU is idle before any benchmark/timing run.
- Latency claims rot fast: the think-off default made "q3 ≈12s/classify, suite needs 30m" stale in several memories/READMEs within one session — swept, but future latency claims should name their think mode.
