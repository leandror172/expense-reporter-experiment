#!/usr/bin/env python3
"""Look up the category for one or more subcategories from the canonical taxonomy.

Source: data/classification/feature_dictionary_enhanced.json (`category_mapping` key)
Loaded at runtime by internal/classifier/classifier.go:LoadTaxonomy().

Usage:
  .claude/tools/lookup-category.py Padaria Combustível Uber/Taxi
  .claude/tools/lookup-category.py --list      # print the full mapping (sorted)

Exit codes:
  0 — always (missing keys are reported as "(not found)", not an error)
  2 — bad usage / taxonomy file missing
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

TAXONOMY_REL = "data/classification/feature_dictionary_enhanced.json"


def repo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in (here.parent, *here.parents):
        if (parent / TAXONOMY_REL).exists():
            return parent
    print(f"error: could not locate {TAXONOMY_REL} from {here}", file=sys.stderr)
    sys.exit(2)


def load_mapping() -> dict[str, str]:
    path = repo_root() / TAXONOMY_REL
    with path.open() as f:
        data = json.load(f)
    mapping = data.get("category_mapping")
    if not isinstance(mapping, dict):
        print(f"error: 'category_mapping' missing or wrong type in {path}", file=sys.stderr)
        sys.exit(2)
    return mapping


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print(__doc__, file=sys.stderr)
        return 2

    mapping = load_mapping()

    if argv[1] == "--list":
        for sub in sorted(mapping):
            print(f"{sub} -> {mapping[sub]}")
        return 0

    width = max(len(s) for s in argv[1:])
    for sub in argv[1:]:
        cat = mapping.get(sub, "(not found)")
        print(f"{sub:<{width}} -> {cat}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
