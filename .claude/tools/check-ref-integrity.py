#!/usr/bin/env python3
"""Validate ref block integrity across all *.md files in the project.

Checks:
  ERROR   Dangling ref    — [ref:KEY] used but no <!-- ref:KEY --> block exists
  ERROR   Unclosed block  — <!-- ref:KEY --> exists but <!-- /ref:KEY --> missing in same file
  ERROR   Duplicate def   — <!-- ref:KEY --> appears in more than one file
  WARNING Orphaned block  — <!-- ref:KEY --> exists but nothing references [ref:KEY]

Usage: .claude/tools/check-ref-integrity.sh [--root /abs/path/to/repo]
Exit codes: 0 = clean or warnings only, 1 = errors found
"""

import re
import sys
import pathlib
import argparse

SKIP_DIRS = {".git", "node_modules", ".venv"}

RE_REFERENCED = re.compile(r'\[ref:([a-z0-9-]+)\]')
RE_OPEN       = re.compile(r'<!--\s*ref:([a-z0-9-]+)\s*-->')
RE_CLOSE      = re.compile(r'<!--\s*/ref:([a-z0-9-]+)\s*-->')


def iter_md_files(root: pathlib.Path):
    for p in root.rglob("*.md"):
        if any(part in SKIP_DIRS for part in p.parts):
            continue
        yield p


def parse_file(path: pathlib.Path) -> tuple[set, list, set]:
    """Return (referenced_keys, defined_keys, closed_keys) for one file.

    Content inside fenced code blocks (``` ... ```) is ignored — those are
    template examples, not live references.
    """
    referenced: set[str] = set()
    defined: list[str] = []    # list preserves order for duplicate detection
    closed: set[str] = set()

    in_fence = False
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if line.strip().startswith("```"):
            in_fence = not in_fence
            continue
        if in_fence:
            continue

        for m in RE_REFERENCED.finditer(line):
            referenced.add(m.group(1))
        for m in RE_OPEN.finditer(line):
            defined.append(m.group(1))
        for m in RE_CLOSE.finditer(line):
            closed.add(m.group(1))

    return referenced, defined, closed


def collect(root: pathlib.Path) -> tuple[set, dict, dict]:
    """Scan all *.md files and return aggregate data structures.

    Returns:
        all_referenced: set of all [ref:KEY] keys used anywhere
        defined_in: {key: [relfile, ...]} — where each key is opened
        closed_in:  {relfile: {key, ...}} — which keys are closed per file
    """
    all_referenced: set[str] = set()
    defined_in: dict[str, list[str]] = {}
    closed_in: dict[str, set[str]] = {}

    for md_file in iter_md_files(root):
        relfile = str(md_file.relative_to(root))
        referenced, defined, closed = parse_file(md_file)

        all_referenced.update(referenced)
        for key in defined:
            defined_in.setdefault(key, []).append(relfile)
        if closed:
            closed_in[relfile] = closed

    return all_referenced, defined_in, closed_in


def find_dangling(all_referenced, all_defined, root) -> tuple[list[str], int]:
    """[ref:KEY] used but no <!-- ref:KEY --> block exists anywhere."""
    msgs = []
    for key in sorted(all_referenced - all_defined):
        files = [
            str(f.relative_to(root))
            for f in iter_md_files(root)
            if RE_REFERENCED.search(f.read_text(errors="replace"))
            and key in {m.group(1) for m in RE_REFERENCED.finditer(f.read_text(errors="replace"))}
        ]
        msgs.append(f"  [ref:{key}]  ←  {', '.join(files)}")
    return msgs, len(msgs)


def find_unclosed(defined_in, closed_in) -> tuple[list[str], int]:
    """<!-- ref:KEY --> present in a file but <!-- /ref:KEY --> missing from same file."""
    msgs = []
    for key, relfiles in defined_in.items():
        for relfile in relfiles:
            if key not in closed_in.get(relfile, set()):
                msgs.append(f"  ref:{key}  in  {relfile}")
    return sorted(msgs), len(msgs)


def find_duplicates(defined_in) -> tuple[list[str], int]:
    """<!-- ref:KEY --> opening tag appears in more than one file."""
    msgs = []
    for key, relfiles in sorted(defined_in.items()):
        if len(relfiles) > 1:
            msgs.append(f"  ref:{key}  defined in: {', '.join(relfiles)}")
    return msgs, len(msgs)


def find_orphaned(all_defined, all_referenced, defined_in) -> tuple[list[str], int]:
    """<!-- ref:KEY --> block exists but nothing references it with [ref:KEY]."""
    msgs = []
    for key in sorted(all_defined - all_referenced):
        relfile = defined_in[key][0]
        msgs.append(f"  ref:{key}  in  {relfile}")
    return msgs, len(msgs)


def print_section(title: str, msgs: list[str]):
    print(f"=== {title} ===")
    for msg in msgs:
        print(msg)
    print()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=None,
                        help="Repo root to check (default: two levels above this script)")
    args = parser.parse_args()

    root = pathlib.Path(args.root) if args.root else pathlib.Path(__file__).resolve().parents[2]
    if not root.is_dir():
        print(f"Error: directory not found: {root}", file=sys.stderr)
        sys.exit(1)

    all_referenced, defined_in, closed_in = collect(root)
    all_defined = set(defined_in.keys())

    total_errors = 0
    total_warnings = 0

    dangling_msgs, n = find_dangling(all_referenced, all_defined, root)
    if dangling_msgs:
        print_section("ERRORS: Dangling [ref:KEY] tags (used but no block defined)", dangling_msgs)
        total_errors += n

    unclosed_msgs, n = find_unclosed(defined_in, closed_in)
    if unclosed_msgs:
        print_section("ERRORS: Unclosed ref blocks (<!-- ref:KEY --> without <!-- /ref:KEY -->)", unclosed_msgs)
        total_errors += n

    dup_msgs, n = find_duplicates(defined_in)
    if dup_msgs:
        print_section("ERRORS: Duplicate ref block definitions", dup_msgs)
        total_errors += n

    orphan_msgs, n = find_orphaned(all_defined, all_referenced, defined_in)
    if orphan_msgs:
        print_section("WARNINGS: Orphaned ref blocks (defined but no [ref:KEY] points to them)", orphan_msgs)
        total_warnings += n

    if total_errors == 0 and total_warnings == 0:
        print(f"✓ All ref blocks are consistent ({len(all_defined)} blocks, {len(all_referenced)} references)")
        sys.exit(0)
    elif total_errors == 0:
        print(f"✓ No errors. {total_warnings} orphaned block(s) — consider removing unused refs.")
        sys.exit(0)
    else:
        print(f"✗ {total_errors} error(s), {total_warnings} warning(s) found.")
        sys.exit(1)


if __name__ == "__main__":
    main()
