# verifier.py

import re
from typing import List, Tuple

class VerifyError(Exception):
    """Exception raised for verification errors."""
    pass


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


def verify(original: str, modified: str, edits: List[Tuple[object, str]]) -> None:
    """Verify that the modified text matches the expected text derived from edits."""
    
    # Overlap guard
    sorted_edits = sorted(edits, key=lambda e: e[0].start)
    for i in range(1, len(sorted_edits)):
        if sorted_edits[i][0].start < sorted_edits[i - 1][0].end:
            raise VerifyError("Overlapping edit regions detected")

    # Independently re-derive the expected text
    expected = original
    for region, content in reversed(sorted_edits):
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
