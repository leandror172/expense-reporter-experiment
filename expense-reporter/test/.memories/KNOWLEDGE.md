# test/ — Knowledge (Semantic Memory)

*Acceptance test harness accumulated decisions. Read on demand by agents.*

<!-- ref:acceptance-dsl -->
## Reading the DSL Before Writing Tests (session 64)

The suite is written in a small DSL, split across two repos. **Read these before adding or
changing a scenario** — writing tests without them is how the suite accumulated 27 Given
bodies for 3 real preconditions.

| # | Read | Why |
|---|---|---|
| 1 | `test/PATTERNS.md` (this repo) | The rules: Given/Then naming, composition, canonical Givens, the config-merge constraint. Start here. |
| 2 | `test/givens_test.go` | The Given vocabulary itself — 4 atomic events + 3 canonical Givens. Every new Given should be a wrapper over one of these. |
| 3 | `test/actions/commands.go` header | The **When** rule, which is the inverse of the Given rule (below). |
| 4 | module `docs/PATTERNS.md` | The generic methodology (three-phase model, zero-copy-read vs copy-to-write, traps). At `$(go env GOMODCACHE)/github.com/leandror172/acceptance-harness@v1.1.0/docs/`, or the clone at `/mnt/i/workspaces/acceptance-harness`. |
| 5 | module `harness/scenario.go` | The whole DSL surface is ~150 lines: `Context`, `Scenario`, `Run`, `Events`, `UseBinary`. Read it once; it answers most "can the harness do X" questions. |

### The invariants a new test must not break

- **The engine owns scenario plumbing.** A Given never sets `ctx.BinaryPath` (from
  `harness.UseBinary` in TestMain) or `ctx.FixtureDir` (from `Scenario.Fixture`), and never
  takes a fixture path — events read `ctx.FixtureDir`.
- **A Given with an empty body means the scenario has no precondition** — omit `Given:`
  rather than keep a no-op with a domain-sounding name. Adopting v1.1 emptied six helpers
  whose entire bodies had been plumbing.
- **Compose inside helper definitions, not at the call site.** `harness.Events(...)` builds a
  named Given from atoms; the scenario still reads as one domain sentence.
- **Config must accumulate, never overwrite.** `domain.SetupBinaryConfig` collects keys in
  `ctx.State` and writes config.json once from a `ctx.BeforeWhen` hook. Two events each
  writing the file would silently drop each other's keys. Never write config.json directly.
- **Given vs When name the opposite things.** For a Given, every argument is plumbing → make
  it implicit. For a When, arguments are the *subject under test* (`RunAdd("Uber Centro;…")`)
  → keep them explicit; only fixture *location* is implicit.
- **Duplicate bodies are the tell** that a name describes assembly rather than a fact. Two
  smells: a Given naming a command flag (`--dry-run` belongs to the When), and a Given taking
  a fixture path it never uses (Go does not flag unused parameters).

### Where a change belongs

Engine capability gap (Context/Scenario/Run) → PR to the module + version bump here. Generic
assertion → module `verify/`. Domain assertion → `expect/`. This boundary is the point of the
extraction; see the section below.
<!-- /ref:acceptance-dsl -->

## Harness Design — Engine Extracted to a Module (2026-03 → T2 Session B, 2026-07-17; v1.1.0 since session 64)
The engine that used to live in `test/harness/` is now the public
`github.com/leandror172/acceptance-harness` module (pinned **v1.1.0**). It was always written
to contain zero expense knowledge, and the extraction proved that: `Context`, `Scenario`,
`Run`, fixture plumbing, `FindModuleRoot`/`BuildBinary` all lifted unchanged.
**Rationale:** a second consumer (career-search's `roles` CLI — Go, deterministic, non-LLM)
wanted the same engine. Two copies of a BDD engine drift; one of them had already grown a
latent bug (`Run` passing `t.Name()` into `os.MkdirTemp` crashes under subtests) that the
module's unit tests caught and fixed for both.
**Implication:** the engine is no longer ours to edit. A change to Context/Scenario/Run is a
PR to the module plus a version bump here — which is the point: it forces the domain-free
boundary to stay honest.

### What stayed, and why (the module's core is lean by design)
| Here | Why not in the module |
|---|---|
| `extern/` (`RequireOllama`) | core is LLM-free and network-free; a generic CLI-test lib must not assume an LLM |
| `domain/` (workbook gate+copy, `SetupBinaryConfig`, fixture-config decode, Env accessors) | encodes *this* CLI's conventions: an Excel workbook, `os.Executable()`-relative config |
| `expect/` (25 domain verifiers) | "what this CLI's bytes mean" |
| `expect/` `readCSVFile` | semicolon delimiters + `#` comments are this project's CSV conventions |

### Two assertion packages (D1) — the module owns `verify`
Scenarios import **both**: `verify.*` is the module's generic Then (CommandSucceeded,
CommandFailed, OutputContains, OutputNotContains, OutputFileExists, OutputIsValidJSON,
OutputJSONHasKey, OutputJSONHasValue); `expect.*` is ours. The local `verify` package was
renamed to `expect` and its 8 duplicates of module functions deleted — they were verified
byte-identical first, so the deletion changed no behavior.
**Rationale:** two same-named packages would force an import alias in every file that
asserts anything (the option career-search explicitly rejected). career-search made the same
call and named its domain package `tracker`.
**Implication:** never re-add a local package named `verify`. When adding an assertion, ask
whether it is generic (belongs upstream in the module) or domain (belongs in `expect/`).

### Domain values ride in `ctx.Env` (D2)
The module's `Context` has no `DataDir`/`WorkbookPath` (and `WorkbookDir` is gone — it had
zero references). Those values live in `ctx.Env` behind accessors in `domain/env.go`
(`SetDataDir`/`DataDir`, `SetWorkbookPath`/`WorkbookPath`). They were never scenario state —
they are flag defaults: every read is in `actions/`, and each becomes `--data-dir`/`--workbook`.
`runCommand` forwards `ctx.Env` to the subprocess, honoring Env's documented contract as the
env-injection seam rather than quietly using it as a private bag; the two keys ride along
harmlessly since the binary takes them as flags.
**Rejected:** a domain wrapper struct around Context — `harness.Run` hands `*harness.Context`
to Given/When/Then, so a wrapper can't be the callback parameter without re-wrapping `Run`.

## Fixture Format (2026-03)
Each fixture is a directory with:
- `config.json` — command, model, threshold, assertion_type, extra_args, accuracy_floor, top_n
- `input.csv` — semicolon-delimited, `#` comments supported
- `expected-classified.csv` — optional, for soft accuracy comparison
- `expected-feedback.jsonl` / `expected-expenses_log.jsonl` — for JSONL log verification
**Key distinction:** classify/auto fixtures use `input.csv` as a scenario table (each row
becomes a separate test invocation). batch-auto fixtures pass `input.csv` directly to the binary.
**Rationale:** Different commands have different input models. Scenario tables let
classify/auto tests run multiple items in one fixture without batch machinery.
**Implication:** The fixture format is a contract — changing it requires updating both
the test code and all existing fixture directories.

## Soft vs Hard Assertions (2026-03)
- **Hard** (`assertion_type: "hard"`) — exact match required, test fails on mismatch
- **Soft** (`assertion_type: "soft"`) — calculates accuracy percentage, fails only below
  `accuracy_floor`. Writes JSON reports to `test/results/` for drift tracking.
**Rationale:** LLM classification is non-deterministic. Hard assertions on classifier
output make tests flaky. Soft assertions with a floor catch regressions without
requiring exact reproducibility.
**Implication:** Mechanics tests (installments, rollover) use hard assertions — they
test deterministic logic. Classification tests use soft assertions.

## Auto-Insert Fixtures — Gate-Passing Items (T-32; was Threshold 0.0)
**`--threshold` is deprecated/ignored since T-32.** Auto-insert is now the AGREEMENT gate,
so a fixture that must exercise the auto-append path needs an item the gate ADMITS: an
item whose keyword has maximum specificity (spec 1.0, unambiguous) AND that the model
predicts to that same subcategory. `threshold: 0.0` in fixture configs no longer forces
auto-insert — it is inert.
**Rationale:** structural tests (installments, rollover, typed log, feedback) assert the
auto-append output; under the strict gate only gate-passing items append.
**Implication:** use gate-passing items (see Canonical Test Items) for any test that must
auto-append; classification-quality tests still use soft assertions.

## Canonical Test Items (2026-03)
- **"Posto Ipiranga"** — gate-PASSING baseline (T-32): keyword `posto`/`ipiranga` spec 1.0 →
  Combustível (unambiguous), model agrees → auto-inserts. Replaced "Uber Centro" in the
  auto/feedback/installment/rollover/typed fixtures.
- **"Uber Centro"** — do NOT use for auto-append tests: keyword `uber` spec 0.8 and AMBIGUOUS
  (Viagens/Uber-Taxi) → FAILS the agreement gate → routes to review, never appends. Still fine
  for `add` (no gate) and classified.csv/structural assertions.
- **"Diarista Letícia"** — reliable for Diarista subcategory. Used in batch tests.
**Rationale:** Empirically discovered that some items are nearly deterministic across
Ollama model versions, while others are sensitive to model changes.
**Implication:** Use canonical items for structural tests. Use diverse items only in
soft-assertion tests where accuracy drift is tracked.

## Composable Then Pattern (2026-04)
Then helpers return `[]func(*harness.Context)`, not single functions. Combined with
`slices.Concat` at the test site. Each helper is scoped to one concern (e.g.,
"classified output has correct columns" vs "accuracy meets floor").
**Rationale:** Monolithic assertion functions hide what's being tested and make it
hard to compose different assertion sets for different fixtures.
**Implication:** When adding a new assertion concern, create a new helper function
that returns `[]func(*Context)`. Never add assertions to existing helpers unless
they're truly part of the same concern.

**Convention (2026-04; package names updated T2 Session B):** `Then:` blocks must contain
only named `then*` helpers — never raw `verify.*`/`expect.*` calls directly. Those calls
belong inside helper bodies. This keeps test intent readable at the scenario level and keeps
assertion details encapsulated. `then*` helpers live in the same `*_test.go` file as their
tests; `expect.*` functions live in `expect/` and `verify.*` comes from the module — both
return a single `func(*harness.Context)`.
`commandSucceeded()` (feedback_test.go, same package) is the shared base — use it
via `slices.Concat` rather than calling `verify.CommandSucceeded()` directly.

**Refinement — name Then helpers by expected RESULT, not artifact (2026-06, PR #35):**
A `Then` helper name should let a reader infer the scenario's behavior without opening the
helper or the fixture. Prefer the *outcome* over the *mechanism*:
`installmentExpandedToNDatedLogLines(fixDir)` over `expenseLogMatchesExpected(fixDir)`;
`crossYearInstallmentNotDivertedToRollover()` over an inline `expect.NoRolloverFileCreated()`.
- **Split on variance:** keep genuinely *invariant* concerns generic (`commandSucceeded()`,
  `classificationsMatchExpected()`); only the *scenario-varying* concern needs the
  outcome-describing name. This reconciles "one concern per helper" with "name the result".
- **Per-scenario wrappers:** when several scenarios share one verifier, wrap it in named
  helpers that just `return expenseLogMatchesExpected(fixDir)`. The body is one line — the
  wrapper's value IS its name; the differing fixture encodes the differing result.
- **Worked example:** `add_log_append_test.go` (typedExpenseRecordedAsSingleLogLine,
  installmentExpandedToNDatedLogLines, crossYearInstallmentLoggedWithNextYearDate,
  crossYearInstallmentNotDivertedToRollover). Rule also documented in `PATTERNS.md`.
- **PENDING SWEEP (not yet done):** only `add_log_append_test.go` was renamed (the file PR #35
  flagged). Other files still call the mechanism-named `expenseLogMatchesExpected(` /
  `classificationsMatchExpected(` and may carry the same smell — candidates: `apply_test.go`,
  `correct_test.go`, `fewshot_test.go`, `generate_income_test.go`, `generate_test.go`,
  `json_output_test.go`, `typed_log_test.go`, `type_routing_cycle_test.go`. Not yet read or
  judged; rename only the scenario-varying call sites, leave the generic invariants alone.

## JSONL Verification Design (2026-03)
File-specific verifiers (not generic string-keyed):
- `expect.ClassificationsMatch(expectedPath)` — checks `classifications.jsonl`
- `expect.ExpenseLogMatches(expectedPath)` — checks `expenses_log.jsonl`
Expected files omit non-deterministic fields (`id`, `timestamp`). For classifier-dependent
tests, `subcategory`/`category` are also omitted from expected files.
**Rationale:** JSONL logs include auto-generated fields (hash IDs, timestamps) that
differ every run. Expected files contain only the deterministic contract.
**Implication:** When adding new fields to JSONL output, update the verifier's skip
list if the field is non-deterministic.
**WS-B slice 3 (2026-06–30):** the log-append path (`add`/`auto`/`batch-auto` via `appender.ExpandAndAppend`)
writes the **`date` field as `DD/MM/YYYY`** (was bare `DD/MM` under the old `logExpense`). Bare-`DD/MM` inputs
get `time.Now().Year()` (`ParseDateFlexible`) — a year time-bomb — so `expected-expenses_log.jsonl` fixtures use
**explicit-year inputs** and pin the deterministic subset `{item, date, value, type}` (subcategory/category left
out where classifier-dependent). Installment rows assert the expanded `item (i/N)` + incremented dates; cross-year
rows pin the real next-year date (e.g. `01/01/2027`) and pair with `NoRolloverFileCreated`.

## Generate-Workbook Acceptance Design (2026-06, session 29)
The `generate-basic` fixture is a NEW sub-format: `taxonomy.json` + `entries.jsonl`
(+ `entries-with-unmapped.jsonl`) + committed `expected-dump-skeleton/` and
`expected-dump-data/` (per-sheet `internal/inspect` JSON dumps). No config.json, no input.csv,
no Ollama (fully deterministic, <1s — safe for the suite-timeout budget).
**Oracle-frozen expectations:** dumps were frozen from the convergence-verified scratch
builder run on the same fixture BEFORE the generator existed, making G2 a converge-to-green
port (healthy RED = command-absent, not expectation-fake).
**Rationale:** placeholder expectations assert nothing and freezing a generator's own output
against itself is circular; an independent already-trusted producer is the only real oracle.
**Limit:** oracle and port can share a bug (hardcoded sheet order emitted invalid D0 refs in
the frozen dumps). When the contract changes, re-freeze and MANUALLY REVIEW the dump delta —
acceptance can't distinguish "both fixed" from "both broken".
**Normalized-subset comparison:** `expect.WorkbookStructureMatches` asserts exact equality on
values/formulas/merges/dims/rowType/rowFill/bgColor/bold/borders and ignores column widths,
row heights, and manifest source (excelize float/serialization noise). Full deep-equality
would turn cosmetic excelize quirks into red tests.
**Implication:** when extending the generator, run the acceptance tests first; if a deliberate
output change is intended, regenerate `expected-dump-*` with the fixed binary + workbook-inspect
and diff old-vs-new dumps before committing.

## Incremental Full-Cycle Test (type-routing-cycle, session 34)
`type_routing_cycle_test.go` proves the batch-auto→review→apply→generate-workbook chain that
validated the `sheets`→`types` fix. Design decisions worth reusing:
- **One CLI step per test, Given accumulates.** Each test's When is one command; prior steps
  become Given preparation. The last test (`_4_…RoutesByType`) seeds the cumulative typed log
  and runs ONLY generate-workbook — its Then is the payoff (ambiguous leaf Dentista ∈
  Variáveis+Extras routes to its chosen sheet, and is ABSENT from the other candidate).
- **Non-CLI steps fold into fixtures**, documented at the fold point: the browser pick+export
  lives in `reviewed.json`. The harness can't drive a browser, so don't model it as a When step.
  **CORRECTED T-54 (session 68):** this bullet used to also list a "true/false→1/0 CSV bridge"
  in `review-input.csv` as a legitimate fold. It was not a fold — it was a **product bug wearing
  a fixture's clothes**. `review.ReadQueue` accepted only `1`/`0` while the only Go producer
  emitted `true`/`false`, so `expense-reporter review classified.csv` hard-errored on every real
  batch-auto file; the hand-conversion in the fixture hid it, and this note made the hiding look
  intentional. **The lesson generalizes: a fixture that TRANSFORMS producer output before feeding
  it to a consumer is not bridging a harness limitation — it is asserting a contract neither side
  implements.** A fold point is legitimate only when the step genuinely cannot run here (a
  browser, a human). Reformatting is not such a step. When you find yourself writing a converter
  into a fixture, the producer and consumer disagree — fix them, don't document the gap.
  Guard: `cmd.TestClassifiedCSV_ReviewReadsWhatBatchAutoWrote` now feeds the REAL writer's bytes
  to the REAL reader, so no fixture has to be kept in step for this to stay honest.
- **Hermetic skeleton trick:** apply's new-row insert needs a workbook with the target slot.
  Instead of committing a binary `.xlsx`, the Given builds one with `generate-workbook` (no
  `--entries`). So generate-workbook is both the *subject* of the last test and a *setup tool*
  in apply's Given — empty skeleton vs `--entries`-filled.
- **Determinism split:** only the batch-auto step needs Ollama (gated, ~38s, asserts just the
  8-column type contract — not LLM values). Review/apply/generate are deterministic (<0.05s),
  so the fix's regression guard never depends on Ollama.
- **Routing assertion without a frozen oracle:** for a single-value routing check, scan the
  generated sheet's cells for the entry's unique value (inline excelize) rather than
  `WorkbookStructureMatches` — cheaper than freezing/maintaining dump fixtures.

## WS-B Acceptance Retarget — batch-auto & apply → expense log (sessions 43–44, consolidated from QUICK.md 2026-07-01)
Both commands stopped writing the workbook; acceptance now asserts the durable log.
- `expect.ExpenseLogMatches(<fixDir>/expected-expenses_log.jsonl)` — field-subset, line-exact,
  skips id/timestamp. `batch-auto-typed` = canonical NON-dry-run append anchor (workbook gate
  dropped). `batch-auto-installments` asserts the N expanded dated log lines.
  `batch-auto-rollover` INVERTED → `expect.NoRolloverFileCreated()` + next-year dates in the
  log (rollover.csv retired).
- **Unit-vs-acceptance split:** append-failure downgrade is a unit test
  (`cmd.TestAppendClassified_DowngradesRowOnAppendFailure`, `cmd.TestAppendNewRows_*`) because
  the pre-flight makes an acceptance-level append failure unreachable. Acceptance covers the
  pre-flight deterministically (no RequireOllama): parent-is-a-file path →
  `TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification`,
  `TestApply_UnwritableLogPath_FailsFast`, `TestApply_UnwritableClassificationsPath_FailsFast`.
- Apply (slice 4): `TestApply_IdempotencyAndFeedback` asserts summary substring
  `"no expense-log change"` + `ExpenseLogNotCreated` (found-only → no append).
  `TestApply_DryRunWritesNothing` pins the found+corrected feedback leak (both logs
  byte-unchanged). `type_routing_cycle_test.go` step 3 dropped `buildSkeletonWorkbook`
  (apply needs no workbook). PR #35 naming sweep applied.
**Why:** JSONL log is the single source of truth (retire-insertion pivot); asserting workbook
rows tested a writer that no longer exists.

## Generate-Workbook Fixture Sub-Format (session 29+, consolidated from QUICK.md)
`generate-basic` / `generate-income` use taxonomy.json + entries.jsonl + oracle-frozen
`expected-dump-*/` — NOT config.json+input.csv (see PATTERNS.md "Generate-Workbook Fixture
Sub-Format"). Assertions: `expect.WorkbookStructureMatches(expectedDumpDir)` over
`internal/inspect` dumps. `generate-income` (WS-C, session 38) covers the 3-level income
route: nested `incomeCategories` + `income-entries.jsonl` (extractor schema) via
`--income-entries`; asserts signed sums (Salário Jan 4150 etc.) + per-Block Listas rollup.
**Coupling:** changing the income/summary render re-freezes BOTH `generate-income` AND
`generate-basic` data+skeleton oracles (income summary shifts Listas rows).

## type-routing-cycle — Incremental Full-Cycle Suite (details, consolidated from QUICK.md)
batch-auto→review→apply→generate-workbook; each test = one CLI step folding prior steps into
its Given; the last seeds the cumulative typed log and runs only generate-workbook, asserting
the ambiguous leaf (Dentista ∈ Variáveis+Extras) routes by type. T1 is Ollama-gated (~38s);
T2–T4 deterministic (<0.05s). apply's skeleton workbook (pre-slice-4) was built hermetically
via generate-workbook in the Given — dropped in slice 4.

## Traps That Masked Regressions (sessions 42–43, consolidated from QUICK.md)
- **Explicit-year time-bomb:** the append path reformats dates to `DD/MM/YYYY`
  (`ParseDateFlexible` fills bare `DD/MM` with `time.Now().Year()`). Non-dry-run
  batch-auto/auto/add fixtures MUST use `DD/MM/YYYY` inputs + explicit-year expected logs, and
  clean-dividing installment values (`90,00/3`→30) to avoid float JSON drift.
- **Dry-run hides the append path:** fixtures with `--dry-run` in `extra_args`
  (batch-auto-basic/exclusions/type-routing-cycle) never append — they prove CSV production
  only. `RequireWorkbook` also SKIPs when the test workbook is absent — that hid a stale
  non-dry-run feedback test through the whole session-42 sweep.
- **Build tag hides breakage:** T-13 made `taxonomy_path` mandatory but many Givens never set
  it; `go test ./...` stayed green. Session 42 repaired 13 tests; fixture taxonomy must cover
  the input CSV's expected leaves (accuracy tests compare subcategory only).
  [[feedback_rename_json_tag_acceptance]]

## --resume / dedup acceptance (T-20, session 60)
- **Seeds go through the REAL append path:** the Given calls `appender.ExpandAndAppend`
  (never hand-written ids), so any prediction/append normalization drift fails loudly.
  Fixtures seed nothing on disk; the Given seeds after `withFeedbackAndTaxonomyConfig`.
- **Fully-seeded fixtures are Ollama-FREE** (`batch-auto-resume-all-seeded`, `-dryrun`):
  every row skips before the model — deterministic resume tests exist because the skip
  precedes classification.
- One-of-two-dups uses gate-failing `Uber Centro` so the non-skipped row deterministically
  hits review. New `expect` counters: `ResumeSkipCount` (counts "already logged" over
  stdout+stderr), `DuplicateWarningCount` (counts "already in expense log" over stderr).
  The batch-auto summary deliberately says `Skipped       :` (not "already logged") so the
  summary line can't inflate `ResumeSkipCount`.
