# Review page — behavior audit (test plan)

**Status:** DRAFT — test plan written, not yet executed
**Raised by:** user, session 77. Three concerns: (1) filters may not work, (2) the Type
dropdown is never filled, forcing manual selection — is that expected?, (3) keyboard
shortcuts are not exhibited anywhere.

## Context

`review` renders an offline HTML page (`internal/review/template/review.html`, 1817 lines,
one `go:embed` template) that a human uses to adjudicate every classified row before
`apply` writes it to the log. It is the ONLY human checkpoint in the close cycle, and the
six 2026-03..08 closes ahead of us will push ~227 rows through it — 5x the largest close
done so far. Any friction or silent misbehavior here is multiplied by that.

The page has never been tested through a browser. Its Go side is well covered
(`queue_test.go`, `render_test.go`, `taxonomy_test.go`, `test/review_test.go`), but
**everything after the JSON is embedded runs in the browser and no test reaches it** —
the same measured gap T-61 files for `exportReviewed()`.

## A. Expected behavior (the spec, read off source + package memory)

### A1. Pre-fill rule (`prefillFor`, template ~line 777)
Documented in the function's own comment block:

| # | Condition | Expected result |
|---|---|---|
| 1 | `predicted.type` present AND hosts (cat,sub) | all 3 pickers filled |
| 2a | type absent/invalid, exactly 1 type hosts (cat,sub) | all 3 filled |
| 2b | >1 type hosts (cat,sub) | cat+sub filled, type EMPTY + flagged "Pick a type" |
| 2c | 0 types host (cat,sub) | all 3 empty + flagged "Unmatched — needs all 3" |

So **case 1 and 2a should arrive pre-filled**. Manual type selection is expected ONLY for
2b and 2c. The user's report that it is *never* filled contradicts the spec.

### A2. Filters (`visibleStates`, ~line 1009)
- **Status segment** — `Needs review` (default, hides `auto_inserted`) / `Auto-inserted` /
  `All`.
- **Confidence range** — two sliders, step 5, inclusive both ends.
- **Search** — substring, case-insensitive, **matches `item` ONLY** (not category/subcategory).
- **`All` sorts** rows that loaded unmatched/ambiguous to the top, by INITIAL flags, so
  resolving a row does not make it jump.
- **Group by** — Queue order / Predicted category / Predicted type / Status.
- Filter state persists in `localStorage` under `expense-review:v1:filters` (global, NOT
  keyed by source).
- **Header counts are GLOBAL, not filtered** — total/reviewed/skipped/remaining count all
  of STATE regardless of the active filter. By design, but a likely source of the
  "filters aren't working" impression.

### A3. Keyboard shortcuts (`keydown` handler, ~line 1475) — 12 bindings
| Key | Action |
|---|---|
| `j` / `k` | next / previous row |
| `a` | accept (mark reviewed) |
| `s` | skip |
| `e` | focus the Category picker of the selected row |
| `Enter` | mark reviewed + advance (requires all 3; toasts otherwise) |
| `1` `2` `3` `4` | set type = Fixas / Variáveis / Extras / Adicionais |
| `Shift+E` | export `reviewed.json` (works even from a text input) |
| `Esc` (on a select) | blur, so `j`/`k` work again |
| `Enter` (on a select) | all 3 set → reviewed + advance; else focus next empty picker |

**Only `⇧E` is shown in the UI** (a `.kbd` span on the export button). The other 11 are
undocumented on the page — confirming the user's third point.

⚠️ Keys 1–4 hardcode the four type names as string literals; they do not read `TAX.types`.
They agree with `config/taxonomy.json` today (`Fixas`, `Variáveis`, `Extras`, `Adicionais`)
but nothing enforces that — the S2 shape.

### A4. Badges and advisory columns
- `installmentBadge` — ONE row with a `xN` marker, never N rows (UX decided s62).
- `keyword_hint` (col 9, A1/s72) — amber badge, advisory only, never auto-selected,
  deliberately NOT round-tripped through `exportReviewed()`.
- `already_logged` (col 10, T-80/s77) — TWO distinct badges, "already in log" (solid rule)
  vs "partly logged". Advisory, also not round-tripped.
- Row badge states: Pending / Pick a type / Unmatched — needs all 3 / Confirmed /
  Corrected / Skipped / Skipped (incomplete).

### A5. Export (`exportReviewed`, ~line 1557)
`action` derived by `deriveAction`: skipped / pending / confirmed / corrected. A row marked
reviewed but missing any of the 3 levels downgrades to `skipped` with `reviewed: null`.

## B. Predictions — three defects found by reading, to be CONFIRMED in the browser

These are hypotheses from source reading. The browser run either confirms or refutes each;
none is a finding until it has been seen on a rendered page.

### B1. `prefillFor`'s return key never reaches the state — the Type is always empty
`prefillFor` returns `{ sheet: … }` on all four branches (lines 781/788/792/795). All three
consumers read **`pre.type`**, which the object does not have:

- line 803  `STATE` init:      `sheet: pre.type`   → `undefined`
- line 1752 `replaceData` (drag-drop): `sheet: pre.type` → `undefined`
- line 1667 `Reset` button:    `s.type = pre.type` → `undefined`

**Origin:** commit `47a6dff` *"feat(T-05): persist expense Type end-to-end + domain rename"*
(sessions 33–35) changed `sheet: pre.sheet` → `sheet: pre.type` and left `prefillFor`'s
returned key as `sheet`. A half-rename. Broken since T-05 — every close to date
(January s71, February s75) was reviewed with an always-empty Type dropdown.

### B2. The STATE object writes `sheet:` while the whole app reads `s.type`
Even with B1 fixed, line 803 stores the value under key `sheet`, and all ~30 other sites
read `s.type`. Two independent defects on one line.

**Predicted downstream symptoms — all should be visible on the page:**
- Category and Subcategory selects render **disabled** (`!s.type`, lines 1206/1234), so the
  model's correctly pre-filled category is greyed out and unreadable.
- "Accept auto-inserted" pill always reads `0`, button always disabled
  (`bulkAcceptCandidates` requires `s.type`).
- Group by "Predicted type" buckets every row as `(no type)` / `(ambiguous type)`.
- Every row costs at least one extra keystroke — ~227 for the upcoming close.

Confirmed on real data that the input is NOT at fault: `classified.csv` col 8 is populated
(`Variáveis` 27, `Adicionais` 15, `Fixas` 4, `Extras` 1 in the s75 close). The page
receives the type and discards it.

### B3. The localStorage round-trip loses the type — reviewed rows come back "Skipped (incomplete)"
- `scheduleSaveRows` (line 863) writes `{ sheet: s.type, … }`
- `applySavedRows`  (line 891) reads  `let sheet = r.type, …` → `undefined`

So reopening the page mid-review restores category, subcategory and status but **drops
every type**. `deriveAction` then downgrades those rows to `skipped`, and the badge reads
"Skipped (incomplete)". The work is lost, not corrupted — but silently.

## C. Test data

`review` is read-only apart from its `-o` output: it reads the CSV plus
`config/taxonomy.json` and writes only the HTML. Running it with `-o <scratchpad>/…`
touches NO production file — no log append, no workbook, no classification write.

**Use the REAL `config/taxonomy.json`** (read-only) so the picker tree, the ambiguous
pairs and the type names are the genuine ones. Build a SYNTHETIC 10-column
`classified.csv` in the scratchpad.

Contract (`ReadQueue`, exactly 10 fields):
`item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint;already_logged`
(note: subcategory precedes category; `auto_inserted` must be `true`/`false`.)

Real taxonomy facts that make the scenarios constructible — 4 types, 104 (cat,sub) pairs,
**exactly 5 ambiguous pairs**:

| Pair | Hosted by |
|---|---|
| `Transporte / Estacionamento` | Fixas, Variáveis |
| `Saúde / Dentista` | Extras, Variáveis |
| `Pets / Orion`, `Pets / Lilly`, `Pets / Ambos` | Extras, Fixas, Variáveis |

Unambiguous examples: `Habitação/Aluguel`→Fixas, `Alimentação / Limpeza/Supermercado`→
Variáveis, `Saúde/Médico`→Extras, `Lazer/Viagens`→Adicionais.

### Scenario matrix (~20 rows, one CSV)
| # | Targets | Row shape |
|---|---|---|
| 1 | A1 case 1 | type `Variáveis` + `Alimentação / Limpeza`/`Supermercado` — should prefill all 3 |
| 2 | A1 case 1, other type | type `Fixas` + `Habitação`/`Aluguel` |
| 3 | A1 case 2a | type EMPTY + `Saúde`/`Médico` (only Extras hosts it) — should still prefill all 3 |
| 4 | A1 case 2b | type EMPTY + `Transporte`/`Estacionamento` — cat+sub only, "Pick a type", 2 options |
| 5 | A1 case 2b, 3 options | type EMPTY + `Pets`/`Orion` — 3 options |
| 6 | A1 case 2c | `Nada`/`Inexistente` — all empty, "Unmatched — needs all 3" |
| 7 | A1 case 1 w/ INVALID type | type `Extras` + `Habitação`/`Aluguel` (Extras does not host it) — falls through to 2a → Fixas |
| 8–9 | status filter | `auto_inserted=true`, all 3 valid — the only rows the `Auto-inserted` filter and bulk-accept should see |
| 10–13 | confidence filter | confidences 0.30 / 0.55 / 0.80 / 0.99 spread across the slider range |
| 14 | search filter | a uniquely-named item (`ZZQQ marker item`) |
| 15 | search negative | category `Saúde` but item text with no "saude" — proves search is item-only |
| 16 | installments | value `405,25 x4` (T-64 multiplier) → `x4` badge, ONE row |
| 17 | installments | value `189,12/2` (divide) → `x2` badge |
| 18 | keyword_hint | col 9 filled with a competing path → amber badge, nothing selected |
| 19 | already_logged | col 10 = `logged` → "already in log" badge |
| 20 | already_logged | col 10 = `partial` → "partly logged" badge |
| 21 | unreviewable | empty date AND value → dropped from queue, named on stderr |
| 22 | BR thousands | value `1.234,56` (T-72 regression guard) |

## D. Browser procedure (Claude in Chrome)

1. `go build`, render the fixture: `review <scratch>/classified.csv -o <scratch>/review.html`.
   Capture stdout (row count) and stderr (the unreviewable warning, scenario 21).
2. Open the file in a NEW tab. **Clear `localStorage` first** — a stale
   `expense-review:v1:filters` from a prior real close would silently change what loads,
   and that is itself a candidate explanation for "filters not working".
3. **Pre-fill pass (A1/B1/B2):** screenshot the list; for rows 1–7 record what each of the
   three pickers actually shows and whether cat/sub are disabled. This is the single
   decisive observation.
4. **Filter pass (A2):** exercise each segment, both sliders, and the search box; after each,
   count visible rows and compare against the expected set. Check whether the header counts
   move (predicted: they do not — global by design).
5. **Grouping pass:** all four group-by modes, noting the "Predicted type" buckets.
6. **Keyboard pass (A3):** every one of the 12 bindings, including `Esc`/`Enter` on a focused
   select. Verify keys 1–4 write the taxonomy's real type names.
7. **Persistence pass (B3):** set types on 3 rows, mark them reviewed, reload the page,
   and record what comes back.
8. **Export pass (A5):** `Shift+E`, then read the downloaded `reviewed.json` and check
   `action` per row and that `keyword_hint`/`already_logged` are absent by design.

Record every step as observed-vs-expected. A prediction that fails to reproduce is a
finding too — it means the reading was wrong and the real cause is elsewhere.

## E. Out of scope for this pass
Fixing anything. This is an audit: test plan → data → observation → report. The fix
(and whether keyboard shortcuts get a legend, a `?` overlay, or tooltips) is a separate
decision for the user once the report is in.
