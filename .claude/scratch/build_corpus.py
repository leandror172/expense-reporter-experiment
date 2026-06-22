#!/usr/bin/env python3
"""Build the augmented training corpus from merged-deduped.jsonl.

Target schema (matches training_data_complete.json): expenses[] of
{id, item, date(ISO YYYY-MM-DD), value, subcategory, category, source, year}.
Backs up the existing corpus first. Drops records with no date (corpus needs one)
and itemless records (already excluded upstream — guarded here too).
"""
import json, os, shutil, datetime, re

MERGED = "/home/leandror/workspaces/expenses/old/extracted/merged-deduped.jsonl"
CORPUS = "/mnt/i/workspaces/expenses/code/data/classification/training_data_complete.json"
STAMP = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")

def to_iso(date, year):
    if not date:
        return None
    if "-" in date:                      # already ISO (corpus-origin)
        return date
    d, m, y = date.split("/")            # DD/MM/YYYY
    return f"{int(y):04d}-{int(m):02d}-{int(d):02d}"

def is_junk_item(it):
    """Placeholder/non-descriptive item: empty, single char, or no alphanumeric
    (e.g. the '-' dash placeholder). Useless as a training signal."""
    s = str(it or "").strip()
    return len(s) <= 1 or not re.search(r"[A-Za-zÀ-ÿ0-9]", s)

recs = [json.loads(l) for l in open(MERGED)]
out, dropped_nodate, dropped_noitem = [], 0, 0
for r in recs:
    if is_junk_item(r.get("item")):
        dropped_noitem += 1
        continue
    iso = to_iso(r.get("date"), r.get("year"))
    if not iso:
        dropped_nodate += 1
        continue
    out.append({
        "item": r["item"], "date": iso, "value": float(r["value"]),
        "subcategory": r.get("subcategory", ""), "category": r.get("category", ""),
        "source": r.get("source", ""), "year": int(iso[:4]),
    })

# stable order: year, date, then assign sequential ids
out.sort(key=lambda e: (e["year"], e["date"], e["item"]))
for i, e in enumerate(out, 1):
    e_id = i
    e2 = {"id": e_id, **e}
    out[i-1] = e2

cats = {e["category"] for e in out}
subs = {e["subcategory"] for e in out}
from collections import Counter
by_year = dict(sorted(Counter(e["year"] for e in out).items()))

doc = {
    "metadata": {
        "total_expenses": len(out),
        "by_year": by_year,
        "unique_categories": len(cats),
        "unique_subcategories": len(subs),
        "extraction_date": datetime.datetime.now().isoformat(),
        "provenance": "Augmented 2026-06-20: historical workbook extraction (2022-2025) "
                      "deduped + merged with prior 694-entry corpus. See "
                      ".claude/scratch/{extract_old_workbooks,dedup_corpus,build_corpus}.py",
    },
    "expenses": out,
}

# backup then write (only if no prior backup exists — preserve the pristine original)
import glob
if os.path.exists(CORPUS) and not glob.glob(f"{CORPUS}.bak-*"):
    bak = f"{CORPUS}.bak-{STAMP}"
    shutil.copy2(CORPUS, bak)
    print(f"backed up corpus -> {bak}")
else:
    print(f"backup already exists: {glob.glob(f'{CORPUS}.bak-*')}")
with open(CORPUS, "w") as f:
    json.dump(doc, f, ensure_ascii=False, indent=2)

print(f"wrote {CORPUS}")
print(f"  total: {len(out)}  (dropped no-date={dropped_nodate}, itemless={dropped_noitem})")
print(f"  by_year: {by_year}")
print(f"  categories: {len(cats)}  subcategories: {len(subs)}")
