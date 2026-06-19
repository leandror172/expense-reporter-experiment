# Plan — RUI-4: Emit the expense *type* into the classified/review CSV

**Owner:** Sonnet subagent (worktree-isolated).
**Branch base:** `feat/type-routing-improvements` (already contains the shared
`internal/taxonomy/lookup.go` reverse-lookup primitive — DO NOT reimplement it).

## Goal

The classifier produces `(category, subcategory)`. Today the `review` UI must guess
which workbook **sheet** each row belongs to. Add the expense **type** as a column in
the batch-auto output CSVs so the `review` UI can pre-fill the destination
unambiguously (the `review.Predicted.Sheet` field already exists but is never set).

## Domain-translation rule (HARD)

- The concept is **`type`** wherever it is a domain attribute of an expense — this
  includes the new CSV column header (`type`).
- The word **`sheet`** stays reserved for code that directly selects/manipulates a
  workbook sheet. The `review` package builds its taxonomy from workbook mappings and
  uses `Predicted.Sheet` to pre-select a workbook sheet — that field is workbook-facing,
  so it KEEPS the `Sheet` name. The translation `type → Predicted.Sheet` happens exactly
  at `review.ReadQueue` (the boundary). Do not rename `Predicted.Sheet`.

## Required reading before any work (ABSOLUTE paths)

1. `/mnt/i/workspaces/expenses/code/.claude/overlays/local-model-conventions.md`
2. `/mnt/i/workspaces/expenses/code/.memories/QUICK.md`
3. `/mnt/i/workspaces/expenses/code/.memories/KNOWLEDGE.md`
4. `/mnt/i/workspaces/expenses/code/expense-reporter/test/PATTERNS.md`
5. `/mnt/i/workspaces/expenses/code/.claude/index.md`
6. The QUICK/KNOWLEDGE in any folder you edit, BEFORE editing it — at minimum
   `/mnt/i/workspaces/expenses/code/expense-reporter/internal/review/.memories/QUICK.md`
7. The shared primitive: `/mnt/i/workspaces/expenses/code/expense-reporter/internal/taxonomy/lookup.go`
   (`BuildTypeIndex`, `LookupType` → `ErrTypeNotFound` / `ErrTypeAmbiguous`).

## Touchpoints (exact)

- `cmd/expense-reporter/cmd/batch_auto.go`
  - `classifiedRow` struct (~line 64): add a `Type string` field.
  - `classifyLines` (~line 177): after classification yields `(Category, Subcategory)`,
    resolve the type via the shared `taxonomy.TypeIndex` (build it once from the loaded
    taxonomy + thread it in). On `ErrTypeNotFound`/`ErrTypeAmbiguous`, leave `Type` empty
    (the generator's bare-name fallback still routes it) — never fail the row.
  - `writeClassifiedCSV` (~line 358) and `writeReviewCSV` (~line 386): add a `type` column.
    Header becomes `item;date;value;subcategory;category;confidence;auto_inserted;type`
    (append `type` LAST to minimize churn — but verify the consumer below agrees).
- `internal/review/queue.go` `ReadQueue` (~line 50): the 7-field guard must become 8;
  read the new `type` column into `Predicted.Sheet`.
- `internal/review/types.go`: `Predicted.Sheet` already exists — no struct change expected.

## ⚠ Pre-existing inconsistency to resolve FIRST (do not ignore)

`writeClassifiedCSV`/`writeReviewCSV` emit `auto_inserted` as `true`/`false` (`%v`), but
`review.ReadQueue` parses it as `"1"`/`"0"` and errors otherwise. This means the `review`
command does NOT consume batch-auto's CSV directly today (there may be an intermediate
step, e.g. `.claude/tools/reconstruct-csvs.py`). **Before adding the column, trace which
CSV `review` actually reads** so the producer and consumer column orders stay consistent.
If they are genuinely different formats, the `type` column must be added to the format
`review.ReadQueue` consumes — confirm by reading `internal/review/.memories/QUICK.md` and
the `reconstruct-csvs.py` tool. Document what you find in your final report.

## Tests FIRST (TDD — required)

- Unit (`cmd` or wherever `classifyLines` is testable): a classified row whose
  `(category, subcategory)` resolves uniquely gets `Type` set; an ambiguous/absent pair
  leaves `Type` empty. Use a small in-memory taxonomy like
  `internal/taxonomy/lookup_test.go`'s `sampleTypes`.
- Unit `internal/review/queue_test.go`: an 8-column row populates `Predicted.Sheet`;
  malformed column count errors with the updated expected-field message.
- Acceptance (`test/`): update the batch-auto fixtures so `classified.csv`/`review.csv`
  expected files carry the `type` column. Follow PATTERNS.md — deterministic fields only;
  the `type` value is taxonomy-derived (deterministic) so it CAN be asserted, unlike the
  LLM-chosen subcategory. Run `go test -tags=acceptance ./test/...`.

## Codegen rule

All Go you add (>~5 lines) goes through the local model first
(`mcp__ollama-bridge__generate_code`, model `my-go-qcoder`), tests passed as context.
Record a 0/1/2 verdict for every call. See the conventions doc (reading #1).

## Advisor budget

You may call `advisor` up to 3 times. First call AFTER you have read the files above and
traced the CSV consumer, BEFORE writing implementation — sanity-check the column-ordering
decision and the `type → Predicted.Sheet` boundary call.

## Done criteria

- `go build ./...`, `go vet ./...`, `go test ./...`, and
  `go test -tags=acceptance ./test/...` all green.
- New unit + acceptance assertions cover the type column end-to-end.
- Final report: files changed, verdicts, the CSV-consumer finding, and any advisor input.
