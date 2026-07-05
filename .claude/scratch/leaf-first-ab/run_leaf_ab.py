#!/usr/bin/env python3
"""Leaf-first A/B runner (T-30, FS-A: few-shot OFF on BOTH arms).

Two arms, one variable (the enum direction):
  - pathfirst  = control. Current production schema: one `path` enum (112 Type/Category/
                 Subcategory strings + sentinel), split back to (type,cat,sub) by reverse map.
  - leaffirst  = treatment. D1-locked two-field schema: `leaf` enum (104 unique leaves +
                 sentinel) THEN `type` enum (4). Derive (type,cat) upward via a ResolveLeaf
                 port; `type` disambiguates the 5 cross-type collisions and cross-checks the
                 99 uniques (free calibration signal, not gated).

Both arms: NO few-shot (FS-A), same taxonomy TREE in the prompt (D2=tree), NO T-22 type
descriptions (held constant/absent). This isolates the enum direction. It is a SIGNAL CHECK,
not the adoption verdict — D4 waits for an FS-B (few-shot-ON, arm-shaped examples) pass.

Calls Ollama /api/chat directly (no production classifier change, D3=benchmark-only).
Emits the exact record shape score.py/sigtest_t26.py already consume, so they run unchanged:
  {id, model, latency_s, ok, top:{type,category,subcategory,confidence}, candidates:[...]}
Resumable by id (append+flush, skip ids already in the results file). Resume keys on the
results FILENAME, so each arm+mode gets its own file (t14-benchmark gotcha).
"""
import argparse
import json
import logging
import time
import unicodedata
import urllib.request
from collections import defaultdict
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[2]
TAXONOMY = REPO_ROOT / "expense-reporter" / "config" / "taxonomy.json"
DATA_DIR = REPO_ROOT / "data" / "classification"
TOPK_FEWSHOT = 5  # matches classifier.selectExamples
OLLAMA_URL = "http://localhost:11434/api/chat"
SENTINEL = "NENHUMA DAS OPÇÕES"
DIVERSOS = "Diversos"
SENTINEL_CONF = 0.30


# ---------- taxonomy / ResolveLeaf port ----------

def nfc(s: str) -> str:
    return unicodedata.normalize("NFC", s)


def load_taxonomy(path: Path):
    """Return (types, owners): types = ordered type names; owners = leaf -> [(type,cat),...]
    in taxonomy order (NFC-keyed for collision matching, verbatim names preserved in values)."""
    tax = json.loads(path.read_text(encoding="utf-8"))
    types = [t["name"] for t in tax["types"]]
    owners = defaultdict(list)
    for ty in tax["types"]:
        for cat in ty["categories"]:
            for sub in cat["subcategories"]:
                owners[nfc(sub)].append((ty["name"], cat["name"], sub))
    return types, owners


def resolve_leaf(owners, leaf, type_hint):
    """Port of taxonomy.ResolveLeaf, extended to also return (cat, sub_verbatim, crosscheck).
    - 0 matches -> None (off-enum / not found).
    - 1 match  -> that owner; type_hint ignored; crosscheck = (hint == real type) or None.
    - >1       -> filter by type_hint; exactly 1 -> it; else None (ambiguous)."""
    matches = owners.get(nfc(leaf), [])
    if not matches:
        return None
    if len(matches) == 1:
        typ, cat, sub = matches[0]
        cross = None if not type_hint else (nfc(type_hint) == nfc(typ))
        return {"type": typ, "category": cat, "subcategory": sub, "collision": False, "type_crosscheck": cross}
    if not type_hint:
        return None
    filtered = [m for m in matches if nfc(m[0]) == nfc(type_hint)]
    if len(filtered) == 1:
        typ, cat, sub = filtered[0]
        return {"type": typ, "category": cat, "subcategory": sub, "collision": True, "type_crosscheck": True}
    return None


def build_path_map(owners):
    """Reverse map for the pathfirst arm: 'Type/Category/Subcategory' -> (type,cat,sub)."""
    pm = {}
    for matches in owners.values():
        for typ, cat, sub in matches:
            pm[f"{typ}/{cat}/{sub}"] = (typ, cat, sub)
    return pm


# ---------- few-shot (FS-B) — port of classifier.SelectExamples + resolveExamplePaths ----------
# Both arms use the SAME selection (same examples per item); they differ ONLY in how the
# example is RENDERED (leaf+type JSON vs path string) — that reformatting IS part of the
# leaf-first change. Training pool only (no live classifications.jsonl) for reproducibility.

_NONALNUM = None  # compiled lazily


def _tokenize(item):
    import re
    global _NONALNUM
    if _NONALNUM is None:
        _NONALNUM = re.compile(r"[^\w ]+", re.UNICODE)
    cleaned = _NONALNUM.sub(" ", item.lower())
    return [t for t in cleaned.split() if len(t) >= 2]


def load_keyword_index(data_dir):
    raw = json.loads((data_dir / "feature_dictionary_enhanced.json").read_text(encoding="utf-8"))
    return {kw: {"specificity": e.get("specificity", 0.0), "subcategories": e.get("subcategories", [])}
            for kw, e in raw["lexical_features"]["keywords"].items()}


def load_training_pool(data_dir):
    raw = json.loads((data_dir / "training_data_complete.json").read_text(encoding="utf-8"))
    pool = []
    for e in raw["expenses"]:
        src = e.get("source", "")
        type_hint = src.rsplit(":", 1)[-1].strip() if ":" in src else ""
        yyyy_mm_dd = e.get("date", "").split("-")
        date = f"{yyyy_mm_dd[2]}/{yyyy_mm_dd[1]}" if len(yyyy_mm_dd) == 3 else e.get("date", "")
        pool.append({"item": e["item"], "value": e["value"], "date": date,
                     "subcategory": e["subcategory"], "type_hint": type_hint})
    return pool


def select_examples(item, pool, keywords, topk):
    tokens = _tokenize(item)
    if not tokens:
        return []
    scores = {}
    for tok in tokens:
        ent = keywords.get(tok)
        if not ent:
            continue
        for sub in ent["subcategories"]:
            if ent["specificity"] > scores.get(sub, 0.0):
                scores[sub] = ent["specificity"]
    if not scores:
        return []
    ordered = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)

    def bucket(sub):
        return [ex for ex in pool if ex["subcategory"] == sub]  # training-only → stable order

    if ordered[0][1] >= 0.7:
        result = bucket(ordered[0][0])
    else:  # ambiguous: interleave top-2
        buckets = [bucket(s) for s, _ in ordered[:2]]
        result = []
        for i in range(max((len(b) for b in buckets), default=0)):
            for b in buckets:
                if i < len(b):
                    result.append(b[i])
    return result[:topk]


def fewshot_messages(item, arm, cfg):
    """Arm-shaped user/assistant example pairs. Each example's canonical (type,cat,sub) is
    resolved via ResolveLeaf (dropped if unresolvable — mirrors resolveExamplePaths)."""
    msgs = []
    for ex in select_examples(item, cfg["pool"], cfg["keywords"], TOPK_FEWSHOT):
        r = resolve_leaf(cfg["owners"], ex["subcategory"], ex["type_hint"])
        if r is None:
            continue
        if arm == "pathfirst":
            assistant = json.dumps({"results": [{"path": f"{r['type']}/{r['category']}/{r['subcategory']}",
                                                 "confidence": 0.95}]}, ensure_ascii=False)
        else:
            assistant = json.dumps({"results": [{"leaf": r["subcategory"], "type": r["type"],
                                                 "confidence": 0.95}]}, ensure_ascii=False)
        user = f"item: {ex['item']}\nvalue: {ex['value']:.2f}\ndate: {ex['date']}"
        msgs.append({"role": "user", "content": user})
        msgs.append({"role": "assistant", "content": assistant})
    return msgs


# ---------- prompt / schema ----------

def render_tree(path: Path) -> str:
    """type -> category: sub, sub, ...  (mirrors classifier.writeTaxonomyTree WITHOUT the
    T-22 description parenthetical — descriptions are held constant/absent for the A/B)."""
    tax = json.loads(path.read_text(encoding="utf-8"))
    lines = []
    for ty in tax["types"]:
        lines.append(f"{ty['name']}:")
        for cat in ty["categories"]:
            lines.append(f"  {cat['name']}: {', '.join(cat['subcategories'])}")
    return "\n".join(lines) + "\n"


def system_prompt(arm, tree, types, topn):
    head = (
        "You are an expense classifier for Brazilian personal finance.\n"
        f"Return exactly {topn} candidates ranked by confidence (highest first).\n"
        "Confidence is a float between 0.0 and 1.0.\n"
    )
    if arm == "pathfirst":
        head += (
            'Each candidate\'s "path" must be a string copied verbatim from the taxonomy, '
            "in the form Type/Category/Subcategory.\n"
            f'If NO taxonomy path fits the expense, answer with the path "{SENTINEL}" '
            "instead of forcing a bad match.\n"
        )
    else:  # leaffirst
        head += (
            'Each candidate has a "leaf" (the specific subcategory, copied verbatim from the '
            'taxonomy) and a "type" (one of: ' + ", ".join(types) + "). Pick the leaf first, "
            "then the type it belongs to for this expense.\n"
            f'If NO taxonomy leaf fits the expense, answer with the leaf "{SENTINEL}".\n'
        )
    return head + "\nTaxonomy:\n" + tree


def response_schema(arm, leaves, types):
    if arm == "pathfirst":
        item = {
            "type": "object",
            "properties": {
                "path": {"type": "string", "enum": leaves},  # here `leaves` = path enum
                "confidence": {"type": "number"},
            },
            "required": ["path", "confidence"],
        }
    else:  # leaffirst: property order leaf -> type -> confidence is LOAD-BEARING (GBNF-verified)
        item = {
            "type": "object",
            "properties": {
                "leaf": {"type": "string", "enum": leaves},
                "type": {"type": "string", "enum": types},
                "confidence": {"type": "number"},
            },
            "required": ["leaf", "type", "confidence"],
        }
    return {
        "type": "object",
        "properties": {"results": {"type": "array", "items": item}},
        "required": ["results"],
    }


# ---------- classify one item ----------

def classify_once(entry, arm, cfg):
    messages = [{"role": "system", "content": cfg["system"]}]
    if cfg["fewshot"]:
        messages.extend(fewshot_messages(entry["item"], arm, cfg))
    messages.append({"role": "user",
                     "content": f"item: {entry['item']}\nvalue: {entry['value']:.2f}\ndate: {entry['date']}"})
    body = {"model": cfg["model"], "stream": False, "format": cfg["schema"], "messages": messages}
    if not cfg["think"]:
        body["think"] = False

    start = time.monotonic()
    try:
        req = urllib.request.Request(OLLAMA_URL, data=json.dumps(body).encode("utf-8"),
                                     headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=cfg["timeout"]) as r:
            resp = json.loads(r.read().decode("utf-8"))
    except Exception as e:
        return {"id": entry["id"], "model": cfg["model"], "arm": arm,
                "latency_s": round(time.monotonic() - start, 2), "ok": False, "error": str(e)[:300]}
    latency = round(time.monotonic() - start, 2)

    content = resp.get("message", {}).get("content", "")
    try:
        parsed = json.loads(content)
    except json.JSONDecodeError:
        return {"id": entry["id"], "model": cfg["model"], "arm": arm, "latency_s": latency,
                "ok": False, "error": f"bad json: {content[:200]}"}

    candidates = []
    for r in parsed.get("results", []):
        cand = to_candidate(r, arm, cfg)
        if cand is not None:
            candidates.append(cand)
    candidates.sort(key=lambda c: c["confidence"], reverse=True)

    return {"id": entry["id"], "model": cfg["model"], "arm": arm, "latency_s": latency,
            "ok": True, "top": candidates[0] if candidates else None, "candidates": candidates}


def to_candidate(r, arm, cfg):
    """Map one raw model result to a scored candidate {type,category,subcategory,confidence}
    (+ leaf-first extras). Sentinel -> Diversos@0.30. Off-enum -> dropped (returns None)."""
    conf = float(r.get("confidence", 0.0))
    if arm == "pathfirst":
        path = r.get("path", "")
        if path == SENTINEL:
            return sentinel_candidate(cfg)
        owner = cfg["path_map"].get(path)
        if owner is None:
            return None
        typ, cat, sub = owner
        return {"type": typ, "category": cat, "subcategory": sub, "confidence": conf}
    # leaffirst
    leaf = r.get("leaf", "")
    if leaf == SENTINEL:
        return sentinel_candidate(cfg)
    resolved = resolve_leaf(cfg["owners"], leaf, r.get("type", ""))
    if resolved is None:
        return None
    return {"type": resolved["type"], "category": resolved["category"],
            "subcategory": resolved["subcategory"], "confidence": conf,
            "predicted_type": r.get("type", ""), "type_crosscheck": resolved["type_crosscheck"],
            "collision": resolved["collision"]}


def sentinel_candidate(cfg):
    d = cfg["diversos"]
    if d is None:
        return None
    return {"type": d["type"], "category": d["category"], "subcategory": d["subcategory"],
            "confidence": SENTINEL_CONF}


# ---------- io / driver ----------

def read_jsonl(path: Path):
    if not path.exists():
        return []
    out = []
    with path.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                out.append(json.loads(line))
    return out


def done_ids(path: Path):
    return {str(r["id"]) for r in read_jsonl(path)}


def main():
    p = argparse.ArgumentParser(description="Leaf-first A/B runner (FS-A)")
    p.add_argument("--arm", required=True, choices=["pathfirst", "leaffirst"])
    p.add_argument("--think", action="store_true", help="think-on (default: no-think)")
    p.add_argument("--fewshot", action="store_true", help="FS-B: inject arm-shaped few-shot (default OFF = FS-A)")
    p.add_argument("--model", default="my-classifier-q3")
    p.add_argument("--sample", type=Path, default=SCRIPT_DIR / "sample.jsonl")
    p.add_argument("--outdir", type=Path, default=SCRIPT_DIR)
    p.add_argument("--prefix", default="results", help='use "ood-results" for the OOD pass')
    p.add_argument("--top", type=int, default=3)
    p.add_argument("--limit", type=int, default=0)
    p.add_argument("--timeout", type=int, default=240)
    args = p.parse_args()

    types, owners = load_taxonomy(TAXONOMY)
    leaves = sorted(owners)  # 104 unique leaf names
    if args.arm == "pathfirst":
        enum = list(build_path_map(owners).keys()) + [SENTINEL]  # 112 paths + sentinel
    else:
        enum = leaves + [SENTINEL]  # 104 leaves + sentinel
    diversos = resolve_leaf(owners, DIVERSOS, "")  # for the sentinel fallback

    cfg = {
        "model": args.model,
        "think": args.think,
        "timeout": args.timeout,
        "schema": response_schema(args.arm, enum, types),
        "system": system_prompt(args.arm, render_tree(TAXONOMY), types, args.top),
        "path_map": build_path_map(owners),
        "owners": owners,
        "diversos": diversos,
        "fewshot": args.fewshot,
        "pool": load_training_pool(DATA_DIR) if args.fewshot else None,
        "keywords": load_keyword_index(DATA_DIR) if args.fewshot else None,
    }

    mode = "think" if args.think else "nothink"
    fs = "fs-" if args.fewshot else ""  # FS-B files get an "fs-" segment; FS-A names unchanged
    out_path = args.outdir / f"{args.prefix}-{args.arm}-{fs}{mode}-{args.model}.jsonl"
    logging.info("arm=%s mode=%s fewshot=%s enum=%d out=%s diversos=%s",
                 args.arm, mode, args.fewshot, len(enum), out_path.name,
                 diversos and diversos["type"] + "/" + diversos["category"])

    sample = read_jsonl(args.sample)
    already = done_ids(out_path)
    todo = [e for e in sample if str(e["id"]) not in already]
    logging.info("skipped %d already done; %d to do", len(sample) - len(todo), len(todo))

    ok = fail = 0
    with out_path.open("a", encoding="utf-8") as f:
        for i, entry in enumerate(todo):
            if args.limit and i >= args.limit:
                break
            rec = classify_once(entry, args.arm, cfg)
            f.write(json.dumps(rec, ensure_ascii=False) + "\n")
            f.flush()
            ok, fail = (ok + 1, fail) if rec["ok"] else (ok, fail + 1)
            top = rec.get("top") if rec["ok"] else None
            desc = f"{top['subcategory']}@{top['confidence']:.2f}" if top else (rec.get("error", "no-cand")[:40])
            logging.info("%d/%d id=%s ok=%s %.1fs %s", i + 1, len(todo), entry["id"], rec["ok"], rec["latency_s"], desc)

    logging.info("done: ok=%d fail=%d -> %s", ok, fail, out_path.name)


if __name__ == "__main__":
    main()
