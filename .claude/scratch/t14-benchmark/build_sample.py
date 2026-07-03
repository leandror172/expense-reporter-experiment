#!/usr/bin/env python3
"""Build a stratified benchmark sample for the T-14 expense-classifier evaluation."""
import argparse
import json
import random
from collections import Counter, defaultdict
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[3]


def load_taxonomy_paths(taxonomy_path: Path) -> set:
    tax = json.loads(taxonomy_path.read_text(encoding="utf-8"))
    return {
        (typ["name"], cat["name"], sub)
        for typ in tax["types"]
        for cat in typ["categories"]
        for sub in cat["subcategories"]
    }


def load_corpus_entries(corpus_path: Path) -> list:
    return json.loads(corpus_path.read_text(encoding="utf-8"))["expenses"]


def expected_type(entry: dict) -> str:
    return entry["source"].rsplit(":", 1)[-1]


def eligible_entries(entries: list, valid_paths: set) -> list:
    return [
        e for e in entries
        if (expected_type(e), e["category"], e["subcategory"]) in valid_paths
    ]


def normalized_item(item: str) -> str:
    return " ".join(item.lower().split())


def leaky_ids(entries: list) -> set:
    counts = Counter(normalized_item(e["item"]) for e in entries)
    return {e["id"] for e in entries if counts[normalized_item(e["item"])] >= 2}


def group_by_leaf(entries: list) -> dict:
    groups = defaultdict(list)
    for e in entries:
        groups[(e["category"], e["subcategory"])].append(e)
    return dict(groups)


def stratified_sample(groups: dict, n: int, rng: random.Random) -> list:
    remaining = {leaf: list(entries) for leaf, entries in groups.items()}
    sample = []
    while len(sample) < n and any(remaining.values()):
        for leaf in sorted(remaining):
            if len(sample) >= n:
                break
            if remaining[leaf]:
                sample.append(remaining[leaf].pop(rng.randrange(len(remaining[leaf]))))
    return sample


def to_ddmmyyyy(iso_date: str) -> str:
    year, month, day = iso_date.split("-")
    return f"{day}/{month}/{year}"


def sample_record(entry: dict, leaky: set) -> dict:
    return {
        "id": entry["id"],
        "item": entry["item"],
        "value": entry["value"],
        "date": to_ddmmyyyy(entry["date"]),
        "expected_type": expected_type(entry),
        "expected_category": entry["category"],
        "expected_subcategory": entry["subcategory"],
        "leaky": entry["id"] in leaky,
    }


def write_jsonl(records: list, out_path: Path) -> None:
    with out_path.open("w", encoding="utf-8") as f:
        for rec in records:
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--n", type=int, default=300)
    parser.add_argument("--out", type=Path, default=Path(__file__).parent / "sample.jsonl")
    args = parser.parse_args()

    valid_paths = load_taxonomy_paths(REPO_ROOT / "expense-reporter/config/taxonomy.json")
    corpus = load_corpus_entries(REPO_ROOT / "data/classification/training_data_complete.json")
    leaky = leaky_ids(corpus)
    eligible = eligible_entries(corpus, valid_paths)
    sample = stratified_sample(group_by_leaf(eligible), args.n, random.Random(42))
    records = [sample_record(e, leaky) for e in sample]
    write_jsonl(records, args.out)

    leaves = {(r["expected_category"], r["expected_subcategory"]) for r in records}
    print(f"eligible: {len(eligible)}")
    print(f"sampled: {len(records)}")
    print(f"leaves covered: {len(leaves)}")
    print(f"leaky in sample: {sum(r['leaky'] for r in records)}")


if __name__ == "__main__":
    main()
