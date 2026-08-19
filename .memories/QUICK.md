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
`correct`; new `apply-stale-id` is the only fixture that discriminates.**
**T-42 SCOUT RUN (s68) — the chain ran on real 2026 data for the first time**
(`.claude/t42-scout-report.md`; isolated in a scratch install root, real logs verified
byte-identical). It found 2 hard blockers in `review`, both now FIXED:
**T-54** — `ReadQueue` accepted only `1`/`0` for `auto_inserted` while the only producer
emits `true`/`false`; the `1`/`0` spelling had NO producer and was invented by fixtures
(PR #61). **S2** — `review` built its picker from the WORKBOOK's Referência sheet while
everything else routed with `config/taxonomy.json`; the two had drifted **7 leaves**
(`IRFF`/`IRRF`, `Apoia-se 4i20`/`Apoia-se`, …), so a workbook-only pick vanished silently
from the generated workbook — and 2 logged "corrections" had been teaching the classifier
the unroutable spelling (3 real records backfilled). **S1** — an unparseable row is now
skipped and named on stderr instead of killing the step (4 of 69 rows did). **S5** —
`generate-workbook` takes `--taxonomy` from config (`--entries` deliberately NOT defaulted:
omitting it is how you ask for an empty year skeleton).
**`review` now reads NO workbook — `excel.LoadReferenceSheet` has only WS-E dead callers
left, so the workbook-as-source retirement is finally done on the live path.**
**T-21 DONE (s69, PR #63)** — the installment count now rides `ReadQueue` →
`QueueEntry` → page DATA → `exportReviewed()` → `reviewed.json` → `ReviewedEntry` →
`ExpandAndAppend`, so a reviewed `99,90/3` lands as 3 rows instead of 1 (was 10 of 45
real rows, 17 missing entries). `installments` is REQUIRED — an absent key decodes to 0,
so defaulting to 1 would hide a producer that stopped emitting it (the T-54 shape).
The apply summary now counts ROWS, not entries: "Appended: N rows" had been true only
while entries and rows were 1:1. **Measured gap kept open on purpose:** deleting the
field from `exportReviewed()` leaves the whole Go suite green — the export runs in the
browser and no Go test reaches it; a committed browser test needs a Node/`playwright-go`
toolchain in a Go-only repo. Do NOT close it with a hand-authored fixture (that IS T-54).
Also new: apply can now leave a PARTIAL installment series on a mid-append failure, with
no ledger to warn.
**T-42 DONE (s71) — THE FIRST REAL MONTHLY CLOSE IS APPLIED.** 69 rows → 17 auto /
48 review / 4 unparsed; 67 rows appended, reconciliation exact, 0 workbook skips; log
2073 → **2159**, 86 rows dated 2026. T-21 visible in the deliverable (`IPVA 2026 (1/5)…(5/5)`
is five workbook rows). **T-63 shipped (PR #65):** `batch-auto` writes `failed.csv` — the
rejected rows' only durable artifact — reason as a trailing `#` comment on the SAME line, so
the file is repaired in place and re-run as-is. Two findings need decisions, written up in
`.claude/t42-close-findings.md`: an auto-inserted row **cannot be corrected** (T-65), and
**38% correction rate / 92% cross-CATEGORY / the keyword layer already held the answer 60%
of the time** (T-66, which inverts T-27's premise). T-64 filed for the `405,25 x4`
notation — it MULTIPLIES where `total/N` divides, so confirm the direction first.
**s72 — THE GATE IS MEASURED AND A1 IS BUILT (PR #66).**
**Absolute agreement-gate precision = 94.1%** — 1 of 17 gate-passing rows corrected vs
**50.0%** error on the 48 gate-refused rows, an 8.5× discrimination on a full unselected
month, from data already on disk (`.claude/b1-gate-precision-measurement.md`). T-32 called
this unmeasurable because the 649 was a confidence-selected subset; a CLOSE labels
everything, since `close-cycle.sh:220` feeds `review` the FULL `classified.csv`. **This
also CORRECTS `t42-close-findings.md`:** the 17 auto rows were NOT unseen — all were
reviewed, as its own `4x álcool 70` example proves. The defect is persistence (T-65), not
review coverage, so **B1 (gate-to-review) was dropped** — its two justifications were this
measurement and removing unrepairable rows, and the first is now free. WS-D still HELD:
n=17, and confirms on auto rows are weak-positive (the page shows them as already handled).
**A1 SHIPPED (PR #66):** `classifier.KeywordHint` — the advisory sibling of the gate, firing
on model⊕keyword DISAGREEMENT at unambiguous specificity 1.00 — rides a NINTH CSV column
into the review page as an amber badge. Measured: fires on 14 of 65 rows, right 71.4%,
recovers 10 of 25 corrections. **Advisory only** (auto-applying would inject ~3 errors per
14) and deliberately NOT round-tripped through `exportReviewed()` (that would be a second
T-61). **T-64 direction CONFIRMED by the user: `x4` MULTIPLIES** (per-installment value).
Next: T-65 supersede-by-append — **DESIGN ONLY, no code yet** (user's call); then T-64.
**s65 date/year hardening (merged):** resolved-year
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
