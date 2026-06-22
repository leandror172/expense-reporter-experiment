#!/usr/bin/env python3
"""One-off: backfill the `type` field onto type-less entries in the expense logs.

Strategy per type-less entry (keyed on NFC-normalized category+subcategory):
  1. If the taxonomy maps (category, subcategory) to exactly ONE type -> assign it.
  2. If it maps to MULTIPLE types (e.g. Pets/Orion under Variáveis AND Extras) ->
     disambiguate from the type used by TYPED entries sharing the same (cat, sub).
     Assign only if those typed twins agree on one type; otherwise FLAG.
  3. If (category, subcategory) is absent from the taxonomy -> FLAG (cannot backfill).

Dry-run by default (prints proposed assignments). Pass --apply to rewrite the logs
in place (timestamped .bak-<ts> backup written first).
"""
import json, os, sys, unicodedata, datetime
from collections import defaultdict

HERE = os.path.dirname(__file__)
ER = os.path.join(HERE, "..", "..", "expense-reporter")
TAXONOMY = os.path.join(ER, "config", "taxonomy.json")
LOGS = ["expenses_log.jsonl", "expenses_log-2022.jsonl",
        "expenses_log-2023.jsonl", "expenses_log-2024.jsonl"]

def nfc(s):
    return unicodedata.normalize("NFC", s or "")

def key(cat, sub):
    return (nfc(cat), nfc(sub))

# 1. taxonomy: (category, subcategory) -> set of types
tax = json.load(open(TAXONOMY, encoding="utf-8"))
tax_types = defaultdict(set)
for ty in tax["types"]:
    for c in ty["categories"]:
        for s in c["subcategories"]:
            tax_types[key(c["name"], s)].add(nfc(ty["name"]))

# 2. typed-entry observed types per (cat, sub), across all logs
observed = defaultdict(lambda: defaultdict(int))
for fn in LOGS:
    p = os.path.join(ER, fn)
    if not os.path.exists(p):
        continue
    for line in open(p, encoding="utf-8"):
        line = line.strip()
        if not line:
            continue
        e = json.loads(line)
        if e.get("type"):
            observed[key(e.get("category"), e.get("subcategory"))][nfc(e["type"])] += 1

def decide(cat, sub):
    cand = tax_types.get(key(cat, sub), set())
    if len(cand) == 1:
        return next(iter(cand)), "unique-in-taxonomy"
    if len(cand) > 1:
        obs = observed.get(key(cat, sub), {})
        agreeing = [t for t in obs if t in cand]
        if len(agreeing) == 1:
            return agreeing[0], f"twin-disambiguated (seen {obs[agreeing[0]]}x typed)"
        return None, f"AMBIGUOUS across {sorted(cand)}; typed twins={dict(obs)}"
    return None, "NOT-IN-TAXONOMY"

apply = "--apply" in sys.argv
ts = datetime.datetime.now().strftime("%Y%m%d-%H%M%S") if apply else "DRYRUN"
flagged = []
for fn in LOGS:
    p = os.path.join(ER, fn)
    if not os.path.exists(p):
        continue
    lines = open(p, encoding="utf-8").read().splitlines()
    out, changed = [], 0
    for line in lines:
        if not line.strip():
            out.append(line)
            continue
        e = json.loads(line)
        if not e.get("type"):
            t, why = decide(e.get("category"), e.get("subcategory"))
            tag = f"{fn}: item={e.get('item')!r} cat={e.get('category')!r} sub={e.get('subcategory')!r}"
            if t:
                print(f"  ASSIGN type={t!r:14} <- {why}  | {tag}")
                e["type"] = t
                changed += 1
            else:
                print(f"  FLAG   {why}  | {tag}")
                flagged.append(tag)
        out.append(json.dumps(e, ensure_ascii=False))
    if apply and changed:
        bak = f"{p}.bak-{ts}"
        os.rename(p, bak)
        open(p, "w", encoding="utf-8").write("\n".join(out) + "\n")
        print(f"-> wrote {fn} ({changed} backfilled); backup {os.path.basename(bak)}")

print(f"\n{'APPLIED' if apply else 'DRY-RUN'}; {len(flagged)} flagged (left type-less).")
