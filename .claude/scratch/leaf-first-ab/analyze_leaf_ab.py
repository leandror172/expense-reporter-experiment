#!/usr/bin/env python3
"""Analyze the leaf-first A/B (pathfirst control vs leaffirst treatment), one --mode at a time.
Reuses score.py predicates + mcnemar logic. Zero model calls. Adds what score.py lacks:
the "right sheet, wrong leaf" rate (the T-14 dominant error class this change targets) and the
leaf-first type_crosscheck signal. Paired McNemar (same ids) is the significance backbone —
a single 300-item run resolves ~±5pp, so don't over-read any single stratum (T-26 lesson)."""
import argparse
import sys
from math import comb
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent / "t14-benchmark"))
import score  # noqa: E402


def by_id(rows):
    return {str(r["id"]): r for r in rows}


def right_sheet_wrong_leaf(row):
    """type correct AND subcategory wrong — the dominant T-14 error class."""
    return score.is_type_match(row) and not score.is_sub_match(row)


def mcnemar_exact(pairs):
    b = sum(1 for a, bb in pairs if a and not bb)   # control-correct, treatment-wrong
    c = sum(1 for a, bb in pairs if not a and bb)   # control-wrong, treatment-correct
    n = b + c
    k = min(b, c)
    p = min(1.0, 2 * sum(comb(n, i) for i in range(k + 1)) / (2 ** n)) if n else 1.0
    return b, c, p


def acc(rows, fn):
    return sum(1 for r in rows if fn(r)) / len(rows) * 100 if rows else 0.0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mode", default="nothink", choices=["nothink", "think"])
    ap.add_argument("--model", default="my-classifier-q3")
    ap.add_argument("--fewshot", action="store_true", help="read the FS-B (fs-) result files")
    args = ap.parse_args()

    fs = "fs-" if args.fewshot else ""
    sample = score.read_jsonl(HERE / "sample.jsonl")
    ctrl = score.join_results(sample, score.read_jsonl(HERE / f"results-pathfirst-{fs}{args.mode}-{args.model}.jsonl"))
    treat = score.join_results(sample, score.read_jsonl(HERE / f"results-leaffirst-{fs}{args.mode}-{args.model}.jsonl"))
    C, T = by_id(ctrl), by_id(treat)
    ids = sorted(set(C) & set(T))
    print(f"# Leaf-first A/B — mode={args.mode} fewshot={args.fewshot} model={args.model}")
    print(f"paired items (both arms ok-or-not): {len(ids)}  (control n={len(C)}, treatment n={len(T)})\n")

    metrics = [("full-path", score.is_full_match), ("type", score.is_type_match),
               ("leaf (sub)", score.is_sub_match), ("right-sheet-wrong-leaf", right_sheet_wrong_leaf)]
    subsets = [("all", lambda r: True),
               ("leaky", lambda r: r.get("leaky", False)),
               ("clean", lambda r: not r.get("leaky", False))]

    for sname, sfn in subsets:
        rids = [i for i in ids if sfn(C[i])]
        print(f"## {sname} (n={len(rids)})")
        print(f"{'metric':24s} {'control':>9s} {'treat':>9s} {'Δpp':>7s} {'ctl-win':>8s} {'trt-win':>8s} {'McNemar p':>10s}")
        for mname, mfn in metrics:
            cr = [C[i] for i in rids]
            tr = [T[i] for i in rids]
            ca, ta = acc(cr, mfn), acc(tr, mfn)
            b, c, p = mcnemar_exact([(mfn(C[i]), mfn(T[i])) for i in rids])
            sig = "*" if p < 0.05 else " "
            print(f"{mname:24s} {ca:8.1f}% {ta:8.1f}% {ta-ca:+6.1f} {b:8d} {c:8d} {p:9.3f}{sig}")
        print()

    # Leaf-first type_crosscheck: for the 99 unique leaves, model-predicted type vs the leaf's
    # real owner type. A free calibration signal (disagreement flags a shaky prediction).
    print("## leaffirst type_crosscheck (free calibration signal, unique leaves only)")
    tops = [T[i]["top"] for i in ids if T[i].get("ok") and T[i].get("top")]
    uniq = [t for t in tops if t.get("collision") is False and t.get("type_crosscheck") is not None]
    disagree = [t for t in uniq if t["type_crosscheck"] is False]
    print(f"unique-leaf predictions: {len(uniq)}; type disagreements (predicted≠owner): {len(disagree)}")
    # Does a disagreement predict a wrong answer? Compare full-path accuracy agree vs disagree.
    agree_ids = [i for i in ids if T[i].get("ok") and T[i].get("top") and T[i]["top"].get("type_crosscheck") is True]
    dis_ids = [i for i in ids if T[i].get("ok") and T[i].get("top") and T[i]["top"].get("type_crosscheck") is False]
    if agree_ids:
        print(f"full-path acc | type-agree (n={len(agree_ids)}): {acc([T[i] for i in agree_ids], score.is_full_match):.1f}%")
    if dis_ids:
        print(f"full-path acc | type-DISAGREE (n={len(dis_ids)}): {acc([T[i] for i in dis_ids], score.is_full_match):.1f}%")

    # Collision leaves: how did the treatment do where the type field is load-bearing?
    coll_ids = [i for i in ids if T[i].get("ok") and T[i].get("top") and T[i]["top"].get("collision") is True]
    print(f"\ncollision-leaf predictions (type field disambiguates): n={len(coll_ids)}"
          + (f", full-path acc {acc([T[i] for i in coll_ids], score.is_full_match):.1f}%" if coll_ids else ""))


if __name__ == "__main__":
    main()
