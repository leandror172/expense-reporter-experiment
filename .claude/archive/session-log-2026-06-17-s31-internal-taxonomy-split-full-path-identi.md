## 2026-06-17 - Session 31: internal/taxonomy split + full-path identity key (T-02)

### Context

Continued from the workbook-generator session after PR #27 merged; picked up T-02 (real-taxonomy export + the penciled `internal/taxonomy` split).

### What Was Done

- Relocated render config (`dataYear`/`headroomRows`/`perGroupPctRows`) out of `taxonomy.go` into `generate` — T-02 prerequisite (commit 07f395a).
- Extracted `internal/taxonomy` as a pure input layer (domain types + loader), fully qualified all `taxonomy.X` references, deleted the alias shim; cycle-free (commit 21c6d4e).
- Authored the real `config/taxonomy.json` (112 subs) from the user's CSV — Gás consolidated to Variáveis/Habitação, Variáveis/Habitação `Produtos`→`Produtos de casa`, categories grouped by first appearance, Receitas income (Investimentos deferred); gitignored (commit 47b2379).
- Reworked `buildSubcategoryMap` to full-path identity + sticky ambiguity guard, distinct ambiguous skip message in `scanEntries`; relaxed spec §1.1; added decision doc `.claude/plans/taxonomy-identity-key.md` `[ref:taxonomy-identity-key]` (commit 47b2379).
- Updated memory QUICK/KNOWLEDGE across generate, module, and the new taxonomy package (commit 5fa22d8).

### Decisions Made

- T-02 reframed: NO export command/writer — the real taxonomy is a one-shot hand-authored file; long-term direction is DB ingestion, not workbook insertion (user).
- Subcategory identity = full path (sheet/category/sub; income group/label), not bare leaf name; only an exact full-path repeat errors, cross-path repeats are legal (real data repeats `Orion`, `Aluguel`).
- Bare-name routing retained but ambiguous names dropped (warn+skip, exit 0) until full-path entry routing lands — never a silent misroute; routing-by-full-path deferred to T-04.

### Next

- Open + merge the PR for branch `refactor/internal-taxonomy`.
- T-04 (deferred): full-path entry routing — entry contract + classifier + `scanEntries` + fixtures.
- T-03 year-rollover; then TF-IDF (5.R1).

### Gotchas

- `buildSubcategoryMap` is only hit for *validation* on the skeleton path (the map is discarded) — ambiguity/routing coverage comes from an entry-fed unit test, not skeleton generation.
- Local model (`my-go-qcoder`) verdict 0 on the loader change: O(n²) ambiguity scan, truncated output, and used `/` as the path separator when real names contain `/` (`Uber/Taxi`) — wrote from scratch (sanctioned for conceptual defects).
- "112 subs + no dups" is necessary but not sufficient for a hand-transcribed file; the CSV↔JSON symmetric-difference (empty) is the real fidelity check.

