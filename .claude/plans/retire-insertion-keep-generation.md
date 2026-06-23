# Plan — Retire workbook insertion, keep only generation

**Status:** PLAN ONLY (not approved for execution) — drafted session 36, 2026-06-22
**Supersedes/closes:** T-03 ("decide apply/add fate against generated workbooks"), couples T-11 + T-09
**Decisions locked (user, session 36):**
- Commands → **convert to log-append** (stop touching the workbook; append to `expenses_log.jsonl`).
- Income → **combine routes**: income gets its own log entries *and* an explicit income route in
  generation; the income *structure* comes from the taxonomy (revenue blocks already do).
- Sequence → produce this plan first; execution order decided later.

---

## 1. Target architecture

**The JSONL logs become the single source of truth; the workbook is a fully-derived artifact.**
`generate` is the only code path that ever writes a workbook. Nothing mutates an existing book in
place anymore.

> ⚠️ **Premise correction (advisor + empirical, session 36):** "5.R4 made history reproducible"
> is true for **expenses only**. The extractor's `EXPENSE_SHEETS = [Fixas, Variáveis, Extras,
> Adicionais]` **omits Receitas**, and `grep` confirms **zero income entries** in any log (the one
> `receita` hit is `"renovação receita cannabis"` — a Saúde expense). Historical income exists
> **only in the real workbook**. The instant generation becomes the sole writer, all historical
> income vanishes from the regenerated book unless WS-0b extracts it first.

> ⚠️ **Workflow change (state explicitly):** once the workbook is derived, any manual edit a user
> makes in Excel is destroyed on the next `generate`, with no path back to the log. The user drove
> this decision, but it is a real round-trip loss, not just an internal refactor.

```
inputs (CSV / single add) ──▶ classify ──▶ review.html ──▶ apply
                                                              │
                                          append ────────────┤
                                                              ▼
                                   expenses_log.jsonl  (+ classifications.jsonl)
                                                              │
                                                              ▼
                                        generate-workbook ──▶ workbook.xlsx (derived)
```

Why this is safe to do now: **5.R4 already extracted 2022–2025 history into logs**, so generation
can reproduce all history — the living workbook no longer needs to be the thing you edit.

### Dependency finding (already verified)
- `internal/generate`, `internal/inspect`, `internal/apply` do **not** import `internal/excel`.
  The generation path is already fully decoupled from insertion.
- Insertion machinery reachable only via: `internal/excel` (writer/columns/reader),
  `internal/workflow` (`InsertExpense`, `InsertBatch…`), and `internal/batch` insert path.
- Importers to rework: `add.go`, `auto.go`, `batch_auto.go`, `apply.go` (write-half),
  `review.go` (reads taxonomy from workbook via excel), `internal/batch/*`.

---

## 2. Workstreams

### WS-0 — Diff-first spike (PREMISE VALIDATOR — do before any code change)
The divergence diff is not a late safety gate; it is the thing that tells us what the real backlog
is. Run it **first**:

1. **Preserve the real workbook** as the diff oracle *and* the historical-income source — it must
   not be deleted or overwritten by an early `generate` run. Copy it aside read-only.
2. Regenerate the full workbook from current logs → `workbook-inspect` dump.
3. Dump the real workbook → diff the two.
4. The diff is a concrete list of what generation is missing: income (known), and anything else
   (formulas, manual notes, Referência curation). **That list reshapes WS-C and confirms WS-0b.**

#### WS-0 RESULTS (executed session 36, 2026-06-22 — sonnet subagent diff)
Inputs: 4 per-year generated workbooks vs real `Planilha_Normalized_Final.xlsx` + the Jun-9
`.claude/workbook-dump/`. Verified against project decisions afterward (raw report over-flagged):
- ❌ **"Referência de Categorias missing"** and ❌ **"month columns offset (gen col C vs real col D)"**
  are **BY DESIGN**, not gaps — spec-v2 active decisions ("2 label cols; months start col C;
  Referência omitted"). Discard both.
- ✅ **Income is the sole real gap**, and it is two-part:
  1. **Data:** every generated `Receitas` is an empty shell (0 values, all years) — historical
     income was never logged → **WS-0b**.
  2. **Structure:** the taxonomy's revenue side is **8 flat categories** vs the real workbook's
     **~25-line payslip taxonomy** (Salário + 14 sub-lines, 13°, Férias, Presente, Outros) → the
     **revenue taxonomy must be enriched** before generation can even hold that detail. *(NEW —
     fold into WS-0b/WS-C.)*
- ✅ **Currency formatting — NO CHANGE NEEDED (corrected session 37, verified in code).** The
  earlier "generation writes a `R$ 200.00` string → change to numeric + format" finding was a
  **dump-serialization artifact** (it is the *real* workbook that stores currency as strings). The
  generator ALREADY writes a numeric value (`data_sheet.go:108` sets `entry.Value` float) with the
  `R$ #,##0.00` cell format (`styles.go:53` `fmtCurrency`, applied via `st.Currency`). No generator
  work here; drop this from WS-C scope.
- Row-count differences are the intentional derived layout (1 row/subcat) + half-year real dump —
  not data loss.
- Expenses reproduce faithfully → **premise validated for expenses; income is the one blocker.**

#### Income decisions RESOLVED (session 37, 2026-06-23)
The three deferred income questions are now locked:
1. **Income taxonomy shape → 3-level symmetric.** `Receitas → block (Salário/13°/Férias/
   Presente/Outros) → sublines`, mirroring expense `type→category→subcategory`. The extracted
   `income_log.jsonl` already carries `income_category`+`income_label` (3-level). Requires:
   `loader.go` income model change (`Blocks []string` → block→sublines), `incomePath` gains a
   level, `RevenueBlock` model, and the `incomeCategories` shape in `config/taxonomy.json` (merge
   the proposal). This is the symmetry that lets WS-D collapse income/expense routing later.
2. **Deduction sign → keep signed.** Gross lines positive, deductions (INSS/IRRF/contributions)
   negative; net = sum. Matches the extractor's output and the source workbook. Generator just sums.
3. **2022 income → accept as unrecoverable.** 2022 old workbook has no Receitas sheet; generated
   2022 Receitas stays an empty shell. 2022 *expenses* unaffected.

> **Verification (session 37):** `parseDate` (loader.go:302) still rejects non-`DD/MM` → WS-A
> genuinely pending. Income *target* scaffolding exists (income blocks in `byPath` via `incomePath`,
> `subcatTarget{kind:"income"}`, `attachEntry` income branch) BUT `routeEntry` only ever builds
> `expensePath` → no log line can reach an income block today. WS-C = wire the router + recognize
> income entries (`income_marker`/`income_category`/`income_label` field names differ from
> `type/category/subcategory` — scanEntries must learn the income entry variant).

### WS-0b — Historical income extraction (5.R4-for-Receitas)
Parallel to 5.R4 but for the **Receitas** sheet across the old workbooks → income log entries.
Confirmed required by WS-0 (income wholly absent). **Method (user, session 36):**
- **Use a subagent** for the extraction (ask which model per session preference).
- **Source:** old workbooks at `~/workspaces/expenses/old/` — mine them for **both** the revenue
  **taxonomy structure** (the ~25-line payslip breakdown → enrich `config/taxonomy.json` revenue
  side) **and** the reported income **values** (→ income-typed log entries).
- Output: income-typed log entries consumable by WS-C's income route, plus a taxonomy enrichment PR.
- Mirror 5.R4 hygiene: outputs/backups outside the repo or gitignored; no personal values in scripts.

### WS-A — T-11: multi-year log (PREREQUISITE, smallest)
**Goal:** one unified `expenses_log.jsonl` spanning years, feeding generation directly; retire the
per-year split (`expenses_log-{2022,2023,2024}.jsonl`) created in 5.R4.

- `internal/taxonomy` `parseDate`: accept `DD/MM/YYYY` in addition to `DD/MM`.
- Routing uses a **per-entry year** when present; fall back to a `--year`/`Options.Year` default
  for legacy `DD/MM` entries.
- `generate` plumbs the year option through to `LoadTaxonomy`.
- Merge the per-year logs back into one (one-off), confirm generation output is byte-identical via
  oracle dumps.
- **Tests:** unit on `parseDate` (both formats, bad input); acceptance — a single multi-year log
  produces the same dumps as the per-year run.

#### WS-A execution decisions LOCKED (session 37, 2026-06-23)
Grounded in code: `Subcat.Months` is `[12][]Entry` (month-indexed, **no year axis**);
`data_sheet.go:106` stamps every retained entry with package `dataYear`. So "multi-year log"
means **one merged log → generate FILTERS to `--year`**, not a multi-year workbook. Per-entry
year is a *filter key*, not a layout dimension.
- **EQ-A2 → (a):** add a `year int` param to `LoadTaxonomy`/`scanEntries`; filter in the scan
  (`keep iff entryYear==0 || entryYear==targetYear`). Downstream `dataYear` stamping unchanged.
- **EQ-A4 → throwaway:** merge script lives in `.claude/scratch/` (5.R4 precedent), runs once over
  gitignored personal logs; **byte-identical oracle-dump gate** vs pre-merge per-year runs.
- **Invariant (EQ-A3 dup trap):** the merge MUST rewrite every line to `DD/MM/YYYY` (filename =
  year authority) so the merged log has **zero** year-0 lines. The `--year` fallback exists only
  for un-migrated single-year logs (all year-0 → all kept → behaves as today).
- **`parseDate` (EQ-A1):** accept `DD/MM` and `DD/MM/YYYY`; return `(day, month, year, err)` with
  `year==0` sentinel. **Shared with income** — `income_log.jsonl` is already `DD/MM/YYYY`, so this
  is also a hard prereq for WS-C.
- **Acceptance (EQ-A5):** a multi-year fixture log generated with `--year N` dumps identically to
  an N-only fixture.

**WS-A task breakdown:**
1. `parseDate` → both formats + year sentinel; unit tests (DD/MM→0, DD/MM/YYYY→year, malformed).
2. `LoadTaxonomy`/`scanEntries` → target-year param + filter; update the one non-test caller
   (`generate.go:47`) to pass `opts.Year`.
3. Throwaway merge script → per-year logs → one `DD/MM/YYYY` log; byte-identical gate.
4. Acceptance: multi-year-log-filtered-to-year == single-year fixture.

**WS-A — ✅ DONE (session 37, 2026-06-23, branch `chore/income-extraction-tooling`, commits
`0c011e1` + `95dbabb`).** All 4 tasks landed and verified:
- `parseDate(dateStr) (day, month, year, err)` accepts `DD/MM` (year 0) + `DD/MM/YYYY` (year ≥ 1000);
  `LoadTaxonomy`/`loadEntries`/`scanEntries` take `targetYear`; filter = keep iff
  `entryYear==0 || targetYear==0 || entryYear==targetYear`. `generate.go` passes `opts.Year`;
  `auto.go` skeleton path passes `0`. (NOTE: filter `continue` sits *after* `routeEntry`, so an
  out-of-year **type-less** entry can still bump the stderr fallback count before being skipped —
  cosmetic only, but relevant to WS-D's "fallback count ~0" gate.)
- Acceptance `TestGenerateWorkbook_MultiYearLogFiltersToYear` (reuses `generate-basic`
  `expected-dump-data` as oracle; fixture `entries-multiyear.jsonl` = 2026 entries + 2025 noise).
  Unit `TestParseDate_MultiYear`.
- Merge script `.claude/scratch/merge_year_logs.py` → gitignored `expenses_log-allyears.jsonl`
  (2073 records). **Byte-identical gate PASSED** all 4 years (per-year `--year N` == merged `--year N`
  dump, excl. manifest `source`). The per-year split CAN now retire — **but the canonical
  `expenses_log.jsonl` was NOT clobbered**; promoting the merged log to canonical + deleting the
  per-year files is a deferred workflow decision (user's call).

### WS-B — Convert commands to log-append
Each command stops calling `internal/workflow`/`internal/excel` and instead appends to
`expenses_log.jsonl` (+ `classifications.jsonl`) using the same writer `apply` uses.

- `add`   — single expense → one log entry (status `manual`).
- `auto`  — classify → if HIGH, append confirmed entry to log (no workbook write); else print
  candidates as today.
- `batch-auto` — already produces classified/review CSVs; **drop the workbook-insert branch**,
  keep CSV production → review → apply.
- `apply` — keep only the log-writing half; **delete the workbook-write half**
  (`excel.WriteBatchExpenses`, `AllocateEmptyRows`, `FindSubcategoryRowBatch`, row-builders).
- **Entry contract:** every appended entry should carry the full **type/category/subcategory**
  (the typed path) so it routes via `byPath` — this is what lets WS-D retire the bare-name fallback.

### WS-C — Income/revenue route (combine)
Today `LoadTaxonomy` builds `revenueBlocks` structurally but routes **no entries** into them
(`routeEntry` only ever builds `expensePath`). Income *target* scaffolding exists (income blocks in
`byPath` via `incomePath`, `subcatTarget{kind:"income"}`, `attachEntry` income branch) — WS-C wires
the producer/router side and lifts the model to 3 levels.

#### WS-C decisions LOCKED (session 37)
- **3-level symmetric income model** (from income decisions §): `Receitas → block (Salário/13°/
  Férias/Presente/Outros) → subline`. Leaf = subline.
- **Income input → SEPARATE `--income-entries` flag** (NOT the unified `--entries`). Consume the
  extractor's `income_log.jsonl` AS-IS (schema `income_marker`/`income_category`/`income_label`/
  `item_note`/`date`(`DD/MM/YYYY`)/`value`/`year`). Keeps expense vs income schemas clean; unify
  later if ever desired. `income_category`→Block, `income_label`→subline Label, `item_note`→Entry.Item.
- **Signed values** (decided): deduction lines stay negative; generator just sums. No sign handling
  in the router.
- **Currency formatting → NO-OP** (see WS-0 RESULTS correction; generator already numeric + format).
- **2022 income** absent → its Receitas stays an empty shell.

#### WS-C task breakdown (NOT STARTED — for next session, subagent-driven)
1. **Model** (`internal/taxonomy/types.go`): `RevenueBlock` gains a middle level →
   `{Category:"Receitas", Block:"Salário", Label:"INSS"(leaf), Months}`. `incomePath` → 3 segments
   (`income\x00receitas\x00salário\x00inss`).
2. **Loader** (`loader.go`): `IncomeCategories` raw struct `Blocks []string` → block→sublines;
   `incomeCatsToRevenueBlocks` flattens to leaf `RevenueBlock`s; `buildSubcategoryMap` registers
   3-segment `incomePath` keys.
3. **Router** (`scanEntries`/`routeEntry`): a separate income-entry scan (new `--income-entries`
   file) that reads the income schema, NFC-normalizes, builds `incomePath(category, block, label)`,
   and `attachEntry`s the signed value to the matching leaf block's month. parseDate already accepts
   the `DD/MM/YYYY` income dates (WS-A).
4. **Generator** (`internal/generate/revenue_sheet.go`): 3-level grouping (Category → Block → subline
   rows) replacing today's 2-level (Category → block-row). Reuses `writeDataBand`.
5. **Taxonomy data**: merge `.claude/scratch/taxonomy-revenue-proposal.json` into
   `config/taxonomy.json` `incomeCategories` (gitignored; structure only).
6. **CLI / plumbing**: `generate-workbook --income-entries <path>`; thread through `Options`.
7. **Acceptance**: income fixture + a NEWLY FROZEN data-bearing income oracle dump (current income
   dumps are empty shells) — logged income lands in the right Receitas block/month, signed sum
   correct. This freeze is the fiddly part; budget for it.

**Size note:** bigger than WS-A — model change ripples loader→router→generator rendering + a new
frozen oracle. Deferred from session 37 (usage budget). The current `revenue_sheet.go` is 2-level
(`RevenueBlock{Category, Label, Months}`, grouped by Category); the 3-level lift is the core risk.

### WS-D — T-09: retire bare-name fallback
Once WS-B guarantees typed entries and WS-C gives income its own route, the transitional bare-name
`byName` fallback + ambiguous-skip in `internal/taxonomy` `scanEntries` has no remaining producer.

- Gate on the stderr type-less fallback count reaching ~0 across a real regenerate of all history.
- Remove `byName`, the ambiguity guard, and the fallback count plumbing.
- **Tests:** the ambiguous-skip acceptance cases either delete or convert to "typed entry routes
  unambiguously."

### WS-E — Delete dead code
After B/C/D, delete what no longer has importers:
- `internal/workflow` (entire package).
- `internal/excel` **write** side (writer.go, columns.go, the allocation/lookup helpers).
- `internal/batch` insert path + `rollover_writer.go` (installment rollover was an insert-time
  concern — confirm generation handles installment expansion, or move expansion to the
  classify/append step).
- `review.go`'s excel dependency → migrate `review.BuildTaxonomy` to read `taxonomy.json` instead of
  the workbook (removes the last reader dependency, lets `internal/excel` go entirely if `inspect`
  doesn't need it — **verify `inspect` reader usage before deleting reader.go**).

---

## 3. Open questions for execution
1. **Installment expansion** ("99,90/3"): currently an insert-time/batch concern. Where does it live
   once insertion is gone — at classify/append time (expand into N log entries) or at generate time?
   Leaning: expand at append time so the log is the literal truth. **Must cover cross-year
   rollover** (`rollover.csv`, installments crossing the year boundary — KNOWLEDGE.md), not just
   same-year splits.
2. **`internal/excel` reader**: does `internal/inspect` (acceptance verifier) still need any reader
   helper, or can the package be deleted wholesale? Verify before WS-E.
3. **Income CLI surface**: `add --income` flag vs dedicated `income` subcommand.
4. **Backup/idempotency**: generation overwrites the derived workbook; confirm the backup discipline
   (the old insert path made `.bak` copies) is preserved or intentionally dropped.

## 4. Suggested order
**WS-0 (diff spike) → WS-0b (income extraction, if diff confirms) → WS-A (T-11) → WS-C (income
route) → WS-B (commands to log-append) → WS-D (T-09) → WS-E (delete).**
Rationale: WS-0 validates the premise and produces the real backlog before any code changes; 0b
fills the income-history gap WS-0 surfaces; A is a safe enabler; C before B so the entry contract
(incl. income) is final before rewiring commands; D and E only once nothing produces type-less or
uses the dead packages. **WS-E stays gated on a clean WS-0 diff re-run.**

## 5. Risk
- Largest risk is **silent data divergence**: a regenerated workbook that drops content the
  in-place book carried (manual notes, income, formulas). Mitigation: regenerate all history,
  diff against the current real workbook via `workbook-inspect` dumps **before** deleting any
  insert code (WS-E gated on a clean diff).
