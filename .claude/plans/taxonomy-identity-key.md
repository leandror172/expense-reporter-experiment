# Taxonomy Identity Key — Full Path (sheet/category/subcategory)

**Status:** decided & partially implemented (2026-06-16). Validation relaxed +
interim ambiguity guard shipped; full entry-routing redesign **deferred** (task #5).

**Authority:** this document is the decision record for *how a subcategory is
identified* in the taxonomy. It supplements the workbook generator spec
(`.claude/plans/workbook-generator-spec.md`, §1.1) — where they overlap, the spec
defers to this for the identity/routing question.

---

## 1. Problem

The generator's taxonomy was, until now, identified and routed by the **bare
subcategory leaf name** (`subcatMap[entry.Subcategory]` in `internal/taxonomy/loader.go`).
`buildSubcategoryMap` built one global `map[string]subcatTarget` keyed by that bare
name and treated **any** repeated name — even across different sheets/categories —
as a fatal validation error. The fixture taxonomy had unique names, so this was
never exercised against real data.

When the **real** taxonomy was authored from the user's Referência CSV (113 source
rows), the bare-name assumption broke immediately.

## 2. Findings — the real taxonomy repeats leaf names legitimately

7 bare names collide across the real data. They are not data errors; they are the
same *concept* tracked in different expense buckets or as both expense and income:

| Bare name | Full paths it occurs under | Same sheet? |
|---|---|---|
| `Orion` / `Lilly` / `Ambos` | Fixas/Pet, Variáveis/Pets, Extras/Pets (**3×**) | No |
| `Dentista` | Variáveis/Saúde, Extras/Saúde | No |
| `Estacionamento` | Fixas/Transporte, Variáveis/Transporte | No |
| `Gás` | Variáveis/Alimentação-Limpeza, Variáveis/Cuidados pessoais | **Yes** |
| `Produtos` | Variáveis/Cuidados pessoais, Variáveis/Habitação | **Yes** |
| `Aluguel` (income vs expense) | Fixas/Habitação (expense) **and** Receitas/Aluguel (income block) | n/a |

Two distinct sub-problems emerged:

- **Same-sheet collisions** (`Gás`, `Produtos` in Variáveis) are ambiguous even to
  a human reading the workbook — two identically-labelled blocks on one sheet. They
  must be disambiguated *in the data* regardless of routing.
- **Cross-sheet / income-vs-expense collisions** (`Orion`, `Dentista`,
  `Estacionamento`, `Aluguel`) are fine *visually* (each sheet is independent) and
  break only because the routing/validation key was a bare name.

### Why category-qualification is insufficient

A tempting cheaper fix — key on `(category, subcategory)` — does **not** work:
`Pets` appears as a category in **both** Variáveis and Extras, so `Pets/Orion`
still collides across those two sheets. The sheet dimension is load-bearing.
Identity must be the **full three-segment path**.

## 3. Decision

**A subcategory's identity is its full path.**

- Expense: `sheet / category / subcategory`
- Income: `incomeGroup / blockLabel`

Consequences:
- Only an **exact repeated full path** is a validation error (a true duplicate).
- Cross-path repeats of a bare name are **legal**.
- Bare names are no longer a safe routing key on their own (see §5).

## 4. Data fixes applied while authoring `config/taxonomy.json`

The same-sheet collisions were genuine data-modelling issues and were fixed at the
source (these are product decisions by the user, not mechanical dedup):

- **`Gás`** → consolidated to a single `Variáveis/Habitação/Gás`; removed from
  `Alimentação / Limpeza` and `Cuidados pessoais` (gás de cozinha belongs to
  Habitação).
- **`Produtos` (Variáveis/Habitação)** → renamed **`Produtos de casa`** to
  distinguish from `Variáveis/Cuidados pessoais/Produtos`.
- **Category order** → grouped by first appearance in the CSV; subcategories in
  listed order (the CSV interleaves categories, e.g. Educação appears twice in
  Fixas — these are merged into one contiguous block).
- **Income** → one group `Receitas` (8 blocks). `Investimentos` deferred by user.

Result: **112** expense subcategories (113 source rows − 2 `Gás` + 1 consolidated),
zero true full-path duplicates, with the expected legal bare-name repeats remaining
(`Orion`×3, `Lilly`×3, `Ambos`×3, `Dentista`×2, `Estacionamento`×2, plus income
`Aluguel` vs expense `Aluguel`).

The real file is **gitignored** (`.gitignore`: `expense-reporter/config/taxonomy.json`)
because it reveals personal categories; the committed fixture
(`test/fixtures/generate-basic/taxonomy.json`) stays the test input.

## 5. Implemented now — validation relaxation + interim ambiguity guard

`buildSubcategoryMap` (`internal/taxonomy/loader.go`) was reworked:

- Returns `(map[string]subcatTarget, map[string]bool /* ambiguous */, error)`.
- A `seen map[string]bool` of **full paths** detects true duplicates; error only on
  an exact full-path repeat.
- Routing is still keyed by bare name, but **a bare name resolving to >1 full path
  is "ambiguous" and is NOT routable**. `registerTarget` enforces this with three
  branches: already-ambiguous → skip; already-present (a second full path) → delete
  from map + mark ambiguous; else → add.
- `scanEntries` accepts the ambiguous set and emits a **distinct** warning for an
  ambiguous name ("ambiguous across multiple blocks (pending full-path routing)")
  versus a genuinely unknown one. Both skip the entry and exit 0.

### Two traps that bit us (record them)

1. **The 3× re-add trap.** `Orion`/`Lilly`/`Ambos` each appear *three* times. A
   naive "delete from map on collision" silently **re-adds** on the third pass
   (absent → add → present → delete → absent → add again) — re-introducing the
   silent misroute. The fix is an explicit `ambiguous` set that is *sticky*: once a
   name is ambiguous it is never re-added. Covered by
   `TestLoadTaxonomy_AmbiguousEntrySkipped` (a 3× repeat + a matching entry,
   asserting the entry routes to **none** of the three blocks).
2. **The `/` separator trap.** Real subcategory names contain `/`
   (`Uber/Taxi`, `Óleo/flor cannabis`, `Alimentação / Limpeza`,
   `Cinema/teatro`). Joining path segments with `/` can produce spurious
   collisions. Full paths are joined with a **null byte (`\x00`)**, which cannot
   occur in human-typed names, plus a `expense`/`income` kind prefix so a
   3-segment expense path can never equal a 2-segment income path.

### Coverage / contract (tests in `loader_test.go`)

- `TestLoadTaxonomy_SamePathDuplicate` — exact full-path repeat → error (surviving
  half of the old guard).
- `TestLoadTaxonomy_CrossPathDuplicateAllowed` — same bare name in two different
  sheets/categories → loads, no error.
- `TestLoadTaxonomy_AmbiguousEntrySkipped` — 3× repeat + matching entry → loads,
  entry routed to none (guards the re-add trap).

Note: the skeleton path (`LoadTaxonomy(..., "")`) only hits `buildSubcategoryMap`
for validation and **discards the map**, so the ambiguity/routing logic is *not*
exercised by skeleton generation — its real coverage is the entry-fed unit test.

### Verification done

`gofmt` clean · `go build/vet ./...` OK · full unit suite green ·
`generate-workbook --taxonomy config/taxonomy.json` produced a 74 KB xlsx with no
stderr, 6 sheets in correct order, Receitas blocks present — i.e. all 112 subs +
income validated with zero false-duplicate errors.

### Local-model note

The `loader.go` change was first delegated to `my-go-qcoder` (per codegen policy);
**verdict 0 (rejected)** — it produced an O(n²) nonsense ambiguity scan, truncated
output, and used `/` as the separator. Rewritten from scratch (sanctioned for
conceptual defects). The null-byte separator and the sticky-ambiguous
`registerTarget` are exactly the parts the model got wrong.

## 6. Deferred — full-path entry routing redesign (task #5)

What is **not** done: routing logged entries by full path. Today an entry in
`expenses_log.jsonl` carries only a bare `subcategory` string (no sheet, no
category — see `scanEntries`'s entry struct). So an entry tagged `Orion` is
genuinely unroutable and is correctly skipped. To route it, the redesign must touch:

- **Entry contract** — `expenses_log.jsonl` lines (`item/date/value/subcategory`)
  must also carry category + sheet (or a composed full-path key).
- **The classifier** that produces those entries (it knows the path from the
  taxonomy, so it can emit it).
- **`scanEntries` / `buildSubcategoryMap`** — route on the full-path key instead of
  bare name; the `ambiguous` set then disappears (every entry is unambiguous).
- **Existing entry-fed fixtures/tests** — updated to the new entry shape.

This is its own change, advisor-reviewed, because the entry contract ripples into
the classification pipeline. Until it lands, the interim guard above keeps the
behavior **safe** (warn + skip, never silent misroute), so entry-fed generation can
run today — ambiguous entries simply don't populate.

<!-- ref:taxonomy-identity-key -->
**Taxonomy identity = full path.** Expense identity is `sheet/category/subcategory`;
income identity is `incomeGroup/blockLabel`. Only an exact repeated full path is a
validation error; cross-path repeats of a bare leaf name are legal (real data
repeats names, e.g. `Orion` across three Pet blocks, `Aluguel` as expense + income).
Bare names are NOT a safe routing key: a bare name shared by >1 full path is
*ambiguous* and is dropped from routing (warn + skip entry, exit 0) until the
full-path entry-routing redesign lands. Full-path keys are joined with a null byte
(names contain `/`) and a `expense`/`income` kind prefix. The ambiguous set must be
sticky — a name appearing 3+ times must not re-add itself after removal.
<!-- /ref:taxonomy-identity-key -->
