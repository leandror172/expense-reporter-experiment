# payload.py
#
# F7 payload schema + parser + validator for the session-handoff pipeline.
# Pure stdlib. Format: YAML-ish frontmatter (the FIRST TWO `---` lines fence it)
# followed by `## role: <name>` markdown sections. The body may itself contain
# `---` separators and `## ...` headings — only the first two `---` and lines
# matching exactly `## role: <name>` are treated as structural.
#
# P2 — log-entry is now structured sub-slots (context, what_was_done, decisions,
# next, gotchas). Claude emits only values; the pipeline renders all scaffold.
# The old free-block form (where Claude wrote the heading + sections) is rejected.

import re
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple

from mechanics import LogEntry


class PayloadError(Exception):
    """Raised on structurally un-parseable input."""
    pass


@dataclass(frozen=True)
class HandoffPayload:
    session_title: str
    current_layer: str
    blocks: Dict[str, str]          # role_name -> authored content (non-log-entry roles)
    log_entry: Optional[LogEntry]   # structured log-entry slots (None if not in payload)
    checkoffs: List[str]            # task ids to flip done
    raw: str                        # the payload text, verbatim (== input.md)


def parse(text: str) -> HandoffPayload:
    frontmatter, body = _split_frontmatter(text)
    fields = _parse_frontmatter(frontmatter)
    all_blocks = _parse_sections(body)

    # Extract and parse log-entry separately (structured slots, not a free block).
    log_entry: Optional[LogEntry] = None
    other_blocks: Dict[str, str] = {}
    for role, content in all_blocks.items():
        if role == "log-entry":
            log_entry = _parse_log_entry_slots(content)
        else:
            other_blocks[role] = content

    return HandoffPayload(
        session_title=fields["session_title"],
        current_layer=fields["current_layer"],
        blocks=other_blocks,
        log_entry=log_entry,
        checkoffs=fields["checkoffs"],
        raw=text,
    )


def validate(payload: HandoffPayload, register: Dict[str, Dict[str, str]], *, amend: bool = False) -> List[str]:
    errors: List[str] = []
    if amend:
        errors.extend(_amend_role_errors(payload, register))
        # log-entry not allowed in amend mode — would prepend a duplicate session heading
        if payload.log_entry is not None:
            errors.append(
                "amend mode is additive-only; log-entry (mode=prepend) belongs to "
                "the next session's normal run, not an amend"
            )
        # scalars not required in amend mode — header is not written
    else:
        errors.extend(_role_errors(payload, register))
        errors.extend(_scalar_errors(payload))
        # log-entry slot validation (required slots already checked at parse time
        # via LogEntry.__post_init__; this is a belt-and-suspenders check here)
    errors.extend(_checkoff_errors(payload))
    return errors


# ---- frontmatter ------------------------------------------------------------

def _split_frontmatter(text: str) -> Tuple[str, str]:
    """Split into (frontmatter, body) using the first two `---` lines as fences.

    The opening fence must be at the top (no non-empty line before it); the
    second `---` closes the frontmatter. Any later `---` belongs to the body.
    """
    lines = text.splitlines()
    fences = [i for i, line in enumerate(lines) if line.strip() == "---"]
    if len(fences) < 2 or any(line.strip() for line in lines[:fences[0]]):
        raise PayloadError("missing or malformed frontmatter (need opening + closing '---')")
    open_i, close_i = fences[0], fences[1]
    frontmatter = "\n".join(lines[open_i + 1:close_i])
    body = "\n".join(lines[close_i + 1:])
    return frontmatter, body


def _parse_frontmatter(text: str) -> Dict[str, object]:
    """Parse frontmatter `key: value` lines into the recognized fields."""
    raw_fields: Dict[str, str] = {}
    for line in text.splitlines():
        if not line.strip():
            continue
        key, sep, value = line.partition(":")
        if not sep:
            continue
        raw_fields[key.strip()] = value.strip()
    return {
        "session_title": raw_fields.get("session_title", ""),
        "current_layer": raw_fields.get("current_layer", ""),
        "checkoffs": _parse_checkoffs(raw_fields.get("checkoffs", "")),
    }


def _parse_checkoffs(value: str) -> List[str]:
    """Parse a `[T-05, T-06]` bracketed list into ['T-05', 'T-06']; empty -> []."""
    content = value.strip()
    if not content.startswith("[") or not content.endswith("]"):
        return []
    items = content[1:-1].split(",")
    return [item.strip() for item in items if item.strip()]


# ---- sections ---------------------------------------------------------------

def _parse_sections(body: str) -> Dict[str, str]:
    """Split the body into {role: content} at exact `## role: <name>` headers.

    Content (including its blank lines, `---` separators and `## ...` headings)
    is preserved verbatim between one role header and the next.
    """
    collected: Dict[str, List[str]] = {}
    current = None
    for line in body.splitlines():
        role = _role_header(line)
        if role is not None:
            if role in collected:
                raise PayloadError(f"duplicate role section: {role}")
            current = role
            collected[role] = []
        elif current is not None:
            collected[current].append(line)
    return {role: "\n".join(lines) for role, lines in collected.items()}


def _role_header(line: str):
    """Return the role name if the line is exactly `## role: <name>`, else None."""
    match = re.match(r"^## role: (\S+)\s*$", line)
    return match.group(1) if match else None


# ---- log-entry slot parser --------------------------------------------------

# The snake_case slot keys used in the payload authoring format.
_LOG_ENTRY_SLOTS = ("context", "what_was_done", "decisions", "next", "gotchas")

# Old-form detection: a `## <date> - Session N:` line inside log-entry.
_OLD_FORM_HEADING = re.compile(r"^## \d{4}-\d{2}-\d{2} [—\-] Session \d+:")


def _parse_log_entry_slots(content: str) -> LogEntry:
    """Parse the structured sub-slots from a log-entry block body.

    Expects `### <slot_key>` headers (snake_case) followed by their content.
    Valid slot keys: context, what_was_done, decisions, next, gotchas.

    Old-form detection: if a `## <date> - Session N:` heading is present,
    raises PayloadError with a clear migration message.

    Required slots: what_was_done, next (ValueError from LogEntry.__post_init__
    is re-raised as PayloadError with a specific message).
    """
    lines = content.splitlines()

    # Detect old-form: Claude-written heading inside log-entry
    for line in lines:
        if _OLD_FORM_HEADING.match(line.strip()):
            raise PayloadError(
                "log-entry contains a '## <date> - Session N:' heading — "
                "the pipeline now renders the heading automatically. "
                "Replace the log-entry body with structured slots "
                "(### context / ### what_was_done / ### decisions / ### next / ### gotchas)."
            )

    # Also detect old-form by Title-Case section headers
    _title_case_headers = {"### Context", "### What Was Done", "### Decisions Made", "### Next", "### Gotchas"}
    found_title_case = [line.strip() for line in lines if line.strip() in _title_case_headers]
    if found_title_case:
        raise PayloadError(
            f"log-entry uses old-form Title-Case section headers "
            f"({', '.join(found_title_case)}). "
            "Replace with snake_case slot keys: "
            "### context / ### what_was_done / ### decisions / ### next / ### gotchas. "
            "The pipeline now renders all section headings; supply only values."
        )

    # Parse ### <slot_key> sub-sections
    slots: Dict[str, List[str]] = {k: [] for k in _LOG_ENTRY_SLOTS}
    current_slot: Optional[str] = None

    for line in lines:
        slot = _slot_header(line)
        if slot is not None:
            if slot not in _LOG_ENTRY_SLOTS:
                raise PayloadError(
                    f"unknown log-entry slot: '{slot}'. "
                    f"Valid slots: {', '.join(_LOG_ENTRY_SLOTS)}."
                )
            current_slot = slot
        elif current_slot is not None:
            slots[current_slot].append(line)

    # Strip leading/trailing blank lines from each slot's content
    def _strip_lines(lines_list: List[str]) -> List[str]:
        # Remove leading/trailing empty lines
        while lines_list and not lines_list[0].strip():
            lines_list.pop(0)
        while lines_list and not lines_list[-1].strip():
            lines_list.pop()
        return lines_list

    context_lines = _strip_lines(slots["context"])
    context_text = "\n".join(context_lines).strip()

    def _parse_bullets(lines_list: List[str]) -> List[str]:
        """Extract bullet items from lines; strip '- ' prefix."""
        result = []
        for line in _strip_lines(list(lines_list)):
            stripped = line.strip()
            if stripped.startswith("- "):
                result.append(stripped[2:])
            elif stripped:
                result.append(stripped)
        return result

    what_was_done = _parse_bullets(slots["what_was_done"])
    decisions = _parse_bullets(slots["decisions"])
    next_items = _parse_bullets(slots["next"])
    gotchas = _parse_bullets(slots["gotchas"])

    try:
        return LogEntry(
            context=context_text,
            what_was_done=what_was_done,
            decisions=decisions,
            next=next_items,
            gotchas=gotchas,
        )
    except ValueError as exc:
        raise PayloadError(str(exc)) from exc


def _slot_header(line: str) -> Optional[str]:
    """Return the slot key if the line is `### <slot_key>`, else None."""
    match = re.match(r"^### (\S+)\s*$", line)
    return match.group(1) if match else None


# ---- validation -------------------------------------------------------------

# Amend mode is additive-only: only append and checkoff are permitted.
# prepend (log-entry) is excluded — it would add a second "## Session N" heading for an
# already-committed session, creating duplicate-heading pollution.
_AMEND_ALLOWED_MODES = {"append", "checkoff"}


def _amend_role_errors(payload: HandoffPayload, register: Dict[str, Dict[str, str]]) -> List[str]:
    """In amend mode, only append/checkoff write_modes are allowed."""
    errors = []
    for role in payload.blocks:
        if role not in register:
            errors.append(f"unknown role: {role}")
        elif register[role].get("write_mode") == "nomodel":
            errors.append(f"role '{role}' is nomodel — header fields come from scalars, not blocks")
        elif register[role].get("write_mode") not in _AMEND_ALLOWED_MODES:
            mode = register[role].get("write_mode", "unknown")
            errors.append(
                f"amend mode is additive-only; role '{role}' (mode={mode}) belongs to the next session's normal run"
            )
    return errors


def _role_errors(payload: HandoffPayload, register: Dict[str, Dict[str, str]]) -> List[str]:
    errors = []
    for role in payload.blocks:
        if role not in register:
            errors.append(f"unknown role: {role}")
        elif register[role].get("write_mode") == "nomodel":
            errors.append(f"role '{role}' is nomodel — header fields come from scalars, not blocks")
    return errors


def _scalar_errors(payload: HandoffPayload) -> List[str]:
    errors = []
    if not payload.session_title.strip():
        errors.append(
            "session_title is required because this run bumps the Current Session header"
        )
    if not payload.current_layer.strip():
        errors.append(
            "current_layer is required because this run bumps the Current Layer header"
        )
    return errors


def _checkoff_errors(payload: HandoffPayload) -> List[str]:
    return [
        f"malformed checkoff id: {cid}"
        for cid in payload.checkoffs
        if not re.match(r"^[A-Za-z\d][A-Za-z\d.\-]*$", cid)
    ]
