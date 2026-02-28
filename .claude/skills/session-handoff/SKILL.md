---
name: session-handoff
description: End-of-session workflow that updates all tracking files for continuity across Claude Code sessions. Use when wrapping up a session or when the user says they're done for now.
disable-model-invocation: false
argument-hint: "[optional summary of session focus]"
---

# Session Handoff Skill

You are closing out a Claude Code session. Your job is to ensure the next session can resume seamlessly by updating all tracking files.

## Pre-flight context

Current git state:
!`git status -s 2>/dev/null || echo "not a git repo"`

Recent commits this session:
!`git log --oneline -5 2>/dev/null || echo "no commits"`

Current date:
!`date +%Y-%m-%d 2>/dev/null || echo "unknown"`

## Steps (execute in order)

### 1. Gather session summary

Review the conversation history from this session. Identify:
- **What was done** (tasks completed, files created/modified, decisions made)
- **What was decided** (design choices, deferred items, user preferences expressed)
- **What's next** (pending work, next layer/phase/task to start)
- **New gotchas discovered** (if any)
- **Uncommitted changes** (warn the user if git status shows changes)

If the user provided a summary via `$ARGUMENTS`, use it as the starting point but supplement with details from the conversation.

### 2. Rotate and update session-log.md

First, run `.claude/tools/rotate-session-log.sh` to archive old entries (keeps 3 most recent).
Then read `.claude/session-log.md` and add a new entry at the top (below the header) with:

```markdown
## YYYY-MM-DD - Session N: [Brief Title]

### Context
[How this session started — what was the entry point]

### What Was Done
- [Bulleted list of accomplishments]

### Decisions Made
- [Key decisions, with rationale if non-obvious]

### Next
- [What the next session should start with]

---
```

Update the header's "Current Session" date and "Phase" field.

### 3. Update session-context.md

Read `.claude/session-context.md`. Update:
- The **Current Status** section with latest checkpoint
- The **Decisions Made** section if new decisions were recorded
- The **Technical Learnings** section if new gotchas were discovered

### 4. Verify tracking files ARE the handoff

The tracking files (session-log.md, tasks.md, session-context.md) serve as the handoff. **Do NOT create separate handoff files** (`.claude/session-handoff-*.md`).

Verify that the updates from steps 2-3 contain everything a new session needs:
- **session-log.md** has a "Next" pointer saying what to start with
- **tasks.md** shows which tasks are done and which are pending
- **session-context.md** has current status and active decisions
- **index.md** links to any new archive files created this session

If significant research or findings were produced, ensure they're archived in `.claude/archive/` and indexed in `.claude/index.md` (not left only in conversation context).

### 5. Warn about uncommitted changes

If `git status` shows uncommitted changes, tell the user:
- List the changed files
- Ask if they want to commit before ending
- Do NOT auto-commit — just inform

### 6. Confirm completion

Show a summary table of what was updated, then confirm the session is ready to close.

## Important rules

- Do NOT create any new project files (code, docs, configs) — this skill only updates tracking files
- Do NOT proceed to new work after the handoff — the session is ending
- If the conversation was short or trivial, keep the handoff proportionally brief
- Preserve the format and style of existing tracking files — read them first
