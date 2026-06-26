# T-13 (revised) — Classifier predicts the full `(type, category, subcategory)` path

**Status:** planned, not started
**Supersedes:** the original T-13 framing ("single-source category/type resolution from taxonomy.json")
**Blocks:** WS-D (retire bare-name fallback) — this is the prerequisite that drives the
type-less line count toward ~0
**Branch context:** continues the "retire insertion, keep generation" pivot
(`.claude/plans/retire-insertion-keep-generation.md`)

---

## 1. Problem (why the original framing was incomplete)

Today an `expenses_log.jsonl` line needs three taxonomy fields — subcategory, category,
type — but they are resolved from **two independent files**:

| Field | Source | Code |
|-------|--------|------|
| category | `data/classification/feature_dictionary_enhanced.json` → `category_mapping` | `add.go:68` `resolveCategoryFromTaxonomy` → `classifier.LoadTaxonomy` (`classifier.go:34`) |
| type | `config/taxonomy.json` (3-level tree) | `auto.go:198` `resolveExpenseType` → `taxdb.LookupType` (`lookup.go:61`) |

`LookupType` is keyed on `(category, subcategory)`, but the `category` fed in comes from the
feature dict — so the two files must agree on category *spelling*. They don't, and worse, the
model never sees types at all. Result: silent `type: ""` lines that keep the bare-name fallback
alive and block WS-D.

The original T-13 ("make taxonomy.json the single source for resolution") treats the symptom.
The cause is that **the classifier's output is incomplete** — it predicts only
`(subcategory, category)`, never the type, and is shown only a 2-level `sub→cat` list built
from the wrong file (`classifier.go:235-240`).

## 2. Decision

**The classifier predicts the complete path `(type, category, subcategory)`, against
`taxonomy.json` as the single taxonomy it is shown.** No post-hoc category/type resolution.
The persisted log line carries exactly what the model chose, valid-by-construction.

## 3. Empirical feasibility (verified 2026-06-26, session 40)

Leaf-set comparison across `taxonomy.json` (104 leaves, 4 types: Fixas/Variáveis/Extras/Adicionais),
the feature dict (68 subs), and training data (81 distinct subs):

- **Subcategory coverage is complete.** The only apparent gap, `IRRF`, is a **typo collision**:
  taxonomy.json has `IRFF` (transposed) under `Fixas → Impostos`; data uses `IRRF`. One-character
  reconciliation, not a new node.
- **The 4 "category-drift" pairs are the root cause and they vanish under this design.** The
  feature dict's category strings (`Fixas – Impostos`, `Fixas – Saúde`, `Alimentação / Limpeza`,
  `Habitação`/`Produtos`) are type+category mashups. Once the model is fed taxonomy.json directly,
  these strings are never used as authority again.
- **5 subcategories are unresolvable post-hoc — only prediction-time choice works:**
  `Estacionamento` (Fixas|Variáveis/Transporte), `Dentista` (Extras|Variáveis/Saúde),
  `Ambos`/`Lilly`/`Orion` (Extras|Fixas|Variáveis/Pets). `LookupType` returns `ErrTypeAmbiguous`
  for these → `type: ""`. The answer genuinely depends on the expense, not the taxonomy. This is
  the empirical proof the full-path design is **necessary**, not just cleaner.
- **Few-shot type labels are tractable.** Training data's `source` field ends in the origin
  sheet-name, which *is* the type for ~96% of examples (Variáveis 774, Adicionais 630, Fixas 251,
  Extras 59). Only the 74 `user_corrections` lack a sheet; derive their type from taxonomy.json
  where unambiguous, else leave type-unlabeled in few-shot.

## 4. Work breakdown

### 4.1 Taxonomy hygiene (prerequisite, tiny) — ✅ DONE 2026-06-26
- Reconciled `IRFF` → `IRRF` at `taxonomy.json:72` (Fixas/Impostos leaf). Verified: subcategory
  coverage now 100% — no feature-dict or training-data sub is missing from the tree.

### 4.2 Feed the 3-level tree to the prompt
- Replace the `sub→cat` list (`classifier.go:235-240`, `entry{sub,cat}` at `:221-224`) with a
  rendering of taxonomy.json's `type → category → subcategory` tree.
- **DECIDED — Option 2B: classifier depends on `internal/taxonomy`.** No cycle risk verified
  2026-06-26: `internal/taxonomy` imports no internal packages (leaf), `classifier` imports only
  `internal/logger`, neither imports the other — so the dependency is safe and is the correct
  downward direction (lower layer → pure data package). Single parser, single source of truth;
  2A (widen `classifier.Taxonomy`) was rejected because it resurrects the two-parsers-for-one-file
  drift that caused T-13.
  - `classifier` calls `taxonomy.LoadTaxonomy(...)` → `[]ExpenseType`, used to render the prompt
    tree AND build the path enum.
  - Add helpers in `internal/taxonomy`: `PathEnum([]ExpenseType) []string` and
    `SplitPath(s) (type, cat, sub)` so the classifier never hand-parses `"A/B/C"`.

### 4.3 Extend the output contract
- Add `Type` to `classifier.Result` (`classifier.go:15`).
- Add `type` to the structured-output JSON schema (`classifier.go:136-140`) and the
  required-fields list.
- Constrain to valid paths: **DECIDED — Option A, atomic `path` enum.** Collapse the triple into
  one `path` field whose `enum` is the 112 `Type/Category/Subcategory` strings generated from
  taxonomy.json at request time; split back to the 3 fields in Go before writing the entry.
  Validity is guaranteed by construction (`type: ""` structurally impossible) and ambiguous
  leaves resolve at prediction time because each appears as a distinct path.
  **Smoke-test (2026-06-26, `my-classifier-qcoder`, scratchpad/enum_smoketest.py):** 112-path
  enum honored 100% (zero invalid paths), ~6s/call flat (no large-enum slowdown), and the
  context split worked — `Estacionamento shopping` R$18 → `Variáveis/Transporte/Estacionamento`
  0.85. Validate-and-retry (Option B) dropped — not needed. Remaining disambiguation misses
  (`Dentista limpeza`→Consultas, `Estacionamento mensal`→Condomínio) are few-shot quality
  issues (§4.4), not enum-feasibility issues.

### 4.4 Enrich few-shot examples with type
- At example load (`examples.go` / `loader.go`), attach a type to each training example from its
  `source` sheet-name (4-way map). For `user_corrections`, derive via taxonomy where unambiguous.
- Render the type in the injected example lines (currently `sub → cat` at `classifier.go:181`).

### 4.5 Consume the predicted type in the command layer
- `auto.go`: write `entry.Type` from `Result.Type` directly; **delete** `resolveExpenseType`
  + `loadTypeIndex` (`auto.go:198-220`).
- `add.go` (manual, no model): resolve the full path by walking **taxonomy.json**, not the
  feature dict. **DECIDED — Option 4B+4A hybrid for the 5 ambiguous leaves:**
  - Accept a `--type` flag; when the leaf is ambiguous and the flag is present, use it.
  - Flag absent + ambiguous leaf + interactive stdin (TTY) → prompt the user to pick a type.
  - Flag absent + ambiguous leaf + non-interactive → hard error (keeps unattended runs
    deterministic; never silently guesses).
  - Unambiguous leaf (99/104) → resolve uniquely, ignore `--type`.
  - 4C (first-match + warn) rejected: a silently-wrong type routes to the wrong sheet and is
    worse than `type: ""` because WS-D would trust it.
- Delete `resolveCategoryFromTaxonomy` (`add.go:176`) once `add` resolves via taxonomy.

### 4.6 Downstream / tests
- `classified.csv` already carries a Sheet/type column (RUI-4); confirm it's now populated from
  the prediction rather than left blank.
- Acceptance: the existing type-routing-cycle fixture should now show **zero** type-less lines
  reaching generate-workbook. Add an assertion on the stderr type-less fallback count == 0 for
  the model path.
- Unit: classifier tests for the new schema + path validation; ambiguous-leaf cases
  (`Estacionamento`, `Dentista`) get explicit prediction tests.

## 5. Decisions — ALL RESOLVED (2026-06-26, session 40)
1. ✅ `IRFF` → `IRRF`: fixed at `taxonomy.json:72`; coverage now 100%.
2. ✅ Option 2B — classifier depends on `internal/taxonomy` (no cycle; single parser).
3. ✅ Option A — atomic 112-path enum (smoke-tested, 100% valid, ~6s/call).
4. ✅ Option 4B+4A hybrid — `--type` flag + interactive prompt fallback + non-interactive error.

## 6. Sequencing vs WS-D
T-13 lands first → drives model-path type-less count to ~0 → WS-D retires the bare-name
fallback once the historical/manual tail is also typed. The `add` manual path (4.5) is the last
type-less source; close it before WS-D deletes the net.
