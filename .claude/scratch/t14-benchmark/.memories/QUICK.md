# QUICK — t14-benchmark harness

Must-know before running or interpreting anything in this dir.

## Running
- **Build the binary first, explicitly:** `go build ./...` does NOT emit the cmd binary.
  Use `go build -o expense-reporter ./cmd/expense-reporter` (from `expense-reporter/`), or
  just run `run_t26.sh` which builds it for you. Otherwise you benchmark stale code.
- **Current runner: `run_t26.sh <think|nothink>`** (default `think`). Toggles the
  **sidecar** `config/type-descriptions.json` (removed = `nodesc`, present = `descen`).
  `run_t22.sh` is STALE (swaps `taxonomy.json`, a pre-adoption mechanism) — don't use it.
- **Resumable:** `run_benchmark.py` appends+flushes per item and skips ids already in its
  results file — kill and re-run to resume. BUT resume keys on the results *filename*, so
  reusing a prefix from a prior run silently "skips everything." New runs → new prefix
  (`run_t26.sh` encodes the mode: `results-<mode>-<cond>-<model>.jsonl`).
- Preflight Ollama + model before a long run: `curl -s localhost:11434/api/tags`.

## Interpreting (READ THIS BEFORE WRITING A FINDING)
- **A single 300-item run resolves ~±5pp at best.** Subgroup splits (leaky n=184, novel
  n=116, OOD n=20) do NOT clear noise on their own.
- **Run `sigtest_t26.py` before calling any delta real.** Paired McNemar (accuracy) +
  Fisher (sentinel), zero model calls, reuses `score.py` predicates.
- **A subgroup effect that flips SIGN between two runs = noise, not a mechanism.** (T-26
  learned this the hard way — see KNOWLEDGE.)

See `.memories/KNOWLEDGE.md` for the T-26 findings and what survived significance testing.
