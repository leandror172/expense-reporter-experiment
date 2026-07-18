## 2026-07-17 - Session 60: T2 closeout (harness v1.0.0) + red-test repair (suite fully green) + T-20 `batch-auto --resume` (PR #49)

### Context

Started right after the user merged expenses PR #48 (T2 Session B). Next-steps discussion locked a three-phase plan: T2 closeout → red-test repair → T-20 dedup, then full wrap-up (reports, docs, PR, handoff) on user instruction.

### What Was Done

- chore(t2): bump acceptance-harness to v1.0.0 (master) — after verifying module PR #1 + career-search PR #7 were merged and tagging `acceptance-harness` v1.0.0 at `11beb2a` (checked first that the "unused surface" consumer-report signals are all used by career-search — not dead API; the `FileUnchanged` expressiveness gap is additive, so no 1.0 blocker)
- fix(test): repair the 5 red acceptance tests (T-34 + T-12) (master) — Uber Centro→Posto Ipiranga CLI-arg swap; 3 dark tests de-workbooked + given taxonomy config + full-date expectations; full suite 49 pass / 0 fail (1311s), first fully-green roster since T-32
- T-20 `batch-auto --resume` + always-on duplicate-append warning — designed (Option B locked with user), advisor-reviewed (Opus xhigh subagent; blocker + 2 cuts adopted), implemented by an `impl-opus-xhigh` subagent (TDD; my-go-qcoder 4×verdict-2, 2×verdict-1), orchestrator-verified (independent gates + full suite green, 779s). Branch `feat/t20-batch-auto-resume`, **PR #49 open**
- Reports written + indexed: `.claude/t2-closeout-report.md`, `.claude/t34-t12-red-test-repair-report.md`, `.claude/t20-resume-implementation-report.md` (carries `ref:batch-auto-resume` semantics block), `.claude/advisor-t20-resume-dedup.md`
- Docs/memories updated (on the PR branch): README batch-auto `--resume` section, repo+expense-reporter QUICK status, internal/feedback + test KNOWLEDGE sections, index.md registrations incl. a previously-missing `internal/appender` package row
- New agent config `.claude/agents/impl-opus-xhigh.md` (Opus xhigh implementation subagent; advisor rule made availability-conditional); user reloaded plugins mid-session to activate it

### Decisions Made

- T-20 design = Option B: explicit `--resume` + always-on stderr duplicate warning; the log NEVER dedups silently (identical (item,date,value) triples can be legitimate distinct expenses — two same-day bus fares). Count-consumption ledger, not an ID set.
- Advisor adjustments adopted: `PredictEntryIDs` must share `expandEntries` with `ExpandAndAppend` (blocker — raw-string prediction would mismatch every DD/MM/installment id); entry-level partial-series auto-complete CUT (divergent re-classification would split a series across categories) → partial series route to review; single shared ledger invariant.
- Branch scope split (user): test repairs committed to master; PR #49 is pure T-20.
- Fable sessions have no `advisor()` — substitute is an Opus subagent at xhigh effort (memory updated); `impl-opus-xhigh` agent created for the same reason on the implementation side.

### Next

- Merge **PR #49** (T-20). Then product: **WS-D (T-09)** — still HELD, design gate-to-review not silent insert; or WS-E after.
- Pending amend: new backlog candidates T-35 (auto's raw-vs-normalized date → divergent GenerateID join key across the two logs) and T-36 (cosmetic sweep: stale `HighConfidence*` test names, `auto-basic/input.csv` ghost reference, README stale `rollover.csv` line, `internal/batch` gofmt).
- Standing queue: T-24 (--think default), T-25 (q35 grammar bug upstream), T-27/T-28, Apoia-se rename, promote all-years log.

### Gotchas

- `rtk git log`/`fetch` served stale cached output for the acceptance-harness repo — `rev-parse` (via `rtk proxy`) disagreed and was right. Verify with plumbing commands when rtk-filtered git output looks impossible.
- Workflow-tool `args` arrive as a JSON STRING in the script (not an object) — `args.prompt` was undefined and the advisor subagent got an empty task. Embed prompts as consts in the script body.
- Full-suite runtime jumped 813s→1311s because the 5 repaired tests now do real classification instead of failing fast (think-on q3); still inside run-acceptance.sh's 1800s timeout but T-08's timeout-flake class gets closer with every added Ollama test.
- `go test ./...` caching hid nothing this time, but `gofmt -l` still flags 5 pre-existing `internal/batch/` files on master (same CRLF-era class session 59 cleaned in `test/`).
