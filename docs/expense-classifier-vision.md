# Expense Auto-Classification: End-to-End Vision

**Date:** 2026-02-26
**Status:** Long-term goal — to be built iteratively across Layers 5-6+
**Related:** Layer 5 (expense classifier), Layer 6 (OpenClaw/Telegram)

---

## User Scenario (Verbatim from Leandro)

> An expense is posted in the chat ["Uber Centro;15/04;35,50"]
>
> The telegram plugin/openclaw/something receives that, and attempts to classify it (using the built tools, whatever is the way we do it)
>
> It gets a response with a few possibilities, and confidence level for them
>
> It posts in the chat, answering the message, saying something to the effect (in PT-BR) "This expense seems to be one of these categories", and lists the possible options (even low confidence ones)
>
> - Would be interesting to have these options as true clickable options, instead of having the user type a response
>
> The user selects one of the categories
>
> The expense is added to that category (or an error happens; the expense reporter has some mechanism to log errors for batch insert; I don't think it has for single; in any case, errors in any phase should be logged somewhere)
>
> The result is posted in the chat

### Offline / Unavailability Behavior

> In case the system is not running (let's say my PC is turned off, or any other form of non-availability):
>
> - When the chat monitoring is online again, only the oldest, non-handled message is handled
> - No other message is processed until the last one is classified/inserted
> - Since we have a list of expenses, instead of directly sending each after being classified, they can be stored in CSV, and sent using the batch functionality
>
> This has a queue behavior, but it's not mandatory that we use a queue system (MQ/Kafka/etc) from the beginning (unless there is a very simple version).
>
> The same is true for the dictionary and other data: we could have this running on SQLite, Redis (specially when moving on to run everything on docker), but not necessary from the beginning (we can start with files, and have changes to other tooling in the roadmap).

---

## Domain Boundaries

### expense-reporter (Z:\Meu Drive\controle\code\expense-reporter)
- **Owns:** Classification logic, training data, Ollama HTTP calls, Excel writing
- **Why:** Classification is an expense-reporter feature. The LLM is an implementation detail.
- **New commands:** `classify`, `auto`, `batch-auto`

### llm repo (/mnt/i/workspaces/llm)
- **Owns:** Ollama platform, MCP bridge, benchmarking, development scaffolding
- **Why:** Provides the "how to use local LLMs" patterns; reusable across projects
- **Stores:** Auto-category analysis artifacts (reference), prompt engineering patterns, model benchmarks

### Telegram layer (Layer 6 — future)
- **Owns:** Chat monitoring, inline keyboard UI, queue processing, offline backlog
- **Depends on:** expense-reporter classify/add commands

---

## Iterative Build Plan

### Phase 0: MVP — CLI classify (Layer 5.3)
- expense-reporter `classify "item;DD/MM;value"` → returns top-N subcategories with confidence
- Uses Ollama HTTP API with structured output (`format` param)
- Training data: feature_dictionary_enhanced.json + correction rules
- Entry point: 3-field expense string (no subcategory)
- **Testable immediately from terminal**

### Phase 1: Auto pipeline (Layer 5.3-5.4)
- expense-reporter `auto "item;DD/MM;value"` → classify + add if HIGH confidence, else prompt
- expense-reporter `batch-auto expenses.csv` → classify all, split into classified.csv + review.csv
- Error logging for all phases (parse, classify, resolve, write)
- Correction log: when user overrides classification, log {input, predicted, actual}
- **Replaces the manual "add subcategory" step in Notepad++ workflow**

### Phase 2: Few-shot learning (Layer 5.5)
- Correction log feeds back into classification prompts
- Per-request: find top-K similar historical expenses by keyword match
- Inject as few-shot examples alongside feature dictionary
- **Accuracy improves over time without retraining**

### Phase 3: MCP exposure (Layer 5 → 6 bridge)
- MCP tool in llm repo wraps expense-reporter classify/add
- Claude Code can manage expenses conversationally
- Scaffolding pattern documented for future tools

### Phase 4: Telegram integration (Layer 6)
- Chat monitoring (polling or webhook)
- Inline keyboard for category selection (clickable options, not typed)
- Queue behavior: process oldest first, one at a time
- Offline backlog: store unprocessed, resume on reconnect
- PT-BR responses: "Essa despesa parece ser uma dessas categorias:"
- Batch accumulation: collect classified expenses in CSV, batch-insert periodically

### Phase 5: Persistence evolution (Layer 7+)
- File-based → SQLite (training data, correction log, expense queue)
- SQLite → Redis (when Docker deployment happens)
- Full RAG with embeddings for classification (if accuracy needs improvement)

---

## Technical Notes

### Structured Output
Ollama `format` param guarantees valid JSON — 100% reliable, no speed penalty.
Classification response schema:
```json
{
  "subcategory": "Uber/Taxi",
  "confidence": 0.95,
  "alternatives": [
    {"subcategory": "Combustível", "confidence": 0.15},
    {"subcategory": "Diversos", "confidence": 0.05}
  ]
}
```

### Training Data Available
- 694 historical expenses (training_data_complete.json)
- 229 keywords with TF-IDF scores (feature_dictionary_enhanced.json)
- 68 subcategories across 16 top-level categories
- Correction rules: DME→utility, Unimed→health insurance, Layla→therapist, Anita→Empréstimo, Algar→internet
- Algorithm: pattern rules (80%) + statistical TF-IDF (20%) + value Gaussian scoring

### Model Options (to benchmark)
- Qwen3-8B (think: false): Good reasoning, 51-56 tok/s
- Qwen2.5-Coder-7B: Fastest (63-67 tok/s), but coding-optimized not classification (can try creating another persona)
- Classification is a structured-output task — may favor Qwen3-8B's reasoning

### Queue Implementation (future)
Start with file-based queue (one file per pending expense).
Upgrade to SQLite when Docker deployment happens.
No MQ/Kafka needed — volume is low (dozens of expenses/month).

### Expense Persistence & Reproducible ID
Each expense gets a hash ID derived from its raw form (before classification):
```
id = sha256(normalize(item) + "|" + date + "|" + value)[:12]
# "Uber Centro;15/04;35,50" → "a3f7b2c1d8e4"
```
Subcategory excluded from hash — it's determined by the pipeline, not part of identity.
Duplicates (same expense twice): accept collision for now; append sequence number if needed later.

**Status lifecycle (final vision):**
  pending → classifying → classified → confirmed → inserting → inserted
  Error branches: failed_classify | failed_insert | rejected

**Storage evolution:**
- Phase 0: JSON Lines file (expenses_log.jsonl) — appended on successful insert
- Phase 1: Full status tracking, record created on first contact, updated through states
- Phase 2: SQLite (id, raw, status, subcategory, confidence, alternatives, telegram_msg_id, timestamps, error)
- Phase 3: Redis stream or sorted set for queue; SQLite remains as audit log

**Location:** alongside expense-reporter config/workbook (expense domain artifact, not llm domain).
