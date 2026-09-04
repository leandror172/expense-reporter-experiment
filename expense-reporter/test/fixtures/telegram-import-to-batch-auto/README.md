# telegram-import-to-batch-auto

The step-3 gate of the T-75 plan, end to end: `telegram-import` writes `expenses-2025-05.csv`
into the work dir, then `batch-auto --dry-run` (config.json is `batch-auto-basic`'s) reads
THAT file. Passing means every converted line — including the repaired one, which carries a
trailing `# repaired from: …` comment — parses through the real reader, so no `failed.csv`
appears. Classification runs, so this scenario needs Ollama and lives in the `-full` group;
the deterministic half of the same claim is the round-trip unit test in
`cmd/.../telegram_import_write_test.go`, which feeds the converter's lines to `parse3FieldLine`.

`result.json` is a copy of `telegram-import-dry-run`'s; `fixture-taxonomy.json` is
`batch-auto-basic`'s.
