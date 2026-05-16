# Implementation Plan — `review` Command

> **Status:** ready to execute in a new session.
> **Prereqs to read first:** `docs/plans/review-ui-design-brief.md` (data contracts),
> `docs/plans/review-ui-fixtures/*.json` (concrete shapes).
> **Workflow:** acceptance test first, then implementation, then unit tests. Pause
> between phases for user approval (CLAUDE.md workflow rule 1). Try local model
> (`my-go-q25c14`) for bounded Go files before escalating.

## 1. Goal

Add `expense-reporter review` — a CLI subcommand that turns a classified-expense CSV
into a ready-to-open `review.html`. It reads the CSV, builds the 3-level taxonomy tree
from the Excel workbook, and bakes both into an HTML template (produced separately by
claude.ai/design). The user opens the HTML, reviews, and exports `reviewed.json`.

```
expense-reporter review classified.csv -o review.html
```

This command is **read-only** with respect to the workbook and feedback logs — it only
reads the workbook and writes one HTML file. Applying `reviewed.json` back to the
workbook is a **separate later command** (`apply`) — out of scope here.

## 2. Inputs & data sources

### 2.1 The classified CSV
Format (header row + `;`-separated, 7 columns):
```
item;date;value;subcategory;category;confidence;auto_inserted
Delivery brod's;01/05;53,98;Delivery;Diversos;0.95;1
```
- `date` — `DD/MM` (no year).
- `value` — Brazilian decimal, comma separator (`53,98`). **May contain installment
  notation** like `250,00/2` — see Open Question O1.
- `confidence` — float `0..1`.
- `auto_inserted` — `1` / `0`.

This is a **new CSV shape** — the existing `parser` package only handles the 4-field
`item;DD/MM;value;subcategory` string. The `review` package needs its own reader.

### 2.2 The taxonomy
Source: the Excel workbook's **"Referência de Categorias"** sheet, already parsed by
`excel.LoadReferenceSheet(workbookPath)` → `map[string][]resolver.SubcategoryMapping`.
Each `SubcategoryMapping` carries `SheetName` (column A "Tipo Principal" — i.e. one of
Fixas/Variáveis/Extras/Adicionais), `Category`, `Subcategory`.

Workbook path: `config.Load()` → `cfg.WorkbookFilePath()`. Add a `--workbook` flag to
override.

### 2.3 The HTML template
Produced by claude.ai/design per `docs/plans/review-ui-design-brief.md`. It contains
the literal placeholder `<script id="review-data" type="application/json">__REVIEW_DATA__</script>`.
**The template does not exist yet** — see Phase 0.

## 3. Package & file layout

```
internal/review/
  types.go        ← ReviewData, QueueEntry, Predicted, Taxonomy, Sheet, Category
  queue.go        ← ReadQueue(csvPath) ([]QueueEntry, error)
  taxonomy.go     ← BuildTaxonomy(mappings) Taxonomy
  render.go       ← Render(template string, data ReviewData) (string, error)
  template/
    review.html   ← go:embed target (Phase 0 stub, real one dropped in later)
cmd/expense-reporter/cmd/
  review.go       ← cobra command wiring
```

`go:embed` the template in `render.go` (or a small `embed.go`):
```go
//go:embed template/review.html
var templateHTML string
```

## 4. Data structures (`types.go`)

Mirror the brief's TypeScript contracts exactly — JSON tags must match:

```go
type ReviewData struct {
    Source      string       `json:"source"`
    GeneratedAt string       `json:"generatedAt"`
    Queue       []QueueEntry `json:"queue"`
    Taxonomy    Taxonomy     `json:"taxonomy"`
}

type QueueEntry struct {
    ID           string    `json:"id"`
    Item         string    `json:"item"`
    Date         string    `json:"date"`
    Value        float64   `json:"value"`
    Confidence   float64   `json:"confidence"`
    AutoInserted bool      `json:"autoInserted"`
    Predicted    Predicted `json:"predicted"`
}

type Predicted struct {
    Sheet       string `json:"sheet,omitempty"`
    Category    string `json:"category"`
    Subcategory string `json:"subcategory"`
}

type Taxonomy struct {
    Sheets []Sheet `json:"sheets"`
}
type Sheet struct {
    Name       string     `json:"name"`
    Categories []Category `json:"categories"`
}
type Category struct {
    Name          string   `json:"name"`
    Subcategories []string `json:"subcategories"`
}
```

## 5. Behaviour spec

### 5.1 `ReadQueue(csvPath)`
- Open CSV, skip the header row.
- Split each line on `;`; expect 7 fields; trim each.
- Parse `value` with `utils.ParseCurrency` (handles `53,98`). See O1 for installments.
- `confidence` → `strconv.ParseFloat`.
- `auto_inserted` → `"1"` true, `"0"` false; anything else is an error.
- `ID` = `feedback.GenerateID(item, date, value)` — reuse the existing hash so the
  later `apply` step can correlate. (Note: `date` is `DD/MM`, no year — acceptable,
  the hash just needs to be stable; document this.)
- `Predicted` = `{Category: category, Subcategory: subcategory}` — `Sheet` left empty
  (today's CSV has no sheet column; the UI's pre-fill rule infers it).
- Skip blank lines; return a clear error with line number on malformed rows.

### 5.2 `BuildTaxonomy(mappings)`
- Iterate every `SubcategoryMapping` value in the map.
- Group: `SheetName` → `Category` → unique `Subcategory` list.
- Deterministic ordering: sort sheets by a fixed preference
  (`Fixas, Variáveis, Extras, Adicionais`, then any others alphabetically), categories
  alphabetically, subcategories alphabetically — so output is stable for tests.
- De-dupe subcategories within a category.

### 5.3 `Render(template, data)`
- `json.Marshal` the `ReviewData` (use `json.Marshal`, not `MarshalIndent` — keep the
  file small; the data is machine-read).
- **Verify** the template contains exactly one `__REVIEW_DATA__` placeholder; error if
  zero or more than one.
- Replace the placeholder with the JSON. Guard: the JSON must not itself contain the
  closing string `</script>` — escape `<` as `<` (Go's `json.Marshal` already
  escapes `<`, `>`, `&` by default — confirm and rely on it).
- Return the final HTML string.

### 5.4 `review.go` command
- `Use: "review <classified.csv>"`, `Args: cobra.ExactArgs(1)`.
- Flags: `-o/--output` (default `review.html`), `--workbook` (override config path).
- Steps: load config → resolve workbook path → `excel.ValidateWorkbook` →
  `LoadReferenceSheet` → `BuildTaxonomy` → `ReadQueue` → assemble `ReviewData`
  (`Source` = base name of the CSV arg, `GeneratedAt` = `time.Now().UTC()` RFC3339) →
  `Render` → write output file (`0o644`).
- Print a one-line summary: `wrote review.html — 42 rows (18 need review)`.

## 6. Edge cases & errors
- Workbook path empty / file missing → clear error naming the config key.
- "Referência de Categorias" sheet missing → surfaced by `LoadReferenceSheet`.
- CSV missing / empty / header-only → error ("no rows to review").
- Malformed CSV row → error with 1-based line number.
- Output path not writable → error.
- Template missing the placeholder → error (catches a broken template drop-in).
- Queue includes **all** rows (auto-inserted and not); the `autoInserted` flag lets the
  UI filter. No `--needs-review-only` flag in v1 (add later if files get large).

## 7. Phasing (each phase pauses for approval)

**Phase 0 — template stub.** Add `internal/review/template/review.html` as a minimal
valid stub: a real `<html>` doc containing only the `__REVIEW_DATA__` placeholder
script tag. Lets the command be built and tested before claude.ai/design delivers the
real template. The real template replaces this file later — no code change needed.

**Phase 1 — acceptance test (first, per `feedback_acceptance_first`).** Add a scenario
under `expense-reporter/test/` (build tag `acceptance`). Fixture: a small classified
CSV + a tiny test workbook (or reuse an existing test workbook fixture — check
`test/` for one). Scenario asserts: running `review` produces an HTML file that
contains a `<script id="review-data">` block whose JSON parses, has the expected queue
length, and a taxonomy with the expected sheet names. Describe previous behaviour
(command does not exist) and new behaviour in the test write-up.

**Phase 2 — `internal/review` package.** Implement `types.go`, `queue.go`,
`taxonomy.go`, `render.go`. Bounded single-file tasks — try `my-go-q25c14` via
ollama-bridge first, record ACCEPTED/IMPROVED/REJECTED verdicts.

**Phase 3 — `cmd/review.go`.** Wire the cobra command; register in `init()`.

**Phase 4 — unit tests (testify).** Per `feedback_testify`: `queue_test.go` (good row,
malformed row, blank lines, installment value per O1, bad `auto_inserted`),
`taxonomy_test.go` (grouping, de-dupe, deterministic ordering, ambiguous category
across sheets), `render_test.go` (placeholder replaced, missing-placeholder error,
`</script>` not present in injected JSON).

**Phase 5 — verify.** `go build ./...`, `go vet ./...`, `go test ./...`, acceptance
suite. Run `review` against the real `/home/leandror/workspaces/expenses/classified.csv`
and confirm the output HTML opens.

## 8. Out of scope (separate future plans)
- The `apply` command that ingests `reviewed.json` into the workbook + feedback logs.
- The HTML template's UI itself (claude.ai/design owns it).
- Any change emitting the full 3-level path into the classified CSV — when that lands,
  `ReadQueue` should populate `Predicted.Sheet`; the `Predicted` struct already has the
  optional field, so it's a one-line change.

## 9. Open questions for the implementing session
- **O1 — installment values.** The CSV `value` column can hold `250,00/2`.
  `utils.ParseCurrency` likely rejects it. Decide: (a) resolve to the per-installment
  number, (b) resolve to the full number, or (c) add a `rawValue string` field to
  `QueueEntry` and carry it through. Check how `batch`/`batch-auto` already handle
  installment notation and stay consistent. Surface to the user before coding.
- **O2 — test workbook fixture.** Confirm whether `expense-reporter/test/` already has
  a workbook fixture with a "Referência de Categorias" sheet; if not, create a minimal
  one for the acceptance test.
- **O3 — `id` and the `apply` step.** `GenerateID` hashes `DD/MM` (no year). Confirm
  the future `apply` command can still correlate; if year matters, the CSV may need a
  full date first.
