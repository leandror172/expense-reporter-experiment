# registry_io.py
"""
Loads the session-handoff register's roles mapping from a YAML file.
"""

try:
    import yaml
except ImportError as exc:  # pragma: no cover - environment guard
    raise ImportError(
        "PyYAML is required for the session-handoff pipeline. "
        "Install it with: pip install pyyaml"
    ) from exc
from pathlib import Path
from typing import Dict, Any

class RegistryError(Exception):
    """Raised for errors loading or parsing the registry."""
    pass

def load_register(path: str | Path) -> Dict[str, Dict[str, Any]]:
    """
    Load and parse a YAML file containing a session-handoff register.

    Args:
        path: A string or Path object pointing to the YAML file.

    Returns:
        The roles mapping from the YAML document.

    Raises:
        RegistryError: If the file does not exist, is invalid, or lacks required structure.
    """
    # Convert input to Path
    path = Path(path)
    
    # Check if file exists
    if not path.exists():
        raise RegistryError(f"File not found: {path}")
    
    # Read and parse YAML content
    try:
        with open(path, "r") as file:
            data = yaml.safe_load(file)
    except yaml.YAMLError as exc:
        raise RegistryError(f"Invalid YAML format in {path}") from exc

    # Validate structure
    if not isinstance(data, dict):
        raise RegistryError(f"Document must be a mapping, got {type(data)}")
    
    if "roles" not in data or not isinstance(data["roles"], dict):
        raise RegistryError("Missing 'roles' key or it is not a mapping")

    # Return only the roles mapping
    return data["roles"]
