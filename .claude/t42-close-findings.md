# T-42 close findings — the two that need decisions (session 71, 2026-08-18)

Measured during and after the **first real monthly close** on 2026 data
(`expenses-2026-01.csv`, 69 rows → 84 log rows dated 2026). The close itself succeeded;
these are the two findings that are not papercuts. Two further findings from the same run
(`finish`'s self-copy abort, and a `~/Downloads` path that does not exist on this machine)
are one-line fixes recorded directly in `tasks.md`.

Source data: `.claude/scratch/close-20260818-120248/` (gitignored — real expense data).

---

<!-- ref:auto-insert-unrepairable -->
## Finding 1 — an auto-inserted row cannot be corrected; the budget stays wrong

**What happened.** `4x álcool 70` (R$42,00, 03/01/2026) passed the agreement gate and was
auto-appended by `batch-auto` as `Combustível / Transporte`. During review it was corrected
to `Supermercado / Alimentação-Limpeza`. Afterwards:

| Artifact | Value | |
|---|---|---|
| `classifications.jsonl` | `status: corrected`, Combustível → **Supermercado** | ✅ |
| `expenses_log.jsonl` | `Combustível / Transporte` | ❌ |
| `workbook-2026.xlsx` | rubbing alcohol filed as **fuel** | ❌ |

`apply` reported it honestly — `⚠ 1 already-applied rows were corrected — feedback logged,
no expense-log change` — so this is loud, not silent. But **no command can repair it**:

- `cmd/correct.go:88` calls `feedback.Append` only. It never touches the expense log.
- `apply.go:176` treats confirmed+already-applied as a no-op, and routes corrected+found
  into `corrections`, which is feedback-only.
- Both logs are append-only by design, and nothing supersedes a prior row.

The only repair available today is hand-editing `expenses_log.jsonl`.

**Why this is structural, not a papercut.** `batch-auto` auto-appended **17 of 69 rows (25%)**
in this close without any human seeing them. T-32's ~95% precision is a *same-sample relative*
validation on a confidence-selected subset — **absolute production precision is unmeasured**,
which is exactly why WS-D (unattended silent auto-insert) is held. So the system writes
unreviewable rows at an unknown error rate into a store with no correction path. The
classifier learns from the correction; the budget never does.

Note the asymmetry that makes this bite: the *review* route is fully correctable (that is what
review is), so the defect is confined to precisely the rows nobody looked at.

**Options weighed:**

- **(a) Supersede-by-append.** A tombstone/replacement record that `generate-workbook` honours
  when folding the log. Preserves append-only and the `GenerateID` join. Costs a schema
  addition and a rule in `scanEntries`.
- **(b) Rewrite in place.** Smallest change at the call site, but breaks the append-only
  property the join-id design and the close-cycle snapshot/undo both lean on.
- **(c) Gate-to-review (WS-D).** Stop auto-inserting; every row is seen. Makes the case
  impossible rather than repairable.

**Recommendation:** (c) is already the held WS-D direction and this is the first *real-data*
argument for it. Do (a) regardless — even at a perfect gate, a human changes their mind about
a category eventually, and today that costs hand-editing a JSONL file.
<!-- /ref:auto-insert-unrepairable -->

---

<!-- ref:close-accuracy-2026-01 -->
## Finding 2 — 38% of reviewed rows needed correcting, and the keyword layer already had the answer 60% of the time

**Headline numbers** (65 reviewed entries: 40 confirmed, 25 corrected, 0 skipped):

| Measure | Value |
|---|---|
| Correction rate | **25/65 = 38%** |
| Corrections that were **cross-category** (wrong branch, not wrong leaf) | **23/25 = 92%** |
| Confidence on wrong answers | median **0.95**, 20/25 ≥ 0.90, min 0.80 |
| Corrections where the **keyword layer already held the reviewer's answer** | **15/25 = 60%** |
| …of those, at maximum specificity (1.00, unambiguous) | **9** |

**The model overrode a correct, maximally-specific keyword.** The clearest case is the pet
leaves. `Lilly` and `Orion` are subcategories in `config/taxonomy.json`
(`types[].categories[].subcategories[]`), have 10 and 14 labelled training examples, and the
keyword dictionary maps `lilly → Lilly` and `orion → Orion` at **specificity 1.00,
unambiguous**. Yet:

```
Consulta Elizabeth cardiologista Lilly   model=Dentista   keyword=Lilly   spec=1.00
Exame pressão Lilly                      model=Dentista   keyword=Lilly   spec=1.00
Exame ecocardio Lilly                    model=Dentista   keyword=Lilly   spec=1.00
Exame eletro Lilly                       model=Exames     keyword=Lilly   spec=1.00
Exames Lilly sinplan                     model=Exames     keyword=Lilly   spec=1.00
Consulta Orion nefro Território animal   model=Dentista   keyword=Orion   spec=1.00
Consulta Lilly nefro Território animal   model=Dentista   keyword=Lilly   spec=1.00
Consulta Lilly nefro Kelly               model=Dentista   keyword=Lilly   spec=1.00
Veterinário Bruno cannábico Orion        model=Óleo/flor  keyword=Orion   spec=1.00
```

Seven consultations routed to **Dentista** — a human dental leaf — for a *pet*. The words the
model latched onto are not even in the dictionary: `exame`, `exames`, `veterinário` and `nefro`
are all **absent**, and `consulta` maps to `Consultas` at only 0.67 specificity.

**The agreement gate behaved correctly throughout.** Model ≠ keyword means the gate refused to
auto-insert, and all of these went to review. That is the gate working as designed. The defect
is one layer up: **the suggestion shown to the human is the model's, even when an unambiguous
maximum-specificity keyword disagrees.**

**Consequences for the existing backlog:**

- **T-27 (per-leaf descriptions) is aimed at the wrong failure.** It targets "right sheet,
  wrong leaf". Measured here, **92% of errors are cross-category** — the wrong branch entirely.
  **T-28** (two-stage: classify type/category first, then the leaf within it) matches this
  data far better. Re-justify T-27 against these rows or drop it.
- **T-32 is confirmed on real data.** Median confidence on *wrong* answers is 0.95. Confidence
  is dead as a signal, independently reproduced outside the 649-replay.
- **5.R6 (regenerate the keyword dictionary) has a concrete payoff.** `exame`/`exames`/
  `veterinário`/`nefro` are missing from a 694-era dictionary that was never rebuilt after
  5.R4 grew the corpus to 1,788.
- **Cheapest candidate intervention:** when a keyword match is unambiguous at specificity 1.00
  and the model disagrees, show (or prefer) the keyword's answer in the review queue. On this
  month that is 9 rows, and 15 by the looser "keyword had the reviewer's answer" measure.

**Method and caveats — read before quoting these numbers.**

- Computed from `reviewed.json` (`predicted` vs `reviewed` objects; note the reviewer's choice
  is NOT at the top level — a top-level `.subcategory` reads `null`) joined to `classified.csv`.
- **The 60% is measured on the corrected subset only, which is selected for model error by
  construction.** It does NOT say keywords beat the model overall — it says that *among rows
  the model got wrong*, the keyword layer already held the right answer 60% of the time. That
  is a recoverable-error measure, not a head-to-head accuracy claim.
- Keyword matching here re-implements `classifier/examples.go`'s tokenisation approximately
  (lowercase, split on non-alphanumerics, accents kept, length ≥ 2). Treat the counts as
  ±1–2, not exact. A production replay would settle it.
- n = 65 rows, one month, one reviewer. Directional, not a benchmark.
<!-- /ref:close-accuracy-2026-01 -->
