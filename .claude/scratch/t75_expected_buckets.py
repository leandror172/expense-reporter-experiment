"""T-75 gate, half 1: per-message-id EXPECTED bucket, computed INDEPENDENTLY of the Go code.
Usage: python3 t75_expected_buckets.py OUT.tsv   (EXPORT_JSON env overrides the export path)

Writes `id <TAB> label <TAB> detail` where label is the Go report's bucket string
(converted / repaired / receipts / ignored / rejected: N fields / rejected: bad date /
rejected: bad value / rejected: ambiguous). Ids and labels only — never message text.

Value parsing comes from the probe's port of pkg/utils/currency.go; the date rule (plan
D1: explicit year wins the SELECTION, 2-digit year → 20YY, and — since the s75 step-5
amendment — the resolved date must land within [-180,+7] days of the message day on BOTH
branches, explicit or bare; year never beyond the current one) and the repair rule (plan D4: single
edits — each ',' → ';', each '/' → ';', four fields → merge the first two — accepted only
when EXACTLY ONE parses with a date inside the window; D5: never for a line that already parses) are re-implemented
here from the plan text, so a bug shared with the Go side would have to be a bug in the
plan itself.
"""
import importlib.util, json, os, re, sys
from collections import Counter
from datetime import date, datetime

HERE = os.path.dirname(os.path.abspath(__file__))
PROBE = os.path.join(HERE, "t75_oracle_probe.py")
EXPORT = os.environ.get("EXPORT_JSON", os.path.join(HERE, "..", "..", "..", "ChatExport_2026-04-20", "result.json"))
OUT = sys.argv[1]
WINDOW_PAST, WINDOW_FUTURE = 180, 7
CURRENT_YEAR = datetime.now().year

spec = importlib.util.spec_from_file_location("probe", PROBE)
probe = importlib.util.module_from_spec(spec); spec.loader.exec_module(probe)
CUR = re.compile(r"(\d+[.,]\d{2}(?!\d)|R\$)")


def resolve_date(field, msg_d):
    """Plan D1, re-implemented. Returns a date or raises ValueError."""
    parts = [p.strip() for p in field.strip().split("/")]
    if len(parts) == 2:
        day, month = int(parts[0]), int(parts[1])
        seen_calendar = False
        for y in (msg_d.year - 1, msg_d.year, msg_d.year + 1):
            try:
                cand = date(y, month, day)
            except ValueError:
                continue
            seen_calendar = True
            if -WINDOW_PAST <= (cand - msg_d).days <= WINDOW_FUTURE:
                return validated(cand)
        raise ValueError("not a calendar date" if not seen_calendar else "no year in window")
    if len(parts) == 3:
        y = parts[2]
        if len(y) == 2: y = "20" + y
        elif len(y) != 4: raise ValueError("year length")
        d = date(int(y), int(parts[1]), int(parts[0]))
        # D1 AMENDMENT (s75, step 5): the window validates BOTH branches, not just the
        # bare one. The explicit year still wins the SELECTION — nothing here re-resolves
        # which year the human meant — but the resolved date must still land near the
        # message, because the boundary's past side is unguarded (1..9999, forward-only)
        # and '21/08/1200' would otherwise convert into its own month file.
        if not (-WINDOW_PAST <= (d - msg_d).days <= WINDOW_FUTURE):
            raise ValueError("explicit year outside the window")
        return validated(d)
    raise ValueError("shape")


def validated(d):
    """The boundary's two resolved-year checks."""
    if d.year < 1 or d.year > 9999 or d.year > CURRENT_YEAR:
        raise ValueError("year refused")
    return d


def line_ok(text, msg_d):
    """parseLine: three fields, a resolvable date, a value the port accepts, non-empty item."""
    parts = text.split(";")
    if len(parts) != 3:
        return "field count %d" % len(parts)
    item, dstr, vstr = (p.strip() for p in parts)
    if not item:
        return "empty item"
    try:
        resolve_date(dstr, msg_d)
    except (ValueError, OverflowError):
        return "bad date"
    try:
        probe.parse_with_installments(vstr)
    except (ValueError, ZeroDivisionError):
        return "bad value"
    return None


def candidate_ok(text, msg_d):
    """A repair candidate must parse AND its resolved date must lie inside the D1 window of
    the message day, explicit year or not (D4 amendment, s75): the comma edit on a slash
    typo reads the value's integer part as a year — '21/08/ 1200' → year 1200 — which the
    boundary accepts because the past side is deliberately unguarded (T-48).
    SUBSUMED since the s75 step-5 D1 amendment, which applies the same window to every
    resolved date — so resolve_date already refuses this and the check below can no longer
    fail. Kept because the two rules have DIFFERENT justifications that merely share a
    threshold: D1's is "a date belongs near its message", D4's is "a guess needs more
    evidence than a statement". If D1's window were ever relaxed, D4 would still want its
    own."""
    if line_ok(text, msg_d) is not None:
        return False
    d = resolve_date(text.split(";")[1], msg_d)
    return -WINDOW_PAST <= (d - msg_d).days <= WINDOW_FUTURE


def candidates(text):
    """Plan D4's single edits, in the Go order: commas, slashes, then the four-field merge."""
    out = [text[:i] + ";" + text[i + 1:] for i, c in enumerate(text) if c == ","]
    out += [text[:i] + ";" + text[i + 1:] for i, c in enumerate(text) if c == "/"]
    f = text.split(";")
    if len(f) == 4:
        out.append(f[0].strip() + " " + f[1].strip() + ";" + ";".join(f[2:]))
    return out


def as_typed_label(reason):
    m = re.match(r"field count (\d+)", reason)
    if m:
        n = int(m.group(1)); return "rejected: %d field%s" % (n, "" if n == 1 else "s")
    return {"bad date": "rejected: bad date", "bad value": "rejected: bad value"}.get(reason, "rejected: other")


def classify(text, att, msg_d):
    text = text.strip()
    if not text:
        return ("receipts" if att != "none" else "ignored"), att
    reason = line_ok(text, msg_d)
    if reason is None:
        return "converted", ""
    if ";" not in text and not CUR.search(text):
        return "ignored", ""
    valid = [c for c in candidates(text) if candidate_ok(c, msg_d)]
    if len(valid) == 1:
        return "repaired", "edit"
    if len(valid) > 1:
        return "rejected: ambiguous", "%d readings" % len(valid)
    return as_typed_label(reason), reason


def expand_lists(text):
    """Plan D10, re-implemented: a message with more than one non-empty line, EVERY one of
    them 3-field-shaped (exactly two ';'), is N messages — one per line. Anything else is
    returned whole. The test is the SEMICOLON COUNT and deliberately not whether the line
    parses: the message that forced this rule carries two `29/12/26` typos among ten good
    lines, and 'every line parses' would have thrown all ten away."""
    lines = [l.strip() for l in text.split("\n") if l.strip()]
    if len(lines) < 2 or any(l.count(";") != 2 for l in lines):
        return [text]
    return lines


msgs = json.load(open(EXPORT, encoding="utf-8"))["messages"]
rows = []
for x in msgs:
    if x.get("type") != "message":
        continue
    text = x.get("text")
    if isinstance(text, list):
        text = "".join(t if isinstance(t, str) else t.get("text", "") for t in text)
    msg_d = date.fromisoformat(x["date"][:10])
    att = "pdf" if x.get("file") else ("photo" if x.get("photo") else "none")
    # One row per LINE since D10, all under the parent id — so an id may occur several
    # times, and in more than one bucket when its lines disagree.
    for line in expand_lists(text or ""):
        label, detail = classify(line, att, msg_d)
        rows.append((x["id"], label, detail))
with open(OUT, "w") as f:
    for r in rows:
        f.write("\t".join(map(str, r)) + "\n")
print("labels:", sorted(Counter(r[1] for r in rows).items()))
print("non-converted ids:", [r[0] for r in rows if r[1] != "converted"])
