#!/usr/bin/env python3
"""Compare no-think vs think-on on the top band (top_score>=0.85).

Question (advisor #4): is the think-on lift UNIFORM across the [0.85,1.0) and [1.0]
sub-bands? Uniform → the specificity discriminator's SHAPE holds under the production
think default (only the level shifts). Bottom-heavy → think-on flattens the gate.

Joins model.jsonl (no-think, all rows) and model-think.jsonl (think-on, top band) by id.
"""
import json, os

HERE = os.path.dirname(os.path.abspath(__file__))
nt = {json.loads(l)["id"]: json.loads(l) for l in open(os.path.join(HERE, "model.jsonl")) if l.strip()}
th = {json.loads(l)["id"]: json.loads(l) for l in open(os.path.join(HERE, "model-think.jsonl")) if l.strip()}

ids = [i for i in th if i in nt]  # top-band rows present in both
print(f"think-on rows={len(th)}  joined with no-think={len(ids)}\n")


def prec(ids_sub, table, field="correct_fullpath"):
    if not ids_sub:
        return 0, 0.0
    c = sum(1 for i in ids_sub if table[i][field])
    return len(ids_sub), 100 * c / len(ids_sub)


bands = [
    ("[0.85, 1.0)", lambda ts: 0.85 <= ts < 1.0),
    ("[1.0]      ", lambda ts: ts >= 1.0),
    (">=0.85 (all)", lambda ts: ts >= 0.85),
]

for field in ("correct_fullpath", "correct_subcat"):
    print(f"=== {field} ===")
    print(f"  {'band':14s} {'n':>4} {'no-think':>9} {'think-on':>9} {'Δ':>7}")
    deltas = {}
    for name, pred in bands:
        sub = [i for i in ids if pred(th[i]["top_score"])]
        n, p_nt = prec(sub, nt, field)
        _, p_th = prec(sub, th, field)
        d = p_th - p_nt
        deltas[name.strip()] = d
        print(f"  {name:14s} {n:>4} {p_nt:>8.1f}% {p_th:>8.1f}% {d:>+6.1f}")
    lo, hi = deltas["[0.85, 1.0)"], deltas["[1.0]"]
    verdict = "UNIFORM (shape holds)" if abs(lo - hi) <= 4 else ("BOTTOM-HEAVY (flattens gate)" if lo > hi else "TOP-HEAVY (sharpens gate)")
    print(f"  → lift spread |Δlow - Δhigh| = {abs(lo-hi):.1f}pp → {verdict}\n")
