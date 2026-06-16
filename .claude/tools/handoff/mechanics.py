# mechanics.py
#
# F5 "mechanics" module for deterministic session-handoff pipeline.
# Pure stdlib, no async, no httpx.

import re
from dataclasses import dataclass, field
from datetime import date as dt_date
from pathlib import Path
from subprocess import CompletedProcess, run
from typing import Any, Dict, List, Tuple

from locator import locate, Region


@dataclass
class LogEntry:
    """Structured slots for a session log entry. what_was_done and next are required."""
    what_was_done: List[str]
    next: List[str]
    context: str = ""
    decisions: List[str] = field(default_factory=list)
    gotchas: List[str] = field(default_factory=list)

    def __post_init__(self):
        if not self.what_was_done:
            raise ValueError("log-entry requires a non-empty 'what_was_done' slot")
        if not self.next:
            raise ValueError("log-entry requires a non-empty 'next' slot")


def render_log_entry(
    log_entry: LogEntry,
    *,
    date: str,
    session_number: int,
    session_title: str,
) -> str:
    """Render a LogEntry into the canonical log-entry markdown block.

    Renders ALL scaffold: heading, section headers, bullet formatting, blank lines.
    Optional empty slots are omitted entirely (no empty section headers).
    The returned string ends with exactly one trailing newline (session-86 contract).

    Heading uses a hyphen: '## <date> - Session <N>: <title>'
    (The Current Session header field uses an em-dash — different string, same values.)
    """
    parts: List[str] = []

    # Heading line + blank line
    parts.append(f"## {date} - Session {session_number}: {session_title}\n")
    parts.append("\n")

    # Optional: Context (plain paragraph, not a bullet list)
    if log_entry.context:
        parts.append("### Context\n")
        parts.append("\n")
        parts.append(f"{log_entry.context}\n")
        parts.append("\n")

    # Required: What Was Done (bullet list)
    parts.append("### What Was Done\n")
    parts.append("\n")
    for item in log_entry.what_was_done:
        parts.append(f"- {item}\n")
    parts.append("\n")

    # Optional: Decisions Made (bullet list)
    if log_entry.decisions:
        parts.append("### Decisions Made\n")
        parts.append("\n")
        for item in log_entry.decisions:
            parts.append(f"- {item}\n")
        parts.append("\n")

    # Required: Next (bullet list)
    parts.append("### Next\n")
    parts.append("\n")
    for item in log_entry.next:
        parts.append(f"- {item}\n")
    parts.append("\n")

    # Optional: Gotchas (bullet list)
    if log_entry.gotchas:
        parts.append("### Gotchas\n")
        parts.append("\n")
        for item in log_entry.gotchas:
            parts.append(f"- {item}\n")
        parts.append("\n")

    result = "".join(parts)
    # Guarantee exactly one trailing newline (session-86 contract: no double-newline glue)
    result = result.rstrip("\n") + "\n"
    return result


class MechanicsError(Exception):
    """Exception raised for mechanics errors."""
    pass


def current_session_number(log_text: str) -> int:
    """Return the highest session number found in log headings (the *current* committed session)."""
    heading_numbers = _extract_heading_numbers(log_text)
    if not heading_numbers:
        return 0
    return max(heading_numbers)


def next_session_number(log_text: str) -> int:
    """Scan the session-log text for entry headings shaped like `## <ISO-date> - Session <N>: ...` and return the next session number."""
    heading_numbers = _extract_heading_numbers(log_text)
    if not heading_numbers:
        return 1
    return max(heading_numbers) + 1


def today(clock: callable = dt_date.today) -> str:
    """Return today's date as an ISO `YYYY-MM-DD` string. Accepts an injectable zero-argument clock callable."""
    return clock().isoformat()


def compute_header_values(
    log_text: str,
    *,
    session_title: str,
    current_layer: str,
    date: str
) -> Dict[str, Any]:
    """Build the two session-log header field values and return a dict with keys: 'session_number', 'current_session', and 'current_layer'."""
    session_number = next_session_number(log_text)
    current_session = f"{date} — Session {session_number}: {session_title}"
    return {
        "session_number": session_number,
        "current_session": current_session,
        "current_layer": current_layer
    }


def header_field_edits(
    log_text: str,
    roles: Dict[str, Dict[str, Any]],
    values: Dict[str, Any]
) -> List[Tuple[Region, str]]:
    """Locate the two header fields and pair each with its new value from `values`."""
    edits = []
    for role_name in ["header-current-session", "header-current-layer"]:
        if role_name not in roles:
            continue
        role = roles[role_name]
        region = locate(role, log_text)
        field_value = values.get(
            "current_session" if role_name == "header-current-session" else "current_layer"
        )
        edits.append((region, field_value))
    return edits


def apply_field(text: str, region: Region, value: str) -> str:
    """Replace exactly the bytes that `region` spans with `value`, leaving everything else untouched."""
    return text[:region.start] + value + text[region.end:]


def rotate(repo_root: Path, keep: int = 1) -> CompletedProcess:
    """Run the rotate-session-log.sh script with --keep argument."""
    script_path = repo_root / ".claude" / "tools" / "rotate-session-log.sh"
    result = run(
        [str(script_path), "--keep", str(keep)],
        capture_output=True,
        text=True
    )
    return result


def _extract_heading_numbers(log_text: str) -> List[int]:
    """Extract all session numbers from heading lines in the log text."""
    pattern = re.compile(r"## \d{4}-\d{2}-\d{2} [—\-] Session (\d+):")
    matches = pattern.finditer(log_text)
    return [int(match.group(1)) for match in matches]
