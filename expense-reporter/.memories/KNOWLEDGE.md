# expense-reporter/ — Knowledge (Semantic Memory)

*Go application accumulated decisions. Read on demand by agents.*

## Command Hierarchy (2026-02 → 2026-03)
Five user-facing commands, each building on the previous:
- `add "item;DD/MM;value;subcategory"` — manual insert with known subcategory
- `batch "file.csv"` — bulk manual inserts from CSV
- `classify "item" value DD/MM` — LLM classification only, no insert (read-only)
- `auto "item" value DD/MM` — classify + **append to expenses_log.jsonl** if confidence ≥ 0.85 (was: workbook insert)
- `batch-auto "file.csv"` — classify CSV batch + **append** high-confidence rows to the log (was: workbook insert)
**Rationale:** Incremental trust — `classify` lets users verify the model before `auto`
appends anything. `batch-auto` is the production workflow.
**Implication:** `auto` and `batch-auto` share the same classifier + decision logic.
The only difference is input source (args vs CSV) and output format.
**Log-append pivot (WS-B, 2026-06–30):** since the session-36 pivot, `add`/`auto`/`batch-auto`/`apply` no
longer touch the workbook — they append typed entries to `expenses_log.jsonl`, and `generate-workbook`
is the single writer (see the dedicated section below). Plain `batch` (manual, non-classifier) still
inserts into the workbook. WS-B is **complete** (slices 1–4); next is WS-D (retire bare-name fallback, T-09).

## Batch Pipeline Optimization (2026-03)
`workflow.InsertBatchExpenses` opens the workbook once, inserts all rows, saves once.
This is 20–28x faster than calling `InsertExpense` per row.
Installments are expanded before insertion; the index mapping tracks which expanded
row maps back to which original input for error reporting.
**Rationale:** Excel file I/O (open + save) dominates per-row cost. Amortizing it across
a batch eliminates the bottleneck.
**Implication:** Single-expense `InsertExpense` delegates to `InsertBatchExpenses` with
a one-element slice — both paths share identical pipeline logic.
**Post-pivot (2026-06–30):** `InsertBatchExpensesFromClassified` (the classified-batch variant) is
**deprecated** (`// Deprecated:`) — `batch-auto` no longer calls it; retained + unit-tested for the
WS-E hard-delete. `InsertBatchExpenses` stays live under the plain `batch` command, so the rollover/
excel machinery is NOT dead and must not be deleted with the classified variant.

## Log-Append Path (WS-B, 2026-06–30)
`internal/appender.ExpandAndAppend(logPath, item, date time.Time, perInstallment, count, type, category, sub)`
is the single append-time writer shared by `add`/`auto`/`batch-auto`/**`apply`** (slice 4, session 44).
`apply` calls it with **count=1** (the reviewed value is a single number; the installment count is lost
upstream at `review.ReadQueue` — T-21) and flips the write order **log-first, feedback-second** with a
**non-destructive** both-path pre-flight: the classifications log (its dedup index) is probed before
`processEntries`, the expense log only when there are new rows and WITHOUT `O_CREATE` (an `O_CREATE` probe
would leave an empty log and break the found-only `ExpenseLogNotCreated` acceptance test). It expands installments into N dated
`ExpenseEntry` lines (`addMonths`; cross-year installments carry their **real next-year date** — there is no
`rollover.csv` anymore) and writes via `feedback.AppendExpense` (plain `O_APPEND`, **no dedup on the hash ID**).
`batch-auto`'s `appendClassified` (cmd) iterates auto-rows, delegates each to `appendOneRow`, and on success
records the confirmed entry to `classifications.jsonl` via `logConfirmedFeedbackForRow`.
**Failure honesty (the load-bearing rule):** because the log is now the only durable persistence, an append
failure for a row **downgrades it in place** (`AutoInserted=false` + `Error` set) — keeping the summary count
honest, sending the row to `review.csv`, and making the command exit non-zero (the returned error is wrapped
by `runBatchAuto`, preserving the already-written CSVs). CSVs are therefore written **after** the append.
**Pre-flight:** `preflightLogPath`/`verifyAppendable` opens the log in `O_APPEND` mode before classifying, so an
unwritable log fails fast (saves ~12 s/row on the model). `--dry-run` skips both pre-flight and append (classify
+ CSVs only). **Date reformat gotcha:** the appender formats dates `DD/MM/YYYY`; bare `DD/MM` inputs get
`time.Now().Year()` via `ParseDateFlexible` — so fixtures/tests must use explicit-year inputs to stay stable.
**Idempotency gap:** no dedup means a re-run after a partial failure double-appends (flagged for a follow-up).

## JSON Output Mode (2026-03, updated 2026-04)
`--json` flag on root command; checked by `classify`, `auto`, and `add` subcommands.
Output structs: `ClassifyOutput`, `AutoOutput`, `AddOutput` — all in `cmd/output.go`.
`auto --json` is read-only (never inserts) and returns `action: "would_insert"|"review"|"excluded"`.
`AutoOutput` includes `classification_id` (sha256[:12] of item+date+value) so MCP callers
can pass it back to `add` for cross-referencing the prior classify call.
**Rationale:** MCP server needs structured, parseable output. Human-readable output
(confidence bars, Unicode symbols) breaks machine consumption.
**Implication:** JSON mode on `auto` is intentionally non-destructive — the MCP server
decides whether to call `add` separately, passing back the prediction context.

## Resolver — Fuzzy Subcategory Matching (2026-02)
`internal/resolver` builds a hierarchical index from the Excel reference sheet.
Matching strategy: exact match → case-insensitive → prefix match → contains match.
The reference sheet is the canonical source of valid subcategories.
**Rationale:** User input varies ("Orion - Consultas" vs "Orion"). Fuzzy matching
reduces friction without requiring exact category knowledge.
**Implication:** Adding a new subcategory only requires adding it to the Excel reference
sheet — no code changes needed.

## Excel Integration (2026-02)
Uses excelize library. Key design: month columns are detected by header pattern
(month names in Portuguese), not hardcoded column indices.
Workbook path resolution: `--workbook` flag → `EXPENSE_WORKBOOK_PATH` env → default path.
Automatic backup before any write operation.
**Rationale:** The workbook layout may change year-to-year. Dynamic column detection
makes the tool resilient to layout changes.
**Implication:** The reference sheet must follow the expected format (category/subcategory
columns), but data sheet column positions are flexible.

## MCP Prediction Feedback Flow (2026-04)
`add` branches its feedback write based on whether prediction flags are present:
- No flags → `status=manual` (backwards-compatible, no model involved)
- `--predicted-subcategory X`, chosen == X → `status=confirmed` (user accepted top candidate)
- `--predicted-subcategory X`, chosen != X → `status=corrected` (user overrode the model)
`--classification-id` is cross-reference only — a miss warns stderr but never blocks the insert.
**Rationale:** Insert is the primary operation; feedback is best-effort. A log concern
should never prevent an expense from being recorded. The `confirmed` write on match was
chosen (not skipped) because it's the only signal that the Telegram flow accepted the
prediction — `auto` is never called in that path, so skipping it would silently discard
a training signal.
**Implication:** `add` now produces all three feedback statuses depending on call context.
The caller (MCP bot) controls which status is written by choosing which flags to pass.

## Feedback Persistence Design (2026-03)
Two append-only JSONL files:
- `classifications.jsonl` — status: confirmed/corrected/manual; includes model name,
  predicted vs actual subcategory. Source for few-shot feedback examples.
- `expenses_log.jsonl` — slim: item, date, value, subcategory, category, timestamp.
  Source for expense history queries.
Shared ID: `sha256(item+date+value+subcategory)[:12]` — correlates entries across files.
**Rationale:** Separate concerns — classification feedback (ML improvement) vs expense
history (data queries). The shared hash enables joins without a database.
**Implication:** Both files grow monotonically. No update/delete operations.

## Config Design (2026-02)
`config/config.json` with `internal/config` loader. Key fields:
- `date_year` (2025) — applied to DD/MM dates during parsing
- `auto_insert_excluded` (["Diversos"]) — subcategories blocked from auto-insert
- `classifications_path`, `expenses_log_path` — JSONL output locations
**Rationale:** Runtime behavior that changes between years or users belongs in config,
not code. The exclusion list was added after discovering "Diversos" false positives.
**Implication:** Year rollover requires updating `date_year` in config (and likely the
workbook path). No code changes needed.

## Workbook Generator Design (2026-06, sessions 26–27)
Spec v2 at `.claude/plans/workbook-generator-spec.md` is the single design authority; it is a
REDESIGN — where it disagrees with the original workbook, the spec wins.
Key decisions:
- **Derived layout** — row positions computed from taxonomy + entry counts; the original
  workbook's dump is only a historical reference. The source's fixed layout contained drift
  (June SUMs over the wrong column, single-cell SUM collapse, fill-down gaps) precisely
  because it was hand-maintained.
- **Merges, not fill-down** — the source filled labels down because hand-inserting rows into
  merges is painful; generated workbooks don't care, so v2 merges col A per category section
  and col B per block (both include total rows).
- **2 label columns everywhere** — sub-item level eliminated ("Orion - Consultas" composed
  strings); months start at col C on all data sheets including Receitas.
- **Referência sheet omitted** — it existed to support manual insertion; the generator knows
  all positions. If `add`/`batch` resolver compat is ever needed, emit a slim A/B/C taxonomy sheet.
- **Golden-master validation** — `.claude/workbook-template/template-reviewed.xlsx`
  (user-curated, fake data) is the convergence target; compare via `workbook-inspect` dumps +
  openpyxl pass (`diff.py`). Never claim convergence by eyeballing.
**Rationale:** generation inverts the dependency — taxonomy input becomes the source of truth,
sheets become projections.
**Implication:** the scratch builder (`.claude/scratch/template-builder/`) is the reference
implementation to port into `internal/generate`; excelize gotchas learned: SetCellFormula takes
no leading `=`; stale-formula fix = UpdateLinkedValue() + SetCalcProps(FullCalcOnLoad).

### Phase B updates (2026-06-10)
- **Convergence target moved** `template-reviewed.xlsx` → `template-data.xlsx` (adds entries,
  typed values) for the data/formula validation pass.
- **Per-group percent rows resolved** (spec §7 Q2): `% sobre despesas` / `% sobre receita`
  emit per categoria group; `perGroupPctRows` ON.
- **Labels centralized + normalized for i18n** (spec §4.4): all generic strings live in a
  `Labels` struct with **English field names** (`PctOfExpenses`, `Investments`, `TotalIncome`)
  and localized pt-BR values; month names included; a `loadLabels(path)` config reader is a
  deferred drop-in. Forces Receita→Revenue / Renda→Income naming. Carry this struct into
  `internal/generate` when porting.

### Phase G — the real generator (2026-06-11, session 29)
- **`internal/inspect`** (G1): extraction core lifted verbatim from `cmd/workbook-inspect`
  (now a thin wrapper). `DumpWorkbook(path, outDir)` + exported dump types. The dump JSON is a
  TEST CONTRACT — `verify.WorkbookStructureMatches` unmarshals it.
- **Input contract (spec §1.1):** taxonomy JSON (sheets→categories→subcategories;
  incomeCategories→blocks; `incomeCategories[].name` is block grouping, NOT the sheet label —
  that's `Labels.RevenueSheet`) + `expenses_log.jsonl` entries (`date` is DD/MM, no year;
  `--year` supplies it). Join layer rules: unknown subcategory → warn to stderr + skip,
  exit 0; on category mismatch the taxonomy wins; duplicate subcategory names = error.
- **Oracle bootstrap (advisor-driven):** the scratch builder was taught to read the fixture
  files and its dumps frozen as the acceptance expectation BEFORE the port — G2 became a
  converge-to-green port (3/3 green first run). Limit of the pattern: oracle and port can
  share a bug — the hardcoded `{Fixas,Variáveis,Extras,Adicionais}` order emitted invalid
  D0/E0 refs for smaller taxonomies and the frozen dumps contained them. Fix: registry records
  `sheetOrder`; dumps re-frozen with a manually reviewed delta.
- **Verifier philosophy:** acceptance compares a NORMALIZED SUBSET (values, formulas, merges,
  dims, rowType/rowFill, bgColor/bold/borders) and deliberately ignores column widths, row
  heights, manifest source — excelize serialization noise classes documented in Phase A.
- **Naming (PR #27 review):** identifiers are English (Revenue*/summary*/balance*/Category);
  pt-BR text lives ONLY in `Labels` values. `RevenueSheet` ("Receitas") appears inside
  cross-sheet formulas — behaves like a schema identifier, not cosmetic text.
- Scratch builder `.claude/scratch/template-builder/` SUPERSEDED (kept as Phase A/B history);
  its fake dataset lives on as `internal/generate/taxonomy_fixture_test.go`.

### Generator internal refactor — style system & file layout (2026-06-12 → 06-15, sessions 30–31)
`internal/generate` was reorganized for domain isolation, behavior-preserving throughout (oracle
dumps byte-identical at every step):
- **styles.go is two layers:** a style *vocabulary* (named constructors + palette/numfmt constants
  naming the workbook's visual language) and a `styleRegistrar` (first-error capture; `family()`
  mints General/currency/percent trios of one fill+font). Sheet builders consume only `styleSet`
  IDs; new styles EXTEND the vocabulary rather than inlining `excelize.Style` literals. Portuguese
  field names anglicized (MonthCorner, TotalText/Value…); dead styles (MonthCovered, IndigoBandCur)
  removed.
- **File homes by domain, not first caller:** pure ref/formula helpers (`cell`, `sheetRef`,
  `needsQuote`) moved into `util.go`; the data-sheet writing vocabulary shared by the expense
  sheets and Receitas moved into new `data_sheet.go`. Two near-duplicate pairs were unified there:
  `calculateBlockRows(row, maxEntries)` (was calculateSubcat/RevenueBlockRows) and
  `writeDataBand(..., rowHeight, lastCol)` (was writeSubcatDataRows vs styleRevenueDataBand+
  fillRevenueEntries) — the sole behavioral difference between the expense and revenue bands is
  row height (12.75 vs 15).
**Rationale:** naming WHAT a style/helper is — not how it's assembled nor where first used — makes
reuse visible and additions self-policing; the oracle-frozen dump made aggressive merges safe (a
mis-parameterized row height fails loudly and specifically).
**Implication:** package stays FLAT (subfolders = separate Go packages = forced exports + cycle
risk; Java-style nesting rejected). The one penciled split — `internal/taxonomy` as a pure input
layer during the T-02 real-taxonomy export — is non-trivial: `taxonomy.go` currently mixes the
domain types (`Entry`/`Subcat`/`Category`/`ExpenseSheet`/`RevenueBlock`, used by every builder)
with mutable RENDER config (`dataYear`/`headroomRows`/`perGroupPctRows`, set by `Generate()` and
read by builders) that must relocate into `generate` before any split.

## internal/taxonomy extraction + full-path identity key (2026-06-16/17)
The penciled split landed (commits 07f395a render-config relocation, 21c6d4e extraction,
47b2379 identity key). Domain types + the loader (`LoadTaxonomy`, join layer) now live in
`internal/taxonomy`; render config relocated into `generate` FIRST so the new package imports
nothing from generate (`go list -deps ./internal/taxonomy` → cycle-free). Generate builders
reference `taxonomy.X` by full qualification (no alias shim).
**Identity key decision** (full record: `.claude/plans/taxonomy-identity-key.md`,
`[ref:taxonomy-identity-key]`): a subcategory's identity is its FULL PATH (sheet/category/sub;
income group/label), NOT the bare leaf name. The old global bare-name guard rejected the real
taxonomy, which legitimately repeats leaves (`Orion` across 3 Pet blocks; `Aluguel` as both a
Fixas expense and a Receitas income block). Now: only an exact repeated full path errors;
cross-path repeats are legal; a bare name shared by >1 full path is *ambiguous* and dropped from
routing (entry warn+skip, exit 0). `registerTarget` keeps the ambiguous set **sticky** (guards
the 3× re-add trap); full paths join on a **null byte** because real names contain `/`
(`Uber/Taxi`, `Óleo/flor cannabis`).
**Rationale:** real-data identity is hierarchical; bare leaf names are not unique.
**Implication:** routing logged entries by full path is **DEFERRED** (task #5) — it changes the
entry contract (entries carry only a bare `subcategory` today), the classifier, `scanEntries`,
and entry-fed fixtures. Until then the ambiguity guard keeps entry-fed generation safe (never a
silent misroute). The real `config/taxonomy.json` (112 subs) is gitignored; the fixture is the
test input. Fidelity verified by CSV↔JSON symmetric-difference (empty) + oracle dumps unchanged.
