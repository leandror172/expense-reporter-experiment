"""T-75 step-1 gate, half 2: compare the Go dry-run report against the probe's per-id buckets.
Usage: python3 t75_gate_compare.py EXPECTED.tsv REPORT.txt [--perturb]
  REPORT.txt = `expense-reporter telegram-import <result.json> --dry-run > REPORT.txt`
Step-1 gate: compare the Go dry-run report against the probe's per-id buckets.
Exit 1 on any per-id difference. --perturb flips one expected label and REQUIRES
exactly one difference, so a silent pass cannot be mistaken for a real one."""
import re, sys
from collections import Counter
TSV, REPORT = sys.argv[1], sys.argv[2]
perturb = "--perturb" in sys.argv

def probe_label(cls, bucket):
    if cls in ("converted", "ignored"): return cls
    if cls == "receipt": return "receipts"
    m = re.match(r"field count (\d+)", bucket)
    if m:
        n = int(m.group(1)); return "rejected: %d field%s" % (n, "" if n == 1 else "s")
    return {"bad date field": "rejected: bad date", "bad value field": "rejected: bad value"}.get(bucket, "rejected: ?")

expected = {}
for line in open(TSV):
    id_, cls, bucket, _ = line.rstrip("\n").split("\t")
    expected[int(id_)] = probe_label(cls, bucket)
if perturb:
    first = min(expected); expected[first] = "rejected: bad value" if expected[first] != "rejected: bad value" else "converted"

got, total, conv_count = {}, None, None
for line in open(REPORT):
    m = re.match(r"\s*(\d+) (.*)$", line.rstrip("\n"))
    if not m: continue
    n, rest = int(m.group(1)), m.group(2)
    if rest == "messages": total = n; continue
    if rest == "converted": conv_count = n; continue
    if re.search(r": [\d ]+$", rest):
        label, ids = rest.rsplit(": ", 1); ids = [int(x) for x in ids.split()]
    else:
        label, ids = rest, []
    if label.startswith(("receipts", "ignored")): label = label.split(" (")[0]
    assert len(ids) == n, "count/id mismatch on: " + line
    for i in ids:
        assert i not in got, "id %d listed twice" % i
        got[i] = label
unknown = set(got) - set(expected)
for i in set(expected) - set(got): got[i] = "converted"
diff = {i: (expected[i], got[i]) for i in expected if expected[i] != got[i]}
print("total line:", total, "expected:", len(expected))
print("converted line:", conv_count, "derived:", sum(1 for v in got.values() if v == "converted"))
print("expected:", sorted(Counter(expected.values()).items()))
print("got     :", sorted(Counter(got.values()).items()))
print("ids not in probe:", sorted(unknown))
print("per-id differences:", len(diff), sorted(diff.items())[:5])
if perturb:
    ok = len(diff) == 1 and not unknown
    print("PERTURBATION", "OK — one flipped id, one difference" if ok else "FAILED — comparison cannot discriminate")
    sys.exit(0 if ok else 1)
ok = total == len(expected) and conv_count == sum(1 for v in got.values() if v == "converted") and not diff and not unknown
print("GATE", "PASS — every one of %d ids in the probe's bucket" % len(expected) if ok else "FAIL")
sys.exit(0 if ok else 1)
