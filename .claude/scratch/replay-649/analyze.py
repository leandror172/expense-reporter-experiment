#!/usr/bin/env python3
"""Analyze the 649-replay retrieval output (T-23 / 5.R1).

Answers:
  - E1 recurrence-strength quality: does top_score (specificity) predict keyword
    correctness, and is it CORRUPTED by novel singletons (freq==1 scoring 1.0)?
  - risk-coverage of a specificity gate vs a frequency-aware gate.
Reads retrieval.jsonl next to this script.
"""
import json, os

HERE = os.path.dirname(os.path.abspath(__file__))
rows = [json.loads(l) for l in open(os.path.join(HERE, "retrieval.jsonl")) if l.strip()]
n = len(rows)


def rate(sub, pred):
    m = [r for r in sub if pred(r)]
    return len(m), (100 * len(m) / len(sub) if sub else 0.0)


print(f"n={n}\n")

hits = [r for r in rows if r["n_examples"] > 0]  # keyword matched
miss = [r for r in rows if r["n_examples"] == 0]
print(f"keyword hits: {len(hits)} ({100*len(hits)/n:.1f}%)   misses: {len(miss)} ({100*len(miss)/n:.1f}%)\n")

# --- Singleton masquerade: high specificity but frequency==1 (novel) -----------
print("=== SINGLETON MASQUERADE (advisor #1) ===")
for thr in (0.7, 0.85, 1.0):
    strong = [r for r in hits if r["top_score"] >= thr]
    singl = [r for r in strong if r["driver_freq"] <= 1]
    c1, p1 = rate(strong, lambda r: r["keyword_top1_correct"])
    cs, ps = rate(singl, lambda r: r["keyword_top1_correct"]) if singl else (0, 0.0)
    print(f"top_score>={thr}: {len(strong):3d} items | "
          f"top1_correct {p1:5.1f}% | of these driver_freq<=1: {len(singl):3d} "
          f"(their top1_correct {ps:5.1f}%)")
print()

# --- Risk-coverage: specificity gate vs frequency-aware gate -------------------
print("=== RISK-COVERAGE among keyword hits (accuracy = keyword_top1_correct) ===")
print("specificity-only gate:")
for thr in (0.0, 0.5, 0.7, 0.85, 1.0):
    sub = [r for r in hits if r["top_score"] >= thr]
    c, p = rate(sub, lambda r: r["keyword_top1_correct"])
    print(f"  top_score>={thr:<4}: coverage {len(sub):3d}/{len(hits)} ({100*len(sub)/len(hits):4.1f}%)  precision {p:5.1f}%")

print("frequency-aware gate (top_score>=0.85 AND driver_freq>=F):")
for F in (1, 2, 3, 5, 10):
    sub = [r for r in hits if r["top_score"] >= 0.85 and r["driver_freq"] >= F]
    c, p = rate(sub, lambda r: r["keyword_top1_correct"])
    print(f"  freq>={F:<3}: coverage {len(sub):3d}/{len(hits)} ({100*len(sub)/len(hits):4.1f}%)  precision {p:5.1f}%")
print()

# --- Frequency distribution among hits -----------------------------------------
print("=== driver_freq buckets among keyword hits ===")
buckets = [("1 (singleton)", lambda f: f <= 1),
           ("2-4", lambda f: 2 <= f <= 4),
           ("5-9", lambda f: 5 <= f <= 9),
           ("10-24", lambda f: 10 <= f <= 24),
           ("25+", lambda f: f >= 25)]
for name, pred in buckets:
    sub = [r for r in hits if pred(r["driver_freq"])]
    c, p = rate(sub, lambda r: r["keyword_top1_correct"])
    print(f"  {name:14s}: {len(sub):3d} items | top1_correct {p:5.1f}%")
print()

# --- Miss characterization -----------------------------------------------------
print("=== MISS pool (no keyword matched — the 5.R2 target) ===")
# how many miss items are RECURRING in the label set (same subcat appears >1x)?
from collections import Counter
subcat_counts = Counter(r["actual_subcategory"] for r in rows)
miss_recurring = [r for r in miss if subcat_counts[r["actual_subcategory"]] > 1]
print(f"  misses whose actual_subcat appears >1x in the 649: {len(miss_recurring)}/{len(miss)}")
print(f"  → these are labels we HAVE examples for but keyword retrieval can't reach (embeddings would).")
