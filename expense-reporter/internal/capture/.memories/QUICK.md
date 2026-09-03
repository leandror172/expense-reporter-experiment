# internal/capture — Quick Memory

*Working memory for the source-agnostic half of expense capture (T-75). Keep under 40 lines.*

## Status
Founded s75, complete s76. Design authority: `.claude/plans/t75-telegram-converter.md`
(decisions D1–D10). **Adapters import this; this imports no adapter.** `internal/telegram`
is one source; a bot is another. That direction is a CORRECTNESS property, not tidiness —
the same message via export or via bot must reach the same verdict.

## Contract
`Message`(id, sentAt, text, attachment) → `Classify` → `Outcome` in one of four classes:
converted / rejected / receipt / ignored. `ClassifyAll` expands lists (D10) then classifies.
`Summarize` folds outcomes for the report. `MonthFiles`/`Rejects` in `output.go` are the PURE
sink — they return values, every `os` call lives in the command.

**The parse is NOT reimplemented.** Converted means `parse.Fields` accepted the fields, so an
emitted `item;DD/MM/YYYY;value` is one `batch-auto` takes by construction. `parseLine` is THE
predicate; the as-typed line and every repair candidate go through it, so a repair can never
be accepted on looser terms than the original.

## Key Rules
- **D5 first, and it is a hard precondition.** `Classify` parses AS TYPED before anything
  else; a line that parses is Converted and NEVER repaired. Without it a well-formed
  `900,00/3` could be "repaired" into something else.
- **D4 exactly-one-reading.** Single-edit candidates (each `,`→`;`, each `/`→`;`, four fields
  → merge the first two). Accept only when EXACTLY ONE parses; two is an error, never a
  guess. Nothing is composed (D6).
- **D1 window `[-180,+7]` validates the resolved date on BOTH branches.** An explicit year
  wins the SELECTION — nothing re-resolves it — but the window then REJECTS. ⚠️ Do not
  "simplify" this away: `internal/parse` allows years 1..9999 and only looks forward, so
  without it `21/08/1200` converts and opens `expenses-1200-08.csv`, exit 0, no rejects file.
  Found by an independent oracle, not by a test. The window is under 365 days so exactly ONE
  year-candidate can fit — that is what makes bare-date resolution unambiguous.
- **D10 splits a multi-line list by SHAPE, not parseability** — every non-empty line must have
  exactly two `;`. "Every line parses" would have thrown away the ten-line backlog list that
  motivated it, two of whose lines carried a year typo. Split lines keep the PARENT id, so one
  id can sit in two buckets: **bucket counts count LINES, the id list counts MESSAGES.**
- **D2: installment tokens pass through RAW.** The count must reach the review queue, and a
  second site computing installment dates is how T-35 happened.
- **A rejection is cheap, a silent misfile is not.** Every rule above prefers refusing to
  guessing, because a reject lands in `telegram-rejects.csv` with its reason for a human,
  while a wrong row looks exactly like a right one.

## Watch out
The plan's decisions have a SECOND implementation in `.claude/scratch/t75_expected_buckets.py`,
an independent re-implementation from the plan TEXT (that independence is what caught the D4
defect). **Amend it whenever a decision changes** — skipping it makes the oracle disagree with
a correct binary, and "the Go must be wrong" is the natural misreading.
