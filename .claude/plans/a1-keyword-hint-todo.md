# A1 (keyword hint in review) + T-65 (supersede-by-append) — working todo

Branch: `feat/a1-keyword-hint` (off master @ 9f981f4). Session 72.
Decisions: `.claude/a1-keyword-hint-precheck.md` (A1 = GO),
`.claude/b1-gate-precision-measurement.md` (B1 dropped, T-65 promoted).

**Conventions in force:** TDD red→green, commit after each green pass; local model first for
any new file/function >~5 lines (`[ref:local-model-conventions]`); acceptance tests first;
characterize-before-extract for untested code; testify for new unit tests; helper names
narrate domain steps, ~15-line bodies.

## Scope decisions already locked

- **One** new CSV column, `keyword_hint` (header 8 → 9). NO `gate_passed` — B1 is dropped, so
  `auto_inserted` still means "the gate passed" and a second column would have no consumer.
- The hint is **advisory** — displayed beside the model's answer, never auto-applied
  (71.4% precision would inject errors on ~3 of 14 rows).
- The hint **must not** round-trip through `exportReviewed()` into `reviewed.json`. Input-only;
  a value crossing the unguarded JS hop would be a fresh T-61.
- **Strict predicate only** (unambiguous, specificity 1.00). Not because a middle setting is
  impossible — `MatchStrength` can structurally return unambiguous-at-0.8 — but because
  `review` loads no keyword index and so has no consumer for the extra signal.

## A1 — ordered

- [ ] **A1.1** `classifier.KeywordHint(signal, modelSubcategory) string` in `decision.go`,
      beside `IsAutoInsertable`. Unit test FIRST (RED), table-driven over the four clauses:
      no match / ambiguous / below 1.00 / agrees-with-model → `""`; unambiguous 1.00 and
      disagrees → the keyword's subcategory.
- [ ] **A1.2** Point `replay_keyword_hint_test.go` at the production `KeywordHint` and re-run.
      **Expect 14 fired / 10 right / recall 10 of 25 unchanged** — a free equivalence check
      that the shipped predicate is the measured one. A different number means one of them moved.
- [ ] **A1.3** `classifiedRow.KeywordHint` + both writers emit the 9th column. Unit test first.
      Mutation: drop the field from the writer → must go red.
- [ ] **A1.4** `review.ReadQueue` field count 8 → 9, read index 8 into `QueueEntry.KeywordHint`.
      Corruption tolerance stays narrow (wrong field count still hard-errors).
- [ ] **A1.5** Extend `classified_csv_seam_test.go` — real writer bytes → real `ReadQueue`,
      hint survives. **The test that would have caught T-54.** Mutation-verify separately from
      the existing assertions.
- [ ] **A1.6** Update the 4 fixtures carrying the header:
      `batch-auto-basic/expected-classified.csv`, `review-basic/input.csv`,
      `review-malformed-rows/input.csv`, `type-routing-cycle/review-input.csv`.
      `expect/accuracy.go` indexes col 6 and is unaffected — confirm, don't assume.
- [ ] **A1.7** Template `internal/review/template/review.html` (edit ONLY this file, never a
      rendered `review*.html`): show the hint beside the model's suggestion when present.
- [ ] **A1.8** Acceptance — deterministic: extend the Ollama-free failed-rows scenario to pin
      the 9-column contract and that `review` still consumes it. **Every row there is unparsed,
      so the hint is empty throughout — this pins the CONTRACT, never the predicate.** Say so in
      the test name/comment so it cannot be mistaken for behavioral coverage.
- [ ] **A1.9** Acceptance — Ollama-gated (`-full`): the column round-trips to the review page.
      Asserts round-trip only, NOT non-emptiness: if the model happens to agree with the
      keyword the predicate correctly does not fire, and a non-emptiness assertion would fail
      spuriously.
- [ ] **A1.10** Verify: `go build`, `go vet`, `go vet -tags=replay`, `go test ./...`,
      `./run-acceptance.sh`, then `-full` (check `/api/ps` for a foreign resident model first —
      one turned a ~110s run into a 3602s timeout).
- [ ] **A1.11** Memories + index: `internal/review/.memories/QUICK.md` (contract now 9 fields),
      `internal/classifier/.memories/QUICK.md`, `test/.memories/KNOWLEDGE.md` if a trap was
      found; `.claude/index.md` rows for anything new.
- [ ] **A1.12** PR against master (no rebase fuss — the user manages review order).

## T-65 — queued, after A1

- [ ] **T-65.1** Characterize FIRST: `taxonomy/loader.go` `scanEntries` folds the log for
      `generate-workbook`. Check its existing coverage; if the fold is untested, write
      characterization tests GREEN against unmodified code before touching it.
- [ ] **T-65.2** Design the supersede record: a replacement keyed by `GenerateID` that the fold
      honours (last-writer-wins per id). Must preserve append-only — `close-cycle.sh`'s
      snapshot-as-complete-undo and the two-log join both depend on it.
- [ ] **T-65.3** Decide the deliberate-exception question: superseding changes what the workbook
      shows for an id whose `classifications.jsonl` history already records the correction. Pin
      which file is authoritative for which consumer, the way s70 pinned the stale-id exception.
- [ ] **T-65.4** Wire `correct` and `apply`'s already-applied path to emit it.
- [ ] **T-65.5** Repair the live `4x álcool 70` row (Combustível → Supermercado) through the new
      path, and re-run `generate-workbook` to confirm the 2026 workbook changes.

## Watch-outs carried in

- A guard is not a guard until broken — mutate, confirm red, and use the **exit code**
  (rtk reshapes `go test` output, so `grep '^--- FAIL'` matches nothing).
- A mutation can fail to APPLY — prefer a Python edit asserting its anchor matched exactly once.
- A guard can fail by never RUNNING — check ordering, and smoke-test the case it exists for.
- Join fixtures: short `DD/MM` only; a full date silently disables the test.
- Backgrounded MCP calls ignore `output_file` and inject no verdict template (measured s72).
  `calls.jsonl` DOES exist at `~/.local/share/ollama-bridge/calls.jsonl` — the s71 note is wrong.
