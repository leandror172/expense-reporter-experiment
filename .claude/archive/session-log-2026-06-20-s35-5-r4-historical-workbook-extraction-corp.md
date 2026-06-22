## 2026-06-20 - Session 35: "5.R4 historical workbook extraction → corpus augmentation (694→1788) + per-year expense logs; PR #33"

### Context

Resumed after PR #32 (type-routing). User directed this session at extracting data from the old workbooks to feed the database under the new taxonomy — task 5.R4 (historical workbook extraction), done as a one-off, not a Go feature.

### What Was Done

- **5.R4 historical extraction (one-off Python, `.claude/scratch/extract_old_workbooks.py`):** header-driven extractor (handles both the standard 2023/24/25 layout and the divergent 2022 layout) lifted clean `{item,date,value,type,category,subcategory}` records from the 2022/2023/2024/2025 old workbooks at `~/workspaces/expenses/old/`. Outputs (outside repo, real values): `extracted-*` (faithful) + `normalized-*` (alias-remapped) + `partial-*` (value-only). 2022=139, 2023=868, 2024=340, 2025=397 full (+80 value-only 2025 Fixas).
- **Investigation findings:** old-workbook labels match the new taxonomy ~98.8%; only 4 drifted tuples → alias map (Gás→Habitação, Produtos→Produtos de casa, Manutenção carro→Extras/.../Carro, Imobiliária→Adicionais/Outros/Diversos). Date rule: band column = authoritative month, source file = authoritative year, cell = day only (kills Excel auto-year ±1yr artifact).
- **Dedup (`dedup_corpus.py`):** day-strict key (NFC-lower item, value, y, m, d) merged extraction with the 694-entry corpus → **merged-deduped.jsonl = 1808** (1797 typed). Corpus 586/694 covered by extraction; 108 net-new (75 user_corrections).
- **Training corpus augmented (`build_corpus.py`):** `training_data_complete.json` rebuilt 694→**1788** (dropped 9 no-date + 11 dash-placeholder junk), 2022–2025, 15 cats / 81 subs. Pristine 694 backed up `.bak-20260620-134514`.
- **Per-year expense logs (`build_logs.py`):** `expenses_log-{2022,2023,2024}.jsonl` (138/853/349) + 2025 merged into base `expenses_log.jsonl` (355+378=733, deduped by GenerateID; backup `.bak-20260620-134855`). All three route through `generate-workbook --year N` (exit 0, real xlsx).
- **PR #33 opened** (`chore/extraction-gitignore-memory` → master): `.gitignore` widened for per-year logs + personal-data backups (`*.bak-*`); stale per-folder memories (feedback/taxonomy/classifier) corrected.

### Decisions Made

- Destination = entries-log JSONL (user choice); year scope 2022–2025; drift handled by alias-table, taxonomy.json left canonical.
- expenses_log stays year-implicit (`parseDate` requires exactly `DD/MM`) → split into per-year files rather than DD/MM/YYYY, which would break routing.
- Dropped dash-placeholder items from training corpus only (noise), kept them in the financial logs (real money).

### Next

- Merge PR #32 (type-routing) then PR #33 (gitignore + memory).
- Then: year-rollover (T-03); year adaptation (T-11) to collapse per-year logs; retire bare-name fallback (T-09, now gated on giving income its own `incomePath` route, not the classifier); 5.R4 id collision (T-10); TF-IDF (5.R1).

### Gotchas

- The feedback/taxonomy memories claimed type emission was "apply-path only / classifier can't emit type" — stale since 5.R4 landed in PR #32. Corrected this session.
- `*.bak` in .gitignore does NOT match timestamped `.bak-YYYYMMDD…` suffixes — the training-corpus and log backups (real expense data) were unignored until widened to `*.bak-*`.
- 2022 workbook only holds Nov–Dec (tracking started late 2022); 2022 Fixas sheet is genuinely empty.
