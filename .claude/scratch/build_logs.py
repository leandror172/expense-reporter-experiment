#!/usr/bin/env python3
"""Build per-year expenses_log files from merged-deduped.jsonl.

expenses_log is year-implicit (DD/MM), consumed by generate-workbook per --year.
So: one file per historical year (2022/2023/2024), and 2025 merged into the existing
expenses_log.jsonl (deduped by ExpenseEntry id). Schema matches feedback.ExpenseEntry:
{id, item, date(DD/MM), value, subcategory, category, type(omitempty), timestamp}.
id = sha256(lower(trim(item)) + "|" + date + "|" + "%.2f"%value)[:12]  (feedback.GenerateID).
Records with no date are skipped (cannot form DD/MM). Junk items are KEPT (real money).
Backs up the existing 2025 log before merging.
"""
import json, os, shutil, datetime, hashlib

MERGED = "/home/leandror/workspaces/expenses/old/extracted/merged-deduped.jsonl"
LOGDIR = "/mnt/i/workspaces/expenses/code/expense-reporter"
LOG2025 = os.path.join(LOGDIR, "expenses_log.jsonl")
STAMP = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
TS = datetime.datetime.now().replace(microsecond=0).isoformat() + "Z"

def gen_id(item, date, value):
    norm = str(item).strip().lower()
    h = hashlib.sha256(f"{norm}|{date}|{value:.2f}".encode()).hexdigest()
    return h[:12]

def ddmm(date):
    if not date:
        return None
    if "-" in date:                       # ISO YYYY-MM-DD (corpus-origin)
        y, m, d = date.split("-")
    else:                                 # DD/MM/YYYY
        d, m, y = date.split("/")
    return f"{int(d):02d}/{int(m):02d}"

def to_entry(r):
    date = ddmm(r.get("date"))
    if date is None:
        return None
    e = {"id": gen_id(r["item"], date, float(r["value"])),
         "item": r["item"], "date": date, "value": float(r["value"]),
         "subcategory": r.get("subcategory", ""), "category": r.get("category", "")}
    if r.get("type"):
        e["type"] = r["type"]
    e["timestamp"] = TS
    return e

# bucket merged records by year
recs = [json.loads(l) for l in open(MERGED)]
by_year, skipped_nodate = {2022: [], 2023: [], 2024: [], 2025: []}, 0
for r in recs:
    yr = int(r["year"]) if r.get("year") else (int(r["date"].split("/")[2]) if r.get("date") else None)
    e = to_entry(r)
    if e is None:
        skipped_nodate += 1
        continue
    if yr in by_year:
        by_year[yr].append(e)

# historical years -> fresh per-year files (dedup by id within year)
for yr in (2022, 2023, 2024):
    seen, out = set(), []
    for e in by_year[yr]:
        if e["id"] in seen:
            continue
        seen.add(e["id"]); out.append(e)
    path = os.path.join(LOGDIR, f"expenses_log-{yr}.jsonl")
    with open(path, "w") as f:
        for e in out:
            f.write(json.dumps(e, ensure_ascii=False) + "\n")
    print(f"{yr}: wrote {len(out)} -> {os.path.basename(path)}")

# 2025 -> merge into existing log (dedup by id), backup first
existing, existing_ids = [], set()
if os.path.exists(LOG2025):
    shutil.copy2(LOG2025, f"{LOG2025}.bak-{STAMP}")
    print(f"backed up 2025 log -> expenses_log.jsonl.bak-{STAMP}")
    for line in open(LOG2025):
        line = line.strip()
        if not line:
            continue
        o = json.loads(line)
        existing.append(o); existing_ids.add(o.get("id"))
added, seen2025 = 0, set(existing_ids)
with open(LOG2025, "a") as f:
    for e in by_year[2025]:
        if e["id"] in seen2025:
            continue
        seen2025.add(e["id"]); added += 1
        f.write(json.dumps(e, ensure_ascii=False) + "\n")
print(f"2025: existing {len(existing)}, added {added} new (deduped), total {len(existing)+added}")
print(f"skipped no-date across all years: {skipped_nodate}")
