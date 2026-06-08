# runlog.py
#
# Per-run logging module for a deterministic session-handoff pipeline.
# Pure stdlib implementation using pathlib, dataclasses, and datetime.

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


def create_run_dir(repo_root: Path, session_number: int, *, clock: callable = datetime.datetime.now) -> Path:
    """Create a per-run directory under repo_root with session-<N>-<timestamp> name."""
    timestamp = clock().strftime("%Y%m%d-%H%M%S")
    run_dir = repo_root / ".claude" / "local" / "handoff-runs" / f"session-{session_number}-{timestamp}"
    run_dir.mkdir(parents=True, exist_ok=True)
    return run_dir


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
