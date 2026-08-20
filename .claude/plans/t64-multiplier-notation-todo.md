# T-64 — multiplier installment notation (`405,25 x4`) — working todo

Session 73. Branch: TBD (`feat/t64-multiplier-notation`).
Companion docs: `.claude/tasks.md` (T-64), `.claude/t42-scout-report.md` § S4,
`expense-reporter/test/fixtures/batch-auto-failed-rows/README.md`.

---

## Locked scope decisions

### D1 — the two notations, and what they mean

| Token | Written number is | Result |
|---|---|---|
| `405,25/4` | the **TOTAL** → divide | 4 rows of 101,3125 |
| `405,25 x4` | **PER-INSTALLMENT** → multiply | 4 rows of 405,25 |

Same digits, **4× apart in the budget**, in opposite directions. This is the whole
reason T-64 is not a small parser tweak. Direction CONFIRMED by the user (s72, re-confirmed
s73).

### D2 — mode switch is the PRESENCE of `x`/`X`, not a positional grammar

The user's reframe (s73), adopted. Two decisions that had been welded together are split:

- **Mode** — "is this a multiplier?" → presence of `x`/`X`. Unambiguous, no grammar needed.
- **Split** — "which part is the value, which the count?" → a separate parsing sub-problem.

### D3 — the split is decided by trying all readings and requiring EXACTLY ONE

Split the field at the `x`, trim both sides, then test:

| | Reading | Valid when |
|---|---|---|
| **A** | `value x count` | left parses as currency AND right is an integer 1–60 |
| **B** | `value count x` | right is **empty**, and left splits **on whitespace** into currency + integer |
| **C** | `count x value` | left is an integer 1–60 AND right parses as currency |

- exactly one valid → use it
- **more than one valid → ERROR (ambiguous)**
- none valid → ERROR

**Why this and not a grammar.** A grammar answers "what does this mean?"; try-all-readings
answers "is there more than one thing this could mean?" — which is the actual question,
since the hazard is two readings 4× apart. Ambiguity is DETECTED, never resolved by fiat.

Rejecting is cheap here in a way it usually is not: T-63 built `failed.csv` precisely so a
row returns to the human with its reason on the same line. One hand-edit versus a silently
4×-wrong budget row is not a close call.

### D4 — worked examples (these become the unit table)

| Input | Readings valid | Expected |
|---|---|---|
| `405,25 x4` | A | 4 × 405,25 |
| `405,25x4` | A | 4 × 405,25 |
| `405,25 X4` | A | 4 × 405,25 |
| `646,25 4x` | B | 4 × 646,25 |
| `646,25 4X` | B | 4 × 646,25 |
| `1.234,56 x4` | A | 4 × 1.234,56 |
| `4x405,25` | C | 4 × 405,25 (retail form — free under D3) |
| `646,254x` | none (B needs whitespace) | **error** |
| `4x5` | A and C | **error: ambiguous** |
| `405,25 x0` / `x61` | — | **error** (count bounds, as today) |
| `405,25/4 x2` | — | **error** (both markers) |
| `405,25/4` | *no `x`* → divide path | 4 × 101,3125 (UNCHANGED) |

### D5 — safety property that makes this low-risk

**Every token containing `x` errors out today.** So no currently-accepted token changes
meaning. The existing 13-case `TestParseCurrencyWithInstallments` table is itself the
regression guard and **must pass unmodified**.

### D6 — the BR-thousands seam bug is folded into this task (user's call, s73)

Verified through the REAL producer→consumer seam, not by reading:

```
writeClassifiedCSV("Notebook;15/04;1.234,56") -> review.ReadQueue
  -> line 2: invalid value: invalid value format: 1.234,56
```

`ReadQueue` returns a HARD error, so one BR-thousands row anywhere in a month's CSV kills
the entire `review` step. Reachable, not theoretical: the value column deliberately keeps
the RAW token (s67, so the installment count survives) and `TestAdd_ReadsBrazilianThousandsAmount`
already pins `1.234,56` as legal CLI input.

Root cause: `normalizeThousands` lives in `internal/parse` only, so the two callers of
`ParseCurrencyWithInstallments` disagree about what a value token may look like. Same shape
as T-54 (one function, two callers, one private opinion).

### D7 — RESOLVED (advisor reconcile, s73): normalization stays in the boundary

Add **`parse.Value(valueStr)`**; `review.ReadQueue` calls it instead of `utils` directly.
Normalization stays in `internal/parse` (design **Q3**), `parse.go`'s existing call stays.

An earlier advisor pass had recommended moving `normalizeThousands` into `pkg/utils`; it
withdrew that on seeing Q3 and the "never call utils date/currency parsers" rule. Placement
was decided from the primary sources — **do not re-litigate it.**

`parse.Value` is symmetric with `parse.Date`, which exists for exactly this reason ("a
caller whose other fields are already structured"). `ReadQueue` is that caller for the value
field: its date is already canonical, only the value is still text. This fixes the ROOT (a
second parse site bypassing the boundary), not the symptom.

**Both notations stay together in `utils.ParseCurrencyWithInstallments`** — they are a
matched pair and must be read, and tested, side by side.

### D8 — `Fields` MUST delegate its value half to `Value`

Otherwise D7 relocates the two-normalization-sites problem instead of fixing it. `Value`
owns normalize → parse → wrap; `Fields` calls it and does not re-wrap.

**Verify mechanically, not by reading:** after the change,
`grep -n "normalizeThousands" internal/parse/*.go` must show **exactly one** call site.
Two means `Fields` did not delegate.

### D9 — error wrapping: `Value` wraps, `ReadQueue` drops its literal

`Value` returns the double-`%w` wrap (`ErrInvalidValue` + cause), per the `internal/parse`
sentinel rule — consumers must branch with `errors.Is`, never on English message text.
`ReadQueue` then drops its own `"invalid value: "` literal and keeps only the line frame:
`fmt.Errorf("line %d: %w", lineNumber, err)`.

**The resulting message is byte-identical to today's**, because `ErrInvalidValue` is
literally `errors.New("invalid value")`:

```
line 2: invalid value: invalid value format: 1.234,56
```

So no human-facing message changes, and `ReadQueue` gains
`errors.Is(err, parse.ErrInvalidValue)` for free. **Pin that exact string in the seam
test** — it is what a human reads off a broken close, and byte-identity here is a claim,
not an observation, until asserted.

---

## Ordered steps — ALL COMPLETE (s73)

- [x] **0. Branch** `feat/t64-multiplier-notation` off master.
- [x] **1. RED (unit)** — D4 table added to `TestParseCurrencyWithInstallments`, every error
      row asserting its REASON (W9). Result: **14 pass / 19 fail**, and the 14 were the
      original 13 plus the new `405,25/4` case — D5's "no accepted token changes meaning"
      confirmed empirically rather than argued.
- [x] **2. GREEN** — local model, two attempts. First: **verdict 0**, structural — its
      readings collapsed "not a currency/integer pair" into "pair with an out-of-range
      count", making the specific `must be positive` error unreachable. Second, with
      structural stubs: **verdict 1**, three mechanical fixes (stub header leaked into the
      file, `parseDivisorNotation` defined but never wired, `countRangeError` hardcoded
      `got 0`). Final table: **33 subtests green**.
- [x] **3. Seam tests** — thousands + multiplier round-trips through the REAL
      `writeClassifiedCSV` → REAL `review.ReadQueue`, plus reader-level cases and the D9
      characterization pin.
- [x] **4. Normalization placement (D7/D8/D9)** — `parse.Value` added, `Fields` delegates,
      `ReadQueue` routed through it. D8 grep: exactly ONE call site. D9 message verified
      byte-identical (pin was green before AND after).
- [x] **5. Verify** — `go test ./...` exit 0, `go vet` clean, `gofmt` clean;
      `./run-acceptance.sh` green; **`-full`: 72 PASS / 0 FAIL / 0 SKIP** after unloading
      the 8.4 GB resident qcoder from VRAM.
- [x] **6. `batch-auto-failed-rows` still rejects all 5 rows** — verified by RUNNING it. The
      scenario asserts all five lines verbatim, so the pass is real evidence and not a
      weaker check quietly succeeding. W2 held: both `x`-notation rows fail on FIELD COUNT
      before the value is parsed, so there was never a guard to unwind.
- [x] **7. Repair path smoke-checked** — `Anita Elô ADM;09/01;405,25 x4` parses to 4 × 405,25,
      and still does with a stale `# <reason>` comment attached, which is the route the user
      actually takes from `failed.csv`.
- [x] **8. Doc flips** — scout report § S4, fixture README (which carried a PREDICTION the run
      falsified — it said these rows would have to move out; they did not), `tasks.md` T-64
      → `[x]`, folder memories for `internal/parse`, `internal/review`, `expense-reporter`,
      and `.claude/index.md` (new file + both go-structure rows).
- [x] **9. Out-of-scope findings filed** — T-73 (`classify` bypasses the boundary) and T-74
      (dot-only token resolves 1000× wrong, pre-existing). T-72 filed AND fixed.

### Mutation evidence (the house rule)

Deleting the `normalizeThousands` call inside `parse.Value` — a mutation that still
COMPILES, per W6 — turned **exactly** the two thousands tests red while both multiplier
tests stayed green.

That discriminates two separate things. It proves `ReadQueue` genuinely routes through the
boundary (a test in `internal/review` cannot go red from a change inside `parse.Value`
unless it calls it). And it proves the **multiplier seam test is blind to that routing**,
because `utils` knows the notation either way — so the thousands case is the ONLY guard on
it, and must not be deleted as redundant.

---

## Watch-outs carried in

- **W1 — an `apply`-level acceptance fixture would prove NOTHING.** `apply` reads
  `reviewed.json`, where `value` is already a float and `installments` already an int — it
  never parses a value string. A fixture there tests `ExpandAndAppend` (which already works)
  and passes no matter what the parser does. Planned in s73 and dropped on advisor review.
  Real coverage = unit table + runtime seam test + `-full`.
- **W2 — `batch-auto-failed-rows` is expected to survive untouched.** Line 1 uses `- 1/4`
  (still a typo, and 4 fields); line 4 has `4x` but only **2 fields**, so it fails on field
  count before the value is ever parsed — its own README quotes that error. Nothing in the
  suite currently pins "`x4` is invalid", so there is no guard to unwind. Verify by running.
- **W3 — OUT OF SCOPE, file it:** `1.234 x4` parses as 1,234 — a 1000× error.
  **Pre-existing**: `normalizeThousands` strips dots only when a comma is ALSO present, so
  bare `1.234` already does this today. T-64 does not worsen it. Do not let it expand the diff.
- **W4 — OUT OF SCOPE, file it:** `classify` uses plain `utils.ParseCurrency`
  (`classify.go:46`), so it accepts neither thousands nor installments — inconsistent with
  every other entry point.
- **W5 — the float-drift trap does NOT apply to the multiplier.** s69 forced `90,00/3` over
  `99,90/3` in fixtures because `99,90/3` is `33.300000000000004`. The multiplier form
  performs **no division**, so `405,25 x4` is exact. Multiplier fixtures are safe.
- **W6 — a mutation that fails to COMPILE proves nothing** (s72). Mutate the SOURCE COLUMN
  or a value, never delete a struct field — the compiler going red is not the test going red.
- **W7 — neither January row is unblocked by the parser alone.** Line 19 is really
  `Anita;Elô ADM;09/01;405,25 - 1/4` — a stray semicolon INSIDE the item plus the typo; it
  needs a hand repair to `Anita Elô ADM;09/01;405,25 x4`. Line 45 has no date at all and does
  not reconcile (299,00 + 49,90 = 348,90 ≠ 646,25) — a data decision for the user.
- **W8 — tests build values via the real parser, never struct literals** (`internal/parse`
  QUICK). A hand-set field tests the literal rather than the ladder.
- **W9 — an error for the RIGHT reason and an error for the WRONG reason look identical.**
  D4 has six expected-error rows, and `wantErr: true` alone cannot tell them apart. Each must
  assert on the reason, or `646,254x` could start erroring on a count bound rather than the
  D3 ambiguity, and the table would stay green while the rule it documents had quietly
  stopped holding. The existing table already has an `errContains` field — use it on every
  new error row.
- **W10 — reading C widens what `ReadQueue` accepts.** A `4x405,25` token that today hard-errors
  (killing the whole review step) will parse after this change. That is the intent for real
  multiplier rows; the check is that nothing currently erroring is now accepted with a
  meaning a human would read differently. Two-bare-integer tokens (`4x2`, `2x10`, `10x2`)
  stay errors under D3 — ambiguous — so the net failure surface only shrinks where intended.
