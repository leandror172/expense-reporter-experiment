# Resume Context Loading — Improvement Ideas

**Date:** 2026-03-23
**Status:** Deferred — document options, revisit after 5.8

## Problem

`resume.sh` outputs a compact summary but doesn't load `session-context.md` into the agent's context.
This means TDD directives, user preferences, and active decisions are missed unless the agent
explicitly reads the file — which it often doesn't.

However, not every session needs full context (e.g., quick one-off questions, unrelated tasks).
Loading it unconditionally wastes tokens.

## Options

### A. Pointer in resume.sh output
`resume.sh` appends: `"For full context: read .claude/session-context.md"`
Cheapest change. Agent still has to choose to read it.

### B. /resume skill
A skill that reads `session-context.md` + runs `resume.sh` together.
User invokes explicitly when continuing prior work.
Keeps the default session start lightweight.

### C. Interactive hook
SessionStart hook asks "Continuing prior work? [y/n]".
On yes: loads full context. On no: lightweight start.
More friction but most accurate.

## Recommendation

Option B feels right — a `/resume` skill that the user invokes when they want full context.
The current `resume.sh` stays as the lightweight default for the hook.
