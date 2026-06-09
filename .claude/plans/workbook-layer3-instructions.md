# Layer 3 — Workbook Generator Spec: Cowork Instructions

Continuation brief for whoever runs **Layer 3** of the workbook-mapping plan. Layers 1 and 2
are complete. This file tells you what to read, what prompt to use, and what the output must be.

---

## Where we are

| Layer | Status | Artifact |
|-------|--------|----------|
| **L1 — JSON dump** | ✅ done (PR #25) | `.claude/workbook-dump/manifest.json` + one `<Sheet>.json` per sheet |
| **L2 — visual notes** | ✅ done | `.claude/workbook-visual-notes.md` + `.claude/workbook-screenshots/<sheet>/description.md` |
| **L3 — generator spec** | ⬜ TODO (this brief) | produce `.claude/plans/workbook-generator-spec.md` |

**Goal of Layer 3:** synthesize the dump + visual notes into a sheet-by-sheet spec a future
Claude Code session can implement as a "generate the workbook from the database" Go command —
without re-examining the workbook.

---

## ⚠ Decision first: WHERE to run Layer 3

**Option A — Claude Code, locally (RECOMMENDED).**
Claude Code (Opus) can read `.claude/workbook-dump/*.json` directly from disk. Nothing leaves
the machine. This honors the project's local-first principle (financial data stays private) and
costs no claude.ai usage. The reasoning task is well within Opus's range.

**Option B — claude.ai web (original plan, 2× usage expires 2026-07-05).**
Only choose this if you specifically want the web workspace. **It uploads real expense values to
the cloud.** If you go this route, upload a SANITIZED dump (see "Sanitize before upload" below),
not the raw files.

> Recommendation: do Layer 3 in **Claude Code (Option A)**. The rest of this brief assumes that,
> and notes the Option B deltas inline.

---

## Files to read / feed (in priority order)

1. **`.claude/workbook-dump/manifest.json`** — sheet inventory (names, dims).
2. **`.claude/workbook-dump/<Sheet>.json`** (all 7) — the structural ground truth. Each sheet:
   `dimensions`, `columnWidths`, `rowHeights`, `mergedCells`, `crossSheetRefs`, and `rows[]`.
   Each row: `rowType` (`header-month` / `header-col` / `total-row` / `data-row` /
   `category-label` / `separator`), optional `rowFill`, and `cells[]` with
   `col` / `value` / `formula` / `style{bgColor,bold,border*}`.
   - These are large (Listas/Referência ~1–2 MB). In Claude Code, read selectively: one expense
     sheet in full (e.g. `Extras.json`), then skim `Receitas`, `Listas de itens`, `Referência`.
3. **`.claude/workbook-visual-notes.md`** — render-only facts the JSON can't show (colors vs hex,
   frozen panes, fonts, number/date formats). **Trust the JSON over the visual gap catalogue** for
   black separators (the notes' UPDATE block explains this).
4. **`.claude/workbook-screenshots/<sheet>/description.md`** (all 7) — structure-from-JSON
   summaries; convenient per-sheet digests.
5. **`.claude/plans/workbook-mapping-plan.md`** — the original plan; § "Layer 3" has prompt framing.
6. **`.claude/workbook-map.md`** — older markdown map (background; superseded by the JSON).
7. Memory: `project_workbook_mapping.md` — the reconciled findings list (also summarized below).

**Sanitize before upload (Option B only):** strip cell `value` fields (keep `formula`, `style`,
`rowType`, `rowFill`, positions, merges). A generator spec needs structure and formulas, not the
actual item names/amounts/dates. Quick approach: a small jq/python pass that nulls every
`cells[].value`, or add a `--no-values` flag to `cmd/workbook-inspect`. Do NOT upload raw dumps.

---

## Findings the spec MUST incorporate (reconciled — read before writing)

1. **Two sheet families, different rules:**
   - *Expense sheets* (Fixas, Variáveis, Extras, Adicionais): palette `C0C0C0` month banner /
     `D8D8D8` Item·Data·Valor header / `F2F2F2` total rows. Category (col A) + subcategory
     (col B) are **filled DOWN every row** — NO merges. Blocks delimited by total rows.
   - *Other sheets* (Receitas, Listas de itens, Referência): different palettes AND they USE
     merges. Generator merges here; fills down on expense sheets.
2. **Receitas is shaped differently:** month banner at **row 1** (merged `A1:B1`="Mês"), month
   spans start at **col C** (not D), 3 cols each (Item/Data/Valor).
3. **Variáveis & Adicionais have a 3rd label column (C)** (sub-item); Fixas & Extras are A+B only.
   `B7B7B7` fill appears on col C of Variáveis total rows (darker than `F2F2F2`).
4. **Total rows carry 12 monthly `SUM` formulas** (one per month column), one per block.
5. **Black separator rows** between category sections — now in the JSON as `rowType:"separator"`
   with `rowFill:"000000"`. Read positions from the dump (Extras 39/67/83; Variáveis
   63/86/151/183/223/241; Adicionais 130/147; Fixas 140; Receitas 114/126). Generator emits these.
6. **Listas de itens is a rollup that PULLS, not SUMs:** each cell references the source sheet's
   total cell directly (`Fixas!F19`, `Receitas!E3`, …). `crossSheetRefs` lists the 5 sources.
7. **Referência de Categorias is the row-mapping source of truth:** col D = row in data sheet,
   col E = row in Listas, with `CONCATENATE("'Listas de itens'.F",$E5)`-style formulas that build
   reference STRINGS (ODS dot-notation, inside string literals — NOT real `!` refs). Uses
   **Cambria 14pt** (expense sheets use Open Sans headers / Arial data).
8. **Formatting to canonicalize:** currency is inconsistent in the source (`R$ 200,00` vs bare
   `209,80`); dates are inconsistent (`17/1`, `21/03`, `1/1/2025`). The generator should pick ONE
   canonical currency + date format.
9. **`D9E1F2` render discrepancy (Referência):** JSON hex is light blue but col A renders
   warm orange-yellow — likely theme remap or conditional formatting. **Open question** — the spec
   should flag it for resolution (inspect the workbook's theme XML / conditional-format rules)
   before trusting that hex.

---

## The prompt

### Option A (Claude Code) — paste as the session's task

```
Execute Layer 3 of the workbook-mapping plan: produce .claude/plans/workbook-generator-spec.md.

Read .claude/plans/workbook-layer3-instructions.md first — it lists the input files, the
reconciled findings, and the required output. Read the inputs it points to (start with
.claude/workbook-dump/manifest.json + Extras.json in full, then skim Receitas/Listas de
itens/Referência JSON, then .claude/workbook-visual-notes.md). Everything is local — do NOT
upload anything.

Produce a sheet-by-sheet generator spec that a future Claude Code session can implement as a Go
command that builds the workbook from a database of expense entries per subcategory. The spec
must contain, per sheet:
1. Layout rules (structure, not data): row bands, block pattern, column groups, where category/
   subcategory/data go, merge vs fill-down, separator rows.
2. Formula templates expressed parametrically (e.g. total row month col = SUM(<firstDataRow>:
   <lastDataRow>) for each of the 12 month columns; Listas cells = '<SourceSheet>'!<totalCell>).
3. Style palette per row type: fill hex, bold, borders, font + size — call out per-sheet
   differences (Referência Cambria; B7B7B7 on Variáveis total col C).
4. Cross-sheet wiring: how Listas de itens pulls from the 5 sources; how Referência maps rows.
5. Canonical currency + date format decisions.
6. Open questions / ambiguities to resolve before implementation (incl. the D9E1F2 render
   discrepancy and how to source row positions/block sizes).

Cross-check claims against the JSON as you go. Do NOT commit the dump or visual notes (gitignored).
When done, summarize the spec and the open questions.
```

### Option B (claude.ai) — deltas

Use the same body, but first upload the **sanitized** dump + `.claude/workbook-visual-notes.md` +
`.claude/workbook-map.md`. Prepend: "Here is the complete structural data and visual notes for a
7-sheet Excel workbook. I want to generate it from scratch from a list of expense entries per
subcategory." Then ask for the 6 deliverables above. Save the result to
`.claude/plans/workbook-generator-spec.md`.

---

## Expected output

`.claude/plans/workbook-generator-spec.md` — implementation-ready, no need to re-open the workbook.
One section per sheet (layout + formulas + styles), a cross-sheet wiring section, the canonical
format decisions, and a consolidated "Open questions" list.

## After Layer 3

- Resolve the open questions (esp. the D9E1F2 theme/conditional-format question — inspect the
  xlsx theme XML).
- Decide implementation scope: new `cmd/` command (e.g. `generate-workbook`) reading
  `expenses_log.jsonl` / `classifications.jsonl` as source of truth.
- This replaces the insert-into-existing-workbook approach long-term.
