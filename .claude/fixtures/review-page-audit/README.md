# review-page-audit fixture

`classified.csv` — 22 rows covering every branch of the review page's pre-fill rule plus
its badges, filters and parse paths. Written for the session-77 audit
(`.claude/review-page-audit-report.md`); kept because the page has **no automated browser
test** (T-61), so this is what makes the manual check cheap to repeat.

## Why it is here and not under `expense-reporter/test/fixtures/`
Those are consumed by the acceptance harness. This one is driven by a human in a browser,
so it lives with the audit docs rather than pretending to be an automated fixture.

## Running it
```bash
cd expense-reporter && go build -o /tmp/er ./cmd/expense-reporter
mkdir -p /tmp/erroot/config && cp config/*.json /tmp/erroot/config/ && cp /tmp/er /tmp/erroot/
cd /tmp/erroot && ./expense-reporter review \
    <repo>/.claude/fixtures/review-page-audit/classified.csv -o review.html --force
python3 -m http.server 8931 --bind 127.0.0.1     # Chrome automation cannot open file://
```
The scratch install root matters: `config.Load()` resolves relative paths against
`filepath.Dir(os.Executable())`, so running the repo binary in place would read production
config. `review` itself writes only the `-o` file — no log, no workbook.

Expected on a correct page: `21 rows (19 need review)` on stdout and **one** unreviewable
row (`S21`) named on stderr.

## What each row targets
| Rows | Target |
|---|---|
| S01, S02 | pre-fill case 1 — valid type in col 8, all three pickers filled |
| S03 | case 2a — type absent, exactly one type hosts (cat,sub) → resolves to `Extras` |
| S04, S05 | case 2b — ambiguous pair, type empty + "Pick a type" (2 and 3 candidates) |
| S06 | case 2c — no match, "Unmatched — needs all 3" |
| S07 | invalid type in col 8 → must fall through to the search and resolve to `Fixas` |
| S08, S09 | `auto_inserted=true` — the only rows the Auto filter and bulk-accept may see |
| S10–S13 | confidence spread 0.30 / 0.55 / 0.80 / 0.99 for the range slider |
| S14 (`ZZQQ`) | unique item text for the search box |
| S15 | category `Saúde` with no "saude" in the item — proves search is item-only |
| S16, S17 | installments: `405,25 x4` (T-64 multiplier) and `189,12/2` (divide) |
| S18 | `keyword_hint` populated (A1) → amber badge, nothing selected |
| S19, S20 | `already_logged` = `logged` / `partial` (T-80) → two distinct badges |
| S21 | empty date+value → dropped from the queue, named on stderr (S1) |
| S22 | `1.234,56` BR thousands (T-72 regression guard) |

The ambiguous pairs in S04/S05 are real: `Transporte/Estacionamento` (Fixas, Variáveis) and
`Pets/Orion` (Fixas, Variáveis, Extras). There are exactly **5** such pairs in the live
taxonomy, so this fixture must be re-checked if the taxonomy changes.
