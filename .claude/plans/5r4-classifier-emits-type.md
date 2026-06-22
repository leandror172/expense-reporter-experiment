# Plan — 5.R4: Classifier emits the expense *type* into expenses_log.jsonl

**Owner:** Sonnet subagent (worktree-isolated).
**Branch base:** `feat/type-routing-improvements` (already contains the shared
`internal/taxonomy/lookup.go` reverse-lookup primitive — DO NOT reimplement it).

## Goal

`auto` and `batch-auto` currently write `expenses_log.jsonl` entries with an EMPTY
`type` (see the explicit `TODO(type)` in `logExpense`, `cmd/expense-reporter/cmd/auto.go`).
Those type-less entries fall back to bare-name routing in `generate-workbook` and get
skipped when the bare name is ambiguous. Populate `type` at write time by reverse-looking-up
the classifier's `(category, subcategory)` against the taxonomy. This shrinks the
"N entries routed via the type-less bare-name fallback" count toward zero.

## Domain-translation rule (HARD)

- The persisted field is `ExpenseEntry.Type` (domain term) — already exists with
  `json:"type,omitempty"` (Plan A). You are POPULATING it, not adding it.
- `sheet` stays reserved for workbook-manipulating code. The value you store in `Type`
  is the `ExpenseType.Name` returned by `taxonomy.LookupType` — a domain type, not a
  workbook handle. Do not introduce "sheet" naming in the classifier/auto path.

## Required reading before any work (ABSOLUTE paths)

1. `/mnt/i/workspaces/expenses/code/.claude/overlays/local-model-conventions.md`
2. `/mnt/i/workspaces/expenses/code/.memories/QUICK.md`
3. `/mnt/i/workspaces/expenses/code/.memories/KNOWLEDGE.md`
4. `/mnt/i/workspaces/expenses/code/expense-reporter/test/PATTERNS.md`
5. `/mnt/i/workspaces/expenses/code/.claude/index.md`
6. The QUICK/KNOWLEDGE in any folder you edit, BEFORE editing it — at minimum
   `/mnt/i/workspaces/expenses/code/expense-reporter/internal/feedback/.memories/QUICK.md`
   and `/mnt/i/workspaces/expenses/code/expense-reporter/internal/classifier/.memories/QUICK.md`
7. The shared primitive: `/mnt/i/workspaces/expenses/code/expense-reporter/internal/taxonomy/lookup.go`
   (`BuildTypeIndex`, `LookupType` → `ErrTypeNotFound` / `ErrTypeAmbiguous`).

## Touchpoints (exact)

- `internal/feedback/expense_log.go`: `NewExpenseEntry` builds the entry without a type.
  Decide the cleanest seam — EITHER add a `NewExpenseEntryWithType(..., typ string)` (keep
  the old constructor for callers that legitimately have no type, e.g. income), OR set
  `entry.Type` at the call site after construction. Prefer the smallest, clearest change;
  the field is `omitempty`, so an empty type serializes away exactly as today.
- `cmd/expense-reporter/cmd/auto.go` `logExpense` (~line 190): remove the `TODO(type)`.
  It must resolve the type from `(subcategory, category)` via a `taxonomy.TypeIndex`
  built once from the configured taxonomy path. On `ErrTypeNotFound`/`ErrTypeAmbiguous`,
  leave the type empty (bare-name fallback handles it) — non-fatal, like today's other
  expense-log warnings.
- `cmd/expense-reporter/cmd/batch_auto.go` (~line 297): the `logExpense` call there
  inherits the same behavior; ensure the `TypeIndex` is constructed once for the batch
  (not per row) and threaded in.
- Taxonomy path: resolve it the same way the rest of the app does (see
  `internal/config`). If a taxonomy file is absent/unconfigured, degrade gracefully —
  no type set, no crash; warn once to stderr at most.

## ⚠ Coordination note (shared file)

RUI-4 (the sibling plan) also edits `cmd/expense-reporter/cmd/batch_auto.go`. You are in
an ISOLATED worktree, so just make your change cleanly; the orchestrator reconciles both
branches afterward. Keep your batch_auto.go edits localized to the `logExpense` call site
and the `TypeIndex` construction — do NOT refactor the CSV writers or `classifiedRow`
(those are RUI-4's).

## Tests FIRST (TDD — required)

- Unit `internal/feedback/expense_log_test.go`: constructing an entry with a type sets
  `Type`; the JSON line includes `"type"` only when non-empty (omitempty contract).
- Unit (auto/batch path): given a small in-memory taxonomy (mirror `sampleTypes` from
  `internal/taxonomy/lookup_test.go`), a uniquely-resolvable `(category, subcategory)`
  yields a typed entry; an ambiguous/absent pair yields a type-less entry. Assert the
  reverse-lookup integration, not the LLM.
- Acceptance (`test/`): a batch-auto scenario whose expected `expected-expenses_log.jsonl`
  now carries `"type"` for rows whose `(category, subcategory)` resolves uniquely in the
  fixture taxonomy. Per PATTERNS.md, only assert deterministic fields; the type is
  taxonomy-derived so it is deterministic. Use the existing JSONL verifiers
  (`verify.ExpenseLogMatches`). Run `go test -tags=acceptance ./test/...`.

## Codegen rule

All Go you add (>~5 lines) goes through the local model first
(`mcp__ollama-bridge__generate_code`, model `my-go-qcoder`), tests passed as context.
Record a 0/1/2 verdict for every call. See the conventions doc (reading #1).

## Advisor budget

You may call `advisor` up to 3 times. First call AFTER reading the files above and
choosing the `NewExpenseEntry` seam, BEFORE writing implementation — sanity-check the
constructor change and the once-per-batch index construction.

## Done criteria

- `go build ./...`, `go vet ./...`, `go test ./...`, and
  `go test -tags=acceptance ./test/...` all green.
- A real-ish run shows the type-less fallback count drop for rows that resolve uniquely.
- Final report: files changed, the constructor-seam decision, verdicts, advisor input.
