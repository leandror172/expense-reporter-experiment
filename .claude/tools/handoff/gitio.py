# gitio.py
#
# Tiny git adapter for the handoff orchestrator. Injected so the orchestration
# logic can be unit-tested with a fake, while production uses real git. All
# methods run with cwd = repo_root; paths are repo-relative.

import subprocess
from typing import List


class SubprocessGit:
    """Real-git implementation of the adapter the orchestrator depends on."""

    def __init__(self, repo_root):
        self.repo_root = str(repo_root)

    def _run(self, args: List[str]) -> subprocess.CompletedProcess:
        return subprocess.run(
            ["git", *args],
            cwd=self.repo_root,
            capture_output=True,
            text=True,
            check=True,
        )

    def is_clean(self, paths: List[str]) -> bool:
        """True if none of the given repo-relative paths have uncommitted changes."""
        result = self._run(["status", "--porcelain", "--", *paths])
        return result.stdout.strip() == ""

    def add(self, paths: List[str]) -> None:
        self._run(["add", "--", *paths])

    def commit(self, message: str) -> None:
        self._run(["commit", "-m", message])

    def checkout(self, paths: List[str]) -> None:
        """Restore the given paths to HEAD (the rollback primitive)."""
        self._run(["checkout", "--", *paths])

    def status_short(self) -> str:
        return self._run(["status", "--porcelain"]).stdout

    def log_messages(self, n: int = 5) -> List[str]:
        """Return the subject lines of the last n commits."""
        result = self._run(["log", f"--format=%s", f"-{n}"])
        return result.stdout.strip().splitlines()
