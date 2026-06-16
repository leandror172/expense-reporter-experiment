# handoff.py — session-handoff CLI
# Three paths:
#   --payload <file>         → stage: validate + ingest + in-memory apply + emit JSON (dir stays -pending)
#   --payload <file> --amend → amend-stage: additive-only (append/checkoff roles), no header write
#   --id <handle>            → promote: find pending run + commit + rename dir + emit JSON
#   --abort <handle>         → abort: rename -pending → -aborted, emit JSON

import argparse
import datetime
import json
import shutil
import subprocess
import sys
from pathlib import Path
from typing import List, Optional

from payload import parse, validate, PayloadError
from registry_io import load_register, RegistryError
from orchestrator import run_handoff, stage_and_apply
from runlog import (
    create_run_dir, find_pending_run, promote_run_dir,
    mark_run_failed, mark_run_aborted, count_runs_by_status, peek_session_number,
    write_input, write_report, RunReport, RunNotFoundError,
)
from mechanics import current_session_number as _current_session_number
from gitio import SubprocessGit


def _parse_args(argv=None):
    parser = argparse.ArgumentParser(description="Session handoff CLI")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--payload", type=str, help="Path to payload file (stage)")
    group.add_argument("--id", type=str, help="Handle of pending run (promote)")
    group.add_argument("--abort", type=str, metavar="HANDLE", help="Abort a pending run by handle")
    parser.add_argument("--amend", action="store_true",
                        help="Amend modifier for --payload: additive-only, no header write")
    parser.add_argument("--repo-root", type=str, default=None)
    parser.add_argument("--registry", type=str, default=None)
    return parser.parse_args(argv)


def _default_repo_root() -> Path:
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True, text=True, check=True,
        )
        return Path(result.stdout.strip())
    except (subprocess.CalledProcessError, FileNotFoundError):
        return Path.cwd()


def _resolve_registry(args, repo_root: Path) -> Path:
    if args.registry:
        return Path(args.registry)
    return repo_root / ".claude/handoff/registry.yaml"


def _handle_from_run_dir(run_dir: Path) -> str:
    for suffix in ("-pending", "-success", "-failed"):
        if run_dir.name.endswith(suffix):
            return run_dir.name[:-len(suffix)]
    raise ValueError(f"run dir has no recognised suffix: {run_dir.name}")


def _build_result(handle: str, status: str, report, run_dir: Path, run_counts: dict) -> dict:
    return {
        "handle": handle,
        "status": status,
        "run_dir": str(run_dir),
        "session_number": report.session_number if report else None,
        "regions": [e.role for e in report.edits] if report and report.edits else [],
        "report_path": str(run_dir / "report.md"),
        "reason": report.reason if report else "",
        "run_counts": run_counts,
    }


def _stage_path(args, repo_root: Path, register: dict, git) -> int:
    """--payload path: validate → ingest → in-memory apply → write report → emit JSON.

    The run dir is created with status="pending" and stays -pending after this call.
    No commit happens here. The --id path commits later.

    With --amend: only append/checkoff roles are allowed; scalars not required; no header write.
    Amend mode is persisted to <run_dir>/mode so promote can detect it without user re-input.
    """
    amend = getattr(args, "amend", False)
    payload_path = Path(args.payload)

    # 1. Parse + validate — failure leaves file at well-known path
    try:
        payload = parse(payload_path.read_text())
    except PayloadError as e:
        print(json.dumps({"status": "validation_failed", "reason": str(e)}))
        return 2

    errors = validate(payload, register, amend=amend)
    if errors:
        print(json.dumps({"status": "validation_failed", "reason": "; ".join(errors)}))
        return 2

    # 2. Ingest: create run dir, COPY payload into run dir (not move yet).
    #    Strategy: copy-then-delete-on-success. The copy is crash-safe: the only
    #    destructive op (unlink of the original) is the very last step, so any crash
    #    before that leaves the author's file intact. The run dir always gets input.md
    #    whether the stage succeeds or fails.
    log_rel = register["header-current-session"]["file"]
    if amend:
        session_number = _current_session_number((repo_root / log_rel).read_text())
    else:
        session_number = peek_session_number(repo_root, log_rel)
    run_dir = create_run_dir(repo_root, session_number, status="pending")
    write_input(run_dir, payload.raw)  # copy — original untouched until success

    # 3. Persist amend flag in sidecar so promote knows the mode without user re-input
    if amend:
        (run_dir / "mode").write_text("amend")

    # 4. In-memory apply; on failure mark run dir as failed (original file stays)
    try:
        _, region_edits = stage_and_apply(repo_root, register, payload, clock=datetime.datetime.now, amend=amend)
        report = RunReport(session_number, False, False, "", True, region_edits)
        write_report(run_dir, report)
    except Exception as e:
        run_dir = mark_run_failed(run_dir)
        print(json.dumps({"status": "stage_failed", "reason": str(e)}))
        return 1

    # 5. Success: free the well-known path (delete original — last destructive op)
    payload_path.unlink()

    run_counts = count_runs_by_status(repo_root)
    print(json.dumps(_build_result(_handle_from_run_dir(run_dir), "stage_ok", report, run_dir, run_counts)))
    return 0


def _already_committed(handle: str, payload, run_dir: Path, repo_root: Path, git) -> bool:
    """Return True if the session title is already in recent git history (idempotency guard).

    When True, the caller should return 0 immediately; the promoted dir and JSON result
    have already been printed as a side-effect of this call.
    """
    commit_suffix = f" — {payload.session_title}"
    if not any(m.startswith("chore(session-handoff): session ") and m.endswith(commit_suffix)
               for m in git.log_messages(5)):
        return False
    success_dir = promote_run_dir(run_dir)
    run_counts = count_runs_by_status(repo_root)
    print(json.dumps(_build_result(handle, "committed", None, success_dir, run_counts)))
    return True


def _promote_path(args, repo_root: Path, register: dict, git) -> int:
    """--id path: find -pending run → idempotency check → run_handoff → promote/fail → emit JSON.

    Reads <run_dir>/mode sidecar to detect amend runs (written by --payload --amend at stage time).
    Amend runs skip the idempotency title-suffix check (they never write a session title to the header).
    """
    handle = args.id

    # 1. Find pending run dir
    try:
        run_dir = find_pending_run(repo_root, handle)
    except RunNotFoundError as e:
        print(json.dumps({"status": "error", "reason": str(e)}))
        return 1

    # 2. Read amend sidecar
    amend = (run_dir / "mode").exists() and (run_dir / "mode").read_text().strip() == "amend"

    # 3. Parse payload
    payload = parse((run_dir / "input.md").read_text())

    # 4. Idempotency check (skipped for amend — no header write, no title in commit message)
    if not amend and _already_committed(handle, payload, run_dir, repo_root, git):
        return 0

    # 5. Full commit path
    report = run_handoff(repo_root, register, payload, git=git, run_dir=run_dir, amend=amend)

    if report.committed:
        final_dir = promote_run_dir(run_dir)
        status = "committed"
    else:
        final_dir = mark_run_failed(run_dir)
        status = "failed"

    run_counts = count_runs_by_status(repo_root)
    print(json.dumps(_build_result(handle, status, report, final_dir, run_counts)))
    return 0 if status == "committed" else 1


def _abort_path(args, repo_root: Path, register: dict, git) -> int:
    """--abort path: find -pending run → rename to -aborted → emit JSON."""
    handle = args.abort
    try:
        run_dir = find_pending_run(repo_root, handle)
    except RunNotFoundError as e:
        print(json.dumps({"status": "error", "reason": str(e)}))
        return 1

    aborted_dir = mark_run_aborted(run_dir)
    run_counts = count_runs_by_status(repo_root)
    print(json.dumps({
        "handle": handle,
        "status": "aborted",
        "run_dir": str(aborted_dir),
        "run_counts": run_counts,
    }))
    return 0


def main(argv=None) -> int:
    args = _parse_args(argv)
    repo_root = Path(args.repo_root or _default_repo_root())
    registry_path = _resolve_registry(args, repo_root)
    try:
        register = load_register(registry_path)
    except RegistryError as e:
        print(json.dumps({"status": "error", "reason": str(e)}))
        return 2
    git = SubprocessGit(repo_root)
    if args.payload:
        return _stage_path(args, repo_root, register, git)
    if args.abort:
        return _abort_path(args, repo_root, register, git)
    return _promote_path(args, repo_root, register, git)


if __name__ == "__main__":
    sys.exit(main())
