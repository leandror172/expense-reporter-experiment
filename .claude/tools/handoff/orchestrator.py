# orchestrator.py
#
# F6 — the atomic handoff transaction. Composes the deterministic core:
#   stage (F1 locate) -> apply (F3 / F5 splice) -> verify (F4, combined) ->
#   write -> rotate (F5) -> commit, with git checkout as the rollback net.
#
# Two safety layers: in-memory verify-then-write (bad bytes never hit disk),
# and git checkout for failures only visible after writing (rotate/commit).
# A clean-tree precondition on the tracking files makes the second layer sound.

import datetime
from pathlib import Path
from typing import Callable, Dict, List, Tuple

from locator import locate, Region, LocatorError
from applier import apply as apply_edit
from verifier import verify, VerifyError
from mechanics import compute_header_values, apply_field, rotate as default_rotate, current_session_number
from runlog import (
    peek_session_number,
    write_report,
    RunReport,
    RegionEdit,
)
from payload import HandoffPayload  # F7 schema — defined in payload.py (B4.1)

HEADER_ROLES = ("header-current-session", "header-current-layer")


def run_handoff(
    repo_root,
    register: Dict[str, dict],
    payload: HandoffPayload,
    *,
    git,
    rotate: Callable = default_rotate,
    clock: Callable = datetime.datetime.now,
    run_dir: Path,
    amend: bool = False,
) -> RunReport:
    repo_root = Path(repo_root)
    log_rel = register["header-current-session"]["file"]

    # 1. Precondition: tracking files must be clean (rollback would clobber otherwise).
    touched = _touched_files(register, payload, amend=amend)
    if not git.is_clean(touched):
        return RunReport(
            session_number=peek_session_number(repo_root, log_rel),
            committed=False, rolled_back=False,
            reason="precondition: tracking files have uncommitted changes",
            verify_ok=False, edits=[],
        )

    log_text = (repo_root / log_rel).read_text()
    if amend:
        session_number = current_session_number(log_text)
    else:
        session_number = peek_session_number(repo_root, log_rel)

    # 2. Stage (locate) -> apply in memory -> verify each file's combined edit set (F4).
    try:
        modified_by_file, region_edits = stage_and_apply(
            repo_root, register, payload, clock=clock, amend=amend
        )
    except LocatorError as exc:
        return _fail(run_dir, session_number, f"locate failed: {exc}", verify_ok=False, edits=[])
    except VerifyError as exc:
        return _fail(run_dir, session_number, f"verify failed: {exc}", verify_ok=False, edits=[])

    # 3. Write, rotate, commit — git checkout is the rollback net.
    try:
        _write_all(repo_root, modified_by_file)
        _run_rotation(rotate, repo_root)
        git.add(_commit_paths(repo_root, touched))
        git.commit(_commit_message(session_number, payload, amend=amend))
    except Exception as exc:
        git.checkout(list(modified_by_file.keys()))
        return _fail(run_dir, session_number, f"apply/commit failed: {exc}",
                     verify_ok=True, rolled_back=True, edits=region_edits)

    report = RunReport(session_number, True, False, "", True, region_edits)
    write_report(run_dir, report)
    return report


# ---- staging ----------------------------------------------------------------

def stage_and_apply(
    repo_root, register, payload, *, clock, amend: bool = False
) -> Tuple[Dict[str, str], List[RegionEdit]]:
    """Locate -> apply -> verify, purely in memory. Raises LocatorError / VerifyError.

    The deterministic, side-effect-free half of the transaction.
    Called directly by handoff.py for the --payload (stage) path.
    `verify` is referenced as a module global (not threaded through) so test
    monkeypatching still intercepts.
    """
    grouped = _collect_edits(repo_root, register, payload, clock=clock, amend=amend)

    modified_by_file: Dict[str, str] = {}
    region_edits: List[RegionEdit] = []
    for rel, items in grouped.items():
        original = (repo_root / rel).read_text()
        modified = _apply_all(original, items)
        verify(original, modified, [(region, content) for _, region, content in items])
        modified_by_file[rel] = modified
        region_edits.extend(
            RegionEdit(role, region.mode, region.interior, content)
            for role, region, content in items
        )
    return modified_by_file, region_edits


def _normalize_block(content: str) -> str:
    """Guarantee a single trailing newline on a payload block.

    Block roles (log-entry / replace blocks / tasks-append) are whole-line
    inserts. The payload's LAST `## role:` section has no trailing blank line, so
    payload.py yields content without a closing newline — which would glue the
    splice onto the following line or ref marker. Header fields (nomodel) and
    checkoffs don't pass through here, so their inline content is unaffected.
    """
    return content if content.endswith("\n") else content + "\n"


def _collect_edits(repo_root, register, payload, *, clock, amend: bool = False) -> Dict[str, List[Tuple[str, Region, str]]]:
    cache: Dict[str, str] = {}

    def text_of(rel: str) -> str:
        if rel not in cache:
            cache[rel] = (repo_root / rel).read_text()
        return cache[rel]

    grouped: Dict[str, List[Tuple[str, Region, str]]] = {}

    def add(rel: str, role: str, region: Region, content: str) -> None:
        grouped.setdefault(rel, []).append((role, region, content))

    for role, content in payload.blocks.items():
        role_def = register[role]
        rel = role_def["file"]
        add(rel, role, locate(role_def, text_of(rel)), _normalize_block(content))

    if payload.checkoffs:
        role_def = register["tasks-checkoff"]
        rel = role_def["file"]
        for task_id in payload.checkoffs:
            add(rel, "tasks-checkoff", locate(role_def, text_of(rel), task_id=task_id), "")

    if not amend:
        _add_header_edits(register, payload, text_of, add, clock=clock)
    return grouped


def _add_header_edits(register, payload, text_of, add, *, clock) -> None:
    log_rel = register["header-current-session"]["file"]
    values = compute_header_values(
        text_of(log_rel),
        session_title=payload.session_title,
        current_layer=payload.current_layer,
        date=clock().strftime("%Y-%m-%d"),
    )
    keyed = {"header-current-session": "current_session", "header-current-layer": "current_layer"}
    for role in HEADER_ROLES:
        role_def = register[role]
        region = locate(role_def, text_of(role_def["file"]))
        add(role_def["file"], role, region, values[keyed[role]])


def _apply_all(text: str, items: List[Tuple[str, Region, str]]) -> str:
    for _, region, content in sorted(items, key=lambda it: it[1].start, reverse=True):
        if region.mode == "nomodel":
            text = apply_field(text, region, content)
        else:
            text = apply_edit(text, region, content)
    return text


# ---- side effects -----------------------------------------------------------

def _write_all(repo_root, modified_by_file: Dict[str, str]) -> None:
    for rel, text in modified_by_file.items():
        (repo_root / rel).write_text(text)


def _run_rotation(rotate, repo_root) -> None:
    result = rotate(repo_root)
    if getattr(result, "returncode", 0) != 0:
        raise RuntimeError(f"rotation failed: {getattr(result, 'stderr', '')}")


# ---- helpers ----------------------------------------------------------------

def _touched_files(register, payload, *, amend: bool = False) -> List[str]:
    files = {register[role]["file"] for role in payload.blocks}
    if payload.checkoffs:
        files.add(register["tasks-checkoff"]["file"])
    if not amend:
        for role in HEADER_ROLES:
            files.add(register[role]["file"])
    return sorted(files)


def _commit_paths(repo_root, touched: List[str]) -> List[str]:
    paths = list(touched)
    if (repo_root / ".claude" / "archive").exists():
        paths.append(".claude/archive")
    return paths


def _commit_message(session_number: int, payload: HandoffPayload, *, amend: bool = False) -> str:
    if amend:
        return f"chore(session-handoff): session {session_number} — amend"
    return f"chore(session-handoff): session {session_number} — {payload.session_title}"


def _fail(run_dir, session_number, reason, *, verify_ok, edits, rolled_back=True) -> RunReport:
    report = RunReport(session_number, False, rolled_back, reason, verify_ok, edits)
    write_report(run_dir, report)
    return report
