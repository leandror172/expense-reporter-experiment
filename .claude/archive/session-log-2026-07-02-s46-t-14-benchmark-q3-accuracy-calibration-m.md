## 2026-07-02 - Session 46: T-14 benchmark — q3 accuracy/calibration measured, --think flag, q35 disqualified

### Context

Started from the session-45 handoff: PR #40 (T-19 sentinel) open, T-14 benchmark next. User is reviewing/merging PR #40 themselves; this session executed T-14 on branch `feat/t14-think-flag` (stacked on the T-19 branch).

### What Was Done

- Built the T-14 benchmark harness (`.claude/scratch/t14-benchmark/`): stratified sample builder (300 items over 80/112 leaves from the 1713 taxonomy-exact corpus entries, leakage-flagged), resumable runner driving the real `classify --json` path, scorer (accuracy/leakage split/calibration/OOD), plus a 20-item synthetic out-of-domain probe set.
- Ran q3 full benchmark: 63.0% full-path (leaky 71.7% / clean 49.1%), type 77.0%, mean 14.4 s/item. **Calibration broken: 91% of wrong answers at confidence ≥0.85** — the auto-insert gate filters almost nothing. OOD: T-19 sentinel fires only 2/20.
- feat(classifier): `--think` flag (`Config.NoThink` → `"think":false`, omitted by default) on classify/auto/batch-auto; TDD unit test; commit `e449bdb`.
- Ran q3 `--think=false` full benchmark: 59.7% at **1.5 s/item (10×)**, grammar intact, zero failures — but OOD sentinel drops to 0/20 with absurd confident picks.
- q35 disqualified: thinking-on runs 70–240 s/item (unbounded thinking); `think:false` **silently drops the `format` grammar** on Ollama 0.17.5 (verified at API level); `/no_think` soft switch ignored by qwen3.5. Created `my-classifier-q35-nothink` persona on the false /no_think premise — inert, to delete (T-25).
- qcoder subset skipped as moot (doesn't fit VRAM; nothing suggests a 30B fixes calibration).
- docs(t14): report `.claude/t14-benchmark-report.md`, harness committed (data JSONLs gitignored — real expense descriptions), classifier QUICK/KNOWLEDGE updated, index.md entries; commit `c009dfe`.

### Decisions Made

- **WS-D gate NOT passed** — blocker is calibration, not accuracy: the 0.85 threshold passes ~91% of errors. Do not retire the bare-name fallback until T-23 resolves the gate.
- `--think` default stays **true** (accuracy +3.3 pp and the only nonzero sentinel behavior); the 10× fast lane is opt-in per command. Default revisit = T-24, after T-22.
- T-14 closed with q35/qcoder columns resolved by disqualification rather than measurement; leakage reported both ways rather than held out (production skews recurring).

### Next

- T-22 descriptions A/B — harness ready; ~8 min/condition at no-think speed (same 300-item sample; measure full-path, clean-subset, OOD decline rate).
- T-23 calibration/gate rethink before WS-D.
- Open PR for `feat/t14-think-flag` (stacked on `feat/t19-sentinel-decline` / PR #40, which the user is merging).

### Gotchas

- qwen3.5 + `think:false` silently drops the `format` grammar (Ollama 0.17.5) — structured output returns free-form JSON with out-of-enum labels; grammar-enforcement claims are qwen3-only for no-think.
- A client-side timeout does NOT stop Ollama's server-side generation; abandoned grammar+thinking requests queue-starve every later call (GPU pegged) until `systemctl restart ollama`.
- ollama-bridge `generate_code output_file`/`patch_file` relative paths resolve against the LLM repo (`/mnt/i/workspaces/llm`), not this repo — pass absolute paths.
