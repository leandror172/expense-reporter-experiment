# T-75 — Telegram export → batch-auto CSV converter

**Status: PLAN, session 75 (2026-09-02). Not implemented.**
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

```
.claude/tools/telegram-to-csv.py <result.json> [--out-dir DIR] [--dry-run]
```

No model, no network, no workbook. Deterministic and idempotent.

- **`--dry-run`** — bucket report only, writes nothing. This IS the committed bucket
  classifier; one tool with two modes rather than two tools that drift.
- **default** — writes `expenses-YYYY-MM.csv` per month plus a rejects file to
  `/mnt/i/workspaces/expenses/` (**outside the git repo**, where `expenses-2026-01.csv`
  already lives). Real expense data therefore needs no gitignore rule — it is not in the
  tree at all.

Output line shape is `item;DD/MM/YYYY;value` — three fields, matching
`batch_auto.go:537`'s `strings.SplitN(line, ";", 3)`.

## Decisions

### D1 — Emit the RESOLVED full date, not bare `DD/MM`

The converter holds strictly better year evidence than anything downstream: the message's
own timestamp. `internal/parse`'s ladder ranks an explicit year above `--year`, so this is
well defined and deliberately outranks the flag.

It matters at the year boundary. A message sent 02/01/2026 reporting `31/12` is a 2025
purchase; under `--year 2026` it becomes 31/12/2026, which `validateYearNotBeyondCurrent`
then refuses — loud, but for the wrong reason.

**Resolution rule:** take the candidate `(day, month, message_year)` and step the year
until the date lands inside an acceptance window; **if no year fits, REJECT with "cannot
resolve year"** rather than picking one.

**Window = `[-180, +7]` days, and it is derived from the data, not chosen.** Measured over
all 354 date-bearing messages in the 2025 export: **min −131 days, max +3, 279 of 354 on
the message's own day.** So the past side must be generous — people do report a purchase
months later — while the future side has NO support beyond +3 (+7 leaves headroom for a
scheduled bill, the `Unimed vencimento` shape).

**The load-bearing invariant is that the window is narrower than 365 days** (187 here), which
is what makes at most ONE year-candidate fit and the resolution unique. An earlier draft of
this plan said `[-300, +60]` — 361 days, technically inside the bound with almost no margin,
and loose enough that a mistyped month (`19/01` for `19/10`) resolves silently to the same
year instead of being questioned. **The converter is the last place where "did you mean last
year?" is still answerable**, so an unresolvable date belongs in rejects.

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

## Build sequence / PROGRESS TRACKER

**This section is the live tracker — tick boxes here as work lands. Deliberately the ONLY
list of steps in the repo: a separate todo file would duplicate the gates and drift from
them.** Branch: `feat/t75-telegram-converter`.

- [ ] **1 — source adapter + message stream + `--dry-run` bucket report**
  **Gate:** reproduces the known 2025 buckets exactly — 352 / 20 / 12 / 4 / 2 / 2 / 1 —
  asserted **PER MESSAGE ID, not as totals**. A bug moving one message from `bad value` to
  `bad date` leaves every count identical; the ids are already in hand. This is the s74
  shape exactly: the pairing check passed 717/717 over a lookup that coin-flipped a year
  for 38 ids.
  **Structure it as the adapter split from the start** — retrofitting it after step 2 means
  re-testing the parser through a second door.

- [ ] **2 — repairs (D4 / D5 / D6)**
  **Gate:** the 12 comma rows + the `/` row recovered, and the 8 genuinely ambiguous ones
  still rejected.
  **Mutation (both directions, and each must still RUN):** remove the C0-first rule → a
  valid `900,00/3` gets "repaired" → RED. Remove the exactly-one rule → an ambiguous
  message is silently accepted → RED.

- [ ] **3 — writers: month CSVs + rejects + summary**
  **Gate:** a produced CSV runs clean through `batch-auto --dry-run`; a rejects line, once
  its fields are repaired, re-runs as-is. D8's overwrite refusal proven by pointing it at
  the existing `expenses-2026-01.csv` and confirming it declines and names the file.

- [ ] **4 — repoint `t75_oracle_probe.py` at real converter output**
  **Gate: unexplained == 0.** NOT a match rate — see the validation section.
  ⚠️ **`characterize-first` applies here and nowhere else in this plan.** The probe is
  committed code with NO tests of its own, so pin its current output (the s75 numbers)
  green BEFORE swapping its conversion source. Tests written after the swap would encode
  what the new code does — circular, and worthless as a regression check.

- [ ] **5 — `index.md` row; `git ls-files -s` → `100755`**
  **Convention verified, not assumed:** all four existing `.claude/tools/*.py`
  (`backfill-type`, `check-ref-integrity`, `lookup-category`, `reconstruct-csvs`) are
  `100755`. Contrast the s75 probe at `100644` in `.claude/scratch/`, run via `python3`.
  drvfs forces 777 locally, so only git's copy is evidence.

- [ ] **6 — run it, hand the output to a real close**
  Not a coding step, and the one that actually validates T-75: convert a month, run
  `close-cycle.sh prepare`, and see the rows arrive. Needs the 2026 export from the user.

### Conventions that govern this work

Read at the start of session 75; recorded so the next session does not re-derive them.

- **`test-executable-spec` rule 5 — the SUT is a PURE FUNCTION, so the DSL COLLAPSES.**
  The diagnostic is "does the SUT consume a sequence?" `message text → CSV line | reject`
  does not: no sequence, no state. So there is **no `given/when/then` skeleton here** —
  the vocabulary reduces to **builder-nouns** (`a_message(text=…, sent=…)`) plus
  **verdict-verbs** (`converts_to(…)`, `rejects_with(reason=…)`). The doc names
  over-DSLing a pure-function test as *"the most common way to violate rule 3"* and says
  explicitly to resist a mutation mini-language — so an exotic malformation reads as one
  inline line of data, not as a new combinator.
  (The aggregate bucket report is a fold over independent messages, not a sequence — it
  does not change this.)
- **`function-decomposition` — split on DECISION COUNT, not line count.** A long linear
  path is fine. The repair resolver is the decision-dense part: extract each candidate
  reading as a **pure function returning a value**, unit-testable without the file reader.
  Helper names must narrate an algorithm step; if a name does not describe a step of the
  domain algorithm, do not extract it.
- **`patterns-code-value-or-error`** — internal helpers return `(value, error)`; a human
  readable sentinel string belongs only at the boundary (the rejects file), never as an
  internal return.
- **`patterns-code-return-not-mutate`** — each stage returns its contribution; no shared
  mutable accumulator threaded through the parse path.
- **`characterize-first`** — does NOT apply to the new tool (no existing behavior to
  preserve). It applies to step 4 only. Noted so it is not cargo-culted onto steps 1–3.

## Testing

Python, so stdlib `unittest` — no new dependency in a Go repo.

**The fixture is SYNTHETIC.** The real export is personal data and must never be committed,
the same reasoning that gitignores the training JSONs. One case per defect class, plus each
repair rule's counterexample — a repair rule with only positive tests is the shape that
produced T-54.

Real-data bucket counts are asserted in a test that **SKIPS when the export is absent**,
mirroring how `extern.RequireOllama` gates the Ollama scenarios: drift in the real file is
caught locally without making the suite depend on private data.

Every guard broken and confirmed red, and **the mutation must leave the program runnable** —
s72's deleted field produced a build failure, which is the interpreter talking, not the test.

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
