# Knowledge Index — Expense Reporter

**Purpose:** Map of where all project information lives. Read this to find anything.

<!-- ref:indexing-convention -->
### Indexing Conventions (Two-Tier System)

| Tier | Notation | When to Use | Lookup Method |
|------|----------|-------------|---------------|
| **Active reference** | `<!-- ref:KEY -->` + `[ref:KEY]` | Agent needs this during work; CLAUDE.md rules point here | `.claude/tools/ref-lookup.sh KEY` (machine-lookupable) |
| **Navigation pointer** | `§ "Heading"` | Index/docs pointing to sections for background reading | Open file, find heading (human/agent reads) |

**Active refs** are for high-frequency, runtime lookups (go package layout, test conventions, classification schema).
**§ pointers** are for low-frequency, "read when needed" navigation (research findings, planning docs, historical context).

**Single-responsibility rule:** One ref block per concept.
<!-- /ref:indexing-convention -->

---

## Quick Pointers (Active Work)

| What | Where |
|------|-------|
| Current layer tasks & progress | `.claude/tasks.md` |
| Session log (current) | `.claude/session-log.md` |
| Agent preferences & resume checklist | `.claude/session-context.md` |
| Project rules & constraints | `CLAUDE.md` (repo root) |
| Cross-repo context (LLM infra) | `/mnt/i/workspaces/llm/.claude/` |
| Implementation plan (session 4) | `.claude/plans/acceptance-harness-batch-auto.md` |
| Implementation plan (session 10 — 5.7) | `.claude/plans/5.7-few-shot-injection.md` |
| Implementation plan (session 20 — workbook path fix) | `.claude/plans/fix-workbook-path-resolution.md` |
| Implementation plan (`review` command) | `.claude/plans/review-command.md` |
| T-23 calibration benchmark plan (leaf-first confidence signal, session 49) | `.claude/plans/t23-calibration-benchmark.md` |
| Leaf-first classification plan (standalone accuracy A/B + T-23 precondition, session 49) | `.claude/plans/leaf-first-classification.md` |
| Review UI design brief (for claude.ai/design) | `docs/plans/review-ui-design-brief.md` + `docs/plans/review-ui-fixtures/` |
| Lovable companion suggestion (superseded by local design) | `docs/plans/lovable-suggestion-plan.md` |
| Workbook mapping plan (3-layer, session 26) | `.claude/plans/workbook-mapping-plan.md` |
| Workbook mapping Layer 3 cowork brief | `.claude/plans/workbook-layer3-instructions.md` |
| Workbook structural map (JSON + cell notes, session 25) | `.claude/workbook-map.md` |
| Workbook visual notes (annotated screenshots, session 26) | `.claude/workbook-visual-notes.md` |
| Workbook generator spec v2 (Layer 3 + hand-review, session 27) | `.claude/plans/workbook-generator-spec.md` |
| Workbook generator implementation plan + next-session brief | `.claude/plans/workbook-generator-implementation-plan.md` |
| Taxonomy identity key (full-path decision + ambiguity guard, task #5 deferred) | `.claude/plans/taxonomy-identity-key.md` [ref:taxonomy-identity-key] |
| A1 + T-65 working todo (session 72, branch `feat/a1-keyword-hint`) — locked scope decisions (one `keyword_hint` column, advisory-only, no `exportReviewed()` round-trip, strict predicate) plus the ordered TDD steps and the watch-outs carried in | `.claude/plans/a1-keyword-hint-todo.md` |
| **T-64 + T-72 working todo (session 73, branch `feat/t64-multiplier-notation`)** — locked decisions D1–D9 for the multiplier installment notation (`x`/`X` presence is the mode switch; the value/count split is resolved by trying three readings and requiring EXACTLY ONE, so ambiguity is an ERROR not a guess) plus the ordered TDD steps and 10 watch-outs. **Read W1 before adding any acceptance coverage** — an `apply`-level fixture proves NOTHING here, because `apply` reads `reviewed.json` where value is already a float and installments already an int, so it never parses a value string | `.claude/plans/t64-multiplier-notation-todo.md` |
| **T-80 duplicate routing plan (session 77)** — a duplicate row routes to review instead of auto-appending; decisions D1–D5 OPEN, incl. `apply`'s silent confirmed+found no-op | `.claude/plans/t80-duplicate-routing.md` |
| Generator build order + row counter + oracle coupling | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-build-order] [ref:generate-row-counter] [ref:generate-oracle-coupling] |
| Generator column layout (data sheets vs Listas) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-colmap] |
| Generator block sizing (calculateBlockRows) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-block-sizing] |
| Generator package-level state (dataYear/headroomRows/perGroupPctRows) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-package-state] |
| Generator excelize gotchas | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-excelize-gotchas] |
| Generator income 3-level model + dual-format taxonomy | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-income-symmetry] |
| Plan A — persist expense *type* + rename/JSON migration + backfill (session 32) | `.claude/plans/persist-expense-type.md` |
| Plan B — full-path entry routing (T-04, session 32) | `.claude/plans/full-path-entry-routing.md` |
| Bf real-data verification runbook (execute next session — Bf1/Bf3 + routing proof) | `.claude/plans/bf-real-data-verification-runbook.md` |
| Retire workbook insertion, keep only generation (logs = source of truth; couples T-11/T-09/T-03) | `.claude/plans/retire-insertion-keep-generation.md` |
| WS-B slice 3 — `batch-auto` → log-append (decisions locked, session 43; not yet implemented) | `.claude/plans/ws-b-slice3-batch-auto-log-append.md` |
| WS-B slice 4 — `apply` → log-append (decisions locked, session 44; not yet implemented) | `.claude/plans/ws-b-slice4-apply-log-append.md` |
| T-13 (revised) — classifier predicts full (type,category,subcategory) path; WS-D prerequisite | `.claude/plans/t13-classifier-full-path.md` |
| 5.R2 embedding retrieval — embed-on-miss cascade layer (DONE + ADOPTED session 54; A/B miss full-path 18.8%→52.5%) | `.claude/plans/5r2-embedding-retrieval.md` |
| Harness extraction plan (T2 — standalone `acceptance-harness` repo, career-search repoint, expenses migration, LTG path; session 56) | `.claude/plans/harness-extraction-plan.md` |
| acceptance-test vs harness comparison (companion: old-job pipeline framework — adopt/adapt/discard analysis; session 56) | `.claude/plans/harness-extraction-acceptance-test-comparison.md` |
| Harness extraction Session B — executable migration plan (expenses `test/` → module; measured inventory, D1 verify-namespace + D2 Context-carrier open; session 59) | `.claude/plans/harness-extraction-session-b.md` |
| Parse boundary — structured input, parse-once (year precedence, installments field, T-37/T-38/T-40/T-21 resolutions; FINAL — design session 63, awaiting implementation go) | `.claude/plans/parse-boundary.md` |
| Per-sheet structural digests (Layer 3 inputs, Sonnet fan-out) | `.claude/workbook-dump/digests/*.md` (gitignored with dump) |
| Template golden master (user-curated, fake data) | `.claude/workbook-template/template-reviewed.xlsx` + `template.xlsx` (generated) |
| Template build/convergence reports | `.claude/workbook-template/{ambiguities,review-diff,convergence-report}.md` + `diff.py` |
| Phase B data extract (template-data.xlsx block/column model) | `.claude/workbook-template/phaseB-data-extract.md` |
| Phase B re-review report (data-bearing template, PASS) | `.claude/workbook-template/phaseB-rereview.md` |
| Template builder (SUPERSEDED by `internal/generate`, kept as Phase A/B history) | `.claude/scratch/template-builder/` (standalone Go module) |
| T-14 benchmark report (accuracy/calibration/OOD + --think findings, session 46) | `.claude/t14-benchmark-report.md` |
| T-23 logprob-confidence probe (leaf-first; pre-mask logprobs, session 49) | `.claude/t23-logprob-confidence-probe.md` |
| T-23 strategic implications (gate route → workbook plan + grand vision, session 50) | `.claude/t23-strategic-implications.md` |
| T-32 agreement-gate report (work + findings: replay validation, verifier false-pass, fixture coupling, session 57) | `.claude/t32-agreement-gate-report.md` |
| **T-42 scout report (first real chain run on 2026 data, session 68)** — isolation recipe (scratch install root; no flag redirects the logs), 2 `review` blockers (S1 unparsed-row contradiction, S2 workbook-derived taxonomy + dangling `workbook_path`), T-21 measured at 22% of the queue (10 of 45 rows → 10 log entries where 27 belong), and the drift-corrupted records backfilled. **Note the in-file S4 correction:** `- 1/4` / `4x` are input ERRORS, not notations the parser must learn (`99,90/3` is the only accepted form), so T-21 is plumbing, not parser work | `.claude/t42-scout-report.md` |
| **T-42 close findings (session 71)** — the two non-papercut findings from the first real monthly close: an auto-inserted row cannot be corrected (feedback updates, budget does not) [ref:auto-insert-unrepairable]; and the accuracy picture — 38% correction rate, **92% of errors cross-CATEGORY not wrong-leaf**, median confidence 0.95 on wrong answers, and the keyword layer already holding the reviewer's answer in 60% of corrections [ref:close-accuracy-2026-01]. Referenced by T-65/T-66; read the method caveats before quoting the 60% | `.claude/t42-close-findings.md` |
| **A1 keyword-hint pre-check (session 72)** — GO. Settles the s71 report's approximate 60% with the PRODUCTION `MatchStrength`: the strict predicate (unambiguous, specificity 1.00, keyword ≠ model) fires on **14 of 65** rows at **71.4%** precision and recovers **10 of 25** corrections, with only 2 noise fires. Also finds specificity is effectively **binary** — every row scoring in [0.70, 1.00) is ambiguous, so the 0.70 threshold is inert (mutation-verified: dropping `!Ambiguous` makes the thresholds diverge 14 vs 17, and those 3 add zero recall). Ship the strict predicate only, and keep it ADVISORY — at 71.4% precision, auto-applying would inject errors | `.claude/a1-keyword-hint-precheck.md` |
| **Absolute gate precision — MEASURED (session 72)** — **94.1%** (1 correction in 17 gate-passing rows) vs **50.0%** for gate-refused rows: an 8.5× discrimination on a full unselected month, from data already on disk. T-32's "unmeasurable" rested on the 649 being a confidence-selected subset; a CLOSE labels everything, because `close-cycle.sh:220` feeds `review` the FULL `classified.csv`. **Also corrects `t42-close-findings.md`:** the 17 auto rows were NOT unseen — they were all reviewed, so the defect is persistence (T-65), not review coverage, and B1's rationale largely dissolves. Join is POSITIONAL — item-keyed collapses duplicate items (`San michel` ×3) and over-counts | `.claude/b1-gate-precision-measurement.md` |
| **T-75 converter plan + LIVE PROGRESS TRACKER (session 75, branch `feat/t75-telegram-converter`)** — **COMPLETE s76: all six steps DONE, and the 2026-02 export was converted and CLOSED FOR REAL** — 46 reviewed entries applied, 34 rows appended (0 failed / 0 skipped / 0 pending), installment reconciliation exact, expense log 2163 → 2216 (**R$ 12.081,48**), `workbook-2026.xlsx` + `workbook-2025.xlsx` both written with **empty stderr, so no warn-skips** (the silent failure that hid the S2 taxonomy drift). **The human caught what the tooling could not:** a cancelled `Academia <redacted>` at `1617,00/12` removed during review — verified `auto_inserted=false` and absent from the log before `apply` ran, so **12 phantom installment rows were avoided by a person looking at a page**, which is why `close-cycle.sh` refuses to synthesise `reviewed.json`. Three prepare runs were needed and no rerun was waste (run 1 found a duplicate double-count, run 3 folded in the repaired rejects): ~15 min of machine time, **zero human time**, because the review pass was held back until the input was final. ⚠️ **Step 6 was planned as "not a coding step" and it was not one:** the first real run produced a bucket 2025 never had (`rejected: 21 fields` = ten expenses in one message), which is the Risks entry's *"bucket drift is the signal"* firing, and it bought **D10**. It also proved three things — the step-1 gate passes 39/39 on data neither implementation was fitted to; **the D1 amendment caught a LIVE misfile** (verified against the pre-amendment binary in a throwaway worktree: `29/12/26` sent 2026-02-12 expands to 2026, `validateYearNotBeyondCurrent` passes it because 2026 IS the current year, and `355535f` wrote `expenses-2026-12.csv` — two real expenses in a month that has not happened); and **D8 refused to clobber the already-closed `expenses-2026-01.csv`**, since a February export legitimately carries January-dated rows (D7). The human review pass is deliberately NOT spent yet — it is the one hop no Go test reaches. Step 1: `telegram-import --dry-run` reproduces the 2025 buckets PER ID, 393/393, perturbation-verified. **Step 2: D4 repairs, with an AMENDMENT found by measurement — a candidate's date must lie inside the D1 window, because the boundary accepts year 1200 and `Value("00")` returns 0 with no error, which made the real `21/08/ 1200,00` slash typo read as ambiguous; after it, 6 repaired / 0 ambiguous / 6 two-field messages left (no date ×3, a colon ×1, a space for the first `;` ×2), and both plan mutations compiled and went red on their REQUIRED rows.** **Step 3: the writers** — `expenses-YYYY-MM.csv` per RESOLVED month (D7), T-63 rejects where **no file IS the signal** that nothing was rejected, and an all-or-nothing overwrite refusal that checks every target before the first write (D8/D9). **Step 4: the probe now reads the REAL converter** — matched holds at 346 while side A grows by exactly the 6 repaired lines, 0 SUSPECT, gate PASS in both `--source` modes; the finding is that those 6 (R$ 1.828,96) never reached the workbook at all, so the typo is what kept them out (file beside T-78). **Step 5: build + vet clean, `go test ./...` exit 0 (22 packages, 948 tests), `./run-acceptance.sh -full` 78 passed / 0 failed / 0 skipped in 108 s** — **0 skipped, not 78, is the load-bearing number**: it is what proves the `RequireOllama` gate ran instead of quietly opting out, corroborated by `TestTelegramImport_MonthFileIsAcceptedByBatchAuto` taking 3.26 s when a skip costs 0.00 s. (`internal/capture` + `internal/telegram` + `cmd/.../telegram_import.go`; acceptance `test/telegram_import_test.go` on the synthetic fixture `telegram-import-dry-run`; gate scripts `.claude/scratch/t75_expected_buckets.py` + `t75_gate_compare.py` — re-run them on the 2026 export first). Found on the way: **`parse.Date("25/07/25")` → 25/07/0025, no error** — a boundary defect to file at handoff (the converter expands 2-digit years itself). The tick-boxes live in the plan's "Build sequence / PROGRESS TRACKER" section, deliberately the ONLY step list (a separate todo file would duplicate the gates and drift). **Language decided s75: GO, as a subcommand `expense-reporter telegram-import` — NOT Python.** The decisive verified fact: `internal/parse` already exports `Date(dateStr, Options)` and `Value(valueStr)`, which ARE the two predicates D4's exactly-one-reading rule needs, with clean imports (no `config`). Python would port `currency.go` + `normalizeThousands` + the ladder — one concept, two implementations, the T-54/T-72/T-73 shape — and would break the property the plan rests on: the same message via export or via bot must be interpreted identically. **The probe stays Python on purpose**, becoming an independent cross-check rather than sharing the Go parser's assumptions. **D1 was AMENDED at step 5 (user ruling): the window validates the resolved date on BOTH branches, because an explicit `21/08/1200` otherwise converted into its own month file and both gates passed on it — see the `internal/capture` row.** ⚠️ **D1 is option (a): the converter OWNS year resolution, the boundary VALIDATES** — passing the message year as `Options.Year` instead is silently wrong at the year boundary (a 02/01/2026 message reporting `31/12` becomes 31/12/2026 and PASSES the future-year check). Carries the governing conventions too: **the SUT is a PURE FUNCTION so `test-executable-spec`'s given/when/then skeleton COLLAPSES** to builder-nouns + verdict-verbs (over-DSLing a pure-function test is the doc's named rule-3 violation), and `characterize-first` applies to **step 4 only** — the probe is committed untested code. — Telegram `result.json` → per-month `batch-auto` CSVs. Decisions D1–D8: emit the RESOLVED full date (the message timestamp outranks `--year`, window **[-180,+7]** derived from the measured -131..+3 spread and kept under 365 days so exactly one year-candidate fits); **do NOT expand installments** — the raw token must survive into the review queue (s67 D4 / T-21), and a second site computing installment dates is how T-35 happened; repairs use **T-64's exactly-one-reading rule** lifted to the message layer, safe against the divisor notation ONLY because D5 stops repairs ever running on a line that already parses. **All 20 empty-text messages are receipts (13 PDF + 7 photo, zero stickers)** — user ruling s75: receipts FOR an expense that was also typed, so NOT missing spend (an earlier draft's inference that they were is corrected in the plan); reported as a one-line count, never routed to rejects. Validation criterion is **unexplained == 0, deliberately NOT a match rate** — the log is a second lossy record, so chasing coverage would reproduce its own omissions | `.claude/plans/t75-telegram-converter.md` |
| **T-75 oracle viability — the 2025 Telegram export as a validation oracle (session 75)** — GO, with an **asymmetric** acceptance criterion. A naive conversion of the 352 conforming messages finds a `(date,value)` partner for **90.3%** of its rows, and all **37** misses have a named cause (0 unexplained). **The residue is on the LOG's side:** all **14** installment series in the window are recorded with only their FIRST installment — R$ 7.309,43 of rows absent, cross-checked to the centavo by two independent methods; **11 are genuinely short (R$ 3.594,44) and 3 carry a compensating same-day lump**, so the net understatement is ~R$ 2.634,44 and a blanket backfill would triple-count those 3. User ruling: reported-as-installment should have BEEN an installment, so these are omissions (filed T-78, do NOT fold into T-75). Because the log is itself lossy it cannot adjudicate a mismatch — the criterion is "every miss explainable, unexplained == 0", **never a match rate**, which would push the converter toward reproducing the log's own omissions. **§ "Step 4" (s75): the probe read the REAL converter — matched stays 346 while side A grows by exactly the 6 repaired lines, none of which is in the log (R$ 1.828,96 typed in 2025 that the workbook never received: the typo was what kept them out); "never entered" is now verified by item lookup (unknown / omission / SUSPECT), SUSPECT 0, gate PASS** | `.claude/t75-oracle-viability.md` [ref:t75-oracle] |
| T2 closeout report (v1.0.0 tag rationale + shape decision, session 60) | `.claude/t2-closeout-report.md` |
| T-34/T-12 red-test repair report (5 reds → 49/49 green; latent join-ID divergence finding, session 60) | `.claude/t34-t12-red-test-repair-report.md` |
| Advisor review — T-20 `--resume`/dedup design (Opus xhigh; adopted contract, session 60) | `.claude/advisor-t20-resume-dedup.md` |
| T-20 implementation report (`batch-auto --resume` + duplicate warning; [ref:batch-auto-resume] semantics block, session 60) | `.claude/t20-resume-implementation-report.md` |
| batch-auto resume acceptance (A1–A5: seeded-skip, one-of-two-dups, warning; deterministic seeds via real append path) | `expense-reporter/test/batch_auto_resume_test.go` + `test/expect/resume.go` + `test/fixtures/batch-auto-resume-*`, `batch-auto-dup-warning` |
| **Reviewed-installment acceptance (T-21, s69)** — `apply` expands a reviewed installment into N dated rows. Fixture `apply-reviewed-installments` uses `90,00/3` → **30** because `ExpenseLogMatches` compares `value` and `99,90/3` is `33.300000000000004` (float-drift trap). Companion `apply-installment-resume-join` uses the NON-terminating `100,00/3` and asserts a BEHAVIOR — `batch-auto --resume` recognises the series apply wrote — because comparing apply's ids to `PredictEntryIDs` would be circular (both call `expandEntries`) | `expense-reporter/test/apply_test.go` + fixtures `test/fixtures/apply-reviewed-installments/`, `test/fixtures/apply-installment-resume-join/` (see their READMEs) |
| **Installment-count contract (T-21, s69)** — `reviewed.json`'s `installments` is REQUIRED (`validateEntries` rejects `< 1`; an absent key decodes to 0, the T-54 shape), and apply's summary counts ROWS not entries (`rowsWrittenFor`) so "Appended: N rows" stays literally true | `expense-reporter/internal/apply/reader_test.go` (`TestReadReviewed_RejectsEntryWithoutInstallmentCount`) + `cmd/expense-reporter/cmd/apply_test.go` (`TestAppendNewRows_CountsRowsWrittenNotEntries`, `…DryRunPreviewsTheRowsARealRunWouldWrite`) |
| **The `exportReviewed()` JS hop is UNGUARDED (measured, s69)** — deleting `installments` from the export leaves the whole Go suite green; the export runs in the browser and no Go test can execute it. Do NOT close it with a hand-authored fixture — that IS the T-54 shape. Go-side half pinned by `installmentCountOfferedToTheReviewer` | `expense-reporter/test/.memories/KNOWLEDGE.md` § "The export JS is the one hop no Go test reaches" + `test/review_test.go` |
| **Telegram-import dry-run acceptance (T-75 step 1, s75)** — `TestTelegramImport_DryRunAccountsForEveryMessage` runs `telegram-import result.json --dry-run` on a SYNTHETIC 15-entry export (one message per bucket + a `service` entry the adapter must drop, so the report says 14; step 2 added the `repaired` line — ids 5 9 15 — and the `rejected: ambiguous` bucket, id 14 `Bar; 15/05/ 2025,50`) and asserts **the ids on every report line, not only the counts** — two buckets swapping one message each leave every count unchanged. Deterministic: no model, no workbook, no config, so it lives in the `-short` group. The real export's per-id check is the hand-run gate in `.claude/scratch/t75_gate_compare.py`, deliberately NOT a unit test that would skip when the private file is absent | `expense-reporter/test/telegram_import_test.go` + `test/actions/commands.go` (`RunTelegramImport`) + fixture `test/fixtures/telegram-import-dry-run/` (see its README); unit: `internal/capture/classify_test.go`, `internal/capture/repair_test.go`, `internal/telegram/export_test.go`, `cmd/.../telegram_import_test.go` |
| **Telegram-import WRITE acceptance (T-75 step 3, s75)** — five scenarios in `test/telegram_import_write_test.go`: one month file per RESOLVED expense date (D7 — the January line was sent in July), repaired lines carry `# repaired from: …` inside the month file, the rejects file lists exactly the attempted expenses in the T-63 shape (`expect.FailedRowsCarryTheirReason`), an existing month file is refused without `--force` **and nothing else is written** (all-or-nothing, D8), `--force` replaces it, and a clean export leaves NO rejects file. The fifth, `TestTelegramImport_MonthFileIsAcceptedByBatchAuto`, is the step-3 gate itself — the converter's file read by the real `batch-auto --dry-run`, no `failed.csv` — and is Ollama-gated (`-full`); its deterministic half is the `parse3FieldLine` round-trip unit test in `cmd/.../telegram_import_write_test.go` (which also pins `writeOutputs`' all-or-nothing refusal and the files report) | `expense-reporter/test/telegram_import_write_test.go` + `test/actions/commands.go` (`RunTelegramImportWriting` — `--out-dir` = WorkDir, registers `telegram-rejects.csv`) + fixtures `test/fixtures/telegram-import-clean/` (two good messages: pins the ABSENCE of the rejects file) and `test/fixtures/telegram-import-to-batch-auto/` (the dry-run export + `batch-auto-basic`'s config and taxonomy); sink logic + tests: `internal/capture/output.go` / `output_test.go` |
| **Telegram-import MULTI-LINE acceptance (T-75 / plan D10, s76)** — `TestTelegramImport_MultiLineListBecomesOneExpensePerLine` on a 3-message synthetic fixture. **The trigger was real data**: the 2026-02 export's first run produced `rejected: 21 fields` — one message holding ten well-formed expense lines (a backlog catch-up list) — and flattening lost all ten at once. The scenario is built so both ways of getting D10 wrong go red: message 2 splits into three lines of which ONE cannot parse (`99/99`), so a "split only when every line PARSES" rule drops `3 converted` to 1; message 3 is one expense plus a chatty line and must NOT split, which is why `rejected: bad value: 3` is asserted — under a "SOME line is 3-field-shaped" rule that line would convert and the human's note would vanish. `5 expense lines (1 multi-line list split)` is the assertion that distinguishes a real split from a coincidence, since the bucket counts alone are equally satisfied by three unsplit messages in three buckets; that line prints ONLY when something split, which is why every other scenario's output is byte-unchanged. Deterministic — `-short` group | `expense-reporter/test/telegram_import_test.go` + fixture `test/fixtures/telegram-import-multiline/` (see its README for why each of the three messages is there); unit: `internal/capture/classify_test.go` (`TestExpandLists`, and the `Summarize` row pinning one id in two buckets), `cmd/.../telegram_import_test.go` (the report row where counts are LINES and ids are MESSAGES) |
| **Unparseable-row output `failed.csv` (T-63)** — `batch-auto` writes every row the parse boundary rejected to `<output-dir>/failed.csv`, each with its reason as a **trailing `#` comment on the same line**, so the human repairs the data in place and re-runs THAT FILE. The retired `batch` command's format appended the reason as a 4th field, which `SplitN(line,";",3)` folded into the value — it never re-imported. `stripTrailingComment` fires only when `#` is whitespace-preceded AND the 3 fields are already complete, so `Mesa #5` stays data and a genuine 4-field `item;date;value;subcategory` still fails loudly instead of losing its subcategory. No file is written when nothing was rejected — existence IS the signal | `expense-reporter/internal/batch/failed_rows.go` + `cmd/expense-reporter/cmd/batch_auto.go` (`stripTrailingComment`, `rejectedRows`); tests: `internal/batch/failed_rows_test.go`, `cmd/.../failed_rows_projection_test.go`, `test/batch_auto_failed_rows_test.go` + `test/expect/failed_rows.go` + fixture `test/fixtures/batch-auto-failed-rows/` (see its README) |
| Join-id acceptance (T-35: both logs share one `GenerateID`) | `expense-reporter/test/apply_test.go` + `test/auto_log_append_test.go` (`*_JoinIDMatchesAcrossLogs`), `test/expect/feedback.go` (`JoinIDMatchesAcrossLogs`), fixture `test/fixtures/apply-join-id/` — fixtures use SHORT `DD/MM` on purpose; a full date hides the bug |
| Year-precedence acceptance (T-41 slice 1: `--year` flag, `date_year` fallback, explicit-year wins — deterministic, no Ollama) | `expense-reporter/test/add_year_test.go` + `test/expect/json.go` `OutputJSONHasDate` |
| Correct `--year` join acceptance (bare date + flag → GenerateID hits prior-year seed) + BR thousands CLI pin (`1.234,56`) | `expense-reporter/test/correct_year_test.go` + fixture `test/fixtures/correct-year-flag/`; `test/json_output_test.go` `TestAdd_ReadsBrazilianThousandsAmount` |
| Year-ladder + field-sentinel unit pins (T-41 slice 2, s66 — deterministic, no Ollama; `auto` has no deterministic CLI route since every path calls `Classify`, so the ladder and the error shapes are pinned at the helper level instead) | `expense-reporter/cmd/expense-reporter/cmd/parse_boundary_test.go` (`parseOptions` wiring/precedence, `describeParseFailure` keeps the boundary's reason) + `internal/parse/parse_test.go` (`TestFields_ErrorsNameTheFieldThatFailed`, `…WrappedErrorKeepsTheSpecificCause`) |
| Apply identity-is-derived pins (T-41 slice 4, s67): characterization tests written GREEN against the unmodified `canonicalDate`/`entriesWithCanonicalDates` (which had ZERO coverage) plus the id test held RED until the fix; acceptance `TestApply_StaleReviewedIDIsRecomputedNotTrusted`. **`apply-basic` cannot catch an id regression — it is self-consistent; `apply-stale-id` is the only discriminating fixture, mutation-verified** | `expense-reporter/cmd/expense-reporter/cmd/apply_canonical_test.go` + `test/apply_test.go` + fixture `test/fixtures/apply-stale-id/` (see its README) |
| `--year` WIRING proof for `batch-auto` (s67 — deterministic, no Ollama: bare-dated input + a 2024 seed, so the row can only be recognised as already-logged if `--year 2024` reached `parseOptions`; mutation-verified — `parseOptions(0, appCfg)` drops the skip count to 0). Input dates MUST stay bare: an explicit year outranks `--year` and would pass without the flag being read | `expense-reporter/test/batch_auto_resume_test.go` (`TestBatchAutoResume_YearFlagResolvesTheSeededYear`) + fixture `test/fixtures/batch-auto-resume-year-flag/` |
| CSV column contract for `classified.csv`/`review.csv` (s67 — value column keeps the RAW token so the installment count survives into the review queue, D4/T-21; an unparsed row keeps its raw line with EMPTY date/value cells rather than rendering a zero time as `01/01/0001`). Mutation-verified: returning `pe.Value` turns `99,90/3` into `33.30` and both writers go red | `expense-reporter/cmd/expense-reporter/cmd/batch_auto_test.go` (`TestCSVWriters_PreserveInstallmentNotation`, `TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine`) |
| **Malformed-row contract for `review` (S1, s68 — deterministic)**: `ReadQueue` returns `(entries, unreviewable, error)`; a row batch-auto could not parse (raw text in item, EMPTY date/value) is skipped and named on stderr by `cmd/review.go`, NOT fatal. Tolerance is deliberately narrow — corruption (bad number, bad confidence, wrong field count) still hard-errors. Mutation-verified BOTH halves: remove the skip → red; keep the skip but drop the warning → red (the quiet-loss trap) | `expense-reporter/internal/review/queue_test.go` (`TestReadQueue_UnparsedRowsAreReturnedNotFatal`) + `test/review_test.go` (`TestReview_UnparsedRowsAreSkippedNotFatal`) + fixture `test/fixtures/review-malformed-rows/` + `test/expect/queue.go` |
| **Producer→consumer seam for `classified.csv` (T-54, s68 — deterministic, no Ollama, no fixture: generates the REAL writer's bytes at runtime and feeds them to the REAL `review.ReadQueue`).** Two properties, mutation-verified SEPARATELY: (a) the reader accepts what the writer emits — revert the `true`/`false` spelling and BOTH tests go red; (b) `ReadQueue`'s id equals the expense log's `GenerateID` — revert `dateCell()` to a year-less date and ONLY (b) goes red, proving (a) is blind to it. Input dates MUST stay bare `DD/MM` or (b) silently disables itself | `expense-reporter/cmd/expense-reporter/cmd/classified_csv_seam_test.go` |
| Join-id construction pins (T-40, s67 — deterministic, no Ollama, no fixture: `GenerateID(item, DateString(), Value)` vs `PredictEntryIDs(...)[0]` for `Installments == 1`, plus a case documenting that an installment SERIES legitimately diverges because `expandEntries` suffixes the item). Inputs MUST stay short `DD/MM` | `expense-reporter/cmd/expense-reporter/cmd/join_id_test.go` |
| Batch stale-`date_year` warning shape (T-49, s67 — one line per RUN carrying the row count, not one per row; silent when `date_year` names the current year and when no row used the config rung) | `expense-reporter/cmd/expense-reporter/cmd/parse_boundary_test.go` (`TestWarnIfStaleConfiguredYearInBatch_*`) |
| Stale-`date_year` warning acceptance (s65: warns when the config rung dated the entry; silent when current, when outranked by an explicit year; fires on `correct` despite its lookup failing) | `expense-reporter/test/stale_date_year_test.go` + `test/expect/dateyear.go` (`StaleConfiguredYearWarned`, `NoStaleConfiguredYearWarning` — stderr-specific, since `--json` owns stdout) |
| Date/year semantics across the system (T-35 survey — six parsers, year sources, what NOT to "fix") | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] |
| T-35 implementation report (join-id fix: scope correction, RED evidence, T-37/T-38 findings, session 61) | `.claude/t35-implementation-report.md` |
| Session log archive (sessions 1–2) | `.claude/archive/session-log-2026-03-02-to-2026-03-02.md` |
| Session log archive (sessions 3–5) | `.claude/archive/session-log-2026-03-13-to-2026-03-02.md` |
| Session log archive (session 6 — 2026-03-03) | `.claude/archive/session-log-2026-03-03-to-2026-03-03.md` |
| Session log archive (session 7 — 2026-03-11) | `.claude/archive/session-log-2026-03-11-to-2026-03-11.md` |
| Session log archive (session 17 — 2026-04-23) | `.claude/archive/session-log-2026-04-23-to-2026-04-23.md` |
| Session log archive (session 18 — 2026-04-24) | `.claude/archive/session-log-2026-04-24-to-2026-04-24.md` |
| Run acceptance tests | `expense-reporter/run-acceptance.sh` — deterministic group by default (`-short`, no Ollama); `-full` = whole suite, 3600s ceiling |
| Generate-workbook acceptance fixture (G3, oracle-frozen dumps) | `expense-reporter/test/fixtures/generate-basic/` + `test/expect/workbook_structure.go` |
| Full type-routing cycle acceptance (batch-auto→review→apply→generate-workbook, incremental) | `expense-reporter/test/type_routing_cycle_test.go` + `test/fixtures/type-routing-cycle/` |
| Advisor review — G3 acceptance design | `.claude/advisor-G3-acceptance-design.md` |
| Advisor reviews — apply phase 3, Phase B builder, session 24 | `.claude/advisor-{apply-phase3,phaseB-builder,session24-review}.md` |
| Doc audit (2026-06-09) | `.claude/doc-audit-2026-06-09.md` |
| QUICK.md memory audit — 46K→16K consolidation into KNOWLEDGE.md (2026-07-01) | `.claude/quick-memory-audit-2026-07-01.md` |
| Session 42 postmortem — PR #36 review, model revert (qcoder→q3), acceptance repair (T-17/T-18) | `.claude/session42-postmortem.md` |
| WS-B slice 3 work report — `batch-auto` → log-append (session 43) | `.claude/ws-b-slice3-implementation-report.md` |
| **DSL reading guide — READ BEFORE WRITING ANY TEST** (what to read, in what order, + the invariants) | `expense-reporter/test/.memories/KNOWLEDGE.md` [ref:acceptance-dsl] |
| Shared Given vocabulary (atomic events + 3 canonical Givens, composed with `harness.Events`) | `expense-reporter/test/givens_test.go`; config accumulates via `domain.SetupBinaryConfig` + `ctx.BeforeWhen` |
| `Given`-as-slice upstream analysis (62-scenario survey; verdict: upstream the helper as v1.1.0, NOT the slice) | `.claude/given-slice-upstream-analysis.md` |
| Harness v1.1.0 upstream proposal (PR-ready: `Scenario.Fixture`, `UseBinary`, `ctx.State`+`BeforeWhen`, `harness.Events`; all additive) | `.claude/plans/harness-v1.1-engine-ownership.md` |
| Acceptance test patterns | `expense-reporter/test/PATTERNS.md` — [ref:acceptance-patterns] effort table + ref index |
| Acceptance test architecture | `expense-reporter/test/README.md` — [ref:acceptance-harness], [ref:acceptance-fixtures], [ref:acceptance-verify], [ref:acceptance-run] |

---

<!-- ref:go-structure -->
## Go Package Structure

| Package | Path | Purpose |
|---------|------|---------|
| `main` | `expense-reporter/cmd/expense-reporter/main.go` | Entry point |
| `cmd` | `expense-reporter/cmd/expense-reporter/cmd/` | Cobra CLI subcommands (add, batch, version; +classify/auto/batch-auto in Layer 5; +generate-workbook in Phase G; **+telegram-import** in T-75 — `telegram_import.go`: `--dry-run` prints the per-bucket report with ids (`writeBucketReport`); `--out-dir DIR` (REQUIRED when writing, D9) writes `expenses-YYYY-MM.csv` per RESOLVED month plus `telegram-rejects.csv` via `batch.WriteFailedRows`, ALL-OR-NOTHING — `writeOutputs` checks every target before the first write and refuses any existing file unless `--force`, naming the blockers); `parse_boundary.go` = command-layer helpers over `internal/parse`: `parseOptions` (builds the year ladder from `--year` + config; no seam — all callers want the identical ladder), `warnIfStaleConfiguredYear`, `describeParseFailure` (adds the installment-syntax hint a value rejection needs, and never replaces the boundary's reason) |
| `workbook-inspect` | `expense-reporter/cmd/workbook-inspect/` | Thin CLI wrapper over `internal/inspect` (workbook mapping L1); see its `.memories/QUICK.md` |
| `internal/inspect` | `expense-reporter/internal/inspect/` | Structural-dump core (values/formulas/styles/merges/rowType classifier) shared by workbook-inspect and test verifiers |
| `internal/generate` | `expense-reporter/internal/generate/` | Workbook generator (spec v2 port of the scratch builder): `Generate(Options)`, taxonomy JSON + entries JSONL loader, layout/styles/listas logic |
| `internal/batch` | `expense-reporter/internal/batch/` | Batch import orchestration |
| `internal/cli` | `expense-reporter/internal/cli/` | CLI output helpers |
| `internal/excel` | `expense-reporter/internal/excel/` | Excel workbook read/write (excelize library) |
| `internal/logger` | `expense-reporter/internal/logger/` | Logging abstraction |
| `internal/models` | `expense-reporter/internal/models/` | Domain structs: Expense, Category, SubCategory, etc. |
| `internal/parser` | `expense-reporter/internal/parser/` | CSV parsing, date/decimal normalization (BR format) |
| `internal/resolver` | `expense-reporter/internal/resolver/` | Category resolution: matches expense → (category, subcategory) |
| `internal/workflow` | `expense-reporter/internal/workflow/` | Multi-step workflow: parse → resolve → insert |
| `pkg/utils` | `expense-reporter/pkg/utils/` | Public utility functions (currency, date, format) |
| `config` | `expense-reporter/config/` | Configuration files and constants; `type-descriptions.json` (tracked, non-sensitive sidecar: expense-type → English description, merged into the classifier prompt — T-22) |
| `internal/classifier` | `expense-reporter/internal/classifier/` | Ollama classifier + `IsAutoInsertable` decision logic; `keyword_hint.go` (`KeywordHint` — advisory sibling of the gate: fires on model⊕keyword DISAGREEMENT at unambiguous specificity 1.00, A1/s72); `examples.go` (SelectExamples, KeywordIndex, tokenization); `loader.go` (LoadTrainingExamples, LoadFeedbackExamples, LoadKeywordIndex, MergeExamplePools) |
| `internal/taxonomy` | `expense-reporter/internal/taxonomy/` | Pure-input taxonomy domain types + loader + full-path routing (`path.go`); `descriptions.go` (`LoadTypeDescriptions`, `ApplyDescriptions` — type-description sidecar overlay, T-22) |
| `internal/config` | `expense-reporter/internal/config/` | Config struct + `Load()` + `ClassificationsFilePath()` + `ExpensesLogFilePath()` + `TaxonomyFilePath()` + `TypeDescriptionsFilePath()` |
| `internal/feedback` | `expense-reporter/internal/feedback/` | JSONL feedback logging: `Entry`, `GenerateID`, `Append`, `NewConfirmedEntry`, `NewManualEntry`; `ExpenseEntry`, `NewExpenseEntry`, `AppendExpense` (slim insert log → `expenses_log.jsonl`); `LoadExpenseIDCounts` (id-multiplicity ledger for `--resume`/dup warning — T-20) |
| `internal/appender` | `expense-reporter/internal/appender/` | Log-append path (WS-B): `ExpandAndAppend` (installment expansion → `expenses_log.jsonl`), `PredictEntryIDs` (same `expandEntries` step — T-20 resume drift guard) |
| `internal/parse` | `expense-reporter/internal/parse/` | **The T-41 parse boundary** — the ONE string→struct conversion: `ParsedExpense` (single stored `time.Time` date + `DateString()` = sole identity-bytes site), `Fields` (field-wise core; year ladder explicit > `--year` > `date_year` > most-recent-non-future, grace 0; BR thousands normalization), `ExpenseString` (semicolon wrapper, returns subcategory alongside). `YearSource` reports WHICH rung resolved the year (zero = `Unknown`, never a real rung); `validateRenderableYear` (permanent contract) + `validateYearNotBeyondCurrent` (provisional T-47 policy) both check the RESOLVED year, so one check covers every rung. `ErrInvalidDate`/`ErrInvalidValue` sentinels name the rejected FIELD (double-`%w`, so the cause survives the wrap — a caller that replaces it reports the wrong reason). Consumers: `add`/`correct` (slice 1), `auto` (slice 2), `batch-auto` (slice 3 — `classifiedRow` carries a `ParsedExpense`, so the classified CSVs emit the CANONICAL date that `review.ReadQueue` then hashes), `apply` (slice 4 — **T-41 complete**; `cmd.canonicalDate` retired, and `Date(dateStr, Options)` added as the date-only entry point for a caller whose other fields are already typed, with `Fields` delegating to it so one ladder serves every door). **`Value(valueStr)` (T-72, s73)** is the symmetric value-only door, added for `review.ReadQueue` — which had been calling `utils` directly and so never applied BR thousands normalization, making `writeClassifiedCSV`'s `1.234,56` a HARD error that killed the whole review step. `Fields` delegates to it, so normalization has exactly ONE call site |
| `internal/capture` | `expense-reporter/internal/capture/` | **The source-agnostic half of expense capture (T-75 step 1, s75).** `Message` (id, send time, flattened text, attachment) → `Classify` → `Outcome` in one of four classes — converted / rejected / receipt / ignored (plan D3: a non-conforming text is *rejected* only if it has a `;` or a money-shaped token, else it is conversation). **The parse is NOT reimplemented**: Converted means `parse.Fields` accepted the fields, so an emitted `item;DD/MM/YYYY;value` line is one batch-auto takes by construction, and the raw value token (`900,00/3`, `405,25 x4`) passes through unexpanded (D2). `ResolveDateField` owns the YEAR (D1: an explicit year wins verbatim, a 2-digit year expands to 20YY, a bare DD/MM resolves by proximity to the message's own day within `[-180,+7]`); the boundary then validates, which is what makes the 2026-12-28 → `03/01` → 2027 case a rejection. `Bucket()` refines a rejection by `errors.Is/As` (N fields / bad date / bad value), `Summarize` folds ids per bucket in stream order. **Step 2 (s75): `repair.go`** — D4's single-edit candidates (each `,`→`;`, each `/`→`;`, four fields → merge the first two with one space), accepted only when EXACTLY ONE parses through the same `parseLine` predicate AND its date sits inside the D1 window (the amendment: `Value("00")` is 0 with no error and year 1200 is a legal year, so `21/08/ 1200,00`'s comma edit was a second reading); D5 by construction — `Classify` parses as typed first and never repairs a line that parsed; `Outcome.Repair` carries the repaired text, `BucketRepaired` / `BucketAmbiguous` / `AmbiguousRepairError` report it. **Step 3 (s75): `output.go`, the PURE sink** — `MonthFiles` groups converted lines into `expenses-YYYY-MM.csv` by the RESOLVED date (D7), sorted by name with stream order kept inside each file, repaired lines carrying `   # repaired from: <as typed>`; `Rejects` renders the T-63 shape with `(message N, sent YYYY-MM-DD)` provenance. Both return values — every `os` call lives in the command, which is why the writers are testable without a temp dir. **D1 AMENDMENT (s75, step 5, user ruling): the `[-180,+7]` window now validates the resolved date on BOTH branches.** `resolveExplicitYear` took `sentAt` and rejects an explicit year outside it — the explicit year still wins the SELECTION (nothing re-resolves it), the window is a VALIDITY check on the outcome. Without it `Item; 21/08/1200; 50,00` converted and the command wrote `expenses-1200-08.csv`, exit 0 and no rejects file, because the boundary allows years 1..9999 and only looks forward — and **D5 means a line that parses as typed is never repaired, so it never met D4's window check**; that gap could only be closed here. ⚠️ **Neither T-75 gate could catch it**: the probe's FIRST residue branch labels it `expanded outside the comparison window`, a named cause, so `unexplained` stays 0 and the gate PASSES on a misfiled row. Measured before deciding: of 358 converted 2025 lines only 2 carry an explicit year, both 2-digit, **zero** 4-digit, **zero** outside the window — the unguarded branch is the one branch the corpus never exercises — and the probe characterization is GREEN on all 12 pinned numbers after the change. It INVERTED the test row `"explicit year outranks the timestamp even far outside the window"`. **D10 (s76, user ruling): `expandLists` — a message with >1 non-empty line, EVERY line 3-field-shaped (exactly two `;`), is N messages, one per line, each keeping the parent id/timestamp/attachment.** ⚠️ **The test is the SEMICOLON COUNT, not parseability**: the real message that forced it (2026-02, ten lines, `rejected: 21 fields`) has two `29/12/26` typos among ten good lines, so "every line parses" would have thrown all ten away; all-or-nothing on shape is what stops a one-expense message with a chatty second line being torn in half. `ClassifyAll` expands then classifies, so it is 1:1 for everything 2025 ever held (**measured: 0 multi-line messages in the whole 2025 export**). Because split lines keep the PARENT id, one id can now sit in TWO buckets, so `Summary` gained `Lines` / `Splits` / `Counts` and **bucket counts count LINES while the id list counts MESSAGES** — `Summarize` de-duplicates ids per bucket. Adapters import it, never the reverse. 88 subtests |
| `internal/telegram` | `expense-reporter/internal/telegram/` | **SOURCE ADAPTER for Telegram Desktop's `result.json` (T-75 step 1, s75)** — `ReadExport` / `ParseExport` → `[]capture.Message`, export order kept. Flattens `text` whether it is a plain string or an array of runs (strings mixed with `{"type","text"}` objects), maps `file` / `photo` to attachments (file wins), reads the zone-less timestamp in UTC so the sender's calendar day survives, drops `service` entries (group created…), and fails WHOLE on an unreadable entry, naming its id. Knows nothing about expenses. 26 subtests |
| `internal/review` | `expense-reporter/internal/review/` | Review command package: `ReadQueue` (**9**-field CSV reader — col 9 = `keyword_hint`, A1/s72; `auto_inserted` is `true`/`false` — the writer's own spelling, T-54; `QueueEntry.Installments` keeps the count parsed from the raw `total/N` token, T-21/s69 — it used to be discarded here, which is why `apply` recorded a 3× purchase as one row; **the value cell is parsed via `parse.Value`, NOT `utils` directly (T-72, s73)**, which is what fixed the thousands hard-error and what carries the T-64 multiplier in without teaching it twice), `BuildTaxonomy` (picker tree projected from **`config/taxonomy.json`**, S2/s68 — was the workbook Referência sheet, which had drifted 7 leaves from the file that actually routes; order preserved verbatim, never sorted), `Render` (placeholder injection), `TemplateHTML` (go:embed); types in `types.go`. **Reads no workbook** — `excel.LoadReferenceSheet` now has only WS-E dead callers |
| `harness` (module) | `github.com/leandror172/acceptance-harness` v1.0.0 | **External dep since T2 Session B** — the acceptance engine (Context, Scenario, Run, fixtures, FindModuleRoot/BuildBinary) + generic `verify` Then-closures. `test/harness/` no longer exists. Engine changes = a PR upstream + a version bump here |
| `test/actions` | `expense-reporter/test/actions/` | When-closures: RunClassify, RunAuto, RunBatchAuto, RunAdd; `runCommand` forwards `ctx.Env` |
| `test/expect` | `expense-reporter/test/expect/` | **DOMAIN** Then-closures (was `test/verify/`, renamed so the module owns `verify.*`): FeedbackFile*, ExpenseLogMatches, ClassificationsMatch, OutputFileHas*, SoftAccuracy, HTML*, WorkbookStructureMatches; ResumeSkipCount, DuplicateWarningCount (T-20); UnreviewableRowsReported (S1, s68) |
| `test/domain` | `expense-reporter/test/domain/` | Expense-specific test helpers the lean-core module excludes: RequireWorkbook + CopyWorkbookToWorkDir, SetupBinaryConfig, ExpenseFixtureConfig (`Raw` decode), ctx.Env accessors (DataDir/WorkbookPath) |
| `test/extern` | `expense-reporter/test/extern/` | External-service gates: RequireOllama (module core is LLM-free/network-free by design) |
| `test/fixtures` | `expense-reporter/test/fixtures/` | Fixture data per functional slice (classify-basic, batch-auto-basic, batch-auto-exclusions, batch-auto-feedback) |
<!-- /ref:go-structure -->

---

<!-- ref:testing -->
## Testing Conventions

- **Test runner:** `cd expense-reporter && go test ./...`
- **Framework:** `testing` package (stdlib) + testify (`assert`/`require`) — both in use
- **Unit test style:** testify preferred for new tests; existing stdlib tests left as-is
- **Coverage:** 263 top-level test functions / 647 including table-driven subtests, all green
  (counted 2026-07-21 via `go test ./... -v | grep -c '^=== RUN'`)
- **Pattern:** Each `internal/X/` has `X_test.go` with table-driven cases
- **Test data:** `expense-reporter/expenses.example.csv` — safe, no personal data
- **TDD approach:** Write failing test first, implement to pass (established pattern from Phases 1-4)

**Do NOT:**
- Use `testing.T.Fatal` where `t.Errorf` suffices
- Skip table-driven approach for new commands
<!-- /ref:testing -->

---

<!-- ref:classification -->
## Classification Data

See `data/classification/classification_algorithm.md` for the full algorithm description.
See `data/classification/reproducibility_guide.md` for how to reproduce results.

**Algorithm summary:** Hybrid rule-based + statistical scoring. Priority: precision over recall.
**Training corpus** (`training_data_complete.json`): **1,788** labeled expenses — 15 categories,
81 subcategories. By year: 2022:127 / 2023:853 / 2024:349 / 2025:459. Built 2026-06-20 by the
5.R4 historical extraction (2022–2025 workbooks deduped + merged into the original 694 corpus).
**Keyword dictionary** (`feature_dictionary_enhanced.json`): 229 keywords, 68 subcategories,
71 user-correction override rules.

⚠️ **The two artifacts are OUT OF SYNC.** The dictionary was never regenerated after 5.R4, so it
still describes the 694-era taxonomy: **13 of the corpus's 81 subcategories have no keyword entry
at all** and can never produce a keyword match. Do not quote a single "coverage" number for both —
they are different files with different scopes. This gap feeds the keyword-miss stratum that 5.R2
embedding retrieval targets ([ref:embedding-retrieval]).
**Layer 5 strategy:** Feature dictionary as system context + top-K few-shot examples per request.

**Key ref blocks for Layer 5 agents:**
- `ref-lookup.sh training-data-schema` — JSON schema for `training_data_complete.json` and `feature_dictionary_enhanced.json`; classify command input format
- `ref-lookup.sh confidence-thresholds` — auto-insert AGREEMENT gate (T-32) + legacy confidence scoring
- `ref-lookup.sh classification-overview` — executive summary + algorithm performance stats
<!-- /ref:classification -->

---

<!-- ref:training-data-schema -->
## Training Data Schema

### `training_data_complete.json` (1,788 labeled expenses — gitignored)

```
{
  "metadata": {
    "total_expenses": 1788,       // post-5.R4; was 694 pre-2026-06-20
    "unique_categories": 15,
    "unique_subcategories": 81,
    "by_year": {"2022": 127, "2023": 853, "2024": 349, "2025": 459},
    "extraction_date": "ISO 8601",
    "provenance": "free text — how this corpus was assembled"
  },
  "expenses": [
    {
      "id": 1,                    // sequential integer
      "item": "Diarista <person D>", // raw description as it appears in the spreadsheet
      "date": "2024-01-05",       // ISO 8601 (YYYY-MM-DD); source data is DD/MM/YYYY
      "value": 160.0,             // float, BRL
      "subcategory": "Diarista",  // ground truth label
      "category": "Habitação",    // parent category
      "source": "Filename.xlsx:SheetName",
      "year": 2024
    }
  ]
}
```

### `feature_dictionary_enhanced.json` (229 keywords, 68 subcategories — gitignored)

⚠️ Built from the **694-era** corpus and never regenerated after 5.R4. 13 of the training
corpus's 81 subcategories are absent from `category_mapping` entirely — those leaves cannot
produce a keyword match, and therefore cannot pass the T-32 agreement gate
([ref:confidence-thresholds]).

```
{
  "lexical_features": {
    "keywords": {
      "<lowercase_keyword>": {
        "frequency": 17,                    // occurrence count in training set
        "dominant_subcategory": "Diarista", // most common subcategory for this keyword
        "dominant_count": 17,               // how many times the dominant mapping occurred
        "specificity": 1.0,                 // dominant_count / frequency (1.0 = unambiguous)
        "idf": 3.652,                       // inverse document frequency (rarity signal)
        "subcategories": ["Diarista"]       // all subcategories this keyword appears in
      }
    }
  },
  "value_ranges": {
    "<subcategory>": { "min": 160.0, "max": 219.8, "mean": 176.19,
                       "median": 172.0, "q1": 160.0, "q3": 172.0, "count": 17 }
  },
  "category_mapping": {
    "<subcategory>": "<parent_category>"    // 68 entries; flat lookup
  },
  "user_corrections": {
    "<normalized_item_lowercase>": {
      "category": "Lazer",
      "subcategory": "Delivery",
      "full_item": "Delivery brod's"        // original casing
    }
  }
}
```

### Classify command input (for 5.2)

Three fields passed to the classifier:
- `item` — expense description string (free text, Portuguese)
- `value` — float (BRL)
- `date` — string DD/MM/YYYY (Brazilian format; normalize before lookup)

### How these files are used in Layer 5

| File | Role at classify time |
|------|----------------------|
| `training_data_complete.json` | Source of few-shot examples (top-K by keyword similarity, injected into prompt) |
| `feature_dictionary_enhanced.json` | Fast pre-filter: keyword lookup → candidate subcategory before calling Ollama; also value range plausibility check |
| `algorithm_parameters.json` | Threshold values (tracked) — HIGH ≥ 0.85, MEDIUM ≥ 0.50, LOW < 0.50. **These no longer gate anything** — they are the desktop-era scoring bands, retained as the design record. Auto-insert is decided by the agreement gate ([ref:confidence-thresholds]) |
<!-- /ref:training-data-schema -->

---

<!-- ref:confidence-thresholds -->
## Auto-Insert Gate (agreement gate — T-32)

**As of T-32, auto-insert is decided by keyword AGREEMENT, not confidence.** The
649-replay (session 52) measured that the model's self-reported confidence is
uninformative as a gate (the whole 0.85–0.95 band is a coin flip), so
`IsAutoInsertable` (`internal/classifier/decision.go`) no longer consults confidence.

A candidate is auto-insertable (`auto`/`batch-auto` append it without review) only when
ALL hold:
- a keyword matched the item (a specificity signal exists)
- the match is unambiguous (no tie at the top specificity)
- the top keyword specificity is maximal (`top_score >= 1.0`)
- the model's predicted subcategory equals the keyword's dominant subcategory (agreement)
- the subcategory is not in `auto_insert_excluded` (e.g. `Diversos`)

Everything else routes to review. Confidence is still emitted per candidate (and shown in
`--json`) but does NOT gate; `--threshold` on `batch-auto` is deprecated/ignored.

**Precision — MEASURED ABSOLUTELY (session 72): 94.1%.** On the first real close, 1 of the
17 gate-passing rows was corrected (5.9% error), against 50.0% for the 48 gate-refused rows —
an 8.5× discrimination on a full, unselected month. Report:
`.claude/b1-gate-precision-measurement.md`.

This supersedes the earlier "absolute precision is unmeasured" caveat, which rested on the
649 being a confidence-selected subset with 378 rows bypassing review UNLABELED. A **close**
labels everything, because `close-cycle.sh:220` feeds `review` the FULL `classified.csv` —
so every gate-passing row is in the queue with a recorded reviewer action.

**Read 94.1% as an UPPER bound, and note it does not by itself unblock WS-D.** n = 17, one
month, one reviewer — a single further error drops it to 88.2%. Confirms on auto-inserted
rows are weak-positive: `review.go` excludes them from the `needsReview` count and the page
shows them as already handled, so a reviewer may have skimmed. The 1 correction is a hard
negative. The ~95% figure from the 649-replay remains a same-sample RELATIVE validation
(shipped predicate == measured predicate) and is a separate claim.

**Legacy (pre-T-32) confidence scoring** — retained as the classifier's internal scoring,
no longer a gate. Source `data/classification/algorithm_parameters.json`: HIGH ≥ 0.85,
MEDIUM ≥ 0.50, LOW < 0.50; fallback (no keyword + inconclusive value range) → `Diversos`,
confidence `0.30`. Feature weights: keyword_match 0.50, semantic_similarity 0.30,
value_proximity 0.20.
<!-- /ref:confidence-thresholds -->

---

<!-- ref:classification-overview -->
## Classification Overview

**Dataset:** 1,788 labeled expenses, 2022–2025, Brazilian Portuguese descriptions, BRL values.
**Taxonomy (corpus):** 15 categories → 81 subcategories.
**Keyword dictionary:** 229 terms (lowercase, with IDF and specificity scores) covering only
68 subcategories — 694-era, not regenerated after 5.R4. See the warning in
[ref:training-data-schema].
**User corrections:** 71 override rules (exact normalized item → forced subcategory).

⚠️ The algorithm description below is the **desktop-era hybrid scorer**, retained as the design
record. It is NOT the shipped auto-insert path: since T-32 the gate is keyword agreement, not a
confidence band — see [ref:confidence-thresholds].

**Algorithm (hybrid, priority order):**
1. User correction lookup (exact match on normalized item string) → confidence 1.0
2. Keyword match scoring (TF-IDF weighted, specificity boosted) → up to 0.85
3. Value range plausibility (Gaussian kernel against per-subcategory IQR) → modifier ±0.20
4. Fallback → Diversos, 0.30

**Key preprocessing:**
- Lowercase, remove special chars, normalize whitespace
- Do NOT strip accents (Portuguese keywords retain accents: `gás`, `habitação`)
- Minimum word length: 2 chars

**Performance (from Desktop-era analysis):**
- High-specificity keywords (specificity = 1.0): unambiguous mapping, no LLM needed
- Ambiguous keywords (e.g., `va` maps to 5 subcategories): value range resolves most cases
- Remaining ambiguous: few-shot LLM call via Ollama
<!-- /ref:classification-overview -->

---

<!-- ref:feedback-system -->
## Feedback System & Corrections

**File:** `docs/FEEDBACK_SYSTEM.md`

The feedback system logs classification decisions to `classifications.jsonl` for training, tuning, and audit purposes.

**Key ref blocks:**
- `[ref:feedback-entry-structure]` — JSON entry format, status values (confirmed/corrected/manual)
- `[ref:feedback-sources]` — Which commands create which feedback entries (add, auto, batch-auto)
- `[ref:feedback-training]` — How feedback becomes training examples for next run
- `[ref:feedback-correction-workflow]` — Correction workflow via `correct` command (status="corrected"), closed in Layer 5.9
- `[ref:feedback-file-path]` — Configuration and file resolution logic
- `[ref:feedback-cold-start]` — Behavior when classifications.jsonl doesn't exist

**Quick summary:**
- `add` command → `status="manual"` (user entered, no model)
- `auto`/`batch-auto` commands → `status="confirmed"` (model auto-appended past the agreement gate)
- `correct` command (`cmd/correct.go`) → `status="corrected"` — requires a prior entry to override
- `apply` (review-UI ingestion) → `status="confirmed"` or `"corrected"` per the reviewer's action;
  entries it writes carry `model: "review"`

**Closed in Layer 5.9** — the old "no way to create `corrected`" feature gap no longer exists.
In the live log, `apply` is in fact the dominant producer of both statuses.

<!-- /ref:feedback-system -->

---

## Retrieval Strategy Docs

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/retrieval-strategy.md` | ✅ Yes | High-level retrieval pipeline: cascade diagram, token budget, data source strategy — [ref:retrieval-strategy], [ref:retrieval-token-budget] |
| `data/classification/tfidf-retrieval.md` | ✅ Yes | TF-IDF retrieval layer: existing artifacts, implementation approach, decision criteria — [ref:tfidf-retrieval] |
| `data/classification/embedding-retrieval.md` | ✅ Yes | Embedding/RAG retrieval layer: Ollama API, vector store, multilingual considerations — [ref:embedding-retrieval] |

---

## Classification Data Files

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/classification_algorithm.md` | ✅ Yes | Algorithm description — no personal data |
| `data/classification/reproducibility_guide.md` | ✅ Yes | How to reproduce results |
| `data/classification/algorithm_parameters.json` | ✅ Yes | Parameter values — no personal data |
| `data/classification/AUTO_CATEGORY_README.md` | ✅ Yes | Overview of the auto-categorization effort |
| `data/classification/FINAL_SUMMARY.md` | ✅ Yes | Final analysis summary |
| `data/classification/classification_reasoning.md` | ✅ Yes | Reasoning behind category assignments |
| `data/classification/research_insights.md` | ✅ Yes | Research findings |
| `data/classification/llm_reasoning_meta.md` | ✅ Yes | LLM reasoning analysis |
| `data/classification/feature_dictionary_enhanced.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/feature_dictionary.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/training_data_complete.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/training_data.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/final_classifications.json` | ❌ Gitignored | Classification results |
| `data/classification/*.csv` | ❌ Gitignored | Personal expense CSVs |
| `data/classification/similarity_matrix.json` | ❌ Gitignored | Computed similarity data |
| `data/classification/vector_representations.json` | ❌ Gitignored | Feature vectors |
| `data/classification/statistical_summary.json` | ❌ Gitignored | Statistical results |
| `data/classification/confusion_analysis.json` | ❌ Gitignored | Per item analysis (may contain real descriptions) |
| `data/classification/extraction-aliases.json` | ❌ Gitignored | 5.R4 source→taxonomy alias map (taxonomy-label fragments); loaded by `extract_old_workbooks.py` |

---

## Vision & Planning Docs (docs/)

| File | Content |
|------|---------|
| `docs/expense-classifier-vision.md` | End-to-end Layer 5–6+ vision: user scenario, domain boundaries, iterative build plan (Phases 0–5), technical notes on structured output, persistence, queue. **Primary reference for understanding scope and architecture.** |
| `docs/expense-classifier-data-inventory.md` | Inventory of all auto-category analysis artifacts (feature dict, training data, confusion analysis, etc.) and their priority/role at build time. Also documents expense-reporter architecture as of Layer 5 start. |
| `.claude/plans/review-page-behavior-audit.md` | Test plan for the review page: expected-behavior spec (pre-fill rule, filters, 12 keyboard bindings, badges, export), the 22-row scenario matrix, and the browser procedure. |
| `.claude/review-page-audit-report.md` | **Findings of that audit (session 77).** Filters are correct; the Type pre-fill is broken three ways since T-05 (`47a6dff`) and a reload silently downgrades finished rows to skipped while the header still counts them reviewed. |
| `.claude/fixtures/review-page-audit/` | 22-row `classified.csv` + README covering every pre-fill branch, badge and parse path of the review page, with the run procedure. Manual browser check — the page has no automated test (T-61). |

---

## Historical Docs (docs/archive/)

Desktop-era planning documents — read for context, do not modify.

| File | Content |
|------|---------|
| `ARCHITECTURE_PLAN_Expense_Automation.md` | Original automation architecture plan |
| `ARCHITECTURE_PLAN_Go_Implementation.md` | Go implementation architecture |
| `AUTO_CATEGORIZATION_PROMPT.md` | Original auto-categorization prompt design |
| `DECISIONS_FINALIZED.md` | Finalized design decisions |
| `GO_vs_PYTHON_Comparison.md` | Language selection rationale |
| `PRE_IMPLEMENTATION_CHECKLIST.md` | Pre-implementation verification |
| `PRE_PHASE1_VERIFICATION_CHECKLIST.md` | Phase 1 checklist |
| `QUICK_REFERENCE_Implementation_Summary.md` | Implementation quick reference |
| `SHEET_NORMALIZATION_PROMPT.md` | Sheet normalization prompt |
| `USAGE_CALCULATION.md` | Usage/quota calculations |
| `DOCUMENTATION_INDEX.md` | Original doc index (superseded by this file) |
| `PROJECT_COMPLETE.md` | Phase completion summary |
| `BATCH_IMPORT_COMPLETE.md` | Batch import feature completion |
| `PHASE1-4_COMPLETE.md` | Combined phase completion doc |
| `FEATURES_INVESTIGATION_REPORT.md` | Feature investigation results |
| `FEATURE_INVESTIGATION.md` | Feature investigation notes |
| `TESTING.md` | Testing documentation (current: expense-reporter/README.md) |

---

## Per-Folder Memories

| Folder | Files | Content |
|--------|-------|---------|
| `.memories/` | QUICK.md, KNOWLEDGE.md | Repo-wide status, architecture, cross-repo relationships |
| `expense-reporter/.memories/` | QUICK.md, KNOWLEDGE.md | Go app structure, command hierarchy, batch pipeline, config |
| `expense-reporter/internal/apply/.memories/` | QUICK.md, KNOWLEDGE.md | Review UI ingestion, feedback + expense log output, identity-is-derived rule; installment threading + the required `installments` field (T-21); KNOWLEDGE carries the known limitations incl. the new partial-series hole |
| `expense-reporter/internal/classifier/.memories/` | QUICK.md, KNOWLEDGE.md | Few-shot algorithm, prompt architecture, empirical findings |
| `expense-reporter/internal/feedback/.memories/` | QUICK.md, KNOWLEDGE.md | Two-file JSONL structure (classifications.jsonl + expenses_log.jsonl), GenerateID join key, type persistence gap (Plan A) |
| `expense-reporter/internal/parse/.memories/` | QUICK.md | T-41 parse boundary: contract (single time.Time + DateString identity bytes), year ladder + grace 0, BR thousands rule, migration status (**all six migrated s67**), and — added s76 — the **unguarded past year as a LIVE defect** rather than a policy footnote: `Date("29/12/25")` → year 0025 with no error, invisible afterwards, and it blocks T-63's rejects re-import |
| `expense-reporter/internal/capture/.memories/` | QUICK.md | **T-75 capture core (added s76).** Why adapters import it and never the reverse; D5 as a hard precondition; D4 exactly-one-reading; **D1's window validating BOTH date branches and why it must not be simplified away**; D10 splitting by SHAPE not parseability, with bucket counts = LINES and ids = MESSAGES; D2 raw installment tokens; and the standing warning that every decision has a SECOND implementation in `t75_expected_buckets.py` that must be amended with it |
| `expense-reporter/internal/review/.memories/` | QUICK.md, KNOWLEDGE.md | HTML review page builder, taxonomy re-derivation from CSV, localStorage state, type-aware UI already in place |
| `expense-reporter/internal/taxonomy/.memories/` | QUICK.md, KNOWLEDGE.md | Full-path identity, bare-name routing + ambiguous fallback, classifier/generator taxonomy disconnect (Plan B two-tier routing) |
| `expense-reporter/test/.memories/` | QUICK.md, KNOWLEDGE.md | BDD harness design, fixture format, soft/hard assertions |
| `mcp-server/.memories/` | QUICK.md, KNOWLEDGE.md | Thin wrapper decisions, binary resolution, data-dir fix |
| `expense-reporter/internal/excel/.memories/` | QUICK.md, KNOWLEDGE.md | Reference-sheet columns, boundary detection, and the workbook structural map (sheet families, palette, fill-down vs merge, separators, cross-sheet wiring) |
| `expense-reporter/cmd/workbook-inspect/.memories/` | QUICK.md | workbook-inspect tool: usage, output schema, classifier + row-fill design |
| `expense-reporter/internal/generate/.memories/` | QUICK.md, KNOWLEDGE.md | generator entry points, sheet-order rule, re-freeze discipline, excelize gotchas; KNOWLEDGE: build order, accumulating row counter, cross-fixture oracle coupling, column layout, block sizing, income 3-level model |
| `.claude/workbook-dump/` | *.json (gitignored) | Raw JSON dumps from workbook-inspect; input for Layer 2 visual annotation and Layer 3 spec |

---

## READMEs

| File | Content |
|------|---------|
| `README.md` | Repo root — project overview, components, quick start |
| `expense-reporter/README.md` | Full CLI documentation — all commands, classification system, testing |
| `expense-reporter/test/README.md` | Acceptance test harness architecture, fixtures, verifiers |

---

## Tools

| Tool | Path | Purpose |
|------|------|---------|
| `sonnet-max-subagent.js` | `.claude/workflows/sonnet-max-subagent.js` | Workflow harness: run ONE Sonnet 5 subagent at max/xhigh effort (the effort knob the Agent tool lacks). `Workflow({name:'sonnet-max-subagent', args:'<prompt>'})` or `args:{prompt,effort}` |
| `impl-opus-med` agent | `.claude/agents/impl-opus-med.md` | Opus medium-effort implementation subagent (TDD, directed reading, local-model delegation w/ per-session persona override; copied from latent-topic-graph, adapted session 54) |
| `impl-opus-xhigh` agent | `.claude/agents/impl-opus-xhigh.md` | Opus xhigh-effort implementation subagent for the hardest TDD tasks (multi-file features, invariant-heavy/advisor-contracted designs); same contract as impl-opus-med, advisor rule made availability-conditional (session 60) |
| `close-cycle.sh` | `.claude/tools/close-cycle.sh` | Runs a real monthly close with an undo + 2 reconciliations. `prepare <csv> <year>` snapshots both logs, runs `batch-auto` + `review`, asserts no row is lost; `finish <run-dir> <reviewed.json>` asserts the export is not stale, applies, asserts **appended rows == Σ installments over NEW entries** (T-21 on real data), generates the workbook; `restore <run-dir>` undoes the whole close — and **since s74 it REFUSES a snapshot taken before the id migration**, which would otherwise put year-less ids back into both logs and silently revert it (no error, no lost row, just a join that quietly stops working). Detected by CONTENT (are the snapshot's ids self-consistent?) rather than by the run dir's date, because a copied or renamed dir carries a misleading timestamp; override with `CLOSE_CYCLE_ALLOW_PREMIGRATION_RESTORE=1`. Mutation-verified in four directions **including the must-pass case** — with `die()` on the failure path, a guard that refuses everything is indistinguishable from one that works. The check catches row loss incidentally too: the August snapshot holds 2073 rows against today's 2163. **TWO snapshots, different jobs — do not collapse them:** `logs-before/` (pre-batch-auto) is the restore point; `logs-after-batch/` is the reconciliation baseline, because apply's dedup set is the classifications log *after* batch-auto wrote its auto rows. Using `logs-before/` for both cancels out only while the reviewer confirms every auto-inserted row, and fires spuriously the moment one is **skipped** — mutation-verified in both directions. **Two phases on purpose** — the review step is a human in a browser and `exportReviewed()` has no headless path; synthesizing `reviewed.json` would be the T-54/T-61 shape. All assertions mutation-verified (s70) |
| `resume.sh` | `.claude/tools/resume.sh` | Session-start context summary |
| `ref-lookup.sh` | `.claude/tools/ref-lookup.sh` | Resolve [ref:KEY] tags |
| `rotate-session-log.sh` | `.claude/tools/rotate-session-log.sh` | Archive old session log entries |
| `reconstruct-csvs.py` | `.claude/tools/reconstruct-csvs.py` | Reconstruct classified/review CSVs from batch-auto log + original input CSV (line-matched) |
| `lookup-category.py` | `.claude/tools/lookup-category.py` | Look up canonical category for one or more subcategories (`<sub> [...]` or `--list`) |
| `backfill-type.py` | `.claude/tools/backfill-type.py` | Backfill expense type into log files from reviewed.json exports (Plan A Phase B-fill recovery) |
| `migrate_ids.py` | `.claude/scratch/migrate_ids.py` | **One-off, EXECUTED s74** — re-derived every `id` in both live logs from its own row, retiring the year-less-id exception ([ref:active-decisions] "The one deliberate exception is RETIRED"). Rewrites `id`/`date` **surgically inside each original line** rather than re-serialising: the two logs mix compact (Go `json.Marshal`) and spaced (Python `json.dumps`) formatting, so a re-serialise would cosmetically churn whichever half does not match and destroy any byte-comparison. **Refuses to write anything** — not a partial result — when a row is unresolved OR when its id is ambiguous across years (38 ids carried two dates differing only by year; a `{id: date}` dict silently keeps the last, and the join-pairing check is BLIND to it because before and after use the same wrong lookup). Both guards mutation-verified. Re-runnable and idempotent |
| `t75_oracle_probe.py` | `.claude/scratch/t75_oracle_probe.py` | **Read-only (session 75). THE T-75 VALIDATION GATE — exit code 1 on any SUSPECT residue row or an insensitive perturbation.** Two side-A sources: `--source converter` (default since step 4) runs `expense-reporter telegram-import` into a temp dir (or reads `--out-dir`) and expands each month-file line's installments with the Python port of `currency.go` — the port shares no code with the Go parser, so a divergence inflates the residue instead of hiding; `--source naive` is the step-1 throwaway conversion that reproduces the viability report. ⚠️ **The two modes DIVERGE by design since the s75 step-5 D1 amendment**: naive is a FROZEN reproduction of the viability numbers and still accepts an explicit out-of-window year, converter follows the amended rule and rejects it. That is not drift to repair — naive's job is to keep step 1 reproducible, and it is the control that isolates "did the converter change?" from "did the probe change?" Both were GREEN after the amendment (12/12 and 11/11) because the 2025 corpus contains zero 4-digit explicit years; on a corpus that has some, expect naive and converter to differ HERE and nowhere else. Multiset-matches `(date,value)` against the live log, classifies the residue by named cause — tails / outside window / same-day count / and, by ITEM lookup (all tokens ⊆ one log item), unknown / omission / **SUSPECT** (same item, same value within 7 days or same date: converter error or human edit) — checks installment-series completeness, and ends with the perturbation. 2025 result in converter mode: 358 lines → 389 rows, matched 346, 0 SUSPECT, gate PASS. ⚠️ Match on the multiset, never a set and never on item — s74's 8 same-day duplicates and s72's `San michel` x3 both die in a set |
| `t75_probe_characterize.py` | `.claude/scratch/t75_probe_characterize.py` | **Characterization of the probe (s75, `characterize-first`)** — runs the probe as a subprocess and compares eleven parsed numbers against a pinned set per `--source`: the report's step-1 numbers for `naive`, the step-4 numbers for `converter`. Green on the unmodified probe BEFORE its source changed (`0626c3b`), and proven able to fail (a flipped pin → 1 mismatch, exit 1). Run both after touching the probe; a drift here is a decision to record, never noise |
| `t75_expected_buckets.py` | `.claude/scratch/t75_expected_buckets.py` | **T-75 step-1 gate, half 1 (s75)** — writes a per-message-id EXPECTED bucket as a TSV (`id  class  bucket  reason`; ids and labels only, never text). ⚠️ **It is an INDEPENDENT re-implementation of D1 + D4 + D5 from the plan TEXT** (only the value parser is borrowed, from the probe's port of `currency.go`) — that independence is not incidental, it is the whole mechanism: it is what predicted the D4 defect before Go ran, and it means a bug shared with the Go side has to be a bug in the PLAN. An earlier draft of this row said the expectation was "the probe's verdict and not a second opinion", which describes the pre-rewrite script and inverts its actual purpose. **Therefore it is the SECOND implementation of every plan decision it re-implements, and must be amended whenever one changes** — done for the s75 step-5 D1 window amendment and again for D10's `expand_lists` (s76); skipping the first would have made the oracle disagree with a correct binary on the first 2026 run, with "the Go must be wrong" as the natural misreading. Since D10 it writes **one row per LINE**, all under the parent id, so an id may occur several times and in more than one bucket. Also prints the date/value shape census (which is how the two `DD/MM/25` messages were found). `python3 t75_expected_buckets.py OUT.tsv`; `EXPORT_JSON` overrides the export path |
| `t75_gate_compare.py` | `.claude/scratch/t75_gate_compare.py` | **T-75 step-1 gate, half 2 (s75)** — diffs `telegram-import … --dry-run` output against that TSV PER ID (converted = the complement of every listed id; the total and converted lines are cross-checked), exit 1 on any difference. **`--perturb` flips one expected label and REQUIRES exactly one per-id difference** — the s74 rule: a comparison is evidence only once it has been seen to fail; it now targets a line whose id has exactly ONE line, so the flip is guaranteed to change that id's label set. **Rewritten for D10 (s76): the unit is a LINE, so it compares (a) per id, the SET of non-converted labels, (b) per bucket, the report's LINE count against the TSV's, (c) the messages line against the distinct-id count.** ⚠️ **One thing got weaker and the count check is what covers it:** `converted` still lists no ids (it is the complement), so for a split message whose lines disagree, converted MEMBERSHIP is only checked through the aggregate count. Results: **2025 393/393, 2026-02 39/39**, perturbation moves each by exactly 1. Run both on a fresh export BEFORE trusting the converter on it |
| 5.R4 extraction scripts | `.claude/scratch/{extract_old_workbooks,dedup_corpus,build_corpus,build_logs}.py` | One-off (session 35): 2022–2024 workbooks → deduped corpus + per-year logs. Alias map externalized to gitignored `extraction-aliases.json`. |
| WS-A.3 merge script | `.claude/scratch/merge_year_logs.py` |
| T-14 benchmark harness | `.claude/scratch/t14-benchmark/{build_sample,run_benchmark,score}.py` + `ood.jsonl` (sample/results JSONLs gitignored — real expense data). Sample builder, resumable runner (drives `classify --json`; `--prefix`/`--extra-args`), scorer (accuracy/leakage/calibration/OOD). | One-off (session 37): merge per-year `expenses_log-{2022,2023,2024}.jsonl` + base 2025 → one `expenses_log-allyears.jsonl`, rewriting `DD/MM`→`DD/MM/YYYY`. Output gitignored. Verified byte-identical (excl. manifest source) vs per-year `generate-workbook --year N` for all 4 years. |
| 649-replay harness (T-23 gate + 5.R1) | `expense-reporter/internal/classifier/replay_{retrieval,model}_test.go` (`//go:build replay`) + `.claude/scratch/replay-649/{FINDINGS,FINDINGS-model,FINDINGS-nn,FINDINGS-5r2}.md` + `analyze*.py`/`compare_think.py`/`nn_precondition.py` | Session 52: replays the 649 real labels through production retrieval + classifier (LOO). Verdict: confidence dead as gate, specificity+agreement is the gate, 5.R2 (not 5.R1) is the lever. Session 53: T-31 NN precondition on the 160 misses → GO for 5.R2 (hit@5 62–65% multilingual; arctic-embed2 practical pick). Raw `*.jsonl` gitignored (real data). |
| A1 keyword-hint probe | `expense-reporter/internal/classifier/replay_keyword_hint_test.go` (`//go:build replay`) | Session 72: drives the real `MatchStrength` over a `reviewed.json` export to size the review-UI keyword hint. Paths env-overridable (`REPLAY_REVIEWED_JSON`, `REPLAY_DATA_DIR`, `REPLAY_HINT_OUT`); emits per-row JSONL beside the input. **Measures, never fails on data** — a zero-fire result is a finding. Report: `.claude/a1-keyword-hint-precheck.md` |
| session-handoff skill | `.claude/skills/session-handoff/SKILL.md` | End-of-session tracking workflow |
