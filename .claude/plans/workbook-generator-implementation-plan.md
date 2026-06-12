# Workbook Generator — Implementation Plan & Next-Session Brief

Written 2026-06-10 (session 27), branch `feat/workbook-generator`. This file carries
everything the next session needs so it does NOT have to recompute what session 27 already
established. Target executor: **Sonnet (or Opus for Phase G2)** — the spec and golden
master carry the design; the remaining work is well-bounded.

---

## 0. Preparation — what to read, in order

1. `.claude/tools/resume.sh` output (session-start ritual).
2. **`.claude/plans/workbook-generator-spec.md` (spec v2)** — the single design authority.
   Where spec and original workbook disagree, spec wins (it's a redesign).
3. **`.claude/workbook-template/convergence-report.md`** — current builder↔golden-master
   state: 41 accepted residuals + 6 golden-master hand-edit artifacts. Don't "fix" these
   blindly; each is justified there.
4. `.claude/workbook-template/review-diff.md` — the 14 hand-review patterns (context for
   why v2 looks the way it does). Skim.
5. `.claude/workbook-template/ambiguities.md` — Phase-A dogfood decisions (A1–A14). Skim;
   consult when a spec sentence feels underdetermined — it may already be resolved there.
6. Builder code: `.claude/scratch/template-builder/*.go` (~10 small files). This is the
   REFERENCE IMPLEMENTATION of spec v2 — the real generator ports its logic, not its shape.
7. For acceptance work: `expense-reporter/test/PATTERNS.md` + `ref-lookup.sh
   acceptance-harness` / `acceptance-fixtures` / `acceptance-verify`.
8. `.claude/overlays/local-model-conventions.md` before any codegen (HARD requirement).

Do NOT re-read `.claude/workbook-dump/*.json`, the digests, or visual notes — the spec
already distills them. They exist only for spot-checking a suspected spec error.

## 1. State as of session 27 (done — do not redo)

- Layer 3 complete: spec v1 synthesized from dump+notes via 7 Sonnet digests, patched to
  v2 after user hand-review.
- Phase A complete: standalone builder (`.claude/scratch/template-builder/`) generates
  `.claude/workbook-template/template.xlsx` CONVERGED to the user-curated golden master
  `template-reviewed.xlsx` (verify anytime: `cd .claude/scratch/template-builder && go run .`
  then `python3 ../../workbook-template/diff.py` — see convergence-report header for exact
  invocation; dumps land in gitignored dump-* dirs).
- Key v2 design decisions (full list spec §6.14): months start col C; vertical MERGES
  replace fill-down; sub-item level eliminated (composed "Orion - Consultas" strings);
  Mês A1:B2; freeze C3/D4; Receitas in the same family; Listas 3-col label area, months D–O.
- Referência sheet omitted (not an insertion target). B7B7B7 moot (spec §7 Q-B7).
- Tooling facts learned: excelize `SetCellFormula` takes NO leading `=`; stale-formula fix
  is `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`; `/tmp/workbook-inspect` built
  from `expense-reporter/cmd/workbook-inspect/`; no `jq` on this machine — use python3;
  openpyxl IS available; excelize serialization noise vs openpyxl: `fontcolor None vs
  000000` and empty-cell `font.size 11` are benign.

## 2. Phase B — data validation (next session, first half)

Goal: prove the formula/value layer, not just structure.

1. **User task (blocks B; ask early):** copy `template-reviewed.xlsx` →
   `template-data.xlsx`; hand-fill a few fake entries (2–3 months, ≥1 block with multiple
   entries in the same month, ≥1 Receitas entry). Also ADD the per-group percent rows to
   Listas while editing (spec §4.2 ⚠ — "% sobre Despesas <S>" + "% sobre Receitas",
   CCCCFF, numFmt 0.00%; formulas spec §4.2).
2. Extend the builder: accept entries (hardcode the same fake entries in `taxonomy.go` for
   now — the input-contract plumbing belongs to the real generator, not the scratch
   builder); flip `perGroupPctRows` to true; emit typed values (float for Valor, real date
   for Data) so numFmts render.
3. Converge again with `diff.py` against `template-data.xlsx`. New surface exercised:
   value typing, SUM results, pulls, percent denominators, saldo chain.
4. Render check (user, in LibreOffice/Sheets): white-on-indigo font, merged headroom
   tails with data above them, `DD/MM` and `R$` rendering. Capture verdicts in the spec §7.

## 3. Phase G — the real `generate-workbook` command (next session, second half / session after)

### G1. Extraction library (prerequisite, mechanical)
Lift `cmd/workbook-inspect/main.go`'s extraction core into `internal/inspect` (keep the
cmd as a thin wrapper). Pure refactor — existing behavior, no new features. Good Ollama
target (`my-go-qcoder`) since it's bounded restructuring; unit-test with a tiny fixture
xlsx generated on the fly.

### G2. Generator port
- New: `cmd/expense-reporter/cmd/generate.go` (cobra: `generate-workbook -o <path>
  [--year N] [--headroom N]`) + `internal/generate/` package.
- Port the scratch builder's layout/styles/listas logic. Replace the hardcoded taxonomy
  with the spec §1 input contract:
  - taxonomy source: DECISION NEEDED at implementation time — options: (a) read from
    `Referência` of an existing workbook via `internal/resolver` (reuses code, needs a
    workbook), (b) a JSON/CSV taxonomy file (clean, new format), (c) derive from
    `expenses_log.jsonl` subcategory/category fields (no new file, but only covers
    subcategories that have entries). Recommend (b) with (c) fallback; confirm with user.
  - entries source: `expenses_log.jsonl` (`internal/feedback.ExpenseEntry`; fields item,
    date, value, subcategory, category) filtered by `--year`.
- Style/layout constants in one file, mirroring the builder's `styles.go`/`colmap.go`.

### G3. Acceptance tests (write FIRST per user's acceptance-first preference)
- Fixture: `test/fixtures/generate-basic/` — small taxonomy file + entries JSONL (fake),
  plus expected dump JSONs produced from a committed mini golden master.
- Given `taxonomyAndExpensesRecorded(fixDir)` / When `RunGenerateWorkbook` / Then
  `verify.WorkbookStructureMatches(expectedDumpDir)` (new verifier on `internal/inspect`).
  Fully deterministic — no Ollama, no soft assertions.
- Two scenarios: structure-only (empty entries) and with-data.
- Unit tests (testify, table-driven) during port: colmap math, block-size/position
  computation, formula rendering, merge ranges.

### G4. Wiring & docs
- Register command in `expense-reporter/README.md`; index.md entries; CLAUDE.md untouched
  (no new hard rules).
- Keep `.claude/scratch/template-builder/` until G3 is green, then either delete or mark
  superseded in a final commit.

## 4. Later / explicitly out of scope for next session

- Year-rollover workflow (generate 2027 workbook from taxonomy alone).
- Regenerate-from-logs replacing insert-into-workbook; fate of `apply`/`add` against
  generated workbooks; slim Referência + resolver taxonomy-source question (spec intro).
- Headroom per-subcategory overrides (spec §7.1) — only if Phase B shows the need.

## 5. Open questions inventory (live list: spec §7)

| # | Question | Status |
|---|---|---|
| 1 | headroom default 3 | test in Phase B |
| 2 | per-group percent rows | required; enter golden master + builder in Phase B step 1–2 |
| 3 | B7B7B7 | moot (record only) |
| 4 | merged headroom tail render | check in Phase B step 4 |
| 5 | Dólar row semantics | settle in Phase B saldo work |
| G | taxonomy source for the real generator | ✅ RESOLVED (2026-06-11): (b) dedicated **JSON taxonomy file** as primary skeleton source; entries from `expenses_log.jsonl`; option (a) Referência read demoted to a possible one-time export/bootstrap tool |

## 6. Working agreements that apply (from this session's flow)

- Sonnet subagents for bulk extraction/diffing; keep main-session context lean; digests
  and reports go to FILES, not just agent replies.
- Ollama-first for bounded new Go units (tier list in CLAUDE.md); direct edits OK for
  tightly-coupled iterative work (overlay's multi-file exception) — but record the call.
- Convergence claims must come from `diff.py`/dump diffs, never eyeballing.
- Commit per green step on `feat/workbook-generator`; PR to master when G3 is green.
