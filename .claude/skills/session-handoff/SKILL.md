---
name: session-handoff
description: End-of-session workflow that updates all tracking files for continuity across Claude Code sessions. Use when wrapping up a session or when the user says they're done for now.
disable-model-invocation: false
argument-hint: "[optional summary of session focus]"
---

# Session Handoff Skill

You are closing out a Claude Code session. The tracking files (`session-log.md`,
`tasks.md`, `session-context.md`) ARE the handoff. Your job is narrow:

> **Decide what changed → author ONE payload file → make ONE call to the pipeline → report.**

A deterministic pipeline (`run-handoff.sh`) owns every mechanic: the date, the
session number, log rotation, locating each region, applying the edits, verifying
nothing outside a registered region moved, committing, and rolling back on failure.
**You do NOT read whole tracking files, compute dates or session numbers, run
rotation, or make per-section Edits.** You decide content; the pipeline does the
surgery. If the pipeline's verifier sees a byte change outside a registered region,
it rolls the whole transaction back — so authoring a clean payload is the whole job.

## What you're feeding: roles, modes, scalars

`run-handoff.sh --payload <file>` reads a payload you write and applies each section
to a **registered region**. A per-repo register (`registry.yaml`) maps each *role* to
a file + location + write mode. You author roles; you never name a file or a line.

Payload-authorable roles:

| Role | Mode | What you author |
|------|------|-----------------|
| `log-entry` | prepend | The new session-log entry, inserted newest-first |
| `current-status` | replace | Full updated interior of `ref:current-status` |
| `active-decisions` | replace | Full updated interior of `ref:active-decisions` |
| `reading-guide` | replace | Full updated interior of `ref:session-reading-guide` |
| `user-prefs` | replace | Full updated interior of `ref:user-prefs` (rarely changes) |
| `tasks-append` | append | Newly-discovered tasks, each a fresh `(T-NN)` line |

Plus two frontmatter **scalars** and a **checkoff list**:

- `session_title` → the "Current Session" header field (the pipeline applies it).
- `current_layer` → the "Current Layer" header field (the pipeline applies it).
- `checkoffs: [T-05, T-08]` → task ids completed this session; the pipeline flips
  `[ ]`→`[x]` by id, wherever they live in `tasks.md`.

The header fields and log rotation are **`nomodel`** — script-owned. **Never put them
in the payload as `## role:` sections.** The scalars above are how you influence them;
the pipeline refuses a `nomodel` role appearing as a block.

## Step 1 — Pre-flight (before you author anything)

1. **The tracking tree must be clean.** Run:
   ```
   test -z "$(git status --porcelain -- .claude/session-log.md .claude/tasks.md .claude/session-context.md)" && echo CLEAN || echo DIRTY
   ```
   If it prints `DIRTY`, **STOP** and ask the user to commit or stash those files
   first. The pipeline aborts on a dirty tracking tree — catching it now avoids
   wasting the whole payload-authoring round-trip.

   The pipeline derives the date and session number itself. You do **not** compute
   them. The stage JSON in Step 4 reports `session_number` so you can confirm it
   looks right after the fact.

## Step 2 — Gather the session summary

**First, seed `what_was_done` from the git log** — run this before reviewing the conversation:

```
.claude/tools/handoff-harvest.sh
```

This emits the commit subjects since the last `chore(session-handoff):` commit (or the
last 20 commits with a stderr note if no handoff commit exists). Use those subjects as the
deterministic skeleton for `### what_was_done` in the `log-entry` block. The model only
adds what the commits don't capture (e.g. a decision rationale, an abandoned approach, a
discovery that left no commit). Do not re-derive the commit list from memory.

From the conversation (seeded by `$ARGUMENTS` if the user supplied a focus), identify:

- **What was done** — commits from `handoff-harvest.sh` (above) plus any non-commit context.
- **What was decided** — design choices and deferred items, with rationale if non-obvious.
- **What's next** — the pending work / next task the following session should start with.
- **New gotchas** — anything surprising worth recording.
- **Uncommitted non-tracking changes** — to warn about at the end (never auto-commit).

Keep everything proportional to the session: a short session gets a short handoff.

## Step 3 — Fetch ONLY the replace-mode interiors you will edit

Replace-mode roles need their CURRENT interior so you can produce the full new one.
Fetch the small bounded block — **not the whole file** — via `ref-lookup.sh`:

```
.claude/tools/ref-lookup.sh current-status
.claude/tools/ref-lookup.sh active-decisions
.claude/tools/ref-lookup.sh session-reading-guide
.claude/tools/ref-lookup.sh user-prefs        # only if it actually changed
```

**Reuse resident interiors — do not re-fetch what you already have.** If a replace-mode
block was already pulled into your context earlier this session (e.g. via a `ref-lookup.sh`
call, a `Read`, or a `resume.sh` run that emitted its content), use that resident copy
as the authoring base. Re-running `ref-lookup.sh` for an already-seen interior duplicates
already-resident content in a second form and wastes context. Only fetch interiors you
have not yet seen this session.

**Strip the marker lines.** `ref-lookup.sh` prints the block WITH its surrounding
`<!-- ref:KEY -->` and `<!-- /ref:KEY -->` lines. The payload section must carry ONLY
the **interior** between them — drop those two lines. Replace mode swaps the interior
in place; if you include the markers you duplicate them and the verifier rolls the
entire run back (it enforces a ref-marker-count invariant).

**Omit any replace-role whose content is unchanged this session.** The pipeline applies
only the roles PRESENT in the payload, so an omitted role is left byte-for-byte untouched
(`user-prefs` is usually omitted). Do not re-author a block just to restate it.

## Step 4 — Author the payload, then stage and promote

Write the payload to `.claude/local/handoff-pending.md` (gitignored). Format:
frontmatter fenced by the **first two** `---` lines, then `## role: <name>` sections.
Section bodies may themselves contain `---` and `##` headings — only the first two
`---` and lines matching exactly `## role: <name>` are structural.

```
---
session_title: <topic for the Current Session header>
current_layer: <full Current Layer value>
checkoffs: [T-08, T-12]
---
## role: log-entry

### context
<how this session started / its entry point (optional — omit if obvious)>

### what_was_done
- <accomplishment>
- <accomplishment>

### decisions
- <key decision, rationale if non-obvious (optional)>

### next
- <what the next session starts with>

### gotchas
- <surprise worth recording (optional)>

## role: current-status
<full updated interior of ref:current-status>

## role: reading-guide
<full updated interior of ref:session-reading-guide>

## role: tasks-append
- [ ] (T-NN) **<short label>** — <newly discovered task>
```

Notes on authoring:

**`log-entry` — structured slots (pipeline renders all scaffold):**
- Use **snake_case** sub-headers (`### context`, `### what_was_done`, etc.) — the
  pipeline renders the `### Context`, `### What Was Done`, `### Decisions Made`,
  `### Next`, `### Gotchas` headers and bullet formatting. You supply only the values.
- **Do NOT write** the `## <date> - Session N: <title>` heading. The pipeline derives
  the date and session number and renders the heading from them + `session_title`.
- Required slots: `### what_was_done` (non-empty list) and `### next` (non-empty list).
- Optional slots omitted entirely when empty: `### context`, `### decisions`, `### gotchas`.
- `### context` is a plain paragraph (not a bullet list). All other slots are bullet lists.
- `log-entry` is **prepend** — write only the new entry; the pipeline puts it newest-first.

**Replace roles:**
- Carry ONLY the **interior** (the lines between the ref markers, markers stripped —
  see Step 3); the applier swaps the interior in place.

**`tasks-append`:**
- Adds only NEW tasks. Use the `(T-NN)` convention for new entries (one past the highest
  `T-NN` id in tasks.md): `- [ ] (T-NN) **label** — …`. Tasks discussed but not stored
  anywhere belong here — but this is judgment-based, so **list the candidates to the user
  and confirm before including them.**

**`checkoffs`:**
- Accepts any alphanumeric id (`T-01`, `5.R1`, `RUI-4`, `1.0`). The locator finds the
  id anywhere within the first ~40 chars of an unchecked line, so existing tasks with
  `**ID**` or bare-number formats are checkable without reformatting. Do not restate
  completed tasks as prose — the id list is sufficient.

### Stage

```
run-handoff.sh --payload .claude/local/handoff-pending.md
```

The pipeline validates, applies in-memory, and emits JSON to stdout. Parse it:

- **`status: stage_ok`** — run staged. Check `session_number` matches your log entry.
  The `regions` list names each role that was applied. The payload file is removed from
  its well-known path (moved into the run dir as `input.md`).
- **`status: validation_failed`** — payload has a schema error (e.g. missing scalar,
  unknown role). The payload file is **untouched** — re-edit it and re-run.
- **`status: stage_failed`** — locate/apply/verify raised an error. The payload file is
  **untouched** (copy-don't-move). The failed run dir contains `input.md` for reference.
  Author fresh content or fix the payload and re-stage.

> In the overlay's **home** repo (where the pipeline lives in source, not installed),
> use `overlays/session-tracking/files/handoff/run-handoff.sh` and pass
> `--registry overlays/session-tracking/files/registry.yaml`.

### Promote

Once `stage_ok`:

```
run-handoff.sh --id <handle>
```

where `<handle>` comes from the `handle` field in the stage JSON. This commits all
touched tracking files, then renames the run directory from `-pending` to `-success`.

### Follow-up (amend mode)

If something was missed **after promote** (e.g. a task to append, a checkoff to flip)
use amend mode instead of out-of-band edits:

```
run-handoff.sh --payload <file> --amend
```

Amend mode:
- **Only `append` and `checkoff` write-mode roles are allowed** — no replace-mode
  blocks, no log-entry (those belong in the next session's normal run).
- Scalars (`session_title`, `current_layer`) are **not required** — the header is not
  rewritten.
- Attaches to the **last committed session** (does not bump the session counter).
- Commit message: `chore(session-handoff): session N — amend`.

Then promote the amend run with `--id <handle>` as usual.

### Abort a pending run

To discard a staged run cleanly:

```
run-handoff.sh --abort <handle>
```

This renames the run dir from `-pending` to `-aborted`. **Never `rm` run dirs manually.**

## Step 5 — Report

Relay the pipeline's JSON: committed (with the session number) or rolled-back
(with the reason), and the regions touched. Then, if `git status` shows uncommitted
changes to **non-tracking** files, list them and ask whether the user wants to commit —
do NOT auto-commit. Finish with a short confirmation that the session is ready to close.

## Important rules

- **One payload, one call.** No per-section Edits, no whole-file reads on the write path.
- **Don't recompute** date, session number, or rotation — the pipeline owns them. Never
  write a `## <date> - Session N:` heading inside `log-entry`; the pipeline renders it.
- **Never** put `nomodel` roles (the `header-*` fields, rotation) in the payload as sections.
- **Use snake_case slot headers** in `log-entry` (`### what_was_done`, not `### What Was Done`).
- **Omit** unchanged replace-roles entirely.
- If the run rolls back, read the reason, fix the payload, and re-run — the tracking
  files were restored, so it is safe to retry.
- Do not create separate `session-handoff-*.md` files; the tracking files are the handoff.
- Do not start new project work after the handoff — the session is ending.
