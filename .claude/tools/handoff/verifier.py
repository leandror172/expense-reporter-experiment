# verifier.py

import re
from typing import List, Tuple

class VerifyError(Exception):
    """Exception raised for verification errors."""
    pass


def _region_label(original: str, region) -> str:
    """Build a human-readable label for a region: role(target)@file:line[-line]."""
    lo = original[:region.start].count("\n") + 1
    hi = original[:region.end].count("\n") + 1
    loc = str(lo) if lo == hi else f"{lo}-{hi}"
    role = getattr(region, "role", "") or ""
    target = getattr(region, "target", "") or ""
    file_ = getattr(region, "file", "") or ""
    if role or target or file_:
        return f"{role}({target})@{file_}:{loc}"
    return f"@byte{region.start}-{region.end}"


def _overlap_message(original: str, a, b) -> str:
    return f"{_region_label(original, a)} overlaps {_region_label(original, b)}"


def _segment(region, content) -> str:
    """Generate the segment based on the region's mode and content."""
    if region.mode == "replace" or region.mode == "nomodel":
        return content
    elif region.mode == "prepend":
        return content + region.interior
    elif region.mode == "append":
        return region.interior + content
    elif region.mode == "checkoff":
        return region.interior.replace("[ ]", "[x]", 1)
    else:
        raise VerifyError(f"Unsupported mode: {region.mode}")


def _effective_range(region, mode: str, content: str) -> Tuple[int, int]:
    """(lo, hi) byte range of original text actually mutated by this edit."""
    if mode == "replace" or mode == "nomodel":
        return (region.start, region.end)
    elif mode == "prepend":
        return (region.start, region.start)
    elif mode == "append":
        return (region.end, region.end)
    elif mode == "checkoff":
        offset = region.interior.find("[ ]")
        if offset != -1:
            return (region.start + offset, region.start + offset + 3)
        else:
            return (region.start, region.end)
    else:
        raise VerifyError(f"Unsupported mode: {mode}")


def verify(original: str, modified: str, edits: List[Tuple[object, str]]) -> None:
    """Verify that the modified text matches the expected text derived from edits."""

    # Overlap guard
    effective_ranges = [(region, _effective_range(region, region.mode, content)) for region, content in edits]
    sorted_edits = sorted(effective_ranges, key=lambda e: e[1][0])
    for i in range(1, len(sorted_edits)):
        if sorted_edits[i][1][0] < sorted_edits[i - 1][1][1]:
            a = sorted_edits[i - 1][0]
            b = sorted_edits[i][0]
            raise VerifyError(_overlap_message(original, a, b))

    # Independently re-derive the expected text — use descending sort (matching applier order)
    # so equal-start regions apply in the same sequence the applier uses.
    expected = original
    for region, content in sorted(edits, key=lambda e: e[0].start, reverse=True):
        segment = _segment(region, content)
        expected = expected[:region.start] + segment + expected[region.end:]

    # Check if expected matches modified
    if expected != modified:
        raise VerifyError("Modified text does not match the expected text")

    # Marker check
    def _collect_markers(text: str) -> List[str]:
        return re.findall(r"<!-- ref:[^>]+ -->|<!-- /ref:[^>]+ -->", text)

    original_markers = _collect_markers(original)
    modified_markers = _collect_markers(modified)

    if sorted(original_markers) != sorted(modified_markers):
        raise VerifyError("Ref-marker multisets differ")
