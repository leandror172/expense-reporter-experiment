# test/ — Knowledge (Semantic Memory)

*Acceptance test harness accumulated decisions. Read on demand by agents.*

## Harness Design — Domain-Agnostic Engine (2026-03)
The `harness/` package contains zero expense-reporter knowledge. It provides:
- `Context` — per-scenario state bag (binary path, work dir, artifacts, stdout/stderr, exit code)
- `Scenario` — three-phase struct: Given (setup), When (action), Then (assertions)
- `Run(t, Scenario)` — wraps in `t.Run` for named subtests
Domain knowledge lives in `actions/` (how to invoke CLI commands) and `verify/`
(what to assert about output).
**Rationale:** The engine is intentionally extractable — it could test any CLI tool
that produces file artifacts. Keeping domain out of the harness makes it reusable.
**Implication:** Adding a new command to test means adding a new action function and
possibly new verify helpers — the harness itself doesn't change.

## Fixture Format (2026-03)
Each fixture is a directory with:
- `config.json` — command, model, threshold, assertion_type, extra_args, accuracy_floor, top_n
- `input.csv` — semicolon-delimited, `#` comments supported
- `expected-classified.csv` — optional, for soft accuracy comparison
- `expected-feedback.jsonl` / `expected-expenses_log.jsonl` — for JSONL log verification
**Key distinction:** classify/auto fixtures use `input.csv` as a scenario table (each row
becomes a separate test invocation). batch-auto fixtures pass `input.csv` directly to the binary.
**Rationale:** Different commands have different input models. Scenario tables let
classify/auto tests run multiple items in one fixture without batch machinery.
**Implication:** The fixture format is a contract — changing it requires updating both
the test code and all existing fixture directories.

## Soft vs Hard Assertions (2026-03)
- **Hard** (`assertion_type: "hard"`) — exact match required, test fails on mismatch
- **Soft** (`assertion_type: "soft"`) — calculates accuracy percentage, fails only below
  `accuracy_floor`. Writes JSON reports to `test/results/` for drift tracking.
**Rationale:** LLM classification is non-deterministic. Hard assertions on classifier
output make tests flaky. Soft assertions with a floor catch regressions without
requiring exact reproducibility.
**Implication:** Mechanics tests (installments, rollover) use hard assertions — they
test deterministic logic. Classification tests use soft assertions.

## Threshold 0.0 Strategy (2026-03)
Fixtures testing structural mechanics (installments, rollover) set `threshold: 0.0`.
This means every classified row is auto-inserted regardless of confidence.
**Rationale:** These tests verify installment expansion, rollover detection, and CSV
output format — not classification accuracy. A non-zero threshold would make them
dependent on the classifier's confidence, adding false-negative risk.
**Implication:** Only use threshold 0.0 for tests where the classification result
doesn't matter. Classification quality tests should use realistic thresholds.

## Canonical Test Items (2026-03)
- **"Uber Centro"** — the most reliable test item. Consistently returns Uber/Taxi
  subcategory across models and runs. Used as the baseline in auto and feedback tests.
- **"Diarista Letícia"** — reliable for Diarista subcategory. Used in batch tests.
**Rationale:** Empirically discovered that some items are nearly deterministic across
Ollama model versions, while others are sensitive to model changes.
**Implication:** Use canonical items for structural tests. Use diverse items only in
soft-assertion tests where accuracy drift is tracked.

## Composable Then Pattern (2026-03)
Then helpers return `[]func(*harness.Context)`, not single functions. Combined with
`slices.Concat` at the test site. Each helper is scoped to one concern (e.g.,
"classified output has correct columns" vs "accuracy meets floor").
**Rationale:** Monolithic assertion functions hide what's being tested and make it
hard to compose different assertion sets for different fixtures.
**Implication:** When adding a new assertion concern, create a new helper function
that returns `[]func(*Context)`. Never add assertions to existing helpers unless
they're truly part of the same concern.

## JSONL Verification Design (2026-03)
File-specific verifiers (not generic string-keyed):
- `verify.ClassificationsMatch(expectedPath)` — checks `classifications.jsonl`
- `verify.ExpenseLogMatches(expectedPath)` — checks `expenses_log.jsonl`
Expected files omit non-deterministic fields (`id`, `timestamp`). For classifier-dependent
tests, `subcategory`/`category` are also omitted from expected files.
**Rationale:** JSONL logs include auto-generated fields (hash IDs, timestamps) that
differ every run. Expected files contain only the deterministic contract.
**Implication:** When adding new fields to JSONL output, update the verifier's skip
list if the field is non-deterministic.
