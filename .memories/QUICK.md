# expense-reporter repo — Quick Memory

*Working memory for the repo root. Injected into agents. Keep under 30 lines.*

## Status
Layer 5 done through 5.9; workbook generator COMPLETE. **Pivot: JSONL logs are the
single source of truth; `generate-workbook` the only writer.** Landed: T-13 full-path
classification, WS-A/B/C, 5.R2 embed-on-miss (s54), T-32 agreement gate (s57 —
confidence measured dead; keyword agreement gates), T2 harness module v1.0.0,
T-20 resume+dup warning (s60), T-35 join-id canonicalization (s61), T-39 suite split
+ T-24 think-off default (s62, PR #52). **PARSE BOUNDARY (T-41) UNDERWAY (s63,
branch `feat/t41-parse-boundary`): plan FINAL (all §10 locked — `internal/parse`,
single `time.Time` + `DateString()`, grace 0, internal-first). Slice 1 DONE:
add/correct migrated, `--year` flags, `date_year` live (rung 3), BR thousands
parse. **Slice 2 DONE (s66): `auto` migrated** — single `parse.Fields` call,
`--year`, stale-year warning, `appendExpense(pe ParsedExpense, …)` collapse,
`parse` field sentinels (`ErrInvalidDate`/`ErrInvalidValue`, double-`%w` so the
cause survives). **Slice 3 DONE (s67): `batch-auto` migrated** — `inputRow` and the
unreachable `resumeParseErr` deleted, `--year`, one counted stale-year warning
(T-49), join-id pinned at construction (T-40); the classified CSVs now carry the
canonical date, repairing a silent `review`→`apply` id divergence. **Slice 4 DONE
(s67): `apply` migrated — T-41 COMPLETE, all six parsers route through the boundary.**
`canonicalDate` retired into new `parse.Date` (date-only entry point; `Fields`
delegates to it), and `entry.ID` is recomputed from the canonical date — apply had
been looking up an id it never writes, and that miss is SILENT (unfound → appended →
duplicate). Fixtures swept off the raw-date ids T-18 had already migrated for
`correct`; new `apply-stale-id` is the only fixture that discriminates. Next: T-21 →
T-42 monthly close.** **s65 date/year hardening (merged):** resolved-year
validations (renderable; entry beyond the current year refused), `YearSource`
provenance, stale-`date_year` stderr warning, `date_year` → 2026.
WS-D (HELD: gate-to-review) → WS-E after.
History + measurement detail → KNOWLEDGE.md "Milestone Log".

## Repo Structure
```
expense-reporter/         # Go module — CLI application (Cobra, excelize, Ollama)
mcp-server/               # Python MCP server — thin wrapper calling Go binary
data/classification/      # Taxonomy, training data (JSONs gitignored)
docs/archive/             # Historical planning docs (read-only)
.claude/                  # Session tracking, tools, overlays
```

## Key Rules
- **Go CLI is the product** — all classification logic lives in Go, not the MCP wrapper
- **Brazilian locale** — DD/MM/YYYY dates, comma decimal (1.234,56), BRL currency
- **Hierarchical categories** — Type → Category → Subcategory; identity = full path
- **Local-first ML** — Ollama models for classification; no cloud API calls
- **Sensitive data gitignored** — training JSONs, expense CSVs, personal financial data

## Deeper Memory → KNOWLEDGE.md
Architecture layers · classification strategy · cross-repo relationships · milestone log
