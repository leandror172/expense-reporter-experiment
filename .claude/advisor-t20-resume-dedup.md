# Advisor Review — T-20 `batch-auto --resume` / dedup design

**Date:** 2026-07-17 (session 60)
**Advisor:** Opus subagent @ xhigh effort (Fable session — no `advisor()` tool; Workflow harness substitute)
**Verdict:** proceed-with-adjustments

Reviewed design: Option B (locked) — explicit `--resume` flag on `batch-auto` + always-on
stderr duplicate warning; count-consumption dedup semantics; entry-ID prediction before the
LLM call. Full prompt in the session workflow script (`advisor-t20-wf_d6202057-9c9.js`).

---

## Findings (verbatim)

**A. Count-consumption correctness**

1. [ADJUST] The classify-skip, append-entry-skip, and no-resume warning paths must read/mutate **one shared ID-ledger** in a single defined order — not three re-derived maps. Full-skips consume in the classify pass; partially-present rows consume nothing there and let the append pass consume per-entry; warnings consume in the append pass. State this invariant explicitly or the two-pass split double-counts.

2. [OK] Failure-downgrade is safe: entries 1..k-1 already appended/warned correctly before entry k fails; downgrading the row afterward doesn't unwind the ledger, matching existing partial-series semantics.

**B. Partial-installment resume**

3. [ADJUST] Entry-level skip (point 4) is unsafe: entry IDs omit category/type, so a run-2 divergent reclassification appends the *missing* installment with a **different category than its already-logged siblings** — a silent split-category series in the workbook. This is reachable (run-1 append error on installment k downgrades the row but leaves 1..k-1 logged). Drop entry-level auto-complete; route any partially-present series to review.csv. Simpler and removes the bug.

**C. Normalization mismatch**

4. [BLOCKER] `PredictEntryIDs` must call the **shared expandEntries with parsed inputs** (`ParseDateFlexible`→`formatDate` DD/MM→DD/MM/YYYY, `ParseCurrencyWithInstallments`→perInstallment+count, `formatInstallmentItem` suffix). Predicting from raw row strings mismatches every DD/MM and installment ID → resume silently skips nothing (or wrong rows). Also parse the date in the classify phase now (currently only appendOneRow does); a parse failure must route to error, not skip.

5. [ADJUST] `ParseDateFlexible` infers year from `time.Now().Year()`. A resume crossing a year boundary (run-1 Dec, run-2 Jan) re-infers a different year for DD/MM rows → ID mismatch → double-append. Narrow but real; document or pin the year.

**D. classifications.jsonl residue**

6. [OK] Crash between AppendExpense and logConfirmedFeedback loses only a *training* signal (secondary log); the workbook source is correct and is reconstructable offline from the expense entry. Backfill would require re-running the LLM, defeating the skip. Accept the residue.

**E. Simpler**

7. [SIMPLER] Keep count-consumption — a plain ID-set under-appends legit duplicates on resume when the log holds a partial subset, violating the locked constraint. The real cut is entry-level partial skip (finding 3).

**F. Test plan gaps**

8. [ADJUST] Add: (a) deterministic full N-installment series seeded → --resume skips the whole row (the highest-risk suffix+addMonths+date-format ID path, untested); (b) two identical legit rows, log seeded with **one** → skips exactly one, appends the other (core count-consumption justification); (c) **seed fixtures by running the real append path, not hand-written JSONL IDs**, so normalization drift actually fails; (d) partial-installment → review policy; (e) dry-run+resume prints skips, zero writes; (f) no-resume warning fires once per pre-existing ID and **not** for a 2nd in-CSV legit dup; (g) unit property test: `PredictEntryIDs(count>1)` equals IDs `ExpandAndAppend` writes.

---

## Adopted adjustments (implementation contract)

- **One shared ID-ledger** (`map[id]int`) threaded through classify phase → append phase → warning path; consumption order documented at the declaration site (finding 1).
- **Entry-level partial skip CUT.** A row whose predicted IDs are only *partially* present in the ledger routes to **review.csv** (gate-to-review philosophy), never auto-completes the series (finding 3).
- **`PredictEntryIDs` shares `expandEntries` with `ExpandAndAppend`** and takes parsed inputs; the classify phase parses date/value up front, parse failure → error row (finding 4, blocker).
- **Year-boundary caveat documented** in the `--resume` help text and report; not pinned (finding 5).
- classifications.jsonl residue accepted (finding 6); count-consumption kept (finding 7).
- Test plan extended per finding 8: installment-series skip (deterministic), one-of-two-dups skip, seeds built via `feedback.NewExpenseEntry` (real ID/normalization path), partial-series → review, dry-run+resume, warning-count semantics, predict==append property test.
