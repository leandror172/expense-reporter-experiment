# Session 42 Postmortem — PR #36 (T-13) Code Review & Follow-ups

**Date:** 2026-06-29 · **Branch:** `feat/t13-classifier-full-path` (PR #36)
**Entry point:** `/code-review` of the open PR → cascaded into model-choice correction + acceptance-suite repair.
**Commits:** `f997ea7`, `3dca7b9`, `9962e5c` (code/tests/fixtures). `tasks.md` + memories updated separately for handoff.

---

## 1. What we set out to do
Review PR #36, which makes the classifier predict the full `(type, category, subcategory)` path against
`config/taxonomy.json` via an atomic 112-member enum. The review surfaced two findings; acting on the first
one cascaded into a much larger (and more interesting) investigation.

## 2. Review findings
- **Finding 1 — predicted `type` dropped at the JSON boundary.** T-13 made the model predict `type`, but
  `toCandidates`/`CandidateOutput` (classify/auto `--json`) and `runAddDryRun`/`AddOutput` discarded it.
  Harmless for 99/104 leaves (`add` re-derives type from the unique owner) but breaks the MCP round-trip for
  the 5 cross-type ambiguous leaves (`Estacionamento`/`Dentista`/`Orion`/`Lilly`/`Ambos`).
- **Finding 2 — `classify` default model diverged.** `auto`/`batch-auto` defaulted to `qcoder`,
  `classify` to `q3`. Filed as T-16 with the (then-assumed) fix "move classify to qcoder."

## 3. What actually happened (scope evolution)
1. **Slice 1 of T-15** (surface `type` in JSON) — implemented, 2 deterministic tests. Cheap, additive.
2. **Precondition check exposed a regression class.** Running `-tags=acceptance` (which `go test ./...`
   never compiles) revealed PR #36 had left **13 acceptance tests red** — hidden by the build tag. Two root
   causes: taxonomy config newly mandatory but un-wired in many `Given`s (11), and a pre-existing `correct`
   seed-id bug (2).
3. **Finding 2 inverted under investigation.** A one-probe experiment showed enum validity is **grammar-
   enforced**, not model-dependent → the entire justification for the qcoder default was a measurement
   artifact. Reversed T-16: unify all commands on `q3` (fits VRAM), don't move classify to qcoder.
4. **Repaired all 13 acceptance tests** (T-17 + T-18) and verified green.

## 4. The model investigation (the session's core lesson)
- **Claim under review:** "qcoder honors the 112-enum 100% (session-40 smoke test), so it must be the default."
- **Test:** sent a `format`-enum-constrained request to a 9B model that *fits* the GPU (`my-classifier-q35`).
  It returned valid enum paths AND was *forced* to map an out-of-domain item ("airplane to Tokyo") into the enum.
- **Conclusion:** Ollama compiles the schema `enum` to a GBNF grammar that constrains the sampler. **Validity is
  free and model-independent.** qcoder (qwen3-coder:30b, 20.7 GB) only added cost: it doesn't fit the 12 GB GPU
  (52% VRAM / 48% CPU offload → latency + transient load 500s), with zero validity benefit.
- **Action:** reverted all four defaults (classify/auto/batch-auto flags + `classifier.Config`) to
  `my-classifier-q3`; validated end-to-end on the real 112-path taxonomy (Uber→Uber/Taxi 100%).
- **Residual:** q3 is **accurate but slow** (~12 s/classify, likely qwen3 "thinking" tokens). Accuracy *and*
  speed across q3/q35/qcoder is now the open benchmark (T-14).

## 5. Acceptance repair details
- **T-17 (11, PR #36 regression):** added a `fixture-taxonomy.json` per fixture covering its input leaves;
  wired `withFeedbackAndTaxonomyConfig`/`SetupBinaryConfig` into each `Given`. `requireDataDir` skip-guard
  added for the 2 few-shot classify tests. All green (incl. accuracy ≥50% on q3).
- **T-18 (2, pre-existing, NOT PR #36):** seeds used `date:"15/04"` + `id:f0c3bf1293f3`, but
  `parseExpenseForFeedback` normalizes to `15/04/2026` → `correct` looks up `id:8c434a0b64c0` → miss.
  T-11 (date normalization) orphaned the hardcoded ids; the build tag hid it on master. Rewrote ids/dates +
  added taxonomy so the corrected entry's category resolves.

## 6. Lessons / what to internalize
1. **Build-tag-hidden rot.** `//go:build acceptance` keeps slow tests out of `go test ./...` — and lets a
   whole regression class ship green. **After any change to a mandatory field/config contract, run
   `-tags=acceptance` explicitly.** This is the concrete cost behind `[[feedback_rename_json_tag_acceptance]]`.
2. **Measurement artifacts.** When a choice is justified by "it passed test X," check whether the *harness*
   (here, the GBNF grammar) is what's passing. The qcoder requirement, T-16, and a CLAUDE.md rule all rested
   on a test that measured the grammar, not the model.
3. **Don't assert blockers from assumptions — verify.** I twice claimed a blocker that wasn't real:
   (a) "the Ollama tests are blocked until qcoder loads" — false, the fixtures pin q3; (b) "data/classification
   is absent" — false, it's at the repo root, I checked the wrong relative path. Both were caught by *running
   the thing* and *reading the fixture*. Re-run + read-the-config beats reasoning from memory.
4. **Atomic enum removed a safety net (T-19).** Grammar-forcing means the model can't decline; novel inputs
   get a confident wrong leaf, weakening the 0.85 manual-review threshold the pre-T-13 algorithm relied on.

## 7. Open follow-ups
- **T-14** — benchmark accuracy AND speed across q3/q35/qcoder on real labeled data; pick smallest-that-fits-
  and-is-accurate. q3 works but is slow; q35 fits and may be faster.
- **T-16** — doc cleanup: CLAUDE.md still calls qcoder the "primary" tier "required for 5.7+"; correct it.
- **T-19** — design a "none-of-these" escape (sentinel path / restore `Diversos`+manual-review).
- **Operational** — full acceptance suite needs `-timeout 30m` (or sub-grouping) until the model is faster.
