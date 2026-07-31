# T-42 Scout — first real run of the full chain (session 68, 2026-07-31)

**Purpose:** convert a speculative backlog into a measured one. Ran the real
`~/workspaces/expenses/expenses-2026-01.csv` (69 rows) through
`batch-auto → review → apply → generate-workbook --year 2026`.

**Isolation:** a scratch install root (binary + its own `config/` + a copy of
`data/classification`), because **no flag redirects the logs** — `config.json` and relative
log paths both resolve against `filepath.Dir(os.Executable())`. `--data-dir` is the *training
data* dir, not the log dir; it also owns the 5.R2 embedding cache
(`EmbeddingCachePath(cfg.DataDir, model)`), so pointing it at real data would write there.
**Verified after the run: both real logs byte-identical** (`classifications.jsonl` 649 lines
`2ccdd6cda85b`, `expenses_log.jsonl` 733 lines `b4dc2e57f001`), repo clean.

## Chain status

| Step | Result |
|---|---|
| `batch-auto` | ✅ 1m27s — 20 auto / 45 review / 4 parse errors |
| `review` | ❌ **blocked twice** (S1, S2) |
| `apply` | ✅ 45 appended, 0 failed |
| `generate-workbook` | ✅ all 67 entries present, 0 silent drops — with flag friction (S5) |

## Blockers

### S1 — `review` cannot read a `classified.csv` containing ANY unparsed row
`batch-auto` **deliberately** writes an unparsed row with EMPTY date/value cells (pinned by
`cmd.TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine` — the alternative was rendering a zero
time as `01/01/0001`). `review.ReadQueue` then hard-errors:
`reading queue: line 20: invalid value: value string cannot be empty`.
4 rows out of 69 killed the entire step. Producer contract and consumer contract contradict
each other, deliberately, on both sides.

**This is the same defect family as T-54 in a different column — and the T-54 seam test does
NOT catch it**, because it only feeds successfully-parsed rows. That is a real gap in the guard
shipped this session. Fix = extend `classified_csv_seam_test.go` with an unparsed row, then
decide the reader's contract (skip the row? carry it as unreviewable?). Do it with T-21/T-54
follow-up — same reader.

### S2 — `review` requires a workbook, and the configured one does not exist
`review` builds its taxonomy from the **workbook's Referência sheet**, not from
`config/taxonomy.json` that every other command reads. Meanwhile `config.json`'s
`workbook_path` (`../Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx`) names a file
**absent from disk anywhere**. Only `Planilha_Normalized_Final.xlsx` exists.

Post-pivot this is a leftover coupling: logs are the source of truth and `generate-workbook`
is the only writer, yet the review surface still depends on the OLD workbook. Decide whether
`review` should read `config/taxonomy.json` instead — that would remove the last workbook read
from the close cycle.

## Measured, not guessed

### S3 — T-21 is 22% of the review queue, not an edge case
10 of 45 reviewed rows carry installment notation. `apply` recorded **10 log rows where 27
belong** — 17 missing rows, i.e. the budget is silently under-recorded by that much:

| Item | Value | Rows written / owed |
|---|---|---|
| Consulta Elizabeth cardiologista Lilly | `250,00/3` | 1 / 3 |
| Exame pressão Lilly | `60,00/3` | 1 / 3 |
| Exame eletro Lilly | `280,00/3` | 1 / 3 |
| IPVA 2026 | `323,65/5` | 1 / 5 |
| Petlove sachês | `204,18/2` | 1 / 2 |
| Exames Lilly sinplan | `310,00/3` | 1 / 3 |
| Consulta Orion nefro Território animal | `40,00/2` | 1 / 2 |
| Consulta Lilly nefro Território animal | `40,00/2` | 1 / 2 |
| Consulta Lilly nefro Kelly | `110,00/2` | 1 / 2 |
| Consulta Orion nefro Kelly | `110,00/2` | 1 / 2 |

### S4 — the input formats real usage produces (4/69 = 6% rejected)
These are **human-authored** shapes, which is what the vision says the source is (an expense
posted in chat, Layer 6) — NOT a bank export:

| Raw line | Why it failed |
|---|---|
| `Anita;Elô ADM;09/01;405,25 - 1/4` | semicolon **inside the item** → 4 fields; plus ` - 1/4` installment form |
| `Comida cinema;109,39;14/01` | date and value **transposed** |
| `Café padaria;201/01;13,00` | typo'd day (`201`) |
| `Anita compra chocolate Ruby 299,00 e cacau 49,90;646,25 4x` | no date at all; `4x` installment form |

**The headline: installment notation in real use is `- 1/4` and `4x`, not the `99,90/3` the
parser expects.** T-21 threads a count the parser mostly never gets to see. Any T-21 work should
decide whether the boundary learns these forms — the chat layer will receive exactly these.
The transposed-fields case fails loudly (good: `109,39` is not a date), but see T-52 — `auto`'s
argument order is itself transposed relative to `parse.Fields`.

## Friction (not blocking)

- **S5 — `generate-workbook` ignores config.** Requires `--taxonomy` and `--entries`
  explicitly, while every other command reads `taxonomy_path` / `expenses_log_path` from
  `config.json`. Footgun for a monthly close: the wrong `--entries` silently builds the wrong
  workbook.
- **S6 — `batch-auto --help` is stale.** Still says it auto-inserts "into the workbook" and
  describes `--dry-run` as skipping workbook insertion; WS-B slice 3 made it append to the log.
  T-36 family.
- **S7 — T-55 did not fire and remains unverified**, exactly as predicted: a 2026-only run has
  `apply`'s `--year` default (`time.Now().Year()`) equal to `date_year`, so the unreachable
  config rung is invisible.

## Confirmed working on real data

- **T-54 (this session):** `review` reads real `batch-auto` output once the rows parse — 65
  rows, 45 needing review. The `true`/`false` + canonical-date CSV is correct end-to-end.
- **T-13:** all 45 reviewed entries carried a non-empty type. Zero type-less. Auto rate 20/69 = 29%.
- **T-41 slice 4:** `reviewed.json` written with EMPTY ids applied cleanly (0 failed) — ids
  recomputed from the canonical date, as designed.
- **generate-workbook:** all 67 log entries present in the output workbook, 0 silently dropped
  (checked against sharedStrings, aggregates only).

## Suggested order

1. **S1 + S2** — both are `review`, and S1 shares the reader with T-21/T-54.
2. **T-21** (+ decide S4's installment forms) — same reader again; do it in the same pass.
3. **S5** — cheap, and directly in the close path.
4. Re-run this scout; then the real close.
