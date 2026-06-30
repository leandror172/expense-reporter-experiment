## 2026-06-29 - Session 42: PR #36 (T-13) review — model revert to q3, type-in-JSON, acceptance repair

### Context

Started as `/code-review` of the open PR #36 (T-13 full-path classifier). Two findings cascaded into a model-choice correction and a full acceptance-suite repair — the review's precondition check exposed that PR #36 had left 13 acceptance tests red, hidden behind the `//go:build acceptance` tag.

### What Was Done

- Reviewed PR #36; found two issues: predicted `type` dropped at the JSON boundary (classify/auto/add `--json`), and the `classify` default model diverged from `auto`/`batch-auto`.
- **T-15 Slice 1**: surfaced the already-resolved `type` in `CandidateOutput` + `AddOutput` (`type,omitempty`) with two deterministic tests; Slice 2 (`--predicted-type` flag) deferred.
- **Reverted the classifier default `qcoder`→`my-classifier-q3`** across all four call sites after proving enum validity is GRAMMAR-enforced (Ollama `format`→GBNF), not model-dependent; q3 validated end-to-end on the real 112-path taxonomy (Uber→Uber/Taxi 100%).
- **Repaired 13 build-tag-hidden acceptance regressions**: T-17 (11 — T-13 made `taxonomy_path` mandatory but many `Given`s never set it) + T-18 (2 — pre-existing `correct` seed-id bug from T-11 date normalization). All verified green via GROUPED runs.
- Updated per-package memories (classifier KNOWLEDGE, test QUICK, expense-reporter QUICK) + index, and wrote `.claude/session42-postmortem.md`.

### Decisions Made

- **Default classifier model = `my-classifier-q3`** (fits the 12 GB GPU). qcoder (qwen3-coder:30b, 20.7 GB) was reverted — enum validity is grammar-enforced so a 30B model bought zero validity, only CPU-offload latency + load-time 500s. Accuracy+speed across q3/q35/qcoder → T-14.
- **Verify acceptance in GROUPS, not full-suite-in-one-shot** — q3 is slow (~12 s/classify); the full suite exceeds the 600 s default `go test` timeout. Grouped runs give incremental, durable confirmation (recorded in test QUICK.md).
- **T-16 resolved by unifying on q3** — the opposite of the original "move classify to qcoder" proposal, once the premise collapsed.

### Next

- **WS-B slice 3 (`batch-auto` → log-append)** remains the primary next work — unchanged by this session.
- **T-14**: benchmark accuracy AND speed across q3/q35/qcoder on real labeled data; pick smallest-that-fits-and-is-accurate.
- **T-16 doc cleanup**: CLAUDE.md still calls qcoder the "primary" tier "required for 5.7+" — correct it. **T-19**: add a "none-of-these" escape (sentinel path) so the enum can decline novel inputs.

### Gotchas

- The `//go:build acceptance` tag hides regressions from `go test ./...` — after any change to a mandatory field/config contract, run `-tags=acceptance` explicitly.
- `data/classification` exists at the **repo root** (`expense-reporter/../data/classification`), NOT under `expense-reporter/` — an early `ls` from the wrong relative path made me wrongly think it was absent.
- Don't assert blockers from assumptions — twice I claimed a false blocker (Ollama tests "qcoder-gated"; data dir "absent"), both disproved by running the tests / reading the fixture config.
