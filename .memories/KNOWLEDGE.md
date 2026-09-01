# expense-reporter repo — Knowledge (Semantic Memory)

*Repo-wide accumulated decisions. Read on demand by agents.*

## Project Identity (2026-02)
Go CLI that reads CSV bank exports, classifies expenses using local LLMs (Ollama),
and inserts them into an Excel budget workbook. Built for personal finance automation
with Brazilian financial conventions (BRL, DD/MM/YYYY, comma decimals).
**Rationale:** Automates a tedious manual process — copying bank statement lines into a
categorized Excel budget. Local LLMs keep financial data private.
**Implication:** All ML inference is local (Ollama). No cloud APIs for classification.

## Architecture — Pipeline Layers (2026-02 → 2026-03)
The expense processing pipeline has distinct stages, each handled by a Go package:
1. **Parse** (`internal/parser`) — semicolon-delimited input → `models.Expense` struct
2. **Resolve** (`internal/resolver`) — fuzzy-match subcategory against reference sheet
3. **Classify** (`internal/classifier`) — LLM-based categorization with confidence scores
4. **Decide** (`internal/classifier/decision.go`) — keyword-agreement gate + exclusion list → auto-insert or review (T-32; replaced the confidence threshold)
5. **Append** (`internal/appender`) — installment expansion → append to `expenses_log.jsonl`
6. **Log** (`internal/feedback`) — append to `classifications.jsonl` and `expenses_log.jsonl`
7. **Generate** (`internal/generate`) — build the Excel workbook FROM the logs (`generate-workbook`)

**Post-pivot note (session 36):** the JSONL logs are the single source of truth and
`generate-workbook` is the **only** workbook writer. The old direct-insert path
(`internal/workflow` + the `internal/excel` write side, reached via plain `batch`) is retired
but not yet deleted — removal is WS-E, which also takes `utils.ParseDate` with it. Do not treat
that path as the live pipeline.
**Rationale:** Each stage is independently testable. The pipeline can stop at any stage
(e.g., `classify` stops at step 3; `auto` goes through step 6; `batch-auto` adds CSV I/O).
**Implication:** New features (like TF-IDF retrieval) slot into the pipeline at a specific
stage without affecting others.

## Installment Handling (2026-03)
Expenses with notation like "99,90/3" are expanded into 3 monthly installments.
Installments crossing the year boundary produce rollover rows written to `rollover.csv`.
**Rationale:** Brazilian credit card statements show installment totals; the budget needs
individual monthly entries.
**Implication:** Batch processing must track original→expanded index mapping for error reporting.

## Classification Strategy — Three-Layer Cascade (2026-03)
1. **Layer 1 (keyword):** Tokenize item → look up in feature dictionary → find dominant
   subcategory by specificity score. Implemented in `classifier/examples.go`.
2. **Layer 2 (TF-IDF):** RULED OUT (5.R1, session 52) — the 649-replay showed all retrieval
   misses are zero-lexical-overlap; TF-IDF cannot bridge them.
3. **Layer 3 (embedding):** **BUILT + ADOPTED, default-on** (5.R2, session 54; PR #45 merged
   session 57). On a keyword miss: embed the item (arctic-embed2) → cosine top-5 → inject as
   few-shot. Per-model JSONL disk cache; degrades to nil on any failure. Implemented in
   `internal/classifier/embedding{,_retriever,_fallback}.go`; `Config.EmbedModel` /
   `Config.NoEmbedRetrieval`, zero values = ON. Measured A/B on the miss stratum (n=80 unique):
   full-path 18.8% → 52.5% no-think. T-31 precondition (session 53) had shown NN hit@5 62–65%,
   multilingual models only.
Each layer feeds few-shot examples to the LLM prompt, improving classification accuracy.
**Rationale:** Empirical finding — LLM resolves multi-word context ("VA compras") better
than keyword specificity alone, but keywords select which few-shot examples to inject.
**Implication:** The keyword layer is a retrieval mechanism, not a classifier itself.

## Auto-Insert Gate — Agreement (T-32; was Confidence 2026-03)
`IsAutoInsertable` (`decision.go`) auto-inserts only when the model's predicted subcategory
AGREES with an unambiguous, maximum-specificity keyword match, and the subcategory is not
excluded. Confidence NO LONGER gates — the 649-replay (session 52) proved it uninformative
(the whole 0.85–0.95 band is a coin flip). All other rows route to review.
- Excluded subcategories (e.g., "Diversos") are never auto-inserted regardless of the gate.
- The ~95% subcat precision from the 649 is a same-sample RELATIVE validation (shipped
  predicate == measured predicate) on a confidence-selected subset — a separate claim from
  the one below, and still not an absolute guarantee.
- **ABSOLUTE precision MEASURED (session 72): 94.1%.** On the first real close, 1 of 17
  gate-passing rows was corrected (5.9% error) against 50.0% for the 48 gate-refused rows —
  an 8.5× discrimination on a full, unselected month. The 649 could not show this because
  378 of its 725 rows bypassed review unlabeled; a CLOSE labels everything, since
  `close-cycle.sh:220` feeds `review` the FULL `classified.csv`, so every gate-passing row
  carries a recorded reviewer action. Report: `.claude/b1-gate-precision-measurement.md`.
- **WS-D stays held regardless.** n=17 (one further error → 88.2%), one month, one reviewer,
  and confirms on auto-inserted rows are weak-positive because the page shows them as
  already handled and `review.go` omits them from the `needsReview` count — read 94.1% as an
  UPPER bound. What the measurement DID kill is the "nobody sees 25% of rows" argument: they
  were all reviewed, so the real defect is that a reviewer's correction cannot reach the
  expense log (T-65).
**Rationale:** confidence was measured dead, so the gate switched to keyword agreement
(specificity + model⊕keyword concurrence). "Diversos" exclusion still guards the vaguest leaf.
**Implication:** Exclusion list is in `config.json`. `--threshold` on `batch-auto` is deprecated.

## Feedback Loop (2026-03)
Two JSONL files persist classification results:
- `classifications.jsonl` — full classification context (predicted vs actual, model, status)
- `expenses_log.jsonl` — slim insert log (item, date, value, subcategory, category, hash ID)
Both share a sha256[:12] ID for cross-file correlation.
Confirmed/corrected entries from `classifications.jsonl` are loaded back as few-shot examples,
creating a self-improving feedback loop.
**Rationale:** The system learns from corrections without retraining. Corrected examples get
highest priority in few-shot selection (SourceCorrected > SourceTraining > SourceConfirmed).
**Implication:** Classification accuracy improves with use as the feedback pool grows.

## MCP Integration (2026-03)
Python MCP server (`mcp-server/`) wraps the Go binary as two tools: `classify_expense`
and `add_expense`. Uses FastMCP, calls Go binary as subprocess with `--json` flag.
**Rationale:** MCP provides a standard protocol for Claude Code to invoke the classifier.
The Go binary remains the single source of truth for all logic.
**Implication:** The Python layer is intentionally thin — no classification logic, no state.
Binary path is resolved at server startup (lifespan context).

## Cross-Repo Relationships (2026-03)
- **LLM infrastructure** (`/mnt/i/workspaces/llm/`) — Ollama personas, MCP bridge server,
  platform documentation. Classification models are defined as personas there.
- **Web research** (`/mnt/i/workspaces/web-research/`) — Independent project. Shares the
  `.memories/` convention and MCP-as-integration-layer pattern.
**Rationale:** Each repo owns its domain. Communication happens through MCP, not shared code.
**Implication:** Model configuration (persona definitions, prompt templates) lives in the
LLM repo; this repo only references model names.

## Key Empirical Findings (2026-03)
- LLM resolves multi-word context better than keyword specificity — "VA compras" → 100%
  despite "va" having specificity=0.36 in feature dictionary
- Fallback category "Diversos" at high confidence is a real risk — blocked via exclusion list
- `Transporte` appearing as subcategory at 90% in Uber test case — taxonomy oddity, not urgent
- Cold-start Ollama timeouts are normal — first-call timeout = retry, not a rejection

## Milestone Log (consolidated from QUICK.md, 2026-07-01)
- **Sessions 26–29 (PR #27) — workbook generator.** Mapping L1–L3 → spec v2 → Phase A/B
  convergence (user-blessed data-bearing golden master) → Phase G: `internal/inspect`
  (dump core), `internal/generate` + `generate-workbook` command; acceptance-first with
  oracle-frozen dumps (deterministic, no Ollama). Review follow-ups: English identifiers,
  SOLID extraction, hardcoded sheet-order bug fixed (registry `sheetOrder`). Scratch
  builder SUPERSEDED. Sessions 30–31: behavior-preserving internal refactor (styles
  vocabulary, shared helpers, unified block sizing) — see generate/.memories.
- **Sessions 33–35 — Plan A (T-05) + Plan B (T-04).** Expense type persisted end-to-end;
  `ExpenseSheet`→`ExpenseType` rename + JSON migration; generator two-tier routing
  (full-path for typed, transitional bare-name fallback, NFC keys). 5.R4 historical
  extraction: 2022–2025 old workbooks → corpus 694→1788 + per-year expense logs.
- **Session 36 — pivot decided:** retire workbook insertion, keep only generation. WS-0
  diff validated the premise (expenses reproduce; income was the sole gap); WS-0b
  extracted historical income.
- **Session 37 — WS-A/T-11 done** (branch `chore/income-extraction-tooling`): `parseDate`
  accepts `DD/MM`+`DD/MM/YYYY`; year filtering in `scanEntries`; throwaway
  `merge_year_logs.py` → one multi-year log, byte-identical gate passed all years. Income
  decisions locked (3-level symmetric income, separate `--income-entries`, signed values).
  Currency formatting confirmed a no-op (generator already numeric + `R$ #,##0.00`).
  Open: promote merged log to canonical + retire per-year split (user's call).
- **Session 38 — WS-C income route done** (model→loader→router→generator).
- **Sessions 41–42 — T-13:** classifier predicts full path via grammar-enforced 112-enum;
  default model `my-classifier-q3`; acceptance repairs (see session42-postmortem).
- **Sessions 43–44 — WS-B done:** batch-auto (slice 3) and apply (slice 4) →
  log-append via `appender.ExpandAndAppend`; rollover.csv retired; pre-flights + failure
  honesty. Next: WS-D (retire fallback, T-09), WS-E (delete dead insert code).
- **Sessions 58–59 — T2 harness extraction done.** The acceptance engine left this repo:
  `github.com/leandror172/acceptance-harness` (public, MIT; v0.1.1 at migration time,
  **v1.0.0 tagged session 60** — that is the version `go.mod` pins today) now provides
  Context/Scenario/Run, fixture plumbing, BuildBinary, and the generic `verify.*` Then
  assertions to both this repo and career-search's `roles` CLI. Session B (59) migrated
  expenses: `test/harness/` deleted; local `verify` → `test/expect/` (the module owns the
  `verify` name — scenarios import both); expense-shaped helpers → `test/{domain,extern}/`;
  `Context.{DataDir,WorkbookPath}` → `ctx.Env` accessors. Gate: the full acceptance roster
  is byte-identical pre/post (44 pass / 5 fail), the 5 reds being pre-existing bugs filed
  separately. Details → `expense-reporter/test/.memories/KNOWLEDGE.md`.
  **Rationale:** a second consumer is the only real test of a "domain-agnostic" claim, and
  two copies drift — the module's own unit tests immediately caught a latent `Run` bug that
  had been sitting in both copies.
  **Implication:** engine changes are now a PR upstream + a version bump here, which is the
  point: the boundary can no longer erode quietly.
- **Sessions 60–61 — T-20, T-35/T-36.** T-20: `batch-auto --resume` + always-on duplicate
  warning (id-count ledger `LoadExpenseIDCounts`, `PredictEntryIDs` pre-LLM skip, partially
  logged installment series route to review, never auto-completed). T-35: the two JSONL logs
  joined only on `GenerateID` (sha256 of raw bytes incl. the date string); `auto`/`batch-auto`/
  `apply` fed one log raw `DD/MM` and the other normalized `DD/MM/YYYY` → two ids per expense.
  Fix: canonicalize once per boundary; join-id guard tests MUST use short `DD/MM` fixtures
  (full dates hide the bug class). T-36 staleness sweep. Survey `.claude/t35-date-year-semantics.md`
  [ref:date-year-semantics] mapped six parsers/six year rules → T-37..T-40 filed.
- **Session 62 — T-39/T-24 (PR #52) + strategic reframe + parse-boundary plan.** Suite split:
  `testing.Short()` skip in `extern.RequireOllama`; `run-acceptance.sh` defaults to the
  deterministic group (~8s, no Ollama), `-full` = whole suite (3600s ceiling; measured 128s at
  think-off — was 840–1989s). `--think=false` now default on classify/auto/batch-auto (gate band
  −2.3pp WITH think; 5.R2 miss +2.5pp at 5.5× latency; sentinel covered by the agreement gate).
  Vision re-read → project reframed as **pre-first-use**; organizing milestone = first real
  monthly close on 2026 data. Parse-boundary plan drafted (`.claude/plans/parse-boundary.md`):
  structured input, parse-once; year precedence string > `--year` > config `date_year` >
  most-recent-non-future (grace window TBD); T-21 UX = 1 row + ×N badge; `--year` uniform;
  T-38 resolved = delete `utils.ParseDate` with plain `batch` under WS-E, don't repair.
- **Session 68 — T-54, the T-42 scout, and the close-cycle repairs (S1/S2/S5).** The chain
  ran on real 2026 data for the first time, isolated in a **scratch install root** (binary +
  its own `config/` + a copy of `data/classification`) — the only mechanism that isolates,
  because no flag redirects the logs: `config.json` and relative log paths both resolve
  against `filepath.Dir(os.Executable())`, and `--data-dir` is the TRAINING data dir (which
  also owns the 5.R2 embedding cache). Real logs verified byte-identical afterwards.
  **Rationale for scouting before fixing:** the backlog was speculation until the chain ran;
  one throwaway run converted it into a measured list and reordered the work.
  **What it found:** two hard blockers in `review` — T-54 (`ReadQueue` accepted only `1`/`0`
  for `auto_inserted`, a spelling NO producer ever emitted, invented by fixtures) and S2
  (`review` derived its picker from the workbook's Referência sheet while everything else
  routed with `config/taxonomy.json`; measured **7 divergent leaves**). Plus S1 (one
  unparseable row killed the step; 4 of 69 did) and S5 (`generate-workbook` ignored config).
  **The recurring shape, four more times:** a check that reports success while testing
  nothing. The `1`/`0` parser whose only inputs were fixtures authored to satisfy it; a
  fixture that hand-converted producer output and a memory note calling that a legitimate
  "fold point"; two test-only synthetic workbooks duplicating the fixtures' own taxonomy;
  and the T-54 seam test itself, which fed only parsed rows and so passed while the seam
  stayed broken for unparsed ones. **Mutation is the only cheap discriminator** — every
  guard this session was broken deliberately and confirmed red, and where a mutation did
  NOT go red the reason was recorded rather than assumed to be a gap.
  **Implication:** a fixture that TRANSFORMS producer output is not bridging a harness
  limitation — it is asserting a contract neither side implements. Feed real producer bytes
  to the real consumer instead, and no fixture has to be kept in step.
  **Also:** the drift had inverted the feedback loop — 2 logged "corrections" taught the
  classifier `IRFF` over the routable `IRRF`, and corrected entries are the HIGHEST-priority
  few-shot source. 3 real records backfilled. Report: `.claude/t42-scout-report.md`.
- **Session 72 — the backlog was re-ordered by measurement, not argument (PR #66).**
  Two numbers, both extracted from data session 71 had already written to disk, no new run.
  **(1) Absolute gate precision 94.1%** vs 50.0% for gate-refused rows
  (`.claude/b1-gate-precision-measurement.md`). **(2) The A1 hint predicate** — unambiguous
  specificity 1.00, keyword ≠ model — fires on 14 of 65 rows at 71.4% precision, recovering
  10 of 25 corrections (`.claude/a1-keyword-hint-precheck.md`).
  **Rationale for measuring before building:** the plan of record was A1 + B1 (gate-to-review).
  B1's two justifications were eliminating unrepairable rows and OBTAINING this measurement.
  Ten minutes of joining existing files delivered the second for free and dissolved the
  first — the 17 auto-inserted rows were never unseen, `close-cycle.sh:220` feeds `review`
  the FULL `classified.csv`. B1 was dropped and T-65 promoted; `t42-close-findings.md` was
  corrected in place where readers land.
  **The methodological finding:** the first join keyed on ITEM and reported 26 corrections
  against a known total of 25. That one-row overshoot was the only visible symptom of
  duplicate item names (`San michel` ×3) silently collapsing. Had they fallen the other way
  the arithmetic would have reconciled and the wrong number shipped looking right. Fixed by
  a POSITIONAL join verified on item AND date across all 65 pairs. **Carry a checksum that
  can fail, and treat a small unexplained discrepancy as a defect rather than rounding.**
  **Two coverage traps, both recorded where the reader stands:** a mutation that fails to
  COMPILE proves nothing (deleting a field made the package unbuildable — red from the
  compiler, not the test; swapping the source column kept it compiling and then discriminated
  precisely, failing the seam test while `internal/review` stayed green). And **the `-short`
  group is structurally blind to format changes** — every assertion about `classified.csv`'s
  shape sits behind `RequireOllama`, because producing one means classifying, so six wrong
  assertions stayed green through the whole session until `-full` ran.
- **Session 74 — the id migration, and what it says about inheriting a measurement.**
  Both live logs were migrated so every `id` equals `GenerateID` of its own row and every
  date is `DD/MM/YYYY` (2159/2159 + 717/717 at migration; join **403 → 403**, all 717
  pairings identical). The year-less-id exception — recorded as locked in s70 and cited in
  four places — is retired.
  **Why it was declined for four sessions, and why that was reasonable.** s70's record said
  "recomputing drops the joins to zero", with a measurement attached. That is true — **of a
  one-sided recompute**, rewriting the expense log's ids from its new full dates while
  `classifications.jsonl` still stored `DD/MM`. It is false of a **two-sided** migration:
  read each classification row's year off the join FIRST, rewrite its date, then recompute
  both sides from the same strings. Same inputs, opposite outcome; the second operation was
  never named as an option.
  **Rationale:** a decision record carrying its measurement is far better than one that does
  not, and this repo is unusually good at it. But **a measurement pins down the operation you
  measured.** Inheriting one means asking "which operation produced this number, and is it
  the operation I am considering?" — not merely "is the number right?".
  **Implication:** the exception's justification dissolved, and with it T-65's hardest
  constraint ("key off the STORED id, never recompute" — now moot). A new one replaced it,
  which no amount of migration can remove: `GenerateID` hashes `item|date|value`, so
  collisions fell 46 → **8** and the survivors are genuine same-day duplicate purchases.
  **Consistent is not unique** — an id can name two rows, and anything keyed on one must say
  what that means.
  **The near-miss worth remembering.** 38 expense ids each carried TWO dates differing only
  by year — inevitable when the id hashes a year-less date. A `{id: date}` lookup silently
  keeps the last, so those rows would have been assigned a coin-flip year. **The join-pairing
  check did not catch it**: before and after used the same wrong lookup, so it passed at
  717/717. It was found by asking whether the lookup was ambiguous — a question no existing
  check encoded. Zero classification rows referenced those ids, so nothing was harmed, and
  the migrator now hard-fails rather than choosing. **A check inherits the assumptions of the
  code it checks; agreement between two runs of the same mistake is not evidence.**
  **What made it safe to execute:** a byte-comparison of all five workbooks 2022–2026, on a
  check first proven to MOVE for a value change and NOT for an id change, so "identical"
  carried information rather than merely being reassuring; surgical in-line field rewriting
  rather than re-serialising (the two logs mix Go-compact and Python-spaced JSON, so a
  re-serialise would have churned whichever half did not match and destroyed the comparison);
  refuse-to-write-anything on any unresolved or ambiguous row; and four-direction mutation of
  the new `restore` guard, including the must-pass case that separates a working guard from
  one that refuses everything.
- **Session 74 (cont.) — the bottleneck is capture, not classification.** Chasing "why is
  only one month closed?" found that expenses are typed into a Telegram group ("Gastos",
  exported by hand as `ChatExport/result.json`), and that **no committed tool converts that
  export into a CSV** — the one manual link in an otherwise mature chain, absent from the
  74-item task board, with ~7 months of 2026 behind it. Note also what the four rejected rows
  of the first close actually were: swapped fields, a typo'd day, a stray `;`, and arithmetic
  that does not reconcile — none a classification problem, all artifacts of free-form typing.
  **Implication:** validation belongs at capture, where the human still remembers the
  purchase, not in `batch-auto` months later where `646,25` vs `299,00 + 49,90` is
  unresolvable. `failed.csv` (T-63) is a good repair mechanism aimed at the latest possible
  moment to repair.
