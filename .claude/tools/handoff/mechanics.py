# mechanics.py
#
# F5 "mechanics" module for deterministic session-handoff pipeline.
# Pure stdlib, no async, no httpx.

import re
from datetime import date as dt_date
from pathlib import Path
from subprocess import CompletedProcess, run
from typing import Any, Dict, List, Tuple

from locator import locate, Region


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


def rotate(repo_root: Path, keep: int = 3) -> CompletedProcess:
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
