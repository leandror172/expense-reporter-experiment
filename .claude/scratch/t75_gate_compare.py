"""T-75 step-1 gate, half 2: compare the Go dry-run report against the probe's per-id buckets.
Usage: python3 t75_gate_compare.py EXPECTED.tsv REPORT.txt [--perturb]
  REPORT.txt = `expense-reporter telegram-import <result.json> --dry-run > REPORT.txt`
Exit 1 on any per-id difference or per-bucket line-count difference. --perturb flips one
expected label and REQUIRES exactly one per-id difference, so a silent pass cannot be
mistaken for a real one.

SINCE PLAN D10 the unit is a LINE, not a message. The TSV carries one row per line, all
under the parent message id, so an id may appear several times and in more than one
bucket. The report cannot be inverted back to per-id line counts — it prints a LINE count
and a DEDUPED id list per bucket — so the comparison is:

  * per id, the SET of non-converted labels must match exactly;
  * per bucket, the report's line count must match the TSV's;
  * the messages line must equal the number of distinct ids.

`converted` is still the complement and still lists no ids, so for a split message whose
lines disagree the converted MEMBERSHIP is checked only through the aggregate count. That
is the one thing this gate got weaker at, and it is why the count check above is not
optional."""
import re, sys
from collections import Counter, defaultdict

TSV, REPORT = sys.argv[1], sys.argv[2]
perturb = "--perturb" in sys.argv

# --- expected: one row per LINE -------------------------------------------------------
rows = []
for line in open(TSV):
    id_, label, _detail = line.rstrip("\n").split("\t")
    rows.append((int(id_), label))

if perturb:
    # Flip a line belonging to an id that has exactly ONE line, so the flip is guaranteed
    # to change that id's label SET and the perturbation measures what it claims to.
    single = [i for i, n in Counter(i for i, _ in rows).items() if n == 1]
    target = min(single)
    rows = [(i, ("rejected: bad value" if l != "rejected: bad value" else "converted") if i == target else l)
            for i, l in rows]

expected_lines = Counter(l for _, l in rows)
expected_labels = defaultdict(set)
for i, l in rows:
    if l != "converted":
        expected_labels[i].add(l)
ids = {i for i, _ in rows}

# --- got: the Go report ---------------------------------------------------------------
got_labels, got_lines, total, conv_count = defaultdict(set), Counter(), None, None
for line in open(REPORT):
    m = re.match(r"\s*(\d+) (.*)$", line.rstrip("\n"))
    if not m:
        continue
    n, rest = int(m.group(1)), m.group(2)
    if rest == "messages":
        total = n
        continue
    if rest == "converted":
        conv_count = n
        continue
    if rest.startswith("expense lines"):
        continue
    if re.search(r": [\d ]+$", rest):
        label, id_text = rest.rsplit(": ", 1)
        listed = [int(x) for x in id_text.split()]
    else:
        label, listed = rest, []
    if label.startswith(("receipts", "ignored", "repaired")):
        label = label.split(" (")[0]
    got_lines[label] = n
    for i in listed:
        got_labels[i].add(label)

# --- compare --------------------------------------------------------------------------
unknown = sorted(set(got_labels) - ids)
label_diff = {i: (sorted(expected_labels.get(i, set())), sorted(got_labels.get(i, set())))
              for i in ids | set(got_labels)
              if expected_labels.get(i, set()) != got_labels.get(i, set())}
count_diff = {b: (expected_lines.get(b, 0), got_lines.get(b, 0))
              for b in set(expected_lines) | set(got_lines)
              if b != "converted" and expected_lines.get(b, 0) != got_lines.get(b, 0)}

print("total line:", total, "expected distinct ids:", len(ids))
print("converted line:", conv_count, "expected converted lines:", expected_lines.get("converted", 0))
print("expected:", sorted(expected_lines.items()))
print("got     :", sorted(got_lines.items()))
print("ids not in probe:", unknown)
print("per-id differences:", len(label_diff), sorted(label_diff.items())[:5])
print("per-bucket line-count differences:", len(count_diff), sorted(count_diff.items())[:5])

if perturb:
    ok = len(label_diff) == 1 and not unknown
    print("PERTURBATION", "OK — one flipped line, one per-id difference" if ok
          else "FAILED — comparison cannot discriminate")
    sys.exit(0 if ok else 1)

ok = (total == len(ids)
      and conv_count == expected_lines.get("converted", 0)
      and not label_diff and not count_diff and not unknown)
print("GATE", "PASS — every one of %d ids in the probe's bucket, every bucket's line count agreeing"
      % len(ids) if ok else "FAIL")
sys.exit(0 if ok else 1)
