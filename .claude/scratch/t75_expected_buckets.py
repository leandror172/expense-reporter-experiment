"""T-75 step-1 gate, half 1: per-message-id EXPECTED buckets from the probe.
Usage: python3 t75_expected_buckets.py OUT.tsv   (EXPORT_JSON env overrides the export path)
Per-message-id expected buckets for the step-1 gate. Reuses the probe's own
parsers so the TSV is the probe's verdict, not a second opinion. Writes ids and
labels only — never message text."""
import importlib.util, json, os, re, sys
from collections import Counter, defaultdict
from datetime import date
HERE = os.path.dirname(os.path.abspath(__file__))
PROBE = os.path.join(HERE, "t75_oracle_probe.py")
EXPORT = os.environ.get("EXPORT_JSON", os.path.join(HERE, "..", "..", "..", "ChatExport_2026-04-20", "result.json"))
OUT = sys.argv[1]
spec = importlib.util.spec_from_file_location("probe", PROBE)
probe = importlib.util.module_from_spec(spec); spec.loader.exec_module(probe)
msgs = json.load(open(EXPORT, encoding="utf-8"))["messages"]
CUR = re.compile(r"(\d+[.,]\d{2}(?!\d)|R\$)")
rows, shapes, vshapes = [], Counter(), Counter()
for x in msgs:
    text = x.get("text")
    if isinstance(text, list):
        text = "".join(t if isinstance(t, str) else t.get("text", "") for t in text)
    text = (text or "").strip()
    msg_d = date.fromisoformat(x["date"][:10])
    att = "pdf" if "file" in x else ("photo" if "photo" in x else "none")
    reason = ""
    if not text:
        b = "attachment" if att != "none" else "empty"; reason = att
    else:
        parts = [p.strip() for p in text.split(";")]
        if len(parts) != 3:
            b = "field count %d" % len(parts)
        else:
            item, dstr, vstr = parts
            try:
                probe.parse_date_field(dstr, msg_d)
            except ValueError as e:
                b, reason = "bad date field", str(e)
            else:
                try:
                    probe.parse_with_installments(vstr); b = "conforming"
                except (ValueError, ZeroDivisionError) as e:
                    b, reason = "bad value field", str(e)
            if b == "conforming":
                m = probe.DATE_RE.match(dstr)
                shapes[("D" if len(m.group(1)) == 1 else "DD") + "/" + ("M" if len(m.group(2)) == 1 else "MM")
                       + ("/Y%d" % len(m.group(3)) if m.group(3) else "")] += 1
                vs = re.sub(r"\d", "9", vstr)
                vshapes[vs] += 1
    attempted = (";" in text) or bool(CUR.search(text))
    cls = ("converted" if b == "conforming" else "receipt" if b in ("attachment", "empty")
           else "rejected" if attempted else "ignored")
    rows.append((x["id"], cls, b, reason))
with open(OUT, "w") as f:
    for r in rows: f.write("\t".join(map(str, r)) + "\n")
print("buckets:", Counter(r[2] for r in rows).most_common())
print("classes:", Counter(r[1] for r in rows).most_common())
xt = defaultdict(Counter)
for r in rows: xt[r[2]][r[1]] += 1
for b, c in xt.items(): print("  %-16s %s" % (b, dict(c)))
print("date shapes (conforming):", dict(shapes))
print("value shapes (conforming):", vshapes.most_common())
print("non-converted ids:", [r[0] for r in rows if r[1] != "converted"])
