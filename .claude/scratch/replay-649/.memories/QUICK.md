# QUICK — 649-replay probe (T-23 gate + 5.R1 retrieval, session 52)

Replays the 649 real human-verified labels (`classifications.jsonl`) through the PRODUCTION
retrieval + classifier, leave-one-out. Answers the T-23 gate AND the 5.R1/5.R2 lever in one pass.
Faithful Go build-tagged harness (drives real code), NOT a Python reimplementation.

## Outcome
- **SHIP (bias-robust):** confidence DEAD as gate (52→56% flat); specificity `top_score` replaces it
  (monotone 52→87%); **best gate = AGREEMENT `spec==1.0` ∧ model==keyword-top1 → 95.0% subcat**; gate
  runs no-think (shape robust to think mode, 10× faster).
- **HOLD:** the 649 is a confidence-selected review SUBSET (378/725 expenses bypassed review, unlabeled)
  → absolute precision levels UNMEASURABLE → "no silent auto-insert" is held, not proven.
- **Retrieval:** miss 24.7%, all `no_keyword_match`, 160/160 have in-pool neighbors → 5.R1 ruled out,
  **5.R2 embeddings is the lever** (softened — NEXT = NN-retrieval precondition on the 160 misses).

## Files
- `FINDINGS.md` (retrieval half) · `FINDINGS-model.md` (gate half, incl. advisor reconciliation + think-on)
- Harness (in `internal/classifier/`, `//go:build replay`): `replay_retrieval_test.go` (Ollama-free),
  `replay_model_test.go` (Ollama, resumable; env: `REPLAY_LIMIT`/`REPLAY_MIN_TOPSCORE`/`REPLAY_THINK`)
- Analysis: `analyze.py` (retrieval), `analyze_model.py` (gate risk-coverage), `compare_think.py` (think-on)
- Raw: `retrieval.jsonl`, `model.jsonl` (649 no-think), `model-think.jsonl` (316 top-band think-on)

## Run
```
cd expense-reporter
go test -tags=replay -run TestReplayRetrieval649 ./internal/classifier/ -v            # seconds, no Ollama
go test -tags=replay -run TestReplayModel649 ./internal/classifier/ -v -timeout 60m   # ~16 min no-think
python3 ../.claude/scratch/replay-649/analyze_model.py
```

## Gotchas
- Go-test cwd = PACKAGE dir (`internal/classifier`); `REPLAY_*_OUT` relative paths resolve from there
  (default `../../../.claude/scratch/replay-649/...`). O_CREATE does NOT mkdir parents — a wrong path
  fails fast (that's how the first think-run died).
- LOO key MUST equal `MergeExamplePools` dedup key (`ToLower(TrimSpace(item))`); `self_match=0` is the tripwire.
- `classify` does NOT set FeedbackPath (training-only few-shot); `auto`/`batch-auto` DO — the model harness
  replicates the `auto` config (FeedbackPath + T-22 descriptions) since the gate lives there.
- Keyword-top1 is a pure-keyword proxy, NOT the model; join to model output for the real gate.
