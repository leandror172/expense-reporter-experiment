#!/usr/bin/env python3
"""Paired significance test for the T-26 think-on A/B (nodesc vs descen).
Reuses score.py's exact match predicates. Zero model calls — reads existing JSONL.

McNemar exact (two-sided binomial on discordant pairs) for full-path and type,
overall and on the novel (leaky=False) subgroup. Also a Fisher-ish note on the
sentinel 4-vs-0. Answers: do the report's novel-subgroup and sentinel deltas
clear noise, or should the language be softened?"""
import json
from pathlib import Path
from math import comb
import score  # same directory

HERE = Path(__file__).parent
MODEL = "my-classifier-q3"

def load(prefix):
    return score.read_jsonl(HERE / f"{prefix}-{MODEL}.jsonl")

sample = score.read_jsonl(HERE / "sample.jsonl")
nodesc = score.join_results(sample, load("results-think-nodesc"))
descen = score.join_results(sample, load("results-think-descen"))

def by_id(rows):
    return {str(r["id"]): r for r in rows}

N, D = by_id(nodesc), by_id(descen)
ids = sorted(set(N) & set(D))

def mcnemar_exact(pairs):
    """pairs: list of (a_correct, b_correct). Returns (b, c, n, p_two_sided).
    b = a-correct & b-wrong (A wins), c = a-wrong & b-correct (B wins)."""
    b = sum(1 for a, bb in pairs if a and not bb)
    c = sum(1 for a, bb in pairs if not a and bb)
    n = b + c
    k = min(b, c)
    # two-sided exact binomial at p=0.5
    p = min(1.0, 2 * sum(comb(n, i) for i in range(k + 1)) / (2 ** n)) if n else 1.0
    return b, c, n, p

def run(metric_fn, label, subset=None):
    rows = ids if subset is None else [i for i in ids if subset(N[i])]
    pairs = [(metric_fn(N[i]), metric_fn(D[i])) for i in rows]
    a_acc = sum(1 for a, _ in pairs if a)
    b_acc = sum(1 for _, bb in pairs if bb)
    n = len(pairs)
    bb, cc, nd, p = mcnemar_exact(pairs)
    sig = "SIGNIFICANT" if p < 0.05 else "not sig"
    print(f"{label:28s} n={n:3d} | nodesc {a_acc/n*100:5.1f}%  descen {b_acc/n*100:5.1f}%  "
          f"Δ={(b_acc-a_acc)/n*100:+5.1f}pp | discordant nodesc-wins={bb} descen-wins={cc} "
          f"| McNemar p={p:.3f} [{sig}]")

novel = lambda r: not r.get("leaky", False)
leaky = lambda r: r.get("leaky", False)

print(f"paired items (both conditions): {len(ids)}\n")
print("=== FULL-PATH ===")
run(score.is_full_match, "all")
run(score.is_full_match, "novel (leaky=False)", novel)
run(score.is_full_match, "recurring (leaky=True)", leaky)
print("\n=== TYPE ===")
run(score.is_type_match, "all")
run(score.is_type_match, "novel (leaky=False)", novel)
run(score.is_type_match, "recurring (leaky=True)", leaky)

# Sentinel: 4 (nodesc) vs 0 (descen) out of 20 OOD. Manual two-sided Fisher exact.
def fisher_2x2(a, b, c, d):
    n = a + b + c + d
    def hg(a, b, c, d):
        return comb(a + b, a) * comb(c + d, c) / comb(n, a + c)
    p_obs = hg(a, b, c, d)
    tot = 0.0
    r1, r2, c1 = a + b, c + d, a + c
    for x in range(max(0, c1 - r2), min(r1, c1) + 1):
        p = hg(x, r1 - x, c1 - x, r2 - (c1 - x))
        if p <= p_obs + 1e-12:
            tot += p
    return tot

# 2x2: [declines, non-declines] x [nodesc, descen] = 4/16 vs 0/20
p_sent = fisher_2x2(4, 16, 0, 20)
print(f"\n=== SENTINEL (OOD n=20 each) ===")
print(f"nodesc declines=4/20  descen declines=0/20  Fisher exact two-sided p={p_sent:.3f} "
      f"[{'SIGNIFICANT' if p_sent < 0.05 else 'not sig'}]")
