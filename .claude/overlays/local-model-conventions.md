<!-- ref:local-model-conventions -->
# Local Model Usage Conventions

How to call local models (via `generate_code` / `ask_ollama`) well, and how to
evaluate what comes back. Two parts: what to do **before** a call, and what to do
**after** the output arrives.

## Before you call

### How to prompt: describe behavior, not implementation

Send behavioral intent and let the model write the code. If you spell the logic out
in code form, *you* wrote the code and the model only transcribed it — which defeats
the purpose of delegation.

- **Provide:** function signature, package/module context, behavioral rules, edge
  cases, and the test cases the result must satisfy.
- **Do NOT provide:** if-else branches written out, exact call sequences, or literal
  string values to embed.
- Bad: "If id != '': call FindLatestEntry(path, id). If !found: write Fprintf(stderr, ...)"
- Good: "If an id is provided, verify it exists in the log. If not found, warn to
  stderr and continue — don't block the insert."

### When to call: always attempt the local model

Try the local model first for any new file or function with more than ~5 lines of
logic — for the whole session, not just the opening calls. Only skip for truly
trivial edits (adding a flag, a 2-line branch, a one-liner).

- Don't drift into writing code directly once the session is underway.
- A past `0` (rejected) verdict on a class of task is **not** a reason to skip it
  next time — it's a reason to pass *better* context. Richer prompts are how you
  learn what context shape works for that task class.
- When in doubt, use the local model.
- Refactoring after a green test pass is also worth delegating.

### Serialize calls: respect the VRAM ceiling

- **Serial by default** for codegen. 3+ concurrent non-trivial requests exceed the
  12 GB VRAM budget — the box cannot hold three full inferences at once and will
  silently degrade or time out.
- **Parallel only** for tiny, near-identical asks that each finish in seconds.
- **Different models in parallel is worse** than same-model parallel — each model
  must be loaded into VRAM separately. Always call models sequentially, even for
  benchmarking: tier 1 first; on a `0` verdict, then tier 2.

### Context files: pass what defines the behavior

More signal in `context_files` means a higher verdict tier. Include:

- Existing implementations and related modules.
- The tests that define expected behavior — write tests first, then pass them as
  context for the implementation.
- Relevant plan / pattern / architecture / doc sections.
- **Callers of the code being generated.** When promoting code that other code
  calls, include those callers — omitting them caused a `0` verdict where the model
  rewrote the function signature and broke the caller's API.
- Don't delete a file that could serve as a few-shot example until its replacement
  is written and validated — a working file from the same framework is the
  strongest prompt context you have.

## After you call

### Step 1: Record a verdict for every local model call

After receiving output from `generate_code` or `ask_ollama`, record one of:

Verdict scale: 2 = accepted · 1 = improved · 0 = rejected

- **2** — used as-is. Note the prompt that worked.
- **1** — used with modifications. Note what changed and why.
- **0** — not usable. Note the failure reason (logic error / wrong API / off-task).

On verdicts 2 or 1, add a rough token estimate:
`2 — ~300 est. Claude tokens saved` (formula: `(prompt chars + response chars) / 4`)

### Step 2: If output is imperfect, classify the defect

Do NOT use a line-count threshold. Instead, assess three dimensions:

| Dimension | Inline signal | Escalate signal |
|---|---|---|
| **Defect type** | Mechanical (slip, typo, wrong import) | Structural (missing sections, wrong pattern) or conceptual (wrong behavior) |
| **Fix scope** | 1-2 isolated sites | 3+ sites or interdependent changes |
| **Prompt cost** | Explaining the fix to Ollama costs more than fixing it | Explaining costs less than fixing |

### Step 3: Choose an action based on the classification

- **Mechanical defect** -> 1 (improved), fix inline. Always.
- **Structural, 1-2 isolated sites** -> Fix inline. 1 if trivial, 0 if effort exceeds describing it.
- **Structural, 3+ sites, interface is definable** -> 0 (rejected). Use stubs-then-Ollama (see below).
- **Structural, 3+ sites, interface not definable** -> 0 (rejected). Write from scratch.
- **Conceptual defect** (correct syntax, wrong behavior) -> 0 (rejected). Write from scratch.
- **Prompt cost tiebreaker:** If explaining the fix would cost more effort than the fix itself, fix inline regardless of scope.

### Step 4: Stubs-then-Ollama retry pattern

Use when Ollama got the structure wrong across 3+ sites but the interface is definable:

1. Verdict the first call as 0 (rejected, with reason).
2. Write stub signatures / interface definitions that anchor the structure.
3. Call Ollama again with the stub file provided via `context_files`.
4. The second call gets its own independent verdict (often 2 (accepted)).

Why this works: stubs embed context *structurally* rather than through natural language description.

### Cold-start grace period

A timeout on the **first call to a model in a session** is not a quality verdict.

- Label it `TIMEOUT_COLD_START` — do not record as 0 (rejected).
- Retry the same prompt immediately — the model is now loaded.
- To prevent cold starts: use `warm_model` MCP tool at session start.

How to tell cold start from real timeout:
- First call after session start or model switch -> likely cold start.
- Subsequent calls to the same loaded model -> real timeout (model is struggling).

### Retry budget before escalating

The cold-start retry above is a single warm-up retry. Separately, before giving up
on the local model and writing the code yourself:

- Make **at least 3-4 attempts** — improve the prompt and try again, or try the
  next model tier — before escalating to writing it directly.
- Escalate only on explicit `0` (rejected) verdicts, **not on timeouts**. A timeout
  is not a rejection; a model that is loading is not a model that failed.
<!-- /ref:local-model-conventions -->
