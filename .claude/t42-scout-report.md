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

### S2 — `review` still sources its TAXONOMY from the workbook, and the two taxonomies have drifted
**(corrected + upgraded after user review — this is the severe half of S2, not the missing file.)**

`cmd/review.go:52-70` hard-requires a workbook, validates it, loads the Referência sheet and
builds the picker taxonomy from it. **`review` never reads `config/taxonomy.json` at all.**

**This is NOT covered by WS-E.** WS-E is scoped to the *write* side (`internal/workflow`, the
`excel` write side, `internal/batch`'s insert path, plain `batch`, `utils.ParseDate`).
`review`'s use is a **read on the live path**, so deleting the write side leaves
`excel.LoadReferenceSheet` standing and `review` still calling it. The workbook-as-source
retirement is therefore **not finished**, and finishing WS-E as written would not finish it.

So the close cycle runs on TWO taxonomies, and they disagree — measured this session
(104 leaves in `taxonomy.json`, 103 in the workbook, **7 divergent**):

| Workbook (what the human PICKS from) | `taxonomy.json` (what ROUTES the row) |
|---|---|
| `IRFF` | `IRRF` — T-13's typo fix landed in taxonomy.json only |
| `Apoia-se 4i20` | `Apoia-se` — the reading guide's durability warning, confirmed live |
| `alguma coisa sindicato` | `extra sindicato` |
| — | `Produtos de casa` |

**Harm:** classifier proposes from A, human picks from B, `apply` writes the pick, and
`generate-workbook` routes with A. A pick of `IRFF` / `Apoia-se 4i20` / `alguma coisa sindicato`
hits `scanEntries`' "not in taxonomy" warn-and-skip path — **the row silently vanishes from the
workbook**. Same silent-loss family as the rest of this session's findings.

**Fix direction:** point `review` at `config/taxonomy.json` (which every other command already
uses), which also removes the last workbook read from the close cycle and makes S2's dangling
`workbook_path` irrelevant rather than needing repair.

### S2b — the configured workbook does not exist
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

**CORRECTED after user review.** An earlier draft of this report concluded that `- 1/4` and
`4x` are real-usage installment forms the parser should learn. **That is wrong.** Per the user:
these four lines are simply **mistakes in the file**. The accepted installment format is
`99,90/3` and only that. So all four rows are user input errors, not a missing feature, and
**T-21 needs no new notation work**<sup>[superseded s71 — see below]</sup> — the count is already reachable for correctly-written
input, which is what T-41 slice 3 established.

> **CORRECTION TO THIS CORRECTION (s71, T-64).** The `4x` / `x4` half of the ruling above is
> wrong. The multiplier form is a notation the user had planned but never recorded; the two
> rows using it are real installment purchases, not typos. Critically it is NOT a synonym for
> `total/N`: `x4` means the written value is PER-INSTALLMENT (multiply), while `total/N`
> divides — `405,25 x4` = 1.621,00 where `405,25/4` = 405,25. Filed as T-64, which must
> confirm the direction with the user before implementing. The `- 1/4` half of the ruling
> stands: that one is a typo.

What survives as a finding is narrower but still real: **4 bad rows out of 69 (6%) is the
normal error rate of hand-typed input**, and one such row currently kills the entire `review`
step (S1). The lesson is about tolerating malformed rows, not about parsing more notations.
The transposed-fields case fails loudly and correctly (`109,39` is not a date) — though see
T-52, since `auto`'s own argument order is transposed relative to `parse.Fields`.

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

1. **S2 — repoint `review` at `config/taxonomy.json`.** Highest severity (silent row loss via
   taxonomy drift), and it retires the last workbook read in the close cycle. Do it FIRST:
   it changes what `review` needs, which changes S1's and T-21's context.
2. **S1 — decide the malformed-row contract in `ReadQueue`**, and extend the T-54 seam test
   with an unparsed row (a known gap in the guard shipped this session).
3. **T-21 — thread the installment count.** No new notation work needed (see the S4
   correction); same reader as 1 and 2, so all three touch `queue.go` once.
4. **S5** — cheap, directly in the close path.
5. Re-run this scout; then the real close.
