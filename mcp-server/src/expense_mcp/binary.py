"""Subprocess wrapper for the expense-reporter Go binary."""

import os
import subprocess
from dataclasses import dataclass
from pathlib import Path


class BinaryNotFoundError(Exception):
    """Raised when the expense-reporter binary cannot be located."""


class BinaryError(Exception):
    """Raised when the binary exits with non-zero status."""

    def __init__(self, exit_code: int, stderr: str):
        self.exit_code = exit_code
        self.stderr = stderr
        super().__init__(f"expense-reporter exited {exit_code}: {stderr}")


@dataclass
class BinaryResult:
    """Result of a successful binary invocation."""

    stdout: str
    stderr: str


# Relative paths from this file to repo landmarks
_REPO_ROOT = Path(__file__).parent.parent.parent.parent
_GO_MODULE = _REPO_ROOT / "expense-reporter"
_DATA_DIR = _REPO_ROOT / "data" / "classification"


def find_binary() -> str:
    """Locate the expense-reporter binary.

    Priority:
    1. EXPENSE_REPORTER_BIN env var (must point to an existing file)
    2. Build from Go module root (expense-reporter/cmd/expense-reporter)
    """
    env_path = os.environ.get("EXPENSE_REPORTER_BIN")
    if env_path:
        if not Path(env_path).is_file():
            raise BinaryNotFoundError(
                f"EXPENSE_REPORTER_BIN={env_path} does not exist"
            )
        return env_path

    # Try to build from Go module
    go_module = _GO_MODULE
    if not go_module.is_dir():
        raise BinaryNotFoundError(
            f"Go module not found at {go_module} and EXPENSE_REPORTER_BIN not set"
        )

    binary_path = go_module / "expense-reporter"
    result = subprocess.run(
        ["go", "build", "-o", str(binary_path), "./cmd/expense-reporter"],
        cwd=str(go_module),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise BinaryNotFoundError(f"go build failed: {result.stderr}")

    return str(binary_path)


def run_binary(
    binary_path: str,
    args: list[str],
    timeout: int = 120,
) -> BinaryResult:
    """Run the expense-reporter binary and return its output.

    Raises BinaryError on non-zero exit code.
    """
    result = subprocess.run(
        [binary_path] + args,
        capture_output=True,
        text=True,
        timeout=timeout,
        cwd=str(_GO_MODULE),
    )

    if result.returncode != 0:
        raise BinaryError(result.returncode, result.stderr.strip())

    return BinaryResult(stdout=result.stdout, stderr=result.stderr)
