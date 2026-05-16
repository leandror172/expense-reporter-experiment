<!-- ref:local-model-retry-patterns -->
# Local Model Verdict & Retry Patterns

## Step 1: Record a verdict for every local model call

After receiving output from `generate_code` or `ask_ollama`, record one of:

Verdict scale: 2 = accepted · 1 = improved · 0 = rejected

- **2** — used as-is. Note the prompt that worked.
- **1** — used with modifications. Note what changed and why.
- **0** — not usable. Note the failure reason (logic error / wrong API / off-task).

On verdicts 2 or 1, add a rough token estimate:
`2 — ~300 est. Claude tokens saved` (formula: `(prompt chars + response chars) / 4`)

## Step 2: If output is imperfect, classify the defect

Do NOT use a line-count threshold. Instead, assess three dimensions:

| Dimension | Inline signal | Escalate signal |
|---|---|---|
| **Defect type** | Mechanical (slip, typo, wrong import) | Structural (missing sections, wrong pattern) or conceptual (wrong behavior) |
| **Fix scope** | 1-2 isolated sites | 3+ sites or interdependent changes |
| **Prompt cost** | Explaining the fix to Ollama costs more than fixing it | Explaining costs less than fixing |

## Step 3: Choose an action based on the classification

- **Mechanical defect** -> 1 (improved), fix inline. Always.
- **Structural, 1-2 isolated sites** -> Fix inline. 1 if trivial, 0 if effort exceeds describing it.
- **Structural, 3+ sites, interface is definable** -> 0 (rejected). Use stubs-then-Ollama (see below).
- **Structural, 3+ sites, interface not definable** -> 0 (rejected). Write from scratch.
- **Conceptual defect** (correct syntax, wrong behavior) -> 0 (rejected). Write from scratch.
- **Prompt cost tiebreaker:** If explaining the fix would cost more effort than the fix itself, fix inline regardless of scope.

## Step 4: Stubs-then-Ollama retry pattern

Use when Ollama got the structure wrong across 3+ sites but the interface is definable:

1. Verdict the first call as 0 (rejected, with reason).
2. Write stub signatures / interface definitions that anchor the structure.
3. Call Ollama again with the stub file provided via `context_files`.
4. The second call gets its own independent verdict (often 2 (accepted)).

Why this works: stubs embed context *structurally* rather than through natural language description.

## Cold-start grace period

A timeout on the **first call to a model in a session** is not a quality verdict.

- Label it `TIMEOUT_COLD_START` — do not record as 0 (rejected).
- Retry the same prompt immediately — the model is now loaded.
- To prevent cold starts: use `warm_model` MCP tool at session start.

How to tell cold start from real timeout:
- First call after session start or model switch -> likely cold start.
- Subsequent calls to the same loaded model -> real timeout (model is struggling).
<!-- /ref:local-model-retry-patterns -->
