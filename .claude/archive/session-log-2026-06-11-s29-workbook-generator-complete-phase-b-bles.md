## 2026-06-11 - Session 29: Workbook generator complete — Phase B blessed, Phase G shipped, PR review addressed

### Context
Resumed from session 28 handoff: re-review the regenerated data-bearing template, settle 2 open
questions, then Phase G. Branch `feat/workbook-generator`. Sonnet pre-approved for subagents.

### What Was Done
- **Phase B closed:** open questions resolved (spec §7.6/§7.7: Receitas numFmt = DD/MM + R$
  uniform; Listas section headers navy 333399 — spec wins over the Phase A black accommodation).
  Sonnet re-review of the regenerated `template.xlsx`: **PASS, 0 blocking defects**
  (`.claude/workbook-template/phaseB-rereview.md` + 8-item double-check list). Its one
  actionable observation (section `% sobre receita` guarded on numerator, not the D9 income
  denominator) fixed. **User blessed the data-bearing golden master.** `RevenueSheet` Labels
  field added (sheet name + Listas pulls + section header + sheet ordering).
- **G1:** `internal/inspect` — dump core lifted verbatim from `cmd/workbook-inspect` (now a
  thin wrapper); output verified byte-identical; unit tests (qcoder, verdict 1).
- **G3 (acceptance-first, advisor-reviewed):** Opus advisor review
  (`.claude/advisor-G3-acceptance-design.md`) reshaped the design — input contract pinned in
  spec §1.1 (taxonomy JSON schema; entries DD/MM no-year; unknown subcat → warn+skip exit 0;
  taxonomy authority on category mismatch); **oracle bootstrap**: scratch builder taught to read
  the fixture (`LoadTaxonomy` loader + flags), its dumps frozen as `expected-dump-*`;
  `verify.WorkbookStructureMatches` compares a NORMALIZED SUBSET (ignores widths/heights/
  manifest source). 3 scenarios born RED on `unknown command`. Fixture
  `test/fixtures/generate-basic/` documented in PATTERNS.md (new sub-format).
- **G2:** `internal/generate` (builder port) + `generate-workbook -o --taxonomy [--entries
  --year --headroom]` cobra command (qcoder, verdict 2) — **all 3 acceptance scenarios green
  on first run**. Loader + builder math tests ported (Phase B fake dataset is now
  `taxonomy_fixture_test.go`).
- **G4:** README command docs; scratch builder marked SUPERSEDED; **PR #27** title/body updated
  to full scope (via REST API — `gh pr edit` hits a projects-classic GraphQL bug on this repo).
- **PR review comments addressed** (2 drafts in the user's still-pending review): identifiers
  → English (Categoria→Category, Receitas*→Revenue*, listas*→summary*, despesa*→expense*,
  saldo*→balance*; files renamed summary_sheet.go/revenue_sheet.go; pt-BR strings only in
  `Labels`); revenueSection/expenseSection extracted into single-responsibility `write*` steps.
- **Latent bug found by the refactor:** summary sections + balance block hardcoded the 4-sheet
  order → invalid `D0`/`E0` refs for smaller taxonomies, PRESENT IN THE FROZEN DUMPS (oracle
  shared the bug). Fix: registry `sheetOrder`; dumps re-frozen with a manually reviewed delta
  (exactly 6 phantom rows + 4 bogus refs removed; other sheets byte-identical).
- Memories updated (root/expense-reporter/test/inspect QUICK+KNOWLEDGE; new
  `internal/generate/.memories/QUICK.md`; auto-memory rewritten). 10 commits, all pushed.

### Decisions Made
- **Taxonomy source = (b) dedicated JSON file** + entries from `expenses_log.jsonl`; option (a)
  Referência read demoted to a possible one-time export tool. JSON over CSV (nested structure).
- **Oracle-frozen expectations** (advisor): freeze the trusted scratch builder's dumps BEFORE
  the port → G2 = converge-to-green. Recorded limit: oracle and port can share a bug —
  acceptance can't see it; on contract changes, re-freeze + manually review the dump delta.
- **`incomeCategories[].name` is block grouping**, not the sheet label (that's
  `Labels.RevenueSheet`, which appears inside formulas → schema identifier, not cosmetic).
- Verbatim code moves go to sed/python, NOT the local model (3 warm timeouts on a 530-line
  transcription; zero design value, pure drift risk). Models kept for synthesis: loader (1),
  loader tests (1 — hallucinated 8 fixture literals despite being given them), verifier (1),
  acceptance tests (2), cobra cmd (2).

### Next
- **Merge PR #27** (user: also submit/discard the pending review — drafts are invisible until then).
- One-time export: real 113-subcategory taxonomy (Referência) → `taxonomy.json`.
- Year-rollover workflow; fate of `apply`/`add` vs generated workbooks. Then TF-IDF (5.R1).

### Gotchas
- `gh pr edit`/`pr view --comments` fail on this repo (projects-classic GraphQL deprecation);
  use `gh api repos/.../pulls/27` REST instead.
- The expenses log stores `date` as `DD/MM` — NO year component; `--year` supplies it.
- excelize `f.NewSheet` returns (int, error); style via NewStyle+SetCellStyle (no SetCellFont);
  `MergeCell` not `MergeCells` — recurring local-model API confusions, all caught by compile.

