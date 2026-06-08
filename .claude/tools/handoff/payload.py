# payload.py
#
# F7 payload schema + parser + validator for the session-handoff pipeline.
# Pure stdlib. Format: YAML-ish frontmatter (the FIRST TWO `---` lines fence it)
# followed by `## role: <name>` markdown sections. The body may itself contain
# `---` separators and `## ...` headings — only the first two `---` and lines
# matching exactly `## role: <name>` are treated as structural.

import re
from dataclasses import dataclass
from typing import Dict, List, Tuple


class PayloadError(Exception):
    """Raised on structurally un-parseable input."""
    pass


@dataclass(frozen=True)
class HandoffPayload:
    session_title: str
    current_layer: str
    blocks: Dict[str, str]      # role_name -> authored content
    checkoffs: List[str]        # task ids to flip done
    raw: str                    # the payload text, verbatim (== input.md)


def parse(text: str) -> HandoffPayload:
    frontmatter, body = _split_frontmatter(text)
    fields = _parse_frontmatter(frontmatter)
    return HandoffPayload(
        session_title=fields["session_title"],
        current_layer=fields["current_layer"],
        blocks=_parse_sections(body),
        checkoffs=fields["checkoffs"],
        raw=text,
    )


def validate(payload: HandoffPayload, register: Dict[str, Dict[str, str]]) -> List[str]:
    errors: List[str] = []
    errors.extend(_role_errors(payload, register))
    errors.extend(_scalar_errors(payload))
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


# ---- validation -------------------------------------------------------------

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
        errors.append("session_title is empty")
    if not payload.current_layer.strip():
        errors.append("current_layer is empty")
    return errors


def _checkoff_errors(payload: HandoffPayload) -> List[str]:
    return [
        f"malformed checkoff id: {cid}"
        for cid in payload.checkoffs
        if not re.match(r"^T-\d+$", cid)
    ]
