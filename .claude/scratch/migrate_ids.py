#!/usr/bin/env python3
"""Re-derive every `id` in the two JSONL logs from its own row.

Historically ids were hashed from year-less dates. `expenses_log.jsonl` later had its
dates rewritten to carry years while its ids were deliberately left alone, and
`classifications.jsonl` was never rewritten at all. This makes both files self-consistent
in one step, WITHOUT breaking the cross-file join (records correspond when ids are equal).

Fields are rewritten SURGICALLY inside each original line rather than by re-serialising
the parsed record: the two files mix compact (Go `json.Marshal`) and spaced (Python
`json.dumps`) formatting, so a re-serialise would cosmetically churn whichever half does
not match this script's serializer and destroy any byte-level comparison.

Usage: migrate_ids.py <in-exp> <in-cls> <out-exp> <out-cls>
"""
import hashlib
import json
import re
import sys
from collections import defaultdict


def generate_id(item, date, value):
    """The id as `feedback.GenerateID` defines it: 12 hex chars of sha256(item|date|value)."""
    payload = "%s|%s|%.2f" % (item.strip().lower(), date, value)
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()[:12]


def read_records(path):
    """Return [(raw_line, parsed_dict)] for every non-blank line, in file order."""
    records = []
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            stripped = line.rstrip("\n")
            if stripped.strip():
                records.append((stripped, json.loads(stripped)))
    return records


def has_year(date):
    return len(date.split("/")) == 3


def dates_by_original_id(records):
    """Map each expense record's CURRENT id to the set of dates carried under it."""
    dates = defaultdict(set)
    for _, record in records:
        dates[record["id"]].add(record["date"])
    return dates


def resolved_date(record, dates):
    """The date this record should carry, or a reason it cannot be determined."""
    if has_year(record["date"]):
        return record["date"], None
    candidates = dates.get(record["id"], set())
    if not candidates:
        return None, "no expense row carries id %s" % record["id"]
    if len(candidates) > 1:
        return None, "id %s is ambiguous across years %s" % (record["id"], sorted(candidates))
    return next(iter(candidates)), None


def rewrite_field(line, key, value):
    """Replace one string-valued field inside the raw line, preserving all other bytes."""
    pattern = r'("%s"\s*:\s*")[^"]*(")' % re.escape(key)
    new_line, count = re.subn(pattern, lambda m: m.group(1) + value + m.group(2), line, count=1)
    if count != 1:
        raise ValueError("expected exactly one %r field, found %d in: %s" % (key, count, line[:80]))
    return new_line


def migrated_line(raw, record, date):
    """The record's line with its date and id brought up to date, nothing else touched."""
    line = raw if date == record["date"] else rewrite_field(raw, "date", date)
    return rewrite_field(line, "id", generate_id(record["item"], date, record["value"]))


def assert_only_id_and_date_changed(raw, line):
    """Every other field must survive the surgical rewrite untouched."""
    before, after = json.loads(raw), json.loads(line)
    if list(before) != list(after):
        raise ValueError("key order changed: %s" % line[:80])
    differing = {k for k in before if before[k] != after[k]}
    if not differing <= {"id", "date"}:
        raise ValueError("unexpected field(s) changed %s in: %s" % (sorted(differing), line[:80]))


def pairing(expense_ids, classification_ids):
    """For each classification row, the set of expense LINE INDICES sharing its id."""
    positions = defaultdict(set)
    for index, identifier in enumerate(expense_ids):
        positions[identifier].add(index)
    return [frozenset(positions.get(identifier, ())) for identifier in classification_ids]


def assert_join_preserved(before_exp, before_cls, after_exp, after_cls):
    """The join must survive: same shared-id count, and the SAME pairings."""
    shared_before = len(set(before_exp) & set(before_cls))
    shared_after = len(set(after_exp) & set(after_cls))
    if shared_before != shared_after:
        raise ValueError("shared-id count moved: %d -> %d" % (shared_before, shared_after))
    if pairing(before_exp, before_cls) != pairing(after_exp, after_cls):
        raise ValueError("join pairings changed while the count stayed equal")
    return shared_before


def migrate(records, dates):
    """Return (lines, ids, dates_changed, failures) for one file."""
    lines, ids, changed, failures = [], [], 0, []
    for raw, record in records:
        date, problem = resolved_date(record, dates)
        if problem:
            failures.append("%s (%s): %s" % (record["item"][:40], record["date"], problem))
            continue
        line = migrated_line(raw, record, date)
        assert_only_id_and_date_changed(raw, line)
        lines.append(line)
        ids.append(json.loads(line)["id"])
        changed += date != record["date"]
    return lines, ids, changed, failures


def write_lines(path, lines):
    with open(path, "w", encoding="utf-8") as handle:
        handle.write("\n".join(lines) + "\n")


def main(in_exp, in_cls, out_exp, out_cls):
    expenses, classifications = read_records(in_exp), read_records(in_cls)
    dates = dates_by_original_id(expenses)

    exp_lines, exp_ids, exp_dates_changed, exp_failed = migrate(expenses, dates)
    cls_lines, cls_ids, cls_dates_changed, cls_failed = migrate(classifications, dates)

    failures = exp_failed + cls_failed
    if failures:
        print("UNRESOLVED (%d) — nothing written:" % len(failures), file=sys.stderr)
        for failure in failures[:20]:
            print("   " + failure, file=sys.stderr)
        return 1

    shared = assert_join_preserved(
        [r["id"] for _, r in expenses], [r["id"] for _, r in classifications], exp_ids, cls_ids
    )

    write_lines(out_exp, exp_lines)
    write_lines(out_cls, cls_lines)

    print("expense log        : %d rows, %d ids changed, %d dates changed"
          % (len(exp_lines), sum(1 for (_, r), i in zip(expenses, exp_ids) if r["id"] != i), exp_dates_changed))
    print("classification log : %d rows, %d ids changed, %d dates changed"
          % (len(cls_lines), sum(1 for (_, r), i in zip(classifications, cls_ids) if r["id"] != i), cls_dates_changed))
    print("join preserved     : %d shared ids, pairings identical" % shared)
    return 0


if __name__ == "__main__":
    sys.exit(main(*sys.argv[1:5]))
