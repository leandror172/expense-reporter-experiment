#!/usr/bin/env python3
"""Analyze the 649-replay MODEL half (T-23 gate).

The payload: real (model, not keyword) risk–coverage of the E1 specificity gate
(top_score) vs the model's own confidence gate, stratified clean vs leaky (in_training).
Reads model.jsonl next to this script.
"""
import json, os
from collections import Counter

HERE = os.path.dirname(os.path.abspath(__file__))
rows = [json.loads(l) for l in open(os.path.join(HERE, "model.jsonl")) if l.strip()]
n = len(rows)


def acc(sub, field="correct_fullpath"):
    if not sub:
        return 0, 0.0
    c = sum(1 for r in sub if r[field])
    return c, 100 * c / len(sub)


clean = [r for r in rows if not r["in_training"]]
leaky = [r for r in rows if r["in_training"]]
print(f"n={n}  clean(not in training)={len(clean)}  leaky(in training)={len(leaky)}\n")

print("=== OVERALL ACCURACY (per-row) ===")
for label, field in [("full-path", "correct_fullpath"), ("type", "correct_type"),
                     ("category", "correct_category"), ("subcat", "correct_subcat")]:
    _, pa = acc(rows, field); _, pc = acc(clean, field); _, pl = acc(leaky, field)
    print(f"  {label:10s}: all {pa:5.1f}%  | clean {pc:5.1f}%  | leaky {pl:5.1f}%")
print()


def risk_coverage(pool, key, thresholds, label):
    print(f"--- {label} (accuracy = full-path correct) ---")
    tot = len(pool)
    for thr in thresholds:
        sub = [r for r in pool if r[key] >= thr]
        c, p = acc(sub)
        cov = 100 * len(sub) / tot if tot else 0
        print(f"  {key}>={thr:<5}: coverage {len(sub):3d}/{tot} ({cov:5.1f}%)  precision {p:5.1f}%")


print("=== GATE 1: E1 specificity (top_score) — the external validator ===")
risk_coverage(rows, "top_score", [0.0, 0.5, 0.7, 0.85, 1.0], "ALL rows")
risk_coverage(clean, "top_score", [0.0, 0.5, 0.7, 0.85, 1.0], "CLEAN rows (novel to training)")
print()

print("=== GATE 2: model confidence — expected anti-informative (T-14) ===")
risk_coverage(rows, "confidence", [0.0, 0.7, 0.85, 0.9, 0.95], "ALL rows")
print()

print("=== RECURRENCE framing: driver_freq buckets vs full-path accuracy ===")
buckets = [("0 (no kw match)", lambda f: f == 0), ("2-4", lambda f: 2 <= f <= 4),
           ("5-9", lambda f: 5 <= f <= 9), ("10-24", lambda f: 10 <= f <= 24),
           ("25+", lambda f: f >= 25)]
for name, pred in buckets:
    sub = [r for r in rows if pred(r["driver_freq"])]
    _, p = acc(sub)
    print(f"  driver_freq {name:16s}: {len(sub):3d} rows | full-path {p:5.1f}%")
print()

# The money question: is there a top_score band with usable auto-insert precision?
print("=== HIGH-PRECISION BAND CHECK (auto-insert needs ~>=95%) ===")
for thr in (0.85, 1.0):
    sub = [r for r in rows if r["top_score"] >= thr]
    subc = [r for r in clean if r["top_score"] >= thr]
    _, p = acc(sub); _, pc = acc(subc)
    print(f"  top_score>={thr}: all {p:.1f}% ({len(sub)} rows) | clean {pc:.1f}% ({len(subc)} rows)")
print("\n(Compare: retrieval-half keyword-only top1 was 90.5% at spec=1.0 — is the MODEL better/worse?)")
