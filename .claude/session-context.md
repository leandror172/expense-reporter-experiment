# Session Context — Expense Reporter

**Purpose:** User preferences and working context across Claude Code sessions.

---

<!-- ref:user-prefs -->
## User Preferences

### Interaction Style
- **Output style:** Explanatory (educational insights with task completion)
- **Pacing:** Interactive — pause after each phase for user input
- **Explanations:** Explain the "why" for each step, like a practical tutorial

### Configuration Files
- **Build incrementally:** Never dump full config files at once
- **Explain each setting:** Add a setting, explain what it does, then add the next
- **Ask before proceeding:** Give user options before making non-obvious choices

### Local Model Usage
- **Try local models first** for Go boilerplate, simple functions, test stubs
- **Preferred model:** `my-go-qcoder` (qwen3-coder:30b via ollama-bridge MCP) — benchmarked session 23 (2026-05-18); verdicts 2/2/1/1 on test + cobra generation. Fallback: `my-go-q25c14` (qwen2.5-coder:14b) if qcoder unavailable (qcoder HTTP 500 = host-RAM ENOMEM from 9p/no-mmap, not VRAM — see benchmark note below).
- **Speed vs cost:** 30s for correct 14B output beats 6s for wrong 8B output. Local inference cost is latency only.
- **Verdict pattern:** Always record verdict **2/1/0** after local model output (2=accepted, 1=improved, 0=rejected)

### Shell Scripts
- Always use `./script.sh` not `bash script.sh` — `./` form is whitelistable per-script in Claude Code
<!-- /ref:user-prefs -->

---

## File Management

### Sensitive Data
- **Location:** `.claude/local/` (gitignored) and `data/classification/*.json` (personal expense data)
- **Rule:** Real expense descriptions, training data, personal financial info → always gitignored

### Log Rotation
- **Tool:** `.claude/tools/rotate-session-log.sh` — run at session end via session-handoff skill
- **Policy:** Keep 3 most recent sessions in `session-log.md`; archive the rest

---

<!-- ref:current-status -->

- **Pre-history (Claude Desktop):** Phases 1–11 complete — full CLI (add/batch/version), 190+ tests, v2.1.0
- **Classification analysis:** Complete (auto-category work) — results in `data/classification/`
- **Workbook Generator:** COMPLETE; PR #27 merged. `internal/taxonomy` extracted (T-02) + real `config/taxonomy.json` (gitignored) + full-path identity key.
- **Plan A (T-05) + Plan B (T-04):** MERGED. Expense `Type` persisted end-to-end; two-tier routing — typed by full path (`byPath`), type-less via the **transitional** bare-name fallback (`byName` + ambiguous-skip).
- **5.R4 historical extraction — DONE.** 2022–2025 old workbooks → corpus 694→1788 + per-year expense logs.
- **PIVOT (session 36): retire workbook insertion, keep only generation** — JSONL logs are the single source of truth, `generate-workbook` the only writer.
- **WS-A/T-11, WS-C income route — DONE.** T-13 (full-path 112-enum classifier), WS-B (commands→log-append), T-19 (sentinel decline), T-14/T-22/T-26 (accuracy/descriptions/think-on benchmarks), T-30 (leaf-first A/B → HOLD) — all DONE & merged.
- **T-23 ROUTE REFRAMED (session 51):** gate = **external validators** (E1 recurrence-strength / E2 value-range / E3 retrieval-agreement) over **model-introspection**. Recurrence dwarfs any token signal. PR #43 (docs) merged. NOT decided — deferred.
- **T-23 GATE + RETRIEVAL MEASURED ON REAL DATA (session 52) — the 649-replay.** SHIP: confidence DEAD as gate; **specificity (`top_score`) replaces it**; **best gate = AGREEMENT `spec==1.0` ∧ model==keyword-top1 → 95.0% subcat**. HOLD: the 649 is a confidence-selected review subset → absolute levels unmeasurable; WS-D scoping HELD.
- **T-31 NN PRECONDITION (session 53) — GO for 5.R2.** hit@5 62–65% multilingual on the 160 misses; arctic-embed2 the practical pick. PR #44 merged.
- **5.R2 BUILT + ADOPTED (session 54) — PR #45 MERGED (session 57).** Embed-on-miss cascade layer; per-model JSONL disk cache; degrade-to-nil on failure. Replay A/B (miss stratum, n=80): full-path 18.8%→52.5% no-think. ~35% permanent-review residue remains inside the novel pool.
- **Method-extraction convention ADOPTED module-wide (session 55).**
- **T-32 SHIPPED (session 57, PR #46 MERGED session 58).** `IsAutoInsertable` gates on keyword AGREEMENT; confidence dropped (measured dead). **CAVEAT: same-sample RELATIVE; ABSOLUTE precision UNMEASURED → WS-D silent-insert HELD.**
- **T2 HARNESS EXTRACTION — CLOSED.** `github.com/leandror172/acceptance-harness` published; **v1.1.0** since session 64. Engine changes = PR upstream + version bump here.
- **T-20 MERGED (s61).** `batch-auto --resume` + always-on duplicate-append warning.
- **T-35 + T-36 MERGED (PRs #50/#51).** Join-id divergence: dates canonicalized ONCE at each boundary so both JSONL logs share one `GenerateID`; guard = `expect.JoinIDMatchesAcrossLogs` + deliberately short-`DD/MM` fixtures.
- **SESSION 62 — T-39 + T-24 SHIPPED (PR #52) + STRATEGIC REFRAME.** Suite split (deterministic group ~8s, `-full` ~100–130s at think-off); `--think=false` default. **Project reframed PRE-FIRST-USE; organizing milestone = first real monthly close on 2026 data (T-42).**
- **SESSIONS 63–66 — T-41 DESIGN + SLICES 1–2.** Design locked: home `internal/parse`; SINGLE stored `time.Time` + `DateString()` as the sole identity-bytes site; grace window 0; boundary owns ALL input normalization. Slice 1 (`add`/`correct`) + slice 2 (`auto`) shipped, plus s65 date/year hardening (resolved-year validations, `YearSource` provenance, stale-`date_year` warning, `date_year` → 2026) and the s64 test-architecture work that produced harness v1.1.0.
- **SESSION 67 — T-41 COMPLETE (slices 3 + 4; PRs #59, #60 OPEN) + T-53.** **Slice 3 (`batch-auto`):** the four-times-per-row re-parse collapsed to one — `inputRow` deleted (`ParsedExpense` already held every field plus the installment count it DISCARDED, which was the root), `resumeParseErr` deleted as unreachable (it existed only because the resume check re-parsed), the T-35 hand-canonicalization deleted. `--year`; ONE counted stale-year warning (T-49); join-id pinned at construction (T-40). **The classified CSVs now carry the CANONICAL date**, closing a divergence where `review.ReadQueue` hashed the raw column into `reviewed.json`'s id while both logs hashed canonical. **Slice 4 (`apply`):** `canonicalDate` — the T-35 prototype — retired into `parse.Date` (new date-only entry point; `Fields` delegates, so one ladder serves both doors), and **`entry.ID` is now recomputed from the date it rewrites**: apply had been looking up an id it never writes, and that miss is SILENT (unfound → treated as new → appended → duplicate). Apply inherits the full boundary incl. T-47 (user call). Fixtures swept off raw-date ids **T-18 had already migrated for `correct` in s42** and never swept here; new `apply-stale-id` is the ONLY discriminating fixture (mutation-verified — `apply-basic` is self-consistent and stays green with the fix reverted). **T-53:** `-count=1` in `run-acceptance.sh`. Acceptance 68 pass / 0 fail / 0 skip.
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — personas, MCP server, platform docs; its `patterns-*` / `test-executable-spec` / `patterns-refactoring-*` refs are the code/test conventions and are injectable into local-model prompts via `refs` + `refs_root`. **acceptance-harness at `/mnt/i/workspaces/acceptance-harness` — master + v1.1.0 tagged, clean.** career-search at `/mnt/i/workspaces/career-search` — TC-EXT2 committed locally, UNPUSHED.
<!-- /ref:current-status -->

---

<!-- ref:resume-steps -->
## Quick Resume

Run `.claude/tools/resume.sh` for a compact session-start summary.

Or manually:
1. `ref-lookup.sh current-status` — current layer, next task, branch state
2. Tail of `.claude/session-log.md` — "Next" pointer from most recent session
3. `git log --oneline -3` — recent commits
4. `.claude/index.md` — find any specific file/topic on demand
<!-- /ref:resume-steps -->

---

<!-- ref:active-decisions -->

## Active Decisions

### Identity — locked (session 67)
- **Identity is DERIVED, never trusted.** Any function that rewrites a value must rewrite whatever was computed from it. `apply`'s `entriesWithCanonicalDates` rewrote each entry's date and kept its `ID` — a hash of that date — so apply looked up an id it would never itself write. The miss is SILENT (unfound → treated as new → appended), so the symptom is a duplicate expense, not an error. Third appearance of this shape after T-35 and T-18; treat a value and its derived key as one edit, always.
- **A guard is not a guard until it has been broken.** Session 67 hit four instances of a check reporting success while testing nothing: a cached acceptance pass, a generated test comparing the parser to a second call to itself, the silent id miss above, and a fixture sweep that blinded its own end-to-end test. Mutation — break the guarded thing, confirm red — is the only cheap discriminator, and its result belongs in the commit message.
- **A deliberately-wrong fixture needs its reason where the reader stands.** `apply-stale-id` carries an id hashed from the pre-canonical date on purpose, and is the only fixture that can catch an id regression. That caveat is recorded in the fixture README, the package memory AND the index, because otherwise the next reader "corrects" it.

### Date/year policy — locked (sessions 65, 67)
- **`date_year` is a per-close-cycle switch, not a permanent setting.** Chosen over deleting it and relying on rung 4: the two agree on every past-dated bare input but diverge on a bare date **later in the year**, where rung 4 is wrong for a close.
- **Its staleness must be visible, not silent.** Warn only when BOTH hold: the config rung actually supplied this expense's year AND that year is below the current one. This is *why* `parse` reports `YearSource` — re-deriving the ladder in `cmd` would be a second copy free to drift.
- **The batch form of that warning is ONE line per RUN carrying a count (T-49, s67).** Per-row would print an identical line for every bare-dated row (300 for a 300-row statement) and train the reader to ignore stderr; hoisting above the loop would fire even when every row carries an explicit year. `--resume`-skipped rows are counted deliberately — a stale year gives them wrong predicted ids, so resume silently matches nothing, which is the case the warning most needs to cover.
- **Future-dated ENTRY is refused (T-47, provisional), and `apply` inherits it (s67).** Year-scale, not day-scale. In `apply` a future-dated reviewed row downgrades to `failed` with a non-zero exit rather than being written — loud rather than silent, which is the rule's purpose. Per-row failure honesty preserved. **Past-side bound deliberately NOT added (T-48).**
- **An enum's zero value must not name a real member.** `YearSourceUnknown` sits at zero so a failed parse cannot report a plausible rung to code that reads the field to decide.

### Parse boundary — COMPLETE (session 67)
- **Two doors, one ladder.** `Fields(item, dateStr, valueStr, opts)` for a caller holding three strings; `Date(dateStr, opts)` for one whose other fields are already typed (`apply` reads `reviewed.json`, where the JSON decoder typed item and value). `Fields` DELEGATES to `Date` — the two year validations are what stand between a mistyped year and a row that hashes cleanly into both logs and then vanishes from the workbook, so they must not exist twice.
- **The classified CSVs carry the CANONICAL date.** `review.ReadQueue` hashes that column into `reviewed.json`'s id while both JSONL logs hash canonical; emitting the raw form handed the review queue an id `apply` would never write.
- **The CSV `value` column keeps the RAW token, never the parsed float.** It is the only place an installment count survives into the review queue; writing the float erases it silently.
- **The 3-field CSV split stays in `cmd`** (plan §11 D1) — a different input FORMAT from the 4-field semicolon form, not a drifted copy; merging them would break error locality.

### Test architecture — locked (session 64)
- **`Scenario.Given` must NOT become a slice.** Every scenario names exactly one Given; the name IS the documentation. Compose INSIDE helper definitions with `harness.Events(...)`.
- **The engine owns scenario plumbing (harness v1.1).** No Given sets `ctx.BinaryPath` or `ctx.FixtureDir`. Engine changes stay ADDITIVE.
- **An empty Given body means the scenario has no precondition** → omit `Given:`.
- **Given and When name opposite things.** Given: arguments are plumbing → implicit. When: arguments are the SUBJECT → explicit.
- **Composed setup accumulates and flushes once** — `ctx.State` + `ctx.BeforeWhen`.
- **Tests build parsed values via the real parser, never struct literals** — `YearSource` is recorded by `resolveDate`, so a hand-set field tests the literal rather than the ladder.
<!-- /ref:active-decisions -->

<!-- ref:session-reading-guide -->

| Task | Read first | Notes |
|------|-----------|-------|
| **Writing or changing ANY acceptance test** | `[ref:acceptance-dsl]` (`.claude/tools/ref-lookup.sh acceptance-dsl`) | REQUIRED reading. Gives the order (`test/PATTERNS.md` → `givens_test.go` → `actions/commands.go` header → module `docs/PATTERNS.md` → `harness/scenario.go`) and the invariants. **s65:** pin the SPECIFIC failure — `commandFailed()` alone passes for any failure. **s67: `run-acceptance.sh` now passes `-count=1` (T-53 fixed)**, so a cached pass can no longer masquerade as a run; the underlying cause (TestMain builds the CLI, so the cache key never covers it) is unchanged, so keep `-count=1` on any direct `go test` invocation. Check `curl -s localhost:11434/api/ps` before trusting `-full` timing — a foreign resident model turned a ~110s run into a 3602s timeout. |
| **Verifying that a guard actually guards** | this session's mutation runs; `[ref:judge-sees-the-change]` in the llm repo | **The single most valuable habit of session 67.** Four defects this session shared one shape: a check reporting success while testing nothing (cached pass, a generated test asserting the parser against itself, a silent id miss, and a fixture sweep that blinded its own test). After writing a guard, BREAK the thing it guards and confirm it goes red. Three guards shipped this session are documented with their mutation result. |
| **T-21 — reviewed-installment under-recording (NEXT)** | `internal/review/queue.go` `ReadQueue`; `review.html` export JS; `internal/apply/types.go`; parse-boundary plan §7 | Unblocked by T-41: `Installments`/`RawValue` are boundary fields and slice 3 retains the count at read time. Count is discarded at `queue.go:63` and absent from `reviewed.json`. UX locked s62: 1 row + ×N badge, `apply` expands. **Do T-54 in the same pass** — it touches the same reader, and `review` currently cannot read batch-auto's output at all. |
| **T-42 — first real monthly close on 2026 data (organizing milestone)** | `docs/expense-classifier-vision.md`; `.claude/t23-strategic-implications.md`; parse-boundary plan §9 | Project is PRE-FIRST-USE. After T-21 + T-54: run a real bank CSV through `batch-auto → review → apply → generate-workbook --year 2026`; everything that breaks is the real backlog. Pre-work also: T-03 rollover check, promote all-years log. **`date_year` is 2026 and warns when stale.** `workbook_path` still names a 2025 file (undecided). T-51 (auto `--json` under the T-47 refusal) and T-52 (`auto`'s arg shape) are deliberately deferred to what this run shows. |
| **Date/year semantics — read BEFORE touching any date code** | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics]; `internal/parse/.memories/QUICK.md`; `[ref:active-decisions]` "Date/year policy" | **T-41 COMPLETE (s67): all six parsers now route through `internal/parse`.** `cmd.canonicalDate` is gone. `Fields` (three strings) and `Date` (date only, for a caller whose other fields are already typed) are the two doors; `Fields` delegates so there is one ladder. The survey still lists year behavior that is CORRECT and must not be "fixed" (installments crossing years, installment ids differing between logs, review's year-less session id, taxonomy's `year=0` sentinel). Open: T-48 (past-side bound), T-50 (`--year 0` indistinguishable from unset), T-55 (apply's flag default makes the config rung unreachable). |
| **Adding any join-id / date fixture** | `test/.memories/QUICK.md` Key Rules; `test/fixtures/apply-join-id/README.md`; `test/fixtures/apply-stale-id/README.md` | Join-id fixtures MUST use short `DD/MM` — a full date makes raw==normalized and **silently disables** the test. **s67 adds a second deliberately-wrong fixture:** `apply-stale-id` carries an id hashed from the PRE-canonical date, on purpose, and is the only fixture that can catch an id-recompute regression (`apply-basic` is self-consistent and stays green without it). Do not "fix" either. |
| **Calling any local model** | `[ref:local-model-conventions]`; llm-repo refs `patterns-code-design-index`, `patterns-refactoring-characterize-first`, `patterns-refactoring-duplicate-first`, `test-executable-spec`, `function-decomposition` | Verdict block required per judgeable call; backgrounded calls inject no template — grep `calls.jsonl` and append by hand. **s67: pass llm-repo conventions into prompts via `refs=[...]` + `refs_root=/mnt/i/workspaces/llm`** — server-side, no Claude token cost. **`my-go-qcoder` is 20.7 GB on a 12 GB GPU** → CPU-offloaded; scope asks tightly or they blow the timeout, and `warm_model` the classifier back before `-full`. **The task-notification payload is NOT the file** — one arrived HTML-escaped and shorter than what landed on disk. |
| **Refactoring code that lacks tests** | llm-repo `[ref:patterns-refactoring-characterize-first]` | Used for real in s67 slice 4: `canonicalDate`/`entriesWithCanonicalDates` had ZERO coverage, so five characterization tests were written and confirmed GREEN against the UNMODIFIED code (with the bug's test deliberately RED) before any production line changed. Tests written after a change encode what the new code does — circular, and worthless as a regression check. |
| **WS-D — retire bare-name fallback (T-09)** | `.claude/t32-agreement-gate-report.md` §7; `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries` | T-32 agreement gate LANDED, so novel rows are protected. Absolute precision UNMEASURED (649 = confidence-selected) → design **gate-to-review**, NOT silent insert. |
| **Changing the acceptance ENGINE (not the suite)** | `/mnt/i/workspaces/acceptance-harness` (master, v1.1.0); its `docs/PATTERNS.md` + `docs/ADOPTION.md` | Engine change = PR upstream + version bump here. Keep it ADDITIVE, seed new Context fields BEFORE Given, and prove compatibility via a `replace` directive before merging. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. Scope includes plain `batch`, `utils.ParseDate` (hardcoded 2025), `internal/parser` and `models.NewExpense` — all superseded by `internal/parse`. |
| **5.R2 follow-ups / T-23 / T-27 / T-28** | `.claude/scratch/replay-649/FINDINGS-5r2.md`; `.claude/plans/5r2-embedding-retrieval.md`; `.claude/t14-benchmark-report.md` | CL1 widen the fallback to weak/ambiguous keyword matches; CL2 pre-warm. T-27 undercut by T-30 — re-justify or drop. T-28 two-stage / two-pass grammar, measure on the T-14 sample. |
| Taxonomy `Apoia-se` durability | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | Rename `Apoia-se 4i20`→`Apoia-se` in the export source or a re-export wipes it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. Feeds T-42. |
<!-- /ref:session-reading-guide -->
