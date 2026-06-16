# Advisor — Phase B builder (pre-implementation review), 2026-06-10

Context: about to write scratch-builder changes (entries, typed values, labels/i18n,
per-group percent rows) and converge against `template-data.xlsx`. Advisor = Opus 4.8.

---

Before you write any builder code, there's a structural conflict in your inputs that your plan hasn't caught. It's the thing that will eat the most cycles if you proceed. Lead with it.

## 1. BLOCKING: template-data's block sizing contradicts spec §3.2

Spec §3.2: `data rows = max-entries-per-month + headroomRows (default 3); zero entries → 3 rows`.

But the extraction shows **every block is exactly 3 rows regardless of fill count**:
- Diarista: 3 entries/month, rows 3–5 (3 data rows) — spec would predict 3 + 3 = **6**.
- Netflix: 1 entry/month, rows 14–16 (3 data rows) — spec would predict 1 + 3 = 4.

Template-data has the user typing entries *into* the Phase-A headroom rows, so blocks stay fixed at 3. That is **not** the spec rule. This matters because it determines every downstream row number and the entire diff. Two ways this bites:
- If you implement spec §3.2 (`max-entries + headroom`), Diarista becomes a 6-row block and the diff explodes against template-data's 3-row block.
- If you "reproduce template-data exactly," you silently bake in `block = headroom, entries overwrite it` — which is a *different rule* than the spec, and it's almost certainly wrong for the real generator (a month with 3 entries has zero spare room — the exact pain headroom exists to prevent).

Note spec §7 Q1 **already flags headroom sizing as an open Phase-B question**. So this isn't settled — and template-data appears to encode an unintended reading of it.

**This is a user decision, not one to absorb by "matching the target."** Take it back: either (a) Phase B consciously scopes to fixed-3 blocks, and you annotate the spec that §3.2's `max-entries + headroom` is *deferred/unvalidated* and the real generator (G2) must implement it — don't let the next session port the scratch shortcut as the rule; or (b) refill template-data with entries-plus-spare so it actually exercises the spec rule. (a) is the cheaper, sensible Phase-B scope — but only if it's deliberate and recorded.

## 2. Do NOT reconstruct entries from the `base + k` pattern — extract the literals

Your clean-diff strategy ("any residual is a real bug") depends entirely on the entry literals matching byte-for-byte. The values feed SUM/pull/percent cells, but those are **formulas** — diff.py compares formula strings, not computed values (convergence report note 5). So the *only* convergence-sensitive values are the entry literals in the data cells — and they must be exact.

That makes `base + k` the wrong tool: it's an *inference*, it's already lossy (the Receitas Salário anomaly broke it, and the subagent was vague — "tiny sequential values"), and a single wrong guess produces residuals that *look* structural. Have the extraction subagent emit a **Go entry table with the exact literal values + exact dates straight from the cells** (it's already read them, they're in the report file). Pattern-encoding re-introduces the transcription risk you spawned a subagent to avoid.

## 3. Two foreseeable traps in the convergence step (plan for them now)

- **openpyxl `insert_rows` does not shift formula references.** When the subagent adds per-group rows to template-data, every saldo formula below (`=D24`, `=D9`, …) will keep pointing at the old cells unless it rebuilds them. Don't insert-and-pray — have it rewrite the affected Listas region with corrected refs, or regenerate that sheet's formulas wholesale.
- **You're changing too much in one diff.** Labels refactor + normalized strings + entries + typed values + per-group rows, on *both* sides, in one shot. If the diff comes back messy you won't know which change caused which residual. Stage it: get entries + typed values converging first (against template-data's current labels), then layer normalization, then per-group rows. Each stage = an interpretable diff.

## 4. R6/R7 as residuals — agreed, no change

Keeping the em-dash and Extras dup-labels as justified residuals is consistent with the standing policy ("don't fix the builder toward golden-master hand-edit artifacts"). Just make sure diff.py's residual allowlist names them so they don't read as regressions.

---

**Bottom line:** #1 blocks — resolve the headroom semantics with the user before you write a line, because it sets the structure. #2 is a free correctness win — swap pattern-inference for extracted literals. #3 will save you a thrash in the convergence step. Your R6/R7 calls are right.

One caution on momentum: this session has been heavy on tooling/process (memory, model-routing, label-naming). All defensible, but the actual builder work hasn't started and the highest-risk decision (sizing) was about to be made implicitly. Get #1 answered, then go.
