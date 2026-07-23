# Parse Boundary — Structured Input, Parse-Once

**Date:** 2026-07-20 (session 62 discussion); design session 2026-07-22 (session 63)
**Status:** FINAL — all §10 questions decided in the session-63 design session.
Implementation not started (awaiting explicit user go).
**Origin:** User direction after reading the grand vision: chat-era input (Telegram/
WhatsApp) does not need one monolithic "receive string → do everything" call. A
dedicated parse step converts input into structured data; every other tool takes
fields, not strings. "Input as single string seems to have generated its own problems."
**Sequencing (user):** parse boundary FIRST, T-21 falls out of it.

---

## 1. Why — the string interface is the common ancestor of the data debt

Evidence that the semicolon-string input format has been generating bugs:

| Symptom | How the string format caused it |
|---------|--------------------------------|
| T-35 join-id divergence | Raw strings travel deep; each command re-parses at its own depth → one log got raw `15/04`, the other normalized `15/04/2026` → two ids for one expense |
| T-18 orphaned seed ids | Same class: fixture ids hashed from raw dates that runtime normalized |
| Six date parsers (see `[ref:date-year-semantics]`) | Every entry point re-parses because no shared boundary exists; six different missing-year rules |
| T-21 installment under-recording | `99,90/3` smuggles the count *inside the value field*; `review.ReadQueue` had to re-discover structure and dropped it (`queue.go:63`) |
| `GenerateID` format-sensitivity (t35 survey §5.4) | Identity = sha256 of *unparsed string bytes*, so identity breaks whenever formatting varies |

The T-35 fix ("canonicalize once at the boundary") was this idea applied per-command,
three times. The parse boundary is the generalization: **one boundary, owned in one
place, everything downstream structured.**

## 2. Goals / non-goals

**Goals**
- One string→struct conversion in the codebase; commands and tools consume the struct.
- Year resolution happens ONCE, at the boundary, under an explicit policy (§4).
- Installment count becomes a first-class field (unblocks T-21 by construction).
- Chat-era shape: a parse step whose result can be echoed back for user confirmation
  *before* classify/insert (matches the Telegram UX in the vision — confirm, then act).
- Semicolon CLI form survives as a terminal convenience calling the same parse.

**Non-goals (this effort)**
- Free-form natural-language parsing ("paguei 35,50 no uber ontem") — that is the
  chat layer's job (Layer 6), possibly with its own LLM; the boundary stays
  **deterministic Go** so parse errors are crisp and testable, and the local-model
  budget stays on classification.
- `GenerateID` migration (§6) — door left open, not walked through.
- Retiring plain `batch` (WS-E owns that; see §7 T-38).

## 3. The contract (DECIDED session 63)

A canonical parsed-expense value, produced only by the boundary:

```go
type ParsedExpense struct {
    Item         string    // trimmed, original casing
    Date         time.Time // the SINGLE stored date truth (year resolved at parse time)
    Value        float64   // per-installment value, BR decimal already parsed
    Installments int       // 1 when no /N notation
    RawValue     string    // original value token (e.g. "99,90/3") for display/audit
}

// DateString returns the canonical DD/MM/YYYY form — the identity bytes consumed
// by GenerateID and both JSONL logs. The SOLE formatting site for expense
// identity in the codebase.
func (pe ParsedExpense) DateString() string { return utils.FormatDate(pe.Date) }
```

**Home (decided): new `internal/parse` package.** Not `internal/models` (leaf
package, imports nothing, and `NewExpense` is on the WS-E death list); not a
rehabilitated `internal/parser` (its only callers are the dying plain-`batch`
chain, and its output type carries a subcategory — classification is not parsing;
keeping old/new grep-separable makes the WS-E deletion wholesale, not surgical).

**Date shape (decided): store `time.Time`, derive the string via method** (user
proposal, session 63 — supersedes the earlier "both fields" lean). Rationale
ladder: string-only keeps re-parse sites (wrong-helper class) reachable
downstream; a parse-on-demand method can fail at a distance and throws away the
`time.Time` the constructor already had; two stored fields hold "the forms agree"
only by convention. One stored field + a formatting method makes an inconsistent
pair **unrepresentable** — formatting cannot fail, and the identity bytes become
the output of one named method instead of N write-site calls. Zero value formats
as `01/01/0001` — garishly wrong rather than subtly wrong, as desired.

**API shape (decided): field-wise core + semicolon wrapper.** The semicolon
string only exists at `add`/`correct` (`auto` = 3 args, `batch-auto` = CSV
columns, `apply` = JSON fields), so the boundary is:
- core: `(item, dateStr, valueStr, opts) → (ParsedExpense, error)` — used by all
  commands; year policy (§4) lives here;
- wrapper: 4-field semicolon form → `(ParsedExpense, subcategory, error)` —
  subcategory returned alongside, NOT a struct field (classification ≠ parsing).

**Construction discipline:** exported fields, constructor-only by convention
(struct is `internal/`, so external construction is impossible anyway); tests
build values via the real parser on real inputs, not literals — same discipline
as the T-35 short-`DD/MM` fixtures. The single-stored-truth shape makes drift
structurally impossible; `expect.JoinIDMatchesAcrossLogs` remains the net.

## 4. Year resolution policy (the T-37 wiring)

Precedence, highest wins — **decided in discussion (user):**

1. **Explicit year in the input string** (`15/04/2026`) — always wins; a supplied year
   is a fallback for absence, never an override (T-35 rule, already test-pinned).
2. **Argument** (`--year` flag / tool-call parameter) — per-invocation override.
3. **`config.json` `date_year`** — the configured default (currently DEAD, T-37;
   this is where it comes alive).
4. **Most recent non-future occurrence** (decided session 62) — when nothing above
   applies, a bare `DD/MM` that would land in the FUTURE of the current year resolves
   to LAST year; otherwise current year. Entering `15/12` in January 2027 → 2026 with
   zero config; expense entry is almost always retrospective. Genuinely future-dated
   entries (scheduled payments) must carry an explicit year — which wins anyway (rung 1).
   Replaces blind `time.Now().Year()` — a deliberate behavior change from today.
   *Grace window (DECIDED session 63): **0 days**.* The user never enters
   future-dated transactions, so any bare `DD/MM` landing in the future resolves
   to last year, no exceptions; genuinely future-dated entries must carry an
   explicit year (rung 1 wins anyway). The rule's tests pin both sides of the
   today/tomorrow edge.

**`--year` surface (decided session 62): uniform.** `auto`/`batch-auto`/`add` gain the
flag so precedence rung 2 exists on every date-accepting command (T-16 parity lesson;
today only `apply`/`generate-workbook` have it). Nearly free once the boundary exists.

This closes the "December batch entered in January lands in the wrong year" hole
(batch-auto --resume help currently just warns about it) at the only correct layer:
set `date_year: 2026` while closing 2026, and every bare `DD/MM` resolves right
regardless of when it's entered.

## 5. Migration map (repointing the six parsers)

Current parser → fate under the boundary:

| Parser | Fate |
|--------|------|
| `utils.ParseDateFlexible` | Becomes an internal helper OF the boundary (or absorbed); commands stop calling it directly |
| `cmd.parseExpenseForFeedback` (add.go) | Replaced by the boundary |
| `cmd.canonicalDate` (apply.go, T-35) | Replaced by the boundary (it was the prototype) |
| `utils.ParseDateWithYear` | Absorbed/retired with the above |
| `utils.ParseDate` (hardcoded 2025) | NOT migrated — dies with plain `batch` under WS-E (T-38 resolution, §7) |
| `taxonomy.parseDate` (year=0 sentinel) | UNTOUCHED — generator-side, deliberately refuses to guess; correct as-is (t35 survey §4) |

**Migration order DECIDED (session 63): `add`/`correct` → `auto` → `batch-auto` →
`apply`, one PR per slice**, with the T-35 join-id guards + acceptance suite as the
net (deterministic group ~8s, `-full` ~2min — post-T-39 iteration is cheap). Each
slice manufactures what the next needs: slice 1 founds the package from proven code,
slice 2 proves the field-wise core against the join-id guards, slice 3 uses the
proven core for the highest-multiplicity rewrite, slice 4 touches the cross-language
schema only after the Go side is stable.

Per-slice mechanics (call-site survey, session 63):
1. **`add`/`correct`** — LIFT `parseExpenseForFeedback` (add.go:168; it already does
   split → `ParseDateFlexible` → `FormatDate` → `ParseCurrencyWithInstallments`)
   into `internal/parse` as the founding code; both commands repoint. Year policy
   §4 goes live here (rungs 2–4 are new behavior — test-pin both sides).
2. **`auto`** — replace the two separate parse calls (auto.go:51, :58) with the
   field-wise core. Net: existing `expect.JoinIDMatchesAcrossLogs` + short-`DD/MM`
   fixtures. **Also extract here (deferred from slice 1 per the
   extract-keep-divergence rule — no seam below 3 callers):** a cmd-level
   `parseOptions(yearFlag int, cfg *config.Config) parse.Options` helper — `add`
   and `correct` hand-build identical `parse.Options{...}` literals today, and
   `auto` is the third copy. Cmd-level, NOT in `parse` (the boundary package must
   not import `config`).
3. **`batch-auto`** — the real dedup win: SIX re-parse sites (batch_auto.go:360/364,
   376/385, 448; batch_auto_resume.go:87/92) collapse into **parse-once-per-CSV-row
   at read time**; the row struct carries `ParsedExpense`, downstream consumes
   fields. `PredictEntryIDs` and append read the same parsed struct — the advisor's
   "prediction from raw strings mismatches" bug class becomes unrepresentable.
   **T-40 lands here:** the batch-auto join-id unit seam becomes a boundary test —
   one short-`DD/MM` input through parse → both log writers → one id (no Ollama).
4. **`apply`** — retires `canonicalDate` (apply.go:250, the T-35 prototype absorbed
   by what it prototyped). Deliberately last: `apply`'s input crosses the
   `reviewed.json` schema (HTML export JS), which T-21 is about to change anyway —
   sequencing apply-then-T-21 touches that schema once, with the boundary in place.

## 6. Identity (`GenerateID`) — door open, not walked through

Today: `sha256(lower(trim(item)) + "|" + date-as-raw-string + "|" + %.2f)[:12]`.
With the boundary, every writer hashes the SAME canonical `DD/MM/YYYY` string, so the
T-35 bug class is structurally gone **without changing any existing id**.
Hashing canonical *fields* (e.g. RFC-3339 date) would be format-proof but rewrites
every id in both logs = a migration with a backfill script. DEFERRED; the boundary
should simply route all id generation through one call site so the migration, if ever
taken, is one function + one backfill.

## 7. Coupled backlog items — how this plan resolves them

- **T-37 (dead `date_year`)** → wired as precedence step 3 (§4). Resolved by this work.
- **T-21 (reviewed installments)** → `Installments`/`RawValue` are boundary fields;
  threading them through `review.ReadQueue` → review.html export → `reviewed.json` →
  `apply.ReviewedEntry` is the first consumer slice of the boundary. **UX decided
  (session 62): 1 row + ×N badge** — confirm once, `apply` expands to N log entries;
  matches the statement view, keeps the review pass fast (load-bearing per the vision),
  and mirrors batch-auto's auto path where expansion is a write-time concern.
- **T-38 (`ParseDate` hardcodes 2025)** → RESOLVED BY DECISION (session 62): plain
  `batch` is a pre-pivot artifact (writes the workbook directly; the pivot made
  `generate-workbook` the only writer; no vision path uses it). Do NOT repair its
  parser — delete `ParseDate` together with its caller under WS-E.
- **T-40 (batch-auto join-id coverage)** → becomes a boundary unit test (§5).
- **T-35 bug class** → structurally impossible once all writers consume `ParsedExpense`.

## 8. Surface (chat-era exposure) — DECIDED session 63: internal-first (a)

T-41 ships only Go: `ParsedExpense` + boundary functions, CLI commands repointed.
No `parse_expense` MCP tool now — `mcp-server/` stays as-is; its existing tools
benefit anyway once the commands they shell out to hit the boundary internally.
Deferred, not lost: the vision's parse → echo-back → confirm → act UX needs the
tool only when Layer 6 (chat) starts; at that point it is a thin JSON wrapper over
an already-tested boundary, shaped by a real consumer (the T-15 slice-2 lesson —
no speculative API surface). Bet being made: no real MCP consumer needs
parse-as-a-step before Layer 6; if wrong, pull the tool forward — nothing blocks it.
The confirm-before-commit UX only requires the parse result be *serializable and
echoable*, which the contract gives (note: `Date time.Time` marshals as RFC 3339 —
the wrapper, when built, should emit `DateString()` for display).

## 9. Context: the milestone this serves

Session-62 framing: the project is **pre-first-use**; the organizing milestone is the
**first real monthly close on 2026 data** (`batch-auto → review → apply →
generate-workbook --year 2026` on an actual bank CSV). Pre-work order:
parse boundary (this plan, absorbing T-37/T-40, resolving T-38) → T-21 threading
(first consumer) → T-03 year-rollover check → promote all-years log (user decision).
Everything else surfaced by the dogfood run becomes the real backlog.

## 10. Design-session record — ALL DECIDED (session 63, 2026-07-22)

1. **Package home + struct shape (§3):** new `internal/parse`; single stored
   `Date time.Time` + `DateString()` method (user proposal — inconsistent date
   pair unrepresentable; formatting cannot fail; one named identity-bytes site).
   API = field-wise core + semicolon wrapper returning subcategory alongside.
2. **Migration order (§5):** `add`/`correct` → `auto` → `batch-auto` → `apply`,
   one PR per slice; T-40 in slice 3; T-21 immediately after slice 4.
3. **Currency ownership:** YES — the boundary owns ALL input normalization
   (dates, BR decimals, installments). Anything else recreates the six-parsers
   problem for a second field class.
4. **Surface timing (§8):** internal-first; MCP `parse_expense` deferred to Layer 6.

Decided session 62 (context): year fallback = most recent non-future (§4 rung 4);
grace window = 0 (locked session 63); T-21 UX = 1 row + ×N badge (§7); `--year`
flags = uniform (§4).
