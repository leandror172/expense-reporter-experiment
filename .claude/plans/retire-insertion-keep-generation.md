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
- ✅ **Currency formatting (DECIDED, user, session 36):** generated value cells should hold a **bare
  number**; the **cell number-format** carries `R$ …` (display only). Today generation writes the
  `"R$ 200.00"` string into the value — change it to a numeric value + currency cell format. Small
  generator change (`internal/generate` styles/number-format).
- Row-count differences are the intentional derived layout (1 row/subcat) + half-year real dump —
  not data loss.
- Expenses reproduce faithfully → **premise validated for expenses; income is the one blocker.**

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
Today `LoadTaxonomy` builds `revenueBlocks` structurally but routes **no entries** into them.

- Define an **income entry** in the log (its own `type` = the revenue/`Receitas` root, or a
  dedicated income marker) with a full **income path**.
- Add an income routing step parallel to `routeEntry` that places logged income amounts into the
  matching `RevenueBlock` by income full-path (`incomePath`).
- Structure stays taxonomy-driven (revenue blocks already come from taxonomy); only the **amounts**
  come from logged income entries.
- Provide an input path for income (e.g. `add --income` or a small `income` subcommand appending an
  income-typed log entry) — exact CLI surface TBD in execution.
- **Tests:** acceptance — logged income entries land in the right Receitas block/month.

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
