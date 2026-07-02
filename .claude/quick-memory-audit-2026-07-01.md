# .memories/QUICK.md Audit & Consolidation — 2026-07-01

## Trigger

The career chatbot HF Space injects every `*/.memories/QUICK.md` into its always-loaded
system prompt. This repo's QUICK files had grown to **46,308 chars (~11K tokens)**,
pushing every Groq request past the free tier's 12K tokens-per-minute limit.

## Contract applied

Per the design docs (`~/workspaces/web-research/docs/research/memory-architecture-design.md`,
`memory-layer-design.md`, and the original template
`/mnt/i/workspaces/llm/docs/portfolio/hf-space/prompt-create-memories.md`):

- **QUICK.md** = Tier-0 working memory: ~30 lines, telegram-style, always injected.
  Current status + structure + key rules + pointers only.
- **KNOWLEDGE.md** = semantic memory: durable decisions with rationale, read on demand.
- Episodic material ("session NN did X") consolidates into KNOWLEDGE, never accumulates
  in QUICK — this consolidation pass is the "dream mode" the architecture doc describes.

## Root cause

Session handoffs appended per-session status paragraphs to the always-injected QUICK
files instead of consolidating them into KNOWLEDGE.md. The worst files were ~80% episodic
history. A deferred task was filed in the llm repo (owner of the `session-handoff`
skill/overlay) to make the skill consolidate instead of append.

## Results

Total QUICK.md: **46,308 → 16,193 chars (−65%)** — chatbot prompt contribution drops
from ~11K to ~4K tokens. Heading skeletons preserved (downstream parser splits on
headings). Nothing deleted — everything relocated/merged into sibling KNOWLEDGE.md.

| QUICK.md (folder) | Before | After | Moved to KNOWLEDGE.md |
|---|---:|---:|---|
| expense-reporter/test | 7,875 | 1,538 | WS-B acceptance retarget, generate-workbook fixture sub-format, type-routing-cycle design, session 42–43 traps (dry-run masking, explicit-year time-bomb, build-tag lessons) |
| expense-reporter (module) | 7,465 | 1,568 | New "Milestone Log": Phase G, taxonomy extraction, Plan A/B, session-42 follow-ups, WS-B slices 3–4 |
| internal/taxonomy | 5,460 | 1,542 | Routing mechanics, path.go details, WS-C income routing, WS-A year adaptation (fixed stale "pending" note) |
| internal/generate | 5,034 | 1,329 | New "Code Organization Conventions" (styles vocabulary, file homes, method extraction, income symmetry) |
| repo root (code/) | 4,029 | 1,513 | New "Milestone Log": generator, typing, retire-insertion pivot, WS-A…WS-E |
| internal/classifier | 3,275 | 1,447 | T-13 mechanics, qcoder-revert rationale, 5.R4 corpus expansion |
| internal/apply | 2,759 | 1,462 | **KNOWLEDGE.md created**: log-append pivot, write order/failure honesty, pre-flight, T-20/T-21 |
| internal/feedback | 2,437 | 1,294 | Year-handling update (corrected stale "parseDate 2 parts only" claim) |
| internal/review | 2,391 | 1,205 | **KNOWLEDGE.md created**: sheets→types rename history, RUI-4 type column |
| cmd/workbook-inspect | 2,222 | 1,097 | **KNOWLEDGE.md created**: cell-inclusion/rowType/rowFill/crossSheetRefs design, provenance |
| internal/excel | 1,886 | 1,104 | Reference-sheet D/F dead-code finding, TrimSpace boundary fix |
| mcp-server | 1,475 | 1,094 | Dedup only (details already in KNOWLEDGE) |
| **Total** | **46,308** | **16,193** | |

## Stale facts corrected during consolidation

- taxonomy KNOWLEDGE described the "year adaptation" (`DD/MM/YYYY` support) as pending —
  it landed in WS-A/T-11 (session 37). Superseded with the landed description.
- feedback QUICK claimed `taxonomy.parseDate` requires exactly `DD/MM` — same staleness;
  updated to the WS-A behavior.

## Follow-ups

- **Regrowth risk:** the session-handoff workflow appends to QUICK.md; without a
  convention change the files will regrow. Deferred task filed in
  `/mnt/i/workspaces/llm/.claude/tasks.md` (session-handoff skill owner) to make handoff
  consolidate into KNOWLEDGE.md instead.
- Total is 16.2K vs the ~15K target — every file is at/near the 30-line contract;
  further cuts would remove genuine working memory.
