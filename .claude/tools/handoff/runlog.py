# runlog.py
#
# Per-run logging module for a deterministic session-handoff pipeline.
# Pure stdlib implementation using pathlib, dataclasses, and datetime.

import shutil
from dataclasses import dataclass
from pathlib import Path
import datetime


@dataclass(frozen=True)
class RegionEdit:
    """A region edit with role, mode, before, and after content."""
    role: str
    mode: str
    before: str
    after: str


@dataclass(frozen=True)
class RunReport:
    """Run report with session status and edits."""
    session_number: int
    committed: bool
    rolled_back: bool
    reason: str
    verify_ok: bool
    edits: list[RegionEdit]


class RunNotFoundError(Exception):
    """Raised when a pending run dir cannot be found by handle."""
    pass


def create_run_dir(repo_root: Path, session_number: int, *, status: str, clock: callable = datetime.datetime.now) -> Path:
    """Create a per-run directory under repo_root with session-<N>-<ts>-<status> name."""
    timestamp = clock().strftime("%Y%m%d-%H%M%S")
    run_dir = repo_root / ".claude" / "local" / "handoff-runs" / f"session-{session_number}-{timestamp}-{status}"
    run_dir.mkdir(parents=True, exist_ok=True)
    return run_dir


def find_pending_run(repo_root: Path, handle: str) -> Path:
    """Find the <handle>-pending/ dir under handoff-runs/. Raises RunNotFoundError if missing or ambiguous."""
    runs_folder = repo_root / ".claude" / "local" / "handoff-runs"
    if not runs_folder.exists():
        raise RunNotFoundError(f"no pending run for handle: {handle}")
    matches = [p for p in runs_folder.iterdir()
               if p.name.startswith(handle) and p.name.endswith("-pending")]
    if len(matches) == 0:
        raise RunNotFoundError(f"no pending run for handle: {handle}")
    if len(matches) > 1:
        raise RunNotFoundError(f"ambiguous: multiple pending dirs match handle: {handle}")
    return matches[0]


def promote_run_dir(run_dir: Path) -> Path:
    """Rename <handle>-pending/ to <handle>-success/ using shutil.move. Returns new path."""
    if not run_dir.name.endswith("-pending"):
        raise ValueError(f"not a pending run dir: {run_dir}")
    new_path = run_dir.parent / (run_dir.name[:-len("-pending")] + "-success")
    shutil.move(str(run_dir), str(new_path))
    return new_path


def mark_run_failed(run_dir: Path) -> Path:
    """Rename <handle>-pending/ to <handle>-failed/ using shutil.move. Returns new path."""
    if not run_dir.name.endswith("-pending"):
        raise ValueError(f"not a pending run dir: {run_dir}")
    new_path = run_dir.parent / (run_dir.name[:-len("-pending")] + "-failed")
    shutil.move(str(run_dir), str(new_path))
    return new_path


def count_runs_by_status(repo_root: Path) -> dict:
    """Count run dirs by suffix (-pending/-success/-failed) under handoff-runs/. Missing statuses get 0."""
    runs_folder = repo_root / ".claude" / "local" / "handoff-runs"
    counts = {"pending": 0, "success": 0, "failed": 0}
    if not runs_folder.exists():
        return counts
    for path in runs_folder.iterdir():
        if not path.is_dir():
            continue
        suffix = path.name.rsplit("-", 1)[-1]
        if suffix in counts:
            counts[suffix] += 1
    return counts


def peek_session_number(repo_root: Path, log_rel: str) -> int:
    """Return the next session number by reading the current session log."""
    from mechanics import next_session_number
    return next_session_number((repo_root / log_rel).read_text())


def write_input(run_dir: Path, payload: str) -> Path:
    """Write payload verbatim to input.md in the run directory."""
    input_path = run_dir / "input.md"
    input_path.write_text(payload)
    return input_path


def _format_region(edit: RegionEdit) -> str:
    """Format a single region edit as markdown."""
    return (
        f"- **Role**: {edit.role}, **Mode**: {edit.mode}\n"
        f"  - Before: `{edit.before}`\n"
        f"  - After: `{edit.after}`\n"
    )


def format_report(report: RunReport) -> str:
    """Format a RunReport to markdown with required content."""
    status = "committed" if report.committed else "rolled back"
    verify_line = "✅ Verify OK" if report.verify_ok else "❌ Verify Failed"
    reason_line = f"**Reason**: {report.reason}\n" if report.rolled_back else ""

    regions = "\n".join(_format_region(edit) for edit in report.edits)

    return (
        f"# Session {report.session_number}\n\n"
        f"**Status**: {status}\n"
        f"{reason_line}"
        f"{verify_line}\n\n"
        f"{regions}"
    )


def write_report(run_dir: Path, report: RunReport) -> Path:
    """Format the report and write it to report.md in the run directory."""
    report_path = run_dir / "report.md"
    report_path.write_text(format_report(report))
    return report_path
