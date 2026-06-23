"""Throwaway WS-A merge tool: merge per-year expense-log JSONL files into one
multi-year JSONL, rewriting each record's DD/MM date to DD/MM/YYYY.

Operates on gitignored personal financial data — never prints record values,
items, or amounts; only non-sensitive counts. Inputs are never modified.
"""

import json
import pathlib
import sys
from typing import Dict, List

BASE_DIR = pathlib.Path("/mnt/i/workspaces/expenses/code/expense-reporter")

INPUT_FILES: Dict[str, int] = {
    "expenses_log-2022.jsonl": 2022,
    "expenses_log-2023.jsonl": 2023,
    "expenses_log-2024.jsonl": 2024,
    "expenses_log.jsonl": 2025,
}

OUTPUT_FILE = BASE_DIR / "expenses_log-allyears.jsonl"


def validate_input_files() -> None:
    """Exit non-zero if any declared input file is missing (no partial merge)."""
    for filename in INPUT_FILES:
        if not (BASE_DIR / filename).exists():
            print(f"ERROR: missing required input file: {filename}", file=sys.stderr)
            sys.exit(1)


def read_jsonl(path: pathlib.Path) -> List[Dict]:
    """Return parsed objects from a JSONL file, skipping blank lines."""
    with path.open(encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def rewrite_date(obj: Dict, year: int) -> Dict:
    """Return obj with a DD/MM date promoted to DD/MM/YYYY; other dates untouched."""
    date = obj.get("date")
    if isinstance(date, str) and date.count("/") == 1:
        obj["date"] = f"{date}/{year}"
    return obj


def collect_records() -> List[Dict]:
    """Read every input file in order, rewriting dates with that file's year."""
    records: List[Dict] = []
    for filename, year in INPUT_FILES.items():
        file_records = read_jsonl(BASE_DIR / filename)
        for obj in file_records:
            records.append(rewrite_date(obj, year))
        print(f"{filename}: {len(file_records)} records -> year {year}")
    return records


def write_jsonl(records: List[Dict], path: pathlib.Path) -> None:
    """Write records as one compact JSON object per line (UTF-8, accents kept)."""
    with path.open("w", encoding="utf-8") as f:
        for obj in records:
            f.write(json.dumps(obj, ensure_ascii=False) + "\n")


def main() -> None:
    validate_input_files()
    records = collect_records()
    write_jsonl(records, OUTPUT_FILE)
    print(f"\nTotal: {len(records)} records -> {OUTPUT_FILE}")


if __name__ == "__main__":
    main()
