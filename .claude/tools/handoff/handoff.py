# handoff.py
"""
CLI entrypoint for session-handoff pipeline.
"""

import argparse
import subprocess
import sys
from pathlib import Path
from typing import Optional, List, Any

from payload import parse, validate, PayloadError
from registry_io import load_register, RegistryError
from orchestrator import run_handoff
from gitio import SubprocessGit


def _parse_args(argv: Optional[List[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Session handoff CLI")
    parser.add_argument("--payload", required=True, help="Path to payload markdown file")
    parser.add_argument(
        "--repo-root",
        default=None,
        help="Repository root directory; defaults to git root or current working directory"
    )
    parser.add_argument(
        "--registry",
        default=None,
        help="Path to registry.yaml; defaults to <repo_root>/.claude/handoff/registry.yaml"
    )
    parser.add_argument("--dry-run", action="store_true", help="Run in dry-run mode")
    return parser.parse_args(argv)


def _default_repo_root() -> Path:
    try:
        repo_root = Path(
            subprocess.run(["git", "rev-parse", "--show-toplevel"],
                          capture_output=True, text=True, check=True).stdout.strip()
        )
        return repo_root
    except (subprocess.CalledProcessError, FileNotFoundError):
        return Path.cwd()


def _resolve_registry(args: argparse.Namespace, repo_root: Path) -> Path:
    if args.registry:
        return Path(args.registry)
    registry_path = repo_root / ".claude/handoff/registry.yaml"
    return registry_path


def print_summary(report: Any, git: SubprocessGit, dry_run: bool) -> None:
    status_line = ""
    if report.committed:
        status_line = f"committed (session {report.session_number})"
    elif dry_run and report.verify_ok:
        status_line = f"dry-run OK (session {report.session_number}): validated, not written"
    elif dry_run:
        status_line = f"dry-run FAILED (session {report.session_number}): {report.reason}"
    else:
        status_line = f"rolled back: {report.reason}"
    
    print(status_line)
    print("verify: ok" if report.verify_ok else "verify: FAILED")
    
    if report.edits:
        print("regions touched:")
        for edit in report.edits:
            print(f"  - {edit.role} ({edit.mode})")
    
    try:
        status = git.status_short()
        if status.strip():
            print(f"warning: uncommitted changes outside tracking files:\n{status}")
    except Exception:
        pass  # status check is best-effort; never block the summary on it


def main(argv: Optional[List[str]] = None) -> int:
    args = _parse_args(argv)
    
    payload_path = Path(args.payload)
    if not payload_path.exists():
        print(f"error: payload file {payload_path} does not exist", file=sys.stderr)
        return 1
    
    try:
        payload_text = payload_path.read_text()
        payload = parse(payload_text)
    except PayloadError as e:
        print(f"error: cannot parse payload: {e}", file=sys.stderr)
        return 2
    
    registry_path = _resolve_registry(args, repo_root=Path(args.repo_root or _default_repo_root()))
    
    try:
        register = load_register(registry_path)
    except RegistryError as e:
        print(f"error: {e}", file=sys.stderr)
        return 2
    
    errors = validate(payload, register)
    if errors:
        print("validation failed:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 2
    
    repo_root = Path(args.repo_root or _default_repo_root())
    git = SubprocessGit(repo_root)
    
    try:
        report = run_handoff(
            repo_root=repo_root,
            register=register,
            payload=payload,
            git=git,
            dry_run=args.dry_run
        )
    except Exception as e:
        print(f"error: {e}", file=sys.stderr)
        return 1
    
    print_summary(report, git, args.dry_run)
    
    if report.committed or (args.dry_run and report.verify_ok):
        return 0
    else:
        return 1


if __name__ == "__main__":
    sys.exit(main())
