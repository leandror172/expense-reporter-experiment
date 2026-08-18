#!/usr/bin/env python3
"""
Backfill expense type into classifications.jsonl and expenses_log.jsonl from reviewed.json exports.

When the user exports reviewed.json from the review UI after corrections, the file contains
type information (the expense type: Fixas/Variáveis/Extras/Adicionais) for each reviewed entry.
This tool fills that type back into the log files based on matching entry IDs.

Backfill is PARTIAL by design: only entries that appear in reviewed.json get a type value.
Auto-inserted/batch-auto/add entries have no type (classifier doesn't emit it yet) — those are
handled by Plan B's bare-name fallback. Do not try to backfill unreviewed entries to zero.

Usage:
    python3 backfill-type.py reviewed.json [reviewed2.json ...] \\
        --expenses-log expenses_log.jsonl --classifications classifications.jsonl

Backup files created: *.bak in place. Rewrite is in-place after backup.
"""

import argparse
import json
import shutil
import sys
from collections import defaultdict
from pathlib import Path


def load_reviewed(reviewed_paths):
    """Load type information from reviewed.json exports (one or many)."""
    type_by_id = {}
    for path in reviewed_paths:
        with open(path, "r") as f:
            data = json.load(f)
        for entry in data.get("entries", []):
            entry_id = entry.get("id")
            reviewed = entry.get("reviewed")
            if entry_id and reviewed:
                # Prefer 'type' key (new format), fall back to 'sheet' (legacy)
                typ = reviewed.get("type") or reviewed.get("sheet")
                if typ:
                    type_by_id[entry_id] = typ
    return type_by_id


def backfill_jsonl(log_path, type_by_id):
    """
    Backfill type into a JSONL file. Returns (filled_count, unmatched_count).
    """
    filled = 0
    unmatched = 0
    lines = []

    with open(log_path, "r") as f:
        for line_num, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                lines.append("")
                continue

            try:
                entry = json.loads(line)
            except json.JSONDecodeError as e:
                print(f"Warning: line {line_num} is not valid JSON: {e}")
                lines.append(line)
                continue

            entry_id = entry.get("id")
            if entry_id and entry_id in type_by_id:
                # Fill type if not already present
                if "type" not in entry or not entry.get("type"):
                    entry["type"] = type_by_id[entry_id]
                    filled += 1
            elif entry_id:
                unmatched += 1

            lines.append(json.dumps(entry))

    # Write back
    with open(log_path, "w") as f:
        for line in lines:
            if line:
                f.write(line + "\n")

    return filled, unmatched


def main():
    parser = argparse.ArgumentParser(
        description="Backfill expense type into log files from reviewed.json exports."
    )
    parser.add_argument(
        "reviewed_files",
        nargs="+",
        help="Path(s) to reviewed.json file(s) containing type information",
    )
    parser.add_argument(
        "--expenses-log",
        required=True,
        help="Path to expenses_log.jsonl (will be updated in-place after backup)",
    )
    parser.add_argument(
        "--classifications",
        required=True,
        help="Path to classifications.jsonl (will be updated in-place after backup)",
    )

    args = parser.parse_args()

    # Load type data
    type_by_id = load_reviewed(args.reviewed_files)
    if not type_by_id:
        print("Warning: no type data found in reviewed.json files")
        return 0

    print(f"Loaded {len(type_by_id)} type entries from reviewed.json files")

    for log_path in [args.expenses_log, args.classifications]:
        log_file = Path(log_path)
        if not log_file.exists():
            print(f"Skipping {log_path}: file not found")
            continue

        # Backup
        backup_path = log_file.with_suffix(log_file.suffix + ".bak")
        shutil.copy2(log_file, backup_path)
        print(f"Backed up: {log_file} → {backup_path}")

        # Backfill
        filled, unmatched = backfill_jsonl(log_path, type_by_id)
        print(f"  {log_file.name}: {filled} entries filled, {unmatched} no match")

    print("\nDone. Verify line counts with: wc -l before.bak after.jsonl")
    print("They should be identical (no lines added/lost).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
