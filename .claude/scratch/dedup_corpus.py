#!/usr/bin/env python3
"""Dedup the historical extraction (normalized-*.jsonl, typed, DD/MM/YYYY) against
the existing 694-entry training corpus (type-less, ISO dates), and emit a single
merged, deduplicated dataset.

Join key: (normalized_item, round(value,2), year, month) — tolerant on day, since the
corpus may carry Excel's auto-year/day artifacts the band-anchored extraction discards.
On collision the typed extraction record wins (richer: carries expense `type`).
Corpus-only records (e.g. user_corrections, 2025 rows not re-extracted) are kept; their
`type` is back-filled from taxonomy when (category,subcategory) maps to exactly one type.
"""
import json, os, re, unicodedata
from collections import Counter

OUTDIR = "/home/leandror/workspaces/expenses/old/extracted"
CORPUS = "/mnt/i/workspaces/expenses/code/data/classification/training_data_complete.json"
TAXONOMY = "/mnt/i/workspaces/expenses/code/expense-reporter/config/taxonomy.json"

def norm_item(s):
    s = unicodedata.normalize("NFC", str(s or "")).strip().lower()
    return re.sub(r"\s+", " ", s)

def key(item, value, year, month, day):
    return (norm_item(item), round(float(value), 2), int(year), int(month), int(day))

# (category, subcategory) -> type, only when unambiguous
tax = json.load(open(TAXONOMY))
cs_to_types = {}
for ty in tax["types"]:
    for c in ty["categories"]:
        for sub in c["subcategories"]:
            cs_to_types.setdefault((c["name"], sub), set()).add(ty["name"])
def infer_type(cat, sub):
    t = cs_to_types.get((cat, sub))
    return next(iter(t)) if t and len(t) == 1 else ""

# --- load extraction (typed) ---
SRCFILE = {2022: "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2022.xlsx",
           2023: "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2023.xlsx",
           2024: "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2024.xlsx",
           2025: "Planilha_Normalized_Final_copy.xlsx"}
ext = []
for yr in (2022, 2023, 2024, 2025):
    p = os.path.join(OUTDIR, f"normalized-{yr}.jsonl")
    for line in open(p):
        r = json.loads(line)
        if not r.get("date"):
            r["_key"] = None  # no date -> can't form key; keep, treat as unique
        else:
            d, m, y = r["date"].split("/")
            r["_key"] = key(r["item"], r["value"], y, m, d)
        r["year"] = yr
        r["source"] = f"{SRCFILE[yr]}:{r['type']}"
        r["_src"] = f"extract-{yr}"
        ext.append(r)

# --- load corpus (type-less, ISO dates) ---
corp = []
for e in json.load(open(CORPUS))["expenses"]:
    d = str(e.get("date", ""))
    parts = d.split("-") if "-" in d else None
    k = key(e["item"], e["value"], parts[0], parts[1], parts[2]) if parts and len(parts) >= 3 else None
    corp.append({"item": e["item"], "date": d, "value": float(e["value"]),
                 "type": infer_type(e.get("category", ""), e.get("subcategory", "")),
                 "category": e.get("category", ""), "subcategory": e.get("subcategory", ""),
                 "year": e.get("year"), "source": e.get("source", ""), "_key": k})

# --- dedup ---
ext_keys = {r["_key"] for r in ext if r["_key"]}
# 1. internal extraction dups
ext_seen, ext_dedup, ext_internal_dups = set(), [], 0
for r in ext:
    if r["_key"] and r["_key"] in ext_seen:
        ext_internal_dups += 1
        continue
    if r["_key"]:
        ext_seen.add(r["_key"])
    ext_dedup.append(r)

# 2. corpus rows covered by extraction vs net-new
corp_covered = [c for c in corp if c["_key"] and c["_key"] in ext_keys]
corp_new = [c for c in corp if not (c["_key"] and c["_key"] in ext_keys)]

# 3. merged set = deduped extraction + corpus-only (further internal-deduped)
merged, mseen = [], set()
for r in ext_dedup + corp_new:
    k = r.get("_key")
    if k and k in mseen:
        continue
    if k:
        mseen.add(k)
    merged.append({kk: vv for kk, vv in r.items() if not kk.startswith("_")})

# --- write + report ---
out = os.path.join(OUTDIR, "merged-deduped.jsonl")
with open(out, "w") as f:
    for r in merged:
        f.write(json.dumps(r, ensure_ascii=False) + "\n")

print(f"extraction (normalized):      {len(ext)} records")
print(f"  - internal duplicates:      {ext_internal_dups}")
print(f"  - after internal dedup:     {len(ext_dedup)}")
print(f"corpus:                       {len(corp)} records")
print(f"  - covered by extraction:    {len(corp_covered)}")
print(f"  - net-new (corpus only):    {len(corp_new)}")
print(f"      of which user_corrections: {sum(1 for c in corp_new if c['source']=='user_corrections')}")
print(f"      net-new by year: {dict(Counter(c['year'] for c in corp_new))}")
print(f"\nMERGED deduped total:         {len(merged)}  ->  {out}")
print(f"  with type / type-less:      {sum(1 for r in merged if r.get('type'))} / {sum(1 for r in merged if not r.get('type'))}")
def ryear(r):
    if r.get("year"): return str(r["year"])
    return r["date"].split("/")[2] if r.get("date") else "no-date"
print(f"  merged by year: {dict(sorted(Counter(ryear(r) for r in merged).items()))}")
