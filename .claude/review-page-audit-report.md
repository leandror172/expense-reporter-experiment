# Review page audit — findings (session 77)

**Method:** synthetic 22-row fixture through the real `review` command against the REAL
`config/taxonomy.json`, rendered and driven in Chrome. Test plan and scenario matrix:
`.claude/plans/review-page-behavior-audit.md`. Nothing production was read or written —
isolated scratch install root (binary + own `config/`), page served over
`http://127.0.0.1` because Chrome automation cannot load `file://`.

**Verdict on the three questions asked:**

| Question | Answer |
|---|---|
| Are the filters working? | **Yes — every filter is correct.** The impression has three other causes (F1–F3). |
| Is the empty Type dropdown expected? | **No. It is a bug — three of them.** Broken since session 33. |
| Are keyboard shortcuts exhibited? | **No.** 12 bindings exist, all work, 1 is documented. |

---

## R1 — CRITICAL: the Type is computed correctly and then thrown away (3 defects, one fault line)

`prefillFor()` returns its result under the key **`sheet`** on all four branches. Every
consumer reads **`pre.type`** — a key the object does not have.

Observed on a virgin load (`u()` marks a value JS would omit from JSON):

```
item                     | csvType   | pre.sheet | pre.type      | s.type        | badge
S01 case1 supermercado   | Variáveis | Variáveis | <<undefined>> | <<undefined>> | Pending
S02 case1 aluguel        | Fixas     | Fixas     | <<undefined>> | <<undefined>> | Pending
S03 case2a medico        | (none)    | Extras    | <<undefined>> | <<undefined>> | Pending
S07 case1 tipo invalido  | Extras    | Fixas     | <<undefined>> | <<undefined>> | Pending
```

`pre.sheet` holds the **right answer every time** — including S03, where the type was
absent from the CSV and the taxonomy search correctly resolved it, and S07, where an
invalid type correctly fell through to `Fixas`. The resolution logic is sound. Only the
handoff is broken.

**R1a — the prefill key never reaches the state.** `prefillFor` returns `sheet:`;
lines 803 (`STATE` init), 1752 (`replaceData`) and 1667 (`Reset`) all read `pre.type`.

**R1b — the state object writes a dead key.** Line 803 stores under `sheet:`, while ~30
other sites read `s.type`. Two independent defects on one line; fixing either alone
changes nothing.

**R1c — the localStorage round-trip loses the type.** `scheduleSaveRows` writes
`{sheet: s.type}`; `applySavedRows` reads `r.type`. Measured payload:
`{"sheet":"Variáveis","category":"Alimentação / Limpeza",...}`.

**Origin — a half-rename, pinned by `git log -L 803,803`:**
```
47a6dff feat(T-05): persist expense Type end-to-end + domain rename + backfill tool
-    sheet: pre.sheet,
+    sheet: pre.type,
```
Session 33. Live through every close since, including January (s71) and February (s75).

**The input is not at fault.** `predicted.type` is present in the page's embedded data and
in the export. Real s75 `classified.csv` col 8: `Variáveis` x27, `Adicionais` x15,
`Fixas` x4, `Extras` x1.

### Why it survived two closes: it is invisible exactly where it does damage
`prefillFor` returns `needsSheet`/`noMatch` under their **correct** keys, so those flags
survive. A row that should have been pre-filled therefore has `initialAmbiguousSheet:
false` and no type, and `rowBadge` falls through to **"Pending"** — not "Pick a type".
Only genuinely ambiguous rows are flagged. The page gives no signal that anything is wrong.

### Measured consequences
- **Category and Subcategory render `disabled`** (`!s.type`), greyed out, holding the
  model's correct answer where the reviewer cannot use or trust it. Setting `s.type` by
  hand re-enables both instantly with the right values already in place.
- **"Accept auto-inserted" is permanently dead** — pill reads `0`, button `disabled`,
  with 2 fully-valid auto-inserted rows present. `bulkAcceptCandidates()` requires `s.type`.
- **Group by "Predicted type"** buckets everything: `(no type):17`, `(ambiguous type):2`.
- **Every row costs an extra keystroke.** ~227 rows are queued for 2026-03..08.
- All three test rows, given their correct type, immediately derived **"Confirmed"** — the
  model had been right, and the reviewer was being made to re-enter a correct answer.

## R2 — CRITICAL: a reload silently downgrades finished work, and the header misreports it

Marked S01–S03 reviewed (all "Confirmed"), reloaded:

```
BEFORE:  S01 type=Variáveis  status=reviewed  action=confirmed  badge="Confirmed"
AFTER:   S01 type=<<undefined>> status=reviewed action=skipped  badge="Skipped (incomplete)"
header:  21/3/0/18  ->  21/3/0/18   (unchanged: still claims 3 reviewed)
toast:   "Restored 21 in-progress rows from a previous session."
export:  {"skipped":3,"pending":18}   confirmed=0  corrected=0
```

The header claims **3 reviewed** while the export emits `"reviewed": null` for all three.
A reviewer who trusts the counter loses those expenses silently — `apply` ignores
`skipped`, so they never reach the log. The reassuring toast makes it worse.

**Severity bounded:** `apply` ignores `pending` and `skipped`, so this loses work, it does
not corrupt the log.

**Second-order (from R1c):** in `applySavedRows` the restored `sheet` is always falsy, so
all three `if (sheet && ...)` validation guards are **dead code**. Those guards exist to
drop a saved pick the taxonomy no longer hosts — the S2 drift scenario. A stale
category/subcategory is currently restored unvalidated.

## R3 — Filters: no defect found. Three other causes for the impression

Every filter was exercised through the real DOM handlers and is **correct**:

| Filter | Result |
|---|---|
| Needs review / Auto-inserted / All | 19 / 2 / 21 — correct |
| Search (case-insensitive substring) | correct |
| Confidence band 95–100 / 0–50 | 4 / 1 — correct, inclusive both ends |
| Combined filter + search | correct |
| Grouping (all 4 modes) | correct |
| Saved-filter restore | correct — syncs the UI controls too |

`renderList()` correctly calls `visibleStates()` and passes the result to `groupedView()`.

**F1 — the header counts are global, never filtered.** `21 TOTAL` with 19 rows on screen.
By design, but it reads as "the filter did nothing".
**F2 — search matches the item field ONLY.** Searching `Saúde` returned **0 rows** while
3 rows carry that category. This is the most likely source of the complaint.
**F3 — "Accept auto-inserted 0", permanently disabled, sits in the filter toolbar** and is
collateral damage from R1.

*Method note:* an apparent confidence bug (`99–100` → 0 rows) was **my test artifact** —
`step=5` snapped `99` to `100`. The page is correct. Worth one line: the slider cannot
express 99%, so a 0.99 row is only reachable via a band ending at 100.

## R4 — Keyboard: all 12 bindings work; the legend existed but was unreachable

⚠️ **CORRECTION to this report's first draft, which claimed only `⇧E` was documented.**
A full `.shortcuts` legend was already in the markup — sitting in flow at the **bottom of
the list**, after every row. With 21 fixture rows it is merely far away; with the ~227 rows
queued for 2026-03..08 it is unreachable in practice, which is why it had never been seen.
The defect was DISCOVERABILITY, not absence. Stated as "undocumented" it would have been
fixed by adding a second copy of the legend — the wrong repair.

Verified by dispatch and by physical keypress (clicked S19, pressed `j`, moved to S20):

| Key | Verified |
|---|---|
| `j` / `k` | navigate |
| `1` `2` `3` `4` | Fixas / Variáveis / Extras / Adicionais — match the taxonomy exactly |
| `a` / `s` | accept / skip, with a correct refusal when the row is incomplete |
| `e` | focuses the Category select |
| `Enter` | reviewed+advance; toasts "Can't mark reviewed — need all 3 selected" |
| `Shift+E` | export |

**Only `⇧E` appears in the UI.** The other 11 are undiscoverable.
⚠️ Keys 1–4 hardcode the four type names as string literals rather than reading
`TAX.types`. Correct today, unenforced — the S2 shape.

## R5 — Everything shipped in s72/s77 is intact
`×4` (T-64 multiplier) and `×2`/`×3` (divide) installment badges; the amber `kw-hint`
badge (A1); the two distinct `already in log` / `partly logged` badges (T-80); BR thousands
`R$ 1.234,56` (T-72); and the unreviewable row correctly dropped and named on stderr (S1).
`installments: 4` survives the export hop. **The regressions are confined to T-05-era type
plumbing.**

## Minor
- The first picker's placeholder still reads **"Sheet…"** — pre-T-05 vocabulary in
  user-facing text (T-36 family). Internal identifiers `sheetSel`/`sheetWrap`/
  `initialAmbiguousSheet`/`ambiguousSheetOptions` likewise.
- The `already in log` badge visually overlaps the "REVIEW" label (see screenshot).
- `STORAGE_FILTERS_KEY` is global, not keyed by source, so filters carry across
  different closes. Controls do sync, so it is discoverable — but it is a real cross-close
  leak.

## What this says about the gap T-61 already filed
All three R1/R2 defects live on the Go→JS seam, where no test reaches. T-61 filed that gap
for one field (`installments` in `exportReviewed()`). The same unguarded seam has now cost
**three more**, one of which silently discards the reviewer's finished work. Whether these
become new tasks or ride T-61 is the user's call.


---

# Resolution (session 77, same session as the audit)

All findings that were defects are fixed on branch `fix/review-page-type-prefill`.
Verified in Chrome on the same fixture, before and after, so each claim below is a measured
change of state rather than a reading of the diff.

| Finding | Fix | Verified |
|---|---|---|
| R1a/R1b — Type never pre-filled | `prefillFor` now returns `type:` on all four branches; the three consumers store into `s.type` | S01 `Variáveis`, S02 `Fixas`, S03 `Extras` (case 2a), S07 `Fixas` (invalid type → search). S04/S05 still correctly say "Pick a type"; S06 still "Unmatched" |
| R1b knock-ons | — | Category/Subcategory now **enabled**; bulk pill `0 → 2` and no longer disabled; group-by-type now `Fixas:3 Variáveis:10 Extras:3 (ambiguous):2 (no type):1` |
| R1c/R2 — reload discarded finished work | `scheduleSaveRows` persists `type:`; `applySavedRows` local renamed to match | Reload keeps all three rows "Confirmed"; export `{"confirmed":3,"pending":18}`; header and export now agree |
| R2 second-order — dead validation guards | The three `if (type && ...)` checks are live again now that the key resolves | Documented in the function; they drop a pick the taxonomy no longer hosts (the S2 shape) |
| R4 — legend unreachable | Same markup, now a panel toggled by a toolbar `?` button and the `?` key; `Escape` / outside-click close | One copy in the document; `aria-expanded` tracks; `?` inert while typing in search; **space does not toggle** |
| Minor — stale T-05 wording | Picker placeholder `"Sheet…"` → `"Type…"`; legend "set sheet" → "set type" | No user-facing `Sheet` strings remain |
| Minor — panel overlapped Export | `setLegend` measures the header at open time instead of a constant offset | Zero overlap with any header control |

**New guard:** `TestTemplateUsesTypeVocabularyNotSheet` fails on a bare `sheet` used as a
data key, with a sub-test proving it discriminates in both directions. Mutation-verified —
restoring `sheet: pre.type` turns it red; reverting turns it green; the mutation stays
compilable, so it measures something (s72's rule).

**Deliberately NOT changed — these were findings, not defects:**
- **F1** header counts stay global. Changing them to follow the filter is a UX decision, not
  a bug fix.
- **F2** search still matches the item field only. Widening it to category/subcategory is a
  behaviour change worth making deliberately; it is the most likely cause of the original
  "filters aren't working" impression and is the best candidate for a follow-up.
- **F3** dissolved on its own — the bulk button works now.
- The confidence slider still cannot express 99% (`step=5`).
- `STORAGE_FILTERS_KEY` is still global rather than keyed by source, so filters leak across
  closes.
- Internal identifiers (`sheetSel`, `sheetWrap`, `initialAmbiguousSheet`,
  `ambiguousSheetOptions`, `setSheetOnSelected`) still use the old term. The new guard is
  anchored so these keep passing; renaming them is cosmetic and would bloat this diff.

**Still open:** the behavioural gap is unchanged. Nothing automated exercises this page —
the guard is a vocabulary check that can only catch this one fault line reopening. Closing
it properly is T-61 and needs a browser toolchain the repo does not have. The fixture and
procedure at `.claude/fixtures/review-page-audit/` are the interim answer.
