# Fixture: telegram-import-multiline

Synthetic. Three messages, built to pin **plan D10** — a multi-line message whose every
non-empty line is 3-field-shaped is N messages, one per line.

The shape came from the real 2026-02 export, whose first run produced a bucket the 2025
corpus never had: `rejected: 21 fields`, one message holding ten well-formed expense lines
(a backlog catch-up list). Flattened, all ten were lost at once.

| id | what it is | why it is here |
|---|---|---|
| 1 | one ordinary expense, one line | the no-split path must stay byte-identical |
| 2 | **three 3-field lines**, the third dated `99/99` | splits into 3; **two convert, one is a bad date** |
| 3 | one 3-field line **plus a chatty line** | must **NOT** split — it is one expense, not two |

## The two rows that carry the design

**Message 2 is why the test is SHAPE, not parseability.** Its third line cannot parse
(`99/99` is not a calendar date). Had D10 required every line to parse, this message would
not split and all three expenses would be lost — which is exactly what the real export
would have done, since two of its ten lines carry a `29/12/26` year typo. Shape is the
structural evidence that a human typed a LIST; each line's date and value are then judged
one at a time by the same predicate as any other message.

**Message 3 is the guard against over-splitting.** Its second line has no semicolons, so
the message is judged whole: flattened it reads `Almoco; 12/05; 25,00 vou pagar amanha`,
three fields with an unparseable value, and it lands in `rejected: bad value`. If D10 ever
loosens to "split when SOME line is 3-field-shaped", this row goes green in the wrong way —
the expense would convert and the human's note would vanish silently.

Message 2 also pins the id rule: split lines keep the **parent** id, so id 2 appears in
**two** buckets at once (converted and rejected: bad date). That is what forced `Summarize`
to de-duplicate ids per bucket, bucket counts to count LINES while the id list counts
MESSAGES, and `t75_gate_compare.py` to compare a set of labels per id.

Deterministic: no model, no workbook, no config — the scenario runs in the `-short` group.
