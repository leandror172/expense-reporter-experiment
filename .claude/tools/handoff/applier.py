# applier.py

from typing import Any


class ApplierError(Exception):
    """Custom exception for errors in the applier module."""
    pass


def apply(text: str, region: Any, content: str = "") -> str:
    """
    Apply the given content to the text according to the region's write mode.

    :param text: The original text.
    :param region: An object with attributes `mode`, `start`, `end`, and `interior`.
    :param content: The content to apply (not used for checkoff).
    :return: The modified text.
    :raises ApplierError: If the mode is unknown or nomodel.
    """
    if region.mode == "replace":
        return _apply_replace(text, region, content)
    elif region.mode == "prepend":
        return _apply_prepend(text, region, content)
    elif region.mode == "append":
        return _apply_append(text, region, content)
    elif region.mode == "checkoff":
        return _apply_checkoff(text, region)
    elif region.mode == "nomodel":
        raise ApplierError("Nomodel mode is not allowed.")
    else:
        raise ApplierError(f"Unknown write mode: {region.mode}")


def _apply_replace(text: str, region: Any, content: str) -> str:
    """Replace the interior of the region with the given content."""
    return text[:region.start] + content + text[region.end:]


def _apply_prepend(text: str, region: Any, content: str) -> str:
    """Prepend the given content to the start of the region's interior."""
    return text[:region.start] + content + text[region.start:]


def _apply_append(text: str, region: Any, content: str) -> str:
    """Append the given content to the end of the region's interior."""
    return text[:region.end] + content + text[region.end:]


def _apply_checkoff(text: str, region: Any) -> str:
    """Flip the first '[ ]' to '[x]' inside the region's interior."""
    flipped_interior = region.interior.replace("[ ]", "[x]", 1)
    return text[:region.start] + flipped_interior + text[region.end:]
