# Parse Boundary — Structured Input, Parse-Once (design draft)

**Date:** 2026-07-20 (session 62 discussion; drafted before design session)
**Status:** DRAFT — decisions marked OPEN are not locked; this captures the discussion
so no angle is lost. Design session to follow.
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

## 3. The contract (draft)

A canonical parsed-expense value, produced only by the boundary:

```go
type ParsedExpense struct {
    Item         string  // trimmed, original casing
    Date         string  // ALWAYS canonical DD/MM/YYYY (year resolved at parse time)
    Value        float64 // per-installment value, BR decimal already parsed
    Installments int     // 1 when no /N notation
    RawValue     string  // original value token (e.g. "99,90/3") for display/audit
}
```

OPEN: exact home — a new `internal/parse` package (leading candidate: it will import
`utils` date/currency helpers and `models` imports nothing) vs. `internal/models`.
OPEN: whether `Date` should also carry the parsed `time.Time` alongside the canonical
string (two fields, one truth) or string-only with parse-on-demand.

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
   *Design-session detail:* consider a small grace window (future within ~7 days →
   still current year) so a card transaction posted with tomorrow's date, entered
   today, doesn't flip to last year. Pick the window in the design session; the
   rule's tests should pin both sides of it.

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

Migration order OPEN, but the constraint is: repoint one command at a time with the
T-35 join-id guards + acceptance suite as the net (deterministic group now runs in
~8s — post-T-39 this iteration is cheap). Candidate order: `add`/`correct` (already
normalized, lowest risk) → `auto` → `batch-auto` → `apply`.

**T-40 lands here:** the batch-auto join-id unit seam becomes a boundary test —
one short-`DD/MM` input through parse → both log writers → one id.

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

## 8. Surface (chat-era exposure) — OPEN

Options for exposing parse as a step:
- (a) Internal-only for now; MCP `parse_expense` tool added when Layer 6 starts.
- (b) Add the MCP tool now (mcp-server) so the chat layer's shape is proven early;
  CLI stays string-convenience.
Lean: (a)-then-(b) — build the internal boundary first; the MCP tool is a thin JSON
wrapper over it whenever Layer 6 needs it. The vision's confirm-before-commit UX only
requires that the parse result be *serializable and echoable*, which the contract gives.

## 9. Context: the milestone this serves

Session-62 framing: the project is **pre-first-use**; the organizing milestone is the
**first real monthly close on 2026 data** (`batch-auto → review → apply →
generate-workbook --year 2026` on an actual bank CSV). Pre-work order:
parse boundary (this plan, absorbing T-37/T-40, resolving T-38) → T-21 threading
(first consumer) → T-03 year-rollover check → promote all-years log (user decision).
Everything else surfaced by the dogfood run becomes the real backlog.

## 10. Open questions for the design session

1. Package home + struct shape (§3) — `internal/parse` vs `models`; `time.Time` field?
2. Migration order + per-command repoint mechanics (§5).
3. Does the boundary also own **currency** parsing policy (it already parses BR
   decimals via the value field) — i.e., is `ParsedExpense` the home for ALL input
   normalization, not just dates?
4. Surface timing (§8): internal-first confirmed, or MCP tool now?

Decided session 62 (moved out of this list): year fallback = most recent non-future
(§4 rung 4); T-21 UX = 1 row + ×N badge (§7); `--year` flags = uniform (§4).
