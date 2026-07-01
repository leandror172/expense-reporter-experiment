# review — KNOWLEDGE

*Accumulated decisions. Read on demand.*

## sheets→types JSON Rename (session 34, consolidated from QUICK.md 2026-07-01)
The `sheets`→`types` / `sheet`→`type` JSON-tag rename completed in commit 321dd2f — the
template JS had migrated during Plan A/B but the Go structs still serialized
`sheets`/`sheet`, crashing the page (`TAX.types not iterable`).
**Why it matters:** the embedded JSON is a cross-language key contract (Go ↔ template JS);
`render_test.go` now asserts the JSON carries `"types"`/`"type"`, never
`"sheets"`/`"sheet"`. JSON export key migrated `sheet`→`type` in Plan A R6; apply keeps
legacy `sheet` read-compat via custom `UnmarshalJSON`.
**Boundary rule:** translation happens at output construction — source-side helpers in
`taxonomy.go` (`sheetOrder=[Fixas,Variáveis,Extras,Adicionais]`, `sheetRank`) keep the
"sheet" vocabulary because they read workbook-facing `resolver.SheetName`.

## Type Column in the Classified CSV (RUI-4, session 33, commit 41afa56)
Input CSV (`queue.go`) is 8 fields: item, date, value, subcategory, category, confidence,
auto_inserted, **type**; `ReadQueue` reads `record[7]` into `Predicted.Type`. The page can
also re-derive candidate types from the taxonomy (`typeByName`/`SHEETS_FOR`).
The Go ingest/log loss is fixed (Plan A): apply persists the type; the generator routes
typed entries by full path ([[taxonomy]] two-tier routing). `exportReviewed()` lives at
template ~line 1504.
