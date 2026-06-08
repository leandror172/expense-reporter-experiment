# locator.py

import re
from dataclasses import dataclass, field
from typing import Any, Dict, Optional

@dataclass(frozen=True)
class Region:
    kind: str
    mode: str
    start: int
    end: int
    interior: str = ""

class LocatorError(Exception):
    pass

def locate(role: Dict[str, Any], text: str, *, task_id: Optional[str] = None) -> Region:
    locator_type = role["locator"]["type"]
    mode = role["write_mode"]

    if locator_type == "ref_block":
        return _locate_ref_block(role, text)
    elif locator_type == "field":
        return _locate_field(role, text)
    elif locator_type == "structural":
        return _locate_structural(role, text)
    elif locator_type == "checklist":
        return _locate_checklist(role, text, task_id=task_id)
    else:
        raise ValueError(f"Unknown locator type: {locator_type}")

def _locate_ref_block(role: Dict[str, Any], text: str) -> Region:
    key = role["locator"]["key"]
    open_marker = f"<!-- ref:{key} -->"
    close_marker = f"<!-- /ref:{key} -->"

    start_index = text.find(open_marker)
    end_index = text.find(close_marker)

    if start_index == -1 or end_index == -1:
        raise LocatorError("Missing marker(s)")

    if text.count(open_marker) > 1 or text.count(close_marker) > 1:
        raise LocatorError("Duplicate marker(s)")

    interior_start = text.index("\n", start_index) + 1
    interior_end = end_index

    return Region(
        kind="ref_block",
        mode=role["write_mode"],
        start=interior_start,
        end=interior_end,
        interior=text[interior_start:interior_end]
    )

def _locate_field(role: Dict[str, Any], text: str) -> Region:
    label = role["locator"]["label"]
    pattern = re.compile(rf"^\*\*{re.escape(label)}:\*\*\s*(.*)$", re.MULTILINE)

    matches = list(pattern.finditer(text))

    if len(matches) != 1:
        raise LocatorError("Field not found or duplicated")

    match = matches[0]
    start, end = match.span(1)
    return Region(
        kind="field",
        mode=role["write_mode"],
        start=start,
        end=end,
        interior=text[start:end]
    )

def _locate_structural(role: Dict[str, Any], text: str) -> Region:
    pattern = role["locator"]["pattern"]
    occurrence = role["locator"]["occurrence"] - 1
    position = role["locator"]["position"]

    lines = text.splitlines()
    matches = [i for i, line in enumerate(lines) if re.match(pattern, line)]

    if len(matches) < occurrence + 1:
        raise LocatorError("Pattern occurrence out of range")

    match_index = matches[occurrence]
    line_start = sum(len(line) + 1 for line in lines[:match_index])
    line_end = line_start + len(lines[match_index])

    if position == "after":
        return Region(
            kind="structural",
            mode=role["write_mode"],
            start=line_end + 1,
            end=line_end + 1
        )
    elif position == "before":
        return Region(
            kind="structural",
            mode=role["write_mode"],
            start=line_start,
            end=line_start
        )
    else:
        raise ValueError(f"Unknown position: {position}")

def _locate_checklist(role: Dict[str, Any], text: str, *, task_id: Optional[str] = None) -> Region:
    if not task_id:
        raise ValueError("task_id is required for checklist locator")

    pattern = re.compile(rf'^- \[\s\]\s*\({re.escape(task_id)}\)\s*(.*)$', re.MULTILINE)
    matches = list(pattern.finditer(text))

    if len(matches) != 1:
        raise LocatorError("Checklist item not found or duplicated")

    match = matches[0]
    start, end = match.span(0)
    return Region(
        kind="checklist",
        mode=role["write_mode"],
        start=start,
        end=end,
        interior=text[start:end]
    )
