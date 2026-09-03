# T-75 — Telegram export → batch-auto CSV converter

**Status: BUILT, session 75 (2026-09-03). Steps 1–5 DONE on `feat/t75-telegram-converter`;
step 6 — the real close — is the only one left and needs the 2026 export from the user.**
Prerequisite DONE: `.claude/t75-oracle-viability.md` [ref:t75-oracle] — the 2025 export
validated as an oracle (90.3% A-coverage, 37/37 misses explained, perturbation-checked).

**Why this is the highest-leverage item:** everything downstream (`batch-auto` → `review`
→ `apply` → `generate-workbook`) is built, tested and measured, but a month costs whatever
the hand-extraction costs — which is why exactly ONE month has ever been closed. Expenses
are typed into a private Telegram group ("Gastos") and exported by hand as
`ChatExport_<date>/result.json`. Nothing in the repo reads that file.

**Measured on the export already on disk (393 messages, 2025-05..12):** 352 conform to the
`item; date; value` shape `batch-auto` already eats — **89.6% of all messages, 94.4% of
text messages.** The humans have been typing the CLI's own format for eight months without
being asked to. This is a strict parser plus six named defect classes, not an NLP problem.

---

## Relationship to the vision — settled, session 75

Re-read `docs/expense-classifier-vision.md` because the converter looked like it might be a
detour around the Phase 4 queue. It is not, and the reasoning is worth keeping.

**The vision's own offline design IS this CSV+batch path**, verbatim: *"Since we have a
list of expenses, instead of directly sending each after being classified, they can be
stored in CSV, and sent using the batch functionality"*, and Phase 4's *"Batch
accumulation: collect classified expenses in CSV, batch-insert periodically."* The
converter is the offline-backlog half of Phase 4 built first, not an alternative to it.

**USER RULING (s75): the per-expense conversational flow remains the END GOAL** (subject to
change). `review.html` is the surface for bulk work and for what is already reported — the
two are different entry points onto one pipeline, not competitors. **The bot does not need
pre-join history; the export route covers that.**

**Two Telegram API facts that shape this, both verified rather than assumed:**

1. A **bot cannot read messages sent before it joined** a chat. Permanent architectural
   restriction, not a retention setting. So a listener built today would not process the
   Feb–Aug 2026 backlog at all.
2. Undelivered updates are retained **at most 24 HOURS**. A bot offline for less than a day
   drains its backlog on reconnect exactly as the vision describes; **a bot offline longer
   loses those messages permanently**, with no error and no retry.

**Consequence, and the reason this tool is not scaffolding:** on a personal desktop that is
off overnight, for a weekend, or during a trip, fact 2 fires routinely. **The export route
is therefore the PERMANENT recovery path for any outage over 24 hours**, not a one-time
catch-up to be deleted when the bot ships. Exposure shrinks if the bot ever runs somewhere
always-on (Phase 5 contemplates Docker) or if capture moves to an MTProto user client,
which reads real history and has no 24h window — neither is needed now, and neither removes
the parser built here.

### The shape, end to end

```
  Telegram group "Gastos"
        |
        +-- live, bot online -----------> [Phase 4 bot] --+
        |                                                 |
        +-- history / >24h outage --> [export] --> [converter] --+
                                                                 |
                                              +------------------+
                                              v
                              batch-auto --> classified.csv
                                              |
                        +---------------------+------------------------+
                        v                                              v
              review.html (bulk, backfill)          inline keyboard (per-expense, Phase 4)
                        +---------------------+------------------------+
                                              v
                                 apply --> logs --> generate-workbook
```

**Both capture routes converge on the same CSV, and both confirm surfaces converge on
`apply`.** The converter and the bot are the two halves of CAPTURE, not alternatives; the
review page and the inline keyboard are two surfaces onto CONFIRM. Reading them as
competitors is what made the converter look like a detour.

### What this requires of the design

**Structure the tool as `source adapter → message stream → parse/repair → sink`.** The
`result.json` reader is one adapter (~30 lines); a live-update or MTProto source is
another. **This is a CORRECTNESS property, not tidiness:** the same message typed on a good
day and recovered from an export on a bad one must be interpreted identically, which only
holds if both routes share one parser. It also makes the MTProto/bot question a later,
cheap decision instead of a rewrite.

Note the rejects mechanism has two audiences on the two routes: the converter writes a bad
line to a file for the human to edit, while the bot can **reply to the message in the chat
with the reason** — the same rule delivered at the earliest possible moment to repair,
which is what T-63 reaches for from the other end.

**Out of scope here, and NOT decided:** whether the confirm step becomes an inline keyboard
(Phase 4) or stays in `review.html` for bulk. Both consume the same `classified.csv`.

---

## Contract

**Language: Go. Home: a SUBCOMMAND of the existing binary** (decided s75 — rationale below).

```
expense-reporter telegram-import <result.json> [--out-dir DIR] [--dry-run] [--force]
```

No model, no network, no workbook. Deterministic and idempotent.

- **`--dry-run`** — bucket report only, writes nothing. This IS the committed bucket
  classifier; one command with two modes rather than two tools that drift.
- **default** — writes `expenses-YYYY-MM.csv` per month plus `telegram-rejects.csv` to
  **`--out-dir`, which is REQUIRED when writing** (D9, s75 — an earlier draft defaulted
  to `/mnt/i/workspaces/expenses/`; the destination is now always the human's explicit
  choice). In practice that is `/mnt/i/workspaces/expenses/`, **outside the git repo**,
  where `expenses-2026-01.csv` already lives, so real expense data needs no gitignore
  rule — it is not in the tree at all.

Output line shape is `item;DD/MM/YYYY;value` — three fields, matching
`batch_auto.go:537`'s `strings.SplitN(line, ";", 3)`.

### Why Go, and why not Python

**The decisive fact, verified not assumed:** `internal/parse` already exports exactly the
two predicates the exactly-one-reading rule needs —

```go
func Date(dateStr string, opts Options) (time.Time, YearSource, error)
func Value(valueStr string) (float64, int, error)
```

— and its imports are clean (`errors, fmt, strings, time, pkg/utils`; **no `config`**), so a
new command can call them directly. D4's rule *is* "for each candidate: does `Date` accept
field 2 and `Value` accept field 3?", which in Go is two calls to the PRODUCTION boundary.

**In Python every one of those becomes a port** — `currency.go`'s three readings,
`normalizeThousands`, the year ladder. One concept, two implementations, kept in step by
nobody: the T-54 / T-72 / T-73 shape this repo has been bitten by three times. It also
breaks the correctness property this plan is built on — **the same message arriving via
export or via the bot must be interpreted identically** — which Go gives by construction and
Python gives only by vigilance. Concretely: when T-79 teaches the parser `*`, a Go converter
gains it for free while a Python one silently keeps rejecting `89,90*3`.

Subcommand rather than a second binary (`cmd/workbook-inspect/` is the precedent for the
latter) because this is Phase 4's offline half — a live feature, not an inspection tool.

**The `.claude/tools/` Python convention does not apply**: those four are one-off analysis
and maintenance scripts. **`t75_oracle_probe.py` STAYS Python**, and that is now a feature —
it carries its own independent port of the parser, so at step 4 it cross-checks the Go
implementation instead of sharing its bugs. It is the only thing in the validation path not
built on the Go parser's assumptions.

**MTProto (route B) being Python-first does not flip this.** That client is a *source
adapter*: it fetches messages and hands records to the Go validator. The parser stays in one
place, which is the whole point of the adapter split.

## Decisions

### D1 — Emit the RESOLVED full date, and own the resolution explicitly

The converter holds year evidence nothing downstream has: the message's own timestamp. But
"call the boundary" does not by itself say whether that evidence *overrides* a year the
human typed. It must not. **Resolution order, and it deliberately mirrors the T-41 ladder:**

1. **The date field already carries an explicit year** (`DD/MM/YYYY`) → use it verbatim.
   The human outranks the timestamp, exactly as ladder rung 1 outranks rung 2.
2. **Otherwise** → resolve by proximity to the message timestamp (window below) and format
   `DD/MM/YYYY`.
3. **Either way, validate by calling `parse.Date(resolved, parse.Options{})`.**

**This is option (a): the converter OWNS resolution and the boundary acts as VALIDATOR.**
The rejected alternative (b) was to pass the message year as `Options.Year` and let the
ladder resolve. It reuses more code and is WRONG at the year boundary: rung 2 fires
unconditionally for a bare date, so a message sent 02/01/2026 reporting `31/12` becomes
**31/12/2026** — and because that is not beyond the *current* year,
`validateYearNotBeyondCurrent` passes it and the row is **silently wrong**. Precisely the
failure D1 exists to prevent. Pre-adjusting the year to avoid it is doing (a)'s work anyway.

`Options{}` is correct at step 3 because the string carries a year by then, so no rung
fires — while both resolved-year validations still run.

This does not violate design Q3 ("the boundary owns ALL input normalization"): the year
comes from message METADATA the boundary never sees, and the value token still passes
through `parse.Value` untouched.

**Window = `[-180, +7]` days, derived from the data.** Measured over all 354 date-bearing
messages in the 2025 export: **min −131 days, max +3, and 279 of 354 on the message's own
day.** The past side must be generous — people report a purchase months later — while the
future side has no support beyond +3 (+7 leaves headroom for a scheduled bill, the
`Unimed vencimento` shape). **The load-bearing invariant is that the window is narrower than
365 days** (187 here), which is what makes at most ONE year-candidate fit. An earlier draft
said `[-300, +60]` — 361 days, and loose enough that a mistyped month (`19/01` for `19/10`)
resolves silently to the same year instead of being questioned. **An unresolvable date
REJECTS**; the converter is the last place where "did you mean last year?" is answerable.

**Two-digit years are expanded to this century BEFORE validation (added s75, step 1).**
The census of the 2025 export found 2 conforming messages written `DD/MM/25`. Handing that
to the boundary "verbatim" is not neutral: **`parse.Date("25/07/25")` returns 25/07/0025
with NO error** — probed s75, `YearFromDateString`, both year validations pass (25 is
renderable and not beyond the current year). That is the boundary's quietest failure shape
in its own words, and it is live today for `add`; it is filed for the boundary separately
(see Risks). The converter therefore reads a 2-digit year as `20YY` — still the human's
year, in the notation they used — and rejects any other year length. Rule 1 above stays
"the human outranks the timestamp"; this only fixes which century the human meant.

**AMENDMENT (s75, step 5, USER RULING): the window validates the RESOLVED date on BOTH
branches — explicit year or not.** As first written, rule 1 handed an explicit year to the
boundary with no distance check, and the boundary's past side is unguarded
(`validateRenderableYear` allows 1..9999; `validateYearNotBeyondCurrent` only looks
forward). **Reproduced, not reasoned:** a message sent 2025-08-21 reading
`Item; 21/08/1200; 50,00` CONVERTS, and the converter writes `expenses-1200-08.csv` — exit
0, nothing in the rejects file, the expense silently out of its month. This is the same
pair of boundary facts that forced the D4 amendment, arriving by the one path D4 does not
cover: **D5 means a line that parses as typed is never repaired, so it never meets D4's
window check.**

**Neither T-75 gate can catch it, which is why it had to be found by reading rather than by
running.** In `t75_oracle_probe.py` the window test is the FIRST branch of the residue
split and yields `"expanded outside the comparison window"` — a *named* cause, and only
`SUSPECT` increments `unexplained`. So the gate PASSES on a misfiled row.
`t75_gate_compare.py` agrees too: both sides bucket the message `converted`. A green gate
and a misfiled expense in the same run, with an odd filename in the files report as the
only signal.

**Why this does not contradict rule 1.** Explicit year still wins the SELECTION — the
converter never overrules which year the human typed, and there is no re-resolution. The
window is now a VALIDITY check on the outcome, which is what rule 3 always was; it simply
stops being weaker on the branch that skips resolution. Bare dates use the window to
*choose* among three candidate years, explicit years use it to *reject*. One invariant,
uniformly enforced: **every emitted date lies within `[-180,+7]` of its own message.**

**The asymmetry that settles the trade-off.** The cost is real: a deliberate backfill typed
with an explicit year more than 180 days back is now refused. That is acceptable because a
rejection is VISIBLE — it lands in `telegram-rejects.csv` with its reason and a human fixes
it — while a misfile is SILENT and looks like success. T-63's whole shape is built on that
asymmetry, and it is the same reason "an unresolvable date REJECTS" above.

**Measured before deciding, on the real 2025 export (ids and deltas only):** of 358
converted lines, **2** carry an explicit year, **both 2-digit (`/25`), zero 4-digit**, and
**0** resolve outside the window. So the amendment rejects nothing that has ever been
validated — and the reason the defect survived five sessions is visible in those numbers:
**the unguarded branch is the one branch the corpus never exercises.**

⚠️ **This INVERTS an existing test.** `classify_test.go` carried
`"explicit year outranks the timestamp even far outside the window"` (`12/03/2024` sent
2025-05-01 → accepted), written to pin rule 1 as originally stated. It now expects a
rejection and is renamed. A behavior change with a test asserting the old behavior is a
decision, never a broken test to fix quietly.

⚠️ **The window and `validateYearNotBeyondCurrent` interact, and the result is a rejection.
REQUIRED TEST.** A message sent 2026-12-28 reporting `03/01`: the 2026 candidate is −359
days (outside the window), the 2027 candidate is **+6** (inside), so it resolves to
**2027-01-03** — which the boundary then REFUSES as beyond the current year. Computed, not
reasoned: an earlier read of this case put it at 2026-01-03, which the arithmetic
contradicts. The outcome (reject, loudly) is acceptable under T-47's provisional policy and
is **unreachable for the 2025 backlog** — only a late-December message referencing early
next year trips it. Pin it with a test so the two rules cannot drift apart unnoticed; note
that narrowing the future window to `+0` would also reject it, only with a clearer reason.

### D2 — Do NOT expand installments; pass the RAW token through

`900,00/3` and `405,25 x4` reach the value column verbatim. The column deliberately keeps
the raw token so the count survives into the review queue (s67 D4, T-21) —
`ParsedExpense.RawValue` exists for precisely this. A converter emitting pre-expanded rows
would bypass `review`'s installment handling: the T-54 shape, a producer emitting a shape
no consumer expects.

Corollary worth stating: because the converter does not expand, it never decides which
month installments 2..N belong to. That stays in `appender.expandEntries`, which owns it
and carries the day-overflow clamp. **Two places computing installment dates is exactly how
T-35 happened.**

### D3 — Three output classes, and nothing vanishes silently

| Class | Destination |
|-------|-------------|
| converted | the month CSV |
| **rejected** — an attempted expense that would not parse | rejects file, T-63 shape: reason as a trailing `#` comment on the SAME line |
| **receipts** — a file or image, no parseable text | NOT convertible, but reported on their OWN summary line with message ids |
| **ignored** — conversation | NOT written as rejects; counted and listed by message id in the summary |

The split needs a rule or chatter pollutes the repair queue: **a message is an attempted
expense if it contains a `;` or a currency-shaped token.** On 2025 that puts
`É 450,00 três meses…` in rejects (it discusses money — a human should see it) and
`Não quer mais?` in ignored.

**The `receipts` class is NOT a subdivision of chatter — it was measured.** All 20
empty-text messages in the export carry an attachment: **13 `application/pdf` and 7
photos, ZERO with neither.** Not one is a sticker — each is a receipt or boleto.

**USER RULING (s75): these are receipts FOR AN EXPENSE THAT WAS ALSO TYPED, and nothing
is to be done with them for now.** So they are NOT missing spend and NOT an attempted
expense — the expense itself arrives as its own text message and converts normally.
An earlier draft of this plan claimed they represented "~2.5 a month of spend that never
reaches the CSV"; that was **wrong** and is corrected here rather than quietly dropped,
because the measurement (13 PDF + 7 photo) was right while the inference from it was not.

Handling: report them as a one-line count so the reader can see the export was fully
accounted for (`N attachments — receipts for typed expenses, skipped`), and do NOT route
them to rejects. Revisit only if a future export shows an attachment with no matching
text message.

Following T-63 exactly: **no rejects file is written when nothing was rejected — its
existence IS the signal.**

⚠️ Known limitation, stated rather than hidden: `stripTrailingComment` fires only when `#`
is whitespace-preceded AND the three fields are already complete. A reject with the wrong
field count will not round-trip until the human repairs the fields — at which point the
comment strips correctly. That is the right order anyway.

### D4 — Repairs under T-64's EXACTLY-ONE-READING rule

Candidate generation for one message:

```
C0    the text as-is
C1..  the text with exactly one ','  replaced by ';'
C..   the text with exactly one '/'  replaced by ';'
C..   if it splits into 4 fields: the FIRST ';' replaced by a space   (stray ; inside item)
```

A candidate is VALID when it splits into exactly 3 fields, field 2 parses as a date and
field 3 as a value (BR decimal + both installment notations). **Accept only when EXACTLY
ONE candidate is valid**; two is an ambiguity and therefore an error, never a guess.

Worked against the real defects in the export:

| Message | Valid candidates | Outcome |
|---------|-----------------:|---------|
| `Almoço <redacted>, 19/05; 50,00` | 1 | repaired |
| `Gás 2 bujões; 15/05, 216,00` | 1 | repaired |
| `Tela celular; 25/07/ 90,00` | 1 | repaired |
| `<person C> Elô; ADM; 09/01; 405,25 x4` | 1 | repaired — the s74 case |
| `<person C>; Terpenos coolterps; 346,40; 23/10` | 0 (date/value also swapped) | rejected |
| `Uber; 05/07; 16:20` | 0 | rejected |

This is T-64's machinery lifted one layer up, deliberately: the converter and the parser
then reject for the SAME reason, so the human learns one rule instead of two.

**AMENDMENT (s75, step 2, measured): a candidate's resolved date must ALSO lie inside the
D1 window of the message day — explicit year or not.** As first written, "valid = parses"
made a plain slash typo ambiguous. The real message `Dentista <person A> canal; 21/08/ 1200,00`
has two single edits: `/`→`;` gives 21/08/2025 + 1200,00, and `,`→`;` gives the date field
`21/08/ 1200` + the value `00` — and the boundary ACCEPTS both: year 1200 is inside 1..9999,
the past side is deliberately unguarded (T-48), and **`parse.Value("00")` returns 0 with no
error** (probed). Two readings → rejected, for a message a human reads instantly. The rule
that resolves it is the one already used for a bare date: a repair is the tool's GUESS, not
the human's statement, so it needs the extra evidence, and the window is that evidence. The
as-typed line keeps rule 1 of D1 untouched (an explicit year the human wrote still wins
verbatim). What ambiguity survives the amendment: only a value whose integer part spells an
in-window year — `Bar; 15/05/ 2025,50` — pinned in the fixture and the unit table, and
absent from the 2025 export. **Measured after the amendment, independently on both sides
(Python oracle re-implementing D1+D4 from this text; Go): 6 repaired, 0 ambiguous.**

**What D4 deliberately does NOT repair, with the 2025 counts (6 two-field messages stay
rejected):** no date at all (×3: `Tabacaria, seda…; 25,50`, `Dme vencimento 2025-12; …`,
`Cartão <person C> novembro 2025; …`), a colon where the second `;` belongs (×1), and a SPACE
where the first `;` belongs (×2: `Crédito Tim 02/10; 60,00`, `San michel 22/12; 69,83`).
The last shape is repairable in principle ("the last space before a date-shaped token"),
but it is a NEW candidate rule, not a single-byte substitution, and two messages in eight
months do not justify widening the search. Revisit only if the 2026 export shows the shape
is common. The earlier estimate in the tracker — "12 comma rows + the `/` row recovered; 8
ambiguous" — was arithmetic on the bucket totals, not a measurement, and is retracted.

⚠️ **The `/`-substitution candidate and the divisor installment notation coexist ONLY
because of D5.** A well-formed `Compras; 15/05; 900,00/3` never enters this path, so the
substitution can never split a legitimate `total/N` token. State the dependency here
because the two rules live in different sections and a reader tightening one would not see
the other.

### D5 — If C0 parses, STOP; never generate repairs for a line that already works

A hard precondition, not an optimisation. It is what guarantees a well-formed `900,00/3`
can never be "repaired" into something else, and it is the single most important thing to
mutation-test in this tool.

### D6 — No repair beyond that candidate set

Anything needing two composed transformations (`<person C>; Terpenos coolterps; 346,40; 23/10`
needs a merge AND a date/value swap) is rejected. The search stops being provably-single,
and T-64's asymmetry applies unchanged: a wrong repair writes a budget row wrong by the
installment count and **indistinguishable from a legitimate one**, while a rejection costs
one edit because the row comes back with its reason inline. Volume: 9 rows over 8 months,
roughly one a month.

### D7 — Split by the RESOLVED expense date, not the message date

A purchase on 30/05 reported on 01/06 belongs in May's file.

### D8 — REFUSE to overwrite an existing month file without `--force`

The default output directory holds `expenses-2026-01.csv` and its `-repairs` /
`-repairs-2` variants, which carry **hand repairs from the first real close**. A 2026
conversion would silently destroy them. Refuse, name the file that blocked it, and require
`--force` — the same shape as `close-cycle.sh restore` refusing a pre-s74 snapshot.

### D9 — Step-3 writer decisions (s75), taken while building

- **`--out-dir` is REQUIRED when writing** (dry-run needs none). The contract above said
  the default was `/mnt/i/workspaces/expenses/`; baking that path into the binary, or
  defaulting to the working directory, are both ways to write real expense files
  somewhere nobody chose. An explicit destination costs one flag and removes both.
- **Rejects file: `telegram-rejects.csv`** in the same directory — deliberately NOT
  `expenses-*.csv`, so a glob over the month files never picks up rows that still need a
  human. Written through `batch.WriteFailedRows`, so the T-63 shape has exactly ONE
  producer (its header says "batch-auto could not parse" — the predicate IS batch-auto's
  parse, so the sentence stays true). Each reason carries provenance the human needs to
  fix a year by hand: `<boundary error> (message <id>, sent <YYYY-MM-DD>)`.
- **All-or-nothing (D8, sharpened):** every target file is checked BEFORE the first
  write, the directory must already exist, and a refusal names every blocking file.
  Half an export that looks complete is worse than none.
- **Repaired lines carry their origin INSIDE the month file:** `<line>   # repaired
  from: <text as typed> (message <id>)`. `batch.CSVReader` skips `#` lines and
  `stripTrailingComment` strips a whitespace-preceded `#` after three complete fields
  (T-63), so the audit trail rides in the very file batch-auto reads and costs nothing
  downstream. Each month file also opens with one `#` provenance line.
- **Line breaks inside a message are flattened to spaces** before a text reaches any
  line-oriented file. Zero occurrences in the 2025 export (measured); the rule exists so
  the first one cannot split a row in two.

## Build sequence / PROGRESS TRACKER

**This section is the live tracker — tick boxes here as work lands. Deliberately the ONLY
list of steps: a separate todo file would duplicate the gates and drift from them.**
Branch: `feat/t75-telegram-converter`. Local model: this repo's **Go** tier list
(`my-go-qcoder` primary) — ⚠️ 20.7 GB on a 12 GB card, so it CPU-offloads, runs ~7 min per
bounded file and **will background**, and T-71 says a backgrounded call silently ignores
`output_file` and injects no verdict template. Scope each call to one bounded file.

- [x] **1 — source adapter + message stream + `--dry-run` bucket report** — **DONE s75.**
  **Gate MET, per id.** `telegram-import <2025 export> --dry-run` → 393 messages / 352
  converted / 20 receipts / rejected 1 + 12 + 1 (1, 2, 4 fields) + 4 bad date + 2 bad value /
  1 ignored — and **every one of the 393 ids sits in the bucket the probe assigned it**:
  `.claude/scratch/t75_expected_buckets.py` writes the probe's per-id TSV,
  `t75_gate_compare.py` diffs the report against it (0 differences), and its `--perturb` run
  flips one expected id and reports exactly 1, so the comparison is proven able to fail.
  Acceptance `TestTelegramImport_DryRunAccountsForEveryMessage` green in the `-short` group
  (synthetic 13-entry export, ids asserted per line); unit suite green — `internal/capture`
  41 subtests including the required 2027 case, `internal/telegram` 26, `cmd` report writer 4.
  **Without `--dry-run` the command currently returns an error naming step 3** — deliberate,
  so a run that would write cannot silently do nothing. Re-run the same gate on the 2026
  export the moment it exists: bucket drift there is the signal, not a failure of this step.
  **Gate:** reproduces the known 2025 buckets exactly — 352 / 20 / 12 / 4 / 2 / 2 / 1 —
  asserted **PER MESSAGE ID, not as totals**. A bug moving one message from `bad value` to
  `bad date` leaves every count identical; the ids are already in hand. The s74 shape
  exactly: the pairing check passed 717/717 over a lookup that coin-flipped a year for 38 ids.
  **Build the adapter split from the start** — retrofitting it after step 2 means re-testing
  the parser through a second door.
  **Scope note (s75):** D1's date resolution is BUILT in this step, not step 2 — "does C0
  parse?" cannot be answered without it, and a placeholder would only be deleted. Step 2's
  gate still owns the D1 assertions; step 1 pins them early in `capture/classify_test.go`.
  Layout: `internal/capture` (source-agnostic: `Message`, `Classify`, `ResolveDateField`,
  `Summarize`) ← `internal/telegram` (adapter: `ReadExport` → `[]capture.Message`) ←
  `cmd/.../telegram_import.go` (flags + report rendering). The adapter imports the core,
  never the reverse.

- [x] **2 — repairs (D4 / D5 / D6) + D1 resolution** — **DONE s75.**
  **Gate MET, per id — with the numbers CORRECTED by measurement.** On the 2025 export:
  **6 repaired** (five comma rows and the `Dentista …; 21/08/ 1200,00` slash row), **0
  ambiguous**, **6 two-field messages still rejected** for shapes D4 does not cover (listed
  under the D4 amendment). Two implementations agree id for id — `t75_expected_buckets.py`,
  which now re-implements D1 + D4 from this plan's text, against the Go command: 393/393,
  `--perturb` moves it by 1. An explicit `DD/MM/YYYY` is honoured and the 2026-12-28 →
  `03/01` → 2027 case rejects (both pinned in step 1). **Both mutations run; each COMPILED
  and went RED on exactly its REQUIRED row:** `if false && err == nil` in `Classify` (repairs
  instead of the as-typed parse) fails the D5 row; `case 1, 2, 3…` in `repairedOrRejected`
  (first valid candidate wins) fails the D4 row. The report gained a `repaired` line (ids
  listed — the lines the tool touched deserve a glance) and a `rejected: ambiguous` bucket;
  the fixture gained ids 14 (ambiguous) and 15 (slash repair), and ids 5 / 9 moved from
  `rejected: N fields` to `repaired`. D1 itself landed in step 1 (scope note above).
  **Original gate text, kept for the record:** the 12 comma rows + the `/` row recovered;
  the 8 genuinely ambiguous ones still rejected; an explicit `DD/MM/YYYY` in the text is
  honored, not overwritten by the message year; and the 2026-12-28 → `03/01` → 2027 case
  rejects. (The 12 / 8 split was an estimate from bucket totals — retracted in D4.)
  **Mutation, both directions, each must still COMPILE:** remove the C0-first rule → a valid
  `900,00/3` gets "repaired" → RED. Remove the exactly-one rule → an ambiguous message is
  silently accepted → RED.

- [x] **3 — writers: month CSVs + rejects + summary** — **DONE s75** (decisions in D9).
  **Gate MET.** `TestTelegramImport_MonthFileIsAcceptedByBatchAuto` (Ollama-gated): the
  converter writes `expenses-2025-05.csv` into the work dir and the REAL `batch-auto
  --dry-run` reads it — 9 s, no `failed.csv`, the repaired line with its trailing audit
  comment included. Deterministic half: `TestMonthFileLines_RoundTripThroughBatchAutoReader`
  feeds every produced line to `parse3FieldLine` and gets the same item, date and raw value
  back. Rejects file is `batch.WriteFailedRows`' own T-63 shape (one producer), each reason
  suffixed `(message <id>, sent <date>)`; `expect.FailedRowsCarryTheirReason` pins exactly
  the four attempted expenses of the fixture, so a receipt or a chat line leaking in fails
  the length check; a clean export leaves NO rejects file. **D8 proven three ways:** the
  acceptance scenario (pre-existing May file → refused, named, `--force` offered, and the
  other three month files plus the rejects file NOT written — all-or-nothing), the unit
  table (every blocker named, not only the first), and the real export written twice into a
  scratch dir — the second run refused naming all ten files, `--force` then replaced them.
  The literal "`expenses-2026-01.csv`" collision cannot be produced by the 2025 export (no
  2026 month); the 2026 export will exercise it. **Real 2025 run:** 9 month files
  (2025-04 … 2025-12), 358 body lines = 352 converted + 6 repaired, 6 `# repaired from:`
  comments, `telegram-rejects.csv` with 14 rows. **`./run-acceptance.sh -full` after step 3:
  78 passed / 0 failed / 0 skipped in 105 s** — step 5's "-full if a scenario is added"
  requirement is met for this state; it re-runs at the end because step 4 touches the probe.
  **Original gate text:** a produced CSV runs clean through `batch-auto --dry-run`; a rejects
  line, once its fields are repaired, re-runs as-is. D8's overwrite refusal proven by
  pointing it at the existing `expenses-2026-01.csv` and confirming it declines and names
  the file.

- [x] **4 — repoint `t75_oracle_probe.py` at real converter output** — **DONE s75.**
  **Characterized first:** `t75_probe_characterize.py` pinned eleven numbers black-box,
  green on the unmodified probe, red when one was flipped, committed (`0626c3b`) BEFORE
  the swap. **Then:** `--source converter` (default) runs the Go command into a temp dir
  and reads its month files; `--source naive` keeps step 1 reproducible; the gate is the
  exit code. **Measured:** 358 lines → 389 rows; matched **346 — unchanged**, because not
  one of the 6 repaired lines has a partner; residue 43 = 24 tails + 9 outside + 7 unknown
  + 2 omission + 1 count mismatch; **SUSPECT 0 → GATE PASS**, perturbations 346 → 345 both
  ways. **The catch-all was tautological and is gone:** "never entered" is now verified
  by item lookup (unknown / omission / SUSPECT — only SUSPECT fails), and step 1's "3
  never entered" reads 2 unknown + 1 omission. **Finding:** the six repaired messages
  (R$ 1.828,96) are absent from the 2025 log — the typo was what kept them out. Written up
  in `.claude/t75-oracle-viability.md` § "Step 4"; to file beside T-78 at handoff.
  **Gate: unexplained == 0.** NOT a match rate — see the validation section.
  ⚠️ **`characterize-first` applies HERE AND NOWHERE ELSE in this plan.** The probe is
  committed code with no tests of its own, and its conversion source changes from an internal
  throwaway to "read the Go command's CSV". **Characterization baseline, to hold green before
  the swap** (measured s75 on the 2025 export): `352` conforming messages → `383` rows;
  **346 matched, 37 side-A residue** (24 installment tails / 9 outside window / 3 never
  entered / 1 count mismatch), `153` side-B residue; buckets `20 / 12 / 4 / 2 / 2 / 1`; both
  perturbations move the residue by exactly 1.

- [x] **5 — `go build ./... && go vet ./... && go test ./...` clean; `index.md` row; `-full` acceptance if a scenario is added** — **DONE s75.**
  ⚠️ **`-short` is structurally blind to CSV-shape changes** (s72): every
  `OutputFileHasColumns` assertion sits behind `RequireOllama`. If this touches the CSV
  contract at all, `-full` must run before calling it done.
  **Measured:** build clean, vet clean, `go test ./...` **exit 0 — 22 packages `ok`, 948
  tests**, re-run through `command go test` so the verdict is an exit code and a raw log,
  not rtk's summary line. **`./run-acceptance.sh -full`: 78 passed / 0 failed / 0 skipped
  in 108 s** — unchanged from the post-step-3 run, which is the expected result since step 4
  touched only the probe. **0 skipped is the load-bearing number, not 78:** it is what says
  the Ollama gate actually ran rather than quietly opting out, and
  `TestTelegramImport_MonthFileIsAcceptedByBatchAuto` took **3.26 s** — a skip costs 0.00 s,
  so the duration is the corroborating evidence that the real model was called.
  **`index.md`:** the `internal/capture` package row described only steps 1–2; it now carries
  step 3's `output.go` (pure sink, month grouping by resolved date, T-63 rejects with
  provenance, every `os` call left in the command) and the corrected subtest count —
  **62 → 74**, the 12 new ones being `output_test.go`; `internal/telegram` holds at 26. The
  plan row's status prefix and this file's header were both stale ("steps 1–2 DONE",
  "Status: PLAN … Not implemented") and were corrected in the same pass.
  **Not done here, deliberately:** the probe's two characterization modes were left alone.
  They are step 4's gate, they were green and committed there, and re-running them touches
  the private export for no new information — step 6 re-runs the whole gate on the 2026 file
  anyway.

  **THEN the D1 amendment landed inside step 5** (see D1 for the reasoning; USER RULING).
  Order was plan first, code second — the D4 discipline, because this changes a decision
  rather than fixing a slip. Reproduced end to end before touching anything: the same
  two-message export wrote `expenses-1200-08.csv` with exit 0 and no rejects file; after,
  the message lands in `telegram-rejects.csv` reading *"invalid date: 21/08/1200 is not
  within 180 days before or 7 days after 2025-08-21 (message 1, sent 2025-08-21)"*.
  **TDD, RED first:** 3 rows failed and now pass; a 4th — the `+7` future edge — was written
  to pass in BOTH directions on purpose, because `+7` accepted next to `+8` refused is what
  proves the guard has a LOCATED boundary rather than refusing everything. One existing row
  was INVERTED, which D1 records as a decision.
  **Implementation from `my-go-qcoder`, verdict 1:** all eight worked cases satisfied and
  both existing helpers (`parseDayMonth`, `calendarDate`, `daysFrom`) reused rather than
  reimplemented; it errored early on a non-integer day/month instead of passing through
  (kept — bucket-equivalent `BucketBadDate`, more specific message) and left the
  "validation is NOT done here" doc comment that the change had just falsified (fixed).
  **Re-verified after the amendment:** build + vet clean; `go test ./...` exit 0, 22
  packages; `internal/capture` 74 → **77** subtests; `./run-acceptance.sh -full` **78 / 0 /
  0 in 112 s** with the Ollama gate again at 3.24 s; **probe characterization GREEN in BOTH
  modes — converter 12/12, naive 11/11 (control)**. The converter-mode green is the
  load-bearing one: every pinned number on the real 2025 export is unchanged, which confirms
  the pre-decision census empirically — the amendment rejects nothing ever validated.
  **Fixtures were audited against the new rule BEFORE running:** the only explicit-year
  fixture line is `08/07/2025` at −1 day, and the `<person C> Elô ADM` repair resolves to −178,
  inside `[-180,+7]` by two days — a fair measure of how tight that window is.
  **Consequence for the Risks entry:** the converter no longer depends on the boundary's
  unguarded past side. That defect is still live for `add` and `batch` and still needs
  filing; T-75 is simply no longer exposed to it.

  ⚠️ **THEN the SECOND implementation of D1 had to be amended too — and none of the six
  checks above could see that it hadn't been.** `t75_expected_buckets.py` is an INDEPENDENT
  re-implementation of D1+D4+D5 from this plan's text; that independence is what caught the
  D4 defect, and it means **every D1 amendment has two code sites, not one** — the D4
  discipline was "fix the plan, then BOTH implementations", and step 5 initially fixed only
  the Go. Why the gap was invisible: `go test` and `-full` are Go-only; `--source converter`
  measures the binary against pinned numbers, not against the oracle; `--source naive` never
  touches the Go path; and the two-half gate had **not been re-run**, which on 2025 would
  have reported 393/393 anyway because zero messages exercise the changed branch. **The
  corpus that hid the defect also hides the drift.** Left alone, the first 2026 run would
  have shown the independent oracle disagreeing with a correct binary — and the natural
  reading of that is "the Go is wrong", i.e. pressure to revert a correct fix at the one
  moment there is no other adjudicator. **Fixed and re-run on 2025: `GATE PASS — every one
  of 393 ids in the probe's bucket`, and `--perturb` still moves it by exactly 1.** That
  re-proves the GATE, not the amendment.
  **Also corrected while in there:** `validRepairs`' window check (Go) and `candidate_ok`'s
  (Python) are now SUBSUMED — `parseLine`/`resolve_date` refuse an out-of-window candidate
  before either is reached — and both carried a comment claiming "the as-typed line honours
  an explicit year verbatim", which the amendment had just falsified. Both checks are KEPT,
  with the comment saying why: D1's rule ("a date belongs near its message") and D4's ("a
  guess needs more evidence than a statement") have different justifications and merely
  share a threshold today.

- [ ] **6 — run it, hand the output to a real close**
  Not a coding step, and the one that actually validates T-75: convert a month, run
  `close-cycle.sh prepare`, watch the rows arrive. Needs the 2026 export from the user.

### Conventions that govern this work

Read at the start of session 75; recorded so the next session does not re-derive them.

- **`test-executable-spec` rule 5 — the SUT is a PURE FUNCTION, so the DSL COLLAPSES.**
  The diagnostic is "does the SUT consume a sequence?" `message text → CSV line | reject`
  does not: no sequence, no state. So there is **no given/when/then skeleton here** — the
  vocabulary reduces to **builder-nouns** plus **verdict-verbs**. The doc names over-DSLing a
  pure-function test as *"the most common way to violate rule 3"*: resist a mutation
  mini-language, and let an exotic malformation read as one inline line of data.
  **How that expresses in Go, since the pattern was written for pytest:** the repo's
  `[ref:testing]` requires table-driven subtests with testify, and lists "Skip table-driven
  approach for new commands" under **Do NOT**. The two conventions AGREE rather than compete —
  **a table row IS the builder-noun, and the shared assert helper IS the verdict-verb.** Say
  it that way; do not import pytest-shaped `given_/when_/then_` names into Go.
  (The aggregate bucket report is a fold over independent messages, not a sequence.)
- **`function-decomposition` — split on DECISION COUNT, not line count.** A long linear path
  is fine. The repair resolver is the decision-dense part: extract each candidate reading as a
  **pure function returning a value**, unit-testable without the file reader. Helper names
  must narrate an algorithm step.
- **`patterns-code-value-or-error`** — internal helpers return `(value, error)`; a
  human-readable sentinel string belongs only at the boundary (the rejects file).
- **`patterns-code-return-not-mutate`** — each stage returns its contribution; no shared
  mutable accumulator threaded through the parse path.
- **Repo Go conventions** — one `.go` file per subcommand; `fmt.Errorf("context: %w", err)`;
  method extraction so a multi-step body reads as named delegated steps (≤ ~15 lines).
- **`characterize-first`** — does NOT apply to the new command (no existing behavior to
  preserve). Step 4 only. Noted so it is not cargo-culted onto steps 1–3.

## Testing

Go, testify, table-driven subtests — the repo's existing convention, so no new toolchain and
`go test ./...` covers it. See the conventions block above for how that reconciles with
`test-executable-spec` (it does: a table row is the builder-noun, the assert helper is the
verdict-verb).

**Fixtures are SYNTHETIC.** The real export is personal data and must never be committed —
the same reasoning that gitignores the training JSONs. One case per defect class, plus each
repair rule's **counterexample**: a repair rule with only positive tests is the shape that
produced T-54.

**The real-export bucket assertion lives outside the unit suite.** It needs a private file,
so it belongs in the `--dry-run` gate run by hand (step 1), not in `go test ./...` — a unit
test that silently skips when a file is absent is a check that reports success while testing
nothing, which is the exact failure mode this repo keeps finding.

Every guard broken and confirmed red, and **the mutation must still COMPILE** — s72's deleted
field produced a build failure, which is the compiler talking, not the test.

## Validation — and the criterion that is deliberately NOT a match rate

Step 4 swaps the probe's throwaway conversion for the real converter's output and reuses
everything else: the `(date, value)` multiset match, the residue classification, the
perturbation check.

**Criterion: every miss carries a named cause; unexplained == 0.** NOT "coverage ≥ 90.3%".
Coverage will move — repairs add rows the probe skipped, and some will not match — and
chasing the number would push the converter toward reproducing the log's own omissions,
which is exactly what the T-78 finding warns about. **The log is a second lossy record, not
ground truth, so it cannot adjudicate a mismatch.**

## Out of scope

- **T-78** — the installment backfill. Different work, different risk, and it needs the
  user's knowledge of what was actually paid.
- **T-79** — `*` as a third multiplier spelling, in the Go parser. Small and separate;
  affects one 2025 row.
- **Classification** — untouched. This validates CAPTURE fidelity only.
- **The 2026 export** — only the user can produce it.

## Risks

- **The 2026 vocabulary may differ from 2025's.** `89,90*3` appears only in 2025-11, and
  the parser only learned `x4` in s73. Run `--dry-run` on the fresh export FIRST; **bucket
  drift is the signal.** This is why the classifier ships as a real tool rather than
  remaining the throwaway used in step 1.
- **Corrections posted as replies would double-count.** MEASURED, not assumed: the 2025
  export has 2 replies and NEITHER contains expense syntax; 19 messages were edited and the
  export carries final text, so edits resolve themselves. Re-check on the 2026 export.
- **Two people type into the group** (290 / 103 messages), so habits may differ per author.
- **TO FILE AT HANDOFF — boundary accepts `DD/MM/YY` as year 00YY (found s75).**
  `parse.Date("25/07/25")` → `25/07/0025`, no error: `ParseDateFlexible` takes any integer
  as the year, `validateRenderableYear` allows 1..9999, and `validateYearNotBeyondCurrent`
  only looks forward. An `add` with a 2-digit year therefore writes a row that hashes
  cleanly into both logs and is then never routed by `generate-workbook --year`. Not fixed
  here: the converter expands its own input (D1), and the boundary rule — reject, or expand
  as the converter does — is a parse-boundary decision to take with its own tests.
  **Updated s75 step 5:** the D1 amendment means T-75 is no longer EXPOSED to the unguarded
  past side, but the defect is unchanged and still live for `add` and `batch`.
- **OPEN QUESTION for the 2026 census — a zero-value row converts silently.** The same
  composition shape as the D1 defect, one field over: `parse.Value("00")` returns 0 with no
  error (verified s75, step 2), and the as-typed path runs `parse.Value` unguarded.
  **Reproduced s75:** `Zero token; 21/08/2025; 00` and `Zero comma; 21/08/2025; 0,00` both
  CONVERT, and the month file carries `Zero token;21/08/2025;00` with no rejects file.
  **Deliberately NOT fixed, and it is a DECISION rather than a regression:** the harm is far
  lower than a misfiled month — a R$ 0,00 row is visible in the month file, survives
  `batch-auto` and reaches `review.html` where a human sees it — and "is a zero-value
  expense ever legitimate?" is a question for the user, not an inference. **Add it to the
  2026 census alongside the explicit-year check** (`.claude/scratch/` census, s75, already
  does the date half): count zero-value lines before deciding. Only refuse if the count is
  greater than zero and the user says they are never intentional.
