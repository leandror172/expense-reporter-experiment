#!/usr/bin/env python3
"""T-75 — the 2025 Telegram export as a validation oracle for the converter.

Read-only. Side A has two sources, chosen with --source:

  converter  (default; step 4)  the REAL converter's output — the expenses-YYYY-MM.csv
             files `expense-reporter telegram-import` writes, run into a temp dir unless
             --out-dir names an existing one. Each line's installments are expanded with
             the Python port below, which shares no code with the Go parser: a divergence
             shows up as residue (or as a "DIVERGENCE" skip), the safe direction.
  naive      (step 1)  the probe's own DELIBERATELY DUMB conversion of the 352 conforming
             messages — no repair, no converter — kept so every number quoted in
             `.claude/t75-oracle-viability.md` stays reproducible, and as a second opinion
             beside the converter. Pinned by t75_probe_characterize.py.

The question is NOT "do the two sides reconcile" — they cannot, and that is known
before running: 352 messages against 499 log rows, where the log side came from the
hand-maintained workbook via 5.R4 (alias table + dedup + merge) and holds recurring
bills nobody types into the chat group.

The question is: of the rows side A produces, how many find a (date, value) partner in
the log — and does every miss have a NAMED cause? The log is a second lossy record, not
ground truth, so the criterion is never a match rate. THE GATE IS THE EXIT CODE: 1 when a
residue row's cause could not be named (UNEXPLAINED — its item IS in the log under another
date or value) or when the perturbation stops discriminating; 0 otherwise.

Usage:  python3 .claude/scratch/t75_oracle_probe.py [--source converter|naive] [--out-dir DIR]
Env:    EXPORT_JSON, EXPENSE_LOG override the defaults below.
"""
import json, os, re, subprocess, sys, tempfile, unicodedata
from collections import Counter, defaultdict
from datetime import date, timedelta
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
EXPORT = Path(os.environ.get("EXPORT_JSON",
               REPO.parent / "ChatExport_2026-04-20" / "result.json"))
LOG = Path(os.environ.get("EXPENSE_LOG",
            REPO / "expense-reporter" / "expenses_log.jsonl"))

WIN_LO, WIN_HI = date(2025, 5, 1), date(2025, 12, 31)
MAX_INSTALLMENTS = 60

# ---------------------------------------------------------------- value parsing
# A faithful port of pkg/utils/currency.go + internal/parse.normalizeThousands.
# Ported rather than shelled out to so the probe stays a single read-only file;
# any divergence from the Go parser shows up as residue, which is the safe
# direction (it inflates the miss count rather than hiding one).

def normalize_thousands(s):
    """Dots stripped ONLY when a comma is also present (T-74 records the gap)."""
    return s.replace(".", "") if ("." in s and "," in s) else s

def parse_currency(s):
    s = s.strip()
    if not s:
        raise ValueError("empty")
    v = float(s.replace(",", "."))
    if v < 0:
        raise ValueError("negative")
    return v

def _plausible(n):
    return 0 < n <= MAX_INSTALLMENTS

def _readings(left, right):
    """readingA/B/C — shape only; the count range is applied by the caller."""
    out = []
    try:                                        # A: "<value> x <count>"
        out.append((parse_currency(left), int(right)))
    except ValueError:
        pass
    if right == "":                             # B: "<value> <count> x"
        parts = left.split()                    #    whitespace split is load bearing
        if len(parts) == 2:
            try:
                out.append((parse_currency(parts[0]), int(parts[1])))
            except ValueError:
                pass
    try:                                        # C: "<count> x <value>"
        out.append((parse_currency(right), int(left)))
    except ValueError:
        pass
    return out

def parse_with_installments(s):
    """-> (per_installment, count). Raises on anything the Go parser rejects.

    Two notations, inverses of each other: "300,00/3" states the TOTAL and divides,
    "405,25 x4" states the PER-INSTALLMENT amount and multiplies (T-64). Presence of
    the x/X marker IS the mode switch, and the value/count split resolves only when
    EXACTLY ONE reading holds — ambiguity is an error, never a guess.
    """
    s = normalize_thousands(s.strip())
    if "x" in s or "X" in s:
        if s.count("x") + s.count("X") != 1:
            raise ValueError("invalid installment format")
        i = min(s.index(c) for c in "xX" if c in s)
        possible = [r for r in _readings(s[:i].strip(), s[i+1:].strip())
                    if _plausible(r[1])]
        if len(possible) == 1:
            return possible[0]
        raise ValueError("ambiguous" if possible else "invalid installment format")
    if "/" in s:
        parts = s.split("/")
        if len(parts) != 2:
            raise ValueError("invalid installment format")
        total, count = parse_currency(parts[0]), int(parts[1].strip())
        if not _plausible(count):
            raise ValueError("count out of range")
        return total / count, count
    return parse_currency(s), 1

# ------------------------------------------------------------------ date handling

def add_months(d, n):
    """Port of appender.addMonths, including the day-overflow clamp."""
    y, m, day = d.year, d.month, d.day
    for _ in range(n):
        y, m = (y + 1, 1) if m == 12 else (y, m + 1)
    nxt = date(y + 1, 1, 1) if m == 12 else date(y, m + 1, 1)
    return date(y, m, min(day, (nxt - timedelta(days=1)).day))

def resolve_year(day, month, msg_d):
    """DD/MM carries no year; the message's own timestamp supplies it. A message
    may report a purchase from the previous month, or (near new year) the previous
    year, so the candidate is chosen by proximity rather than by assuming msg_year."""
    for y in (msg_d.year, msg_d.year - 1, msg_d.year + 1):
        try:
            cand = date(y, month, day)
        except ValueError:
            continue
        if -300 <= (cand - msg_d).days <= 60:
            return cand
    return None

DATE_RE = re.compile(r"^(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?$")

def parse_date_field(s, msg_d):
    m = DATE_RE.match(s.strip())
    if not m:
        raise ValueError("bad date")
    day, month = int(m.group(1)), int(m.group(2))
    if m.group(3):
        y = int(m.group(3))
        return date(y + 2000 if y < 100 else y, month, day)
    d = resolve_year(day, month, msg_d)
    if d is None:
        raise ValueError("unresolvable year")
    return d

# ------------------------------------------------------------------- the two sides

def parse_args(argv):
    source, out_dir = "converter", None
    it = iter(argv)
    for a in it:
        if a == "--source":
            source = next(it, "")
        elif a == "--out-dir":
            out_dir = Path(next(it, ""))
        else:
            raise SystemExit("unknown argument: %s (see the module docstring)" % a)
    if source not in ("naive", "converter"):
        raise SystemExit("--source must be naive or converter")
    return source, out_dir

def load_side_a(source, out_dir):
    return load_side_a_naive() if source == "naive" else load_side_a_converter(out_dir)

def strip_trailing_comment(line):
    """Port of cmd.stripTrailingComment (T-63): a '#' preceded by whitespace, once three
    fields are complete, starts a comment. Returns (data, comment)."""
    for i in range(1, len(line)):
        if line[i] == "#" and line[i - 1] in " \t" and line[:i].count(";") >= 2:
            return line[:i].rstrip(" \t"), line[i:]
    return line, ""

def converter_output(out_dir):
    """The directory holding the converter's files: the one given, or a fresh temp dir the
    Go command is run into right now — so the probe always reads THE converter, never a
    stale copy of its output."""
    if out_dir is not None:
        return out_dir
    tmp = Path(tempfile.mkdtemp(prefix="t75-probe-"))
    cmd = ["go", "run", "./cmd/expense-reporter", "telegram-import", str(EXPORT),
           "--out-dir", str(tmp)]
    p = subprocess.run(cmd, cwd=REPO / "expense-reporter", capture_output=True, text=True)
    if p.returncode != 0:
        raise SystemExit("converter failed (%d):\n%s%s" % (p.returncode, p.stdout, p.stderr))
    return tmp

def load_side_a_converter(out_dir):
    """Side A from the converter's month files. Every data line is item;DD/MM/YYYY;value,
    the year already resolved by the converter (plan D1), so only the VALUE goes through
    the Python port — that is the cross-check. A repaired line's "(message N)" comment
    supplies its id; other lines are keyed by file:line, which is enough for the series
    check (one line is one message's expense)."""
    out_dir = converter_output(out_dir)
    rows, skipped = [], Counter()
    for f in sorted(out_dir.glob("expenses-*.csv")):
        for n, raw in enumerate(f.read_text().splitlines(), 1):
            line = raw.rstrip(" \t")
            if not line or line.startswith("#"):
                continue
            data, comment = strip_trailing_comment(line)
            item, dstr, vstr = (p.strip() for p in data.split(";"))
            dd, mm, yy = dstr.split("/")
            d = date(int(yy), int(mm), int(dd))
            try:
                per, count = parse_with_installments(vstr)
            except (ValueError, ZeroDivisionError):
                skipped["value the Go parser took but the Python port rejects (DIVERGENCE)"] += 1
                continue
            m = re.search(r"\(message (\d+)\)", comment)
            msg_id = int(m.group(1)) if m else "%s:%d" % (f.name, n)
            for i in range(count):
                rows.append({"date": add_months(d, i), "cents": round(per * 100),
                             "item": item, "msg_id": msg_id, "n": count, "seq": i + 1,
                             "repaired": bool(comment)})
    rejects = out_dir / "telegram-rejects.csv"
    if rejects.exists():
        skipped["rejected by the converter (telegram-rejects.csv)"] = sum(
            1 for l in rejects.read_text().splitlines() if l.strip() and not l.startswith("#"))
    return rows, skipped

def load_side_a_naive():
    msgs = json.loads(EXPORT.read_text())["messages"]
    rows, skipped = [], Counter()
    for x in msgs:
        text = (x.get("text") or "").strip()
        msg_d = date.fromisoformat(x["date"][:10])
        if not text:
            skipped["empty (photo/file/sticker)"] += 1; continue
        parts = [p.strip() for p in text.split(";")]
        if len(parts) != 3:
            skipped["field count %d" % len(parts)] += 1; continue
        item, dstr, vstr = parts
        try:
            d = parse_date_field(dstr, msg_d)
        except ValueError:
            skipped["bad date field"] += 1; continue
        try:
            per, count = parse_with_installments(vstr)
        except (ValueError, ZeroDivisionError):
            skipped["bad value field"] += 1; continue
        for i in range(count):
            rows.append({"date": add_months(d, i), "cents": round(per * 100),
                         "item": item, "msg_id": x["id"], "n": count, "seq": i + 1})
    return rows, skipped

def load_side_b():
    rows = []
    for line in LOG.read_text().splitlines():
        if not line.strip():
            continue
        r = json.loads(line)
        try:
            d, m, y = r["date"].split("/")
            dd = date(int(y), int(m), int(d))
        except (ValueError, KeyError):
            continue
        if WIN_LO <= dd <= WIN_HI:
            rows.append({"date": dd, "cents": round(float(r["value"]) * 100),
                         "item": r.get("item", "")})
    return rows

# -------------------------------------------------------------------- the match

def key(r):
    return (r["date"], r["cents"])

def match(a_rows, b_rows):
    """Multiset intersection on (date, value).

    NEVER a set and never keyed on item. Duplicate purchases are real — s74 found 8
    genuine same-day duplicates that collide by construction — and a set silently
    eats them. That is the s72 failure exactly: an item-keyed join collapsed
    'San michel' x3 and reported 26 corrections against a known 25.
    """
    ca, cb = Counter(key(r) for r in a_rows), Counter(key(r) for r in b_rows)
    return sum((ca & cb).values()), ca - cb, cb - ca

def norm_tokens(s):
    s = unicodedata.normalize("NFD", s.lower())
    s = "".join(c for c in s if not unicodedata.combining(c))
    return set(re.findall(r"[a-z0-9]{3,}", s))

NEAR_DAYS = 7

def leftover_cause(item, d, cents, log_index):
    """Why a single-row residue entry has no partner — decided by what the log holds for
    the SAME item (every token of the export item inside one log item; 'all but one' was
    tried first and called 'Cartão <person C>' present because some log item contains 'anita').

      SUSPECT   the item is in the log with the same value within NEAR_DAYS days, or on the
                same date with another value: a converter error or a human edit. Someone
                must look, so this FAILS the gate.
      omission  the item recurs in the log but nothing is near this date/value: the human
                did not enter this one. A named cause; the log is lossy by construction.
      unknown   no log item carries the export item at all: never entered.
    """
    toks = norm_tokens(item)
    same = [r for r in log_index if toks and toks <= r["toks"]]
    if not same:
        return "message never entered in the workbook (item unknown to the log)"
    for r in same:
        if r["cents"] == cents and abs((r["date"] - d).days) <= NEAR_DAYS:
            return SUSPECT
        if r["date"] == d:
            return SUSPECT
    return "item recurs in the log, this date/value absent (omission)"

SUSPECT = ("SUSPECT: same item in the log with the same value within %d days, or on the same "
           "date with another value — converter error or human edit, INSPECT" % NEAR_DAYS)

# ------------------------------------------------------------------------- report

def main(argv=None):
    source, out_dir = parse_args(sys.argv[1:] if argv is None else argv)
    a_rows, skipped = load_side_a(source, out_dir)
    b_rows = load_side_b()
    matched, a_only, b_only = match(a_rows, b_rows)
    by_key = defaultdict(list)
    for r in a_rows:
        by_key[key(r)].append(r)

    print("=" * 70)
    print("SIDE A  %s" % ("Telegram export, naive conversion" if source == "naive"
                          else "converter output (expense-reporter telegram-import)"))
    print("=" * 70)
    label = "conforming messages" if source == "naive" else "converted lines    "
    print("  %s : %d" % (label, len({r["msg_id"] for r in a_rows})))
    if source == "converter":
        print("  of which repaired   : %d" % len({r["msg_id"] for r in a_rows if r["repaired"]}))
    print("  rows after expand   : %d" % len(a_rows))
    for b, c in skipped.most_common():
        print("      %3d skipped: %s" % (c, b))

    print()
    print("SIDE B  live expense log, %s .. %s : %d rows"
          % (WIN_LO, WIN_HI, len(b_rows)))

    print()
    print("=" * 70)
    print("MATCH on the (date, value) multiset")
    print("=" * 70)
    print("  matched          : %d" % matched)
    print("  A unmatched      : %d" % sum(a_only.values()))
    print("  B unmatched      : %d" % sum(b_only.values()))
    print("  A coverage       : %.1f%%  <- the number that validates a CAPTURE tool"
          % (100.0 * matched / len(a_rows)))
    print("  B coverage       : %.1f%%  <- expected to be low; the workbook holds"
          % (100.0 * matched / len(b_rows)))
    print("                             rows nobody types into the chat group")

    # --- residue, by named cause -------------------------------------------------
    by_date = defaultdict(list)
    for r in b_rows:
        by_date[r["date"]].append(r["cents"])
    # "never entered" is VERIFIED, not assumed: the catch-all used to be tautological
    # (every leftover row got that label, so "unexplained == 0" could not fail). Now the
    # log is consulted for the same ITEM — see leftover_cause — and a row the log holds
    # nearby is SUSPECT, which fails the run.
    log_index = [dict(r, toks=norm_tokens(r["item"])) for r in b_rows]
    buckets, detail = Counter(), defaultdict(list)
    for k, n in a_only.items():
        d, c = k
        sample = by_key[k][0]
        if not (WIN_LO <= d <= WIN_HI):
            b = "expanded outside the comparison window"
        elif c in by_date.get(d, []):
            b = "count mismatch on the same (date,value)"
        elif sample["n"] > 1:
            b = "installment tail absent from the log"
        else:
            b = leftover_cause(sample["item"], d, c, log_index)
        buckets[b] += n
        if len(detail[b]) < 3:
            detail[b].append("%s %9.2f  %s%s"
                             % (d.strftime("%d/%m/%Y"), c / 100, sample["item"][:36],
                                "  [repaired]" if sample.get("repaired") else ""))
    unexplained = buckets.get(SUSPECT, 0)
    print()
    print("  --- side A residue, by cause (every row must have one) ---")
    for b, c in buckets.most_common():
        print("      %3d  %s" % (c, b))
        for line in detail[b]:
            print("             %s" % line)
    print("  unexplained      : %d   <- the gate; must be 0" % unexplained)

    # --- are installment series complete in the log? ------------------------------
    # The decisive check. An earlier cut asked "does this value appear elsewhere in
    # the log" and was CONFOUNDED: within a series every installment shares a value,
    # so the answer is trivially yes and carries no information. Match on ITEM.
    log25 = [json.loads(l) for l in LOG.read_text().splitlines()
             if l.strip() and json.loads(l).get("date", "").endswith("2025")]
    series = defaultdict(list)
    for r in a_rows:
        series[r["msg_id"]].append(r)
    print()
    print("=" * 70)
    print("INSTALLMENT SERIES — is the log complete?")
    print("=" * 70)
    tally, missing_cents = Counter(), 0
    for _, rows in sorted(series.items(), key=lambda kv: kv[1][0]["date"]):
        if len(rows) < 2:
            continue
        rows.sort(key=lambda r: r["seq"])
        toks = norm_tokens(rows[0]["item"])
        if not toks:
            continue
        hits = [h for h in log25
                if len(toks & norm_tokens(h.get("item", ""))) >= max(1, len(toks) - 1)
                and round(float(h["value"]) * 100) == rows[0]["cents"]]
        state = "complete" if len(hits) >= len(rows) else "PARTIAL (%d of %d)" % (
            len(hits), len(rows))
        tally["complete" if state == "complete" else "PARTIAL"] += 1
        missing_cents += max(0, len(rows) - len(hits)) * rows[0]["cents"]
        print("  %-36s %2dx   %s" % (rows[0]["item"][:36], len(rows), state))
    print("  " + "-" * 66)
    for k, v in tally.most_common():
        print("  %d series %s" % (v, k))
    print("  value of the missing installment rows : R$ %.2f" % (missing_cents / 100))
    print("  total of the %d log rows in the window : R$ %.2f"
          % (len(b_rows), sum(r["cents"] for r in b_rows) / 100))

    # --- perturbation -------------------------------------------------------------
    # A comparison that cannot FAIL carries no information, however good its number
    # looks (s74: the workbook byte-comparison was first proven to move for a value
    # change and not for an id change). Corrupt one row; the residue must move by 1.
    print()
    print("=" * 70)
    print("PERTURBATION — can this comparison fail?")
    print("=" * 70)
    ok = True
    for label, mutate in (
        ("shift ONE value by 1 centavo", lambda r: dict(r, cents=r["cents"] + 1)),
        ("shift ONE date by 1 day", lambda r: dict(r, date=r["date"] + timedelta(days=1))),
    ):
        inter = Counter(key(r) for r in a_rows) & Counter(key(r) for r in b_rows)
        victim = next(i for i, r in enumerate(a_rows) if key(r) in inter)
        mutated = list(a_rows)
        mutated[victim] = mutate(mutated[victim])
        m2, _, _ = match(mutated, b_rows)
        good = (matched - m2) == 1
        ok &= good
        print("  %-31s %d -> %d   %s"
              % (label, matched, m2, "OK" if good else "*** NOT SENSITIVE ***"))
    print()
    print("GATE: %s" % ("PASS — every residue row has a named cause and the comparison can fail"
                        if ok and unexplained == 0 else
                        "FAIL — %d unexplained row(s)%s" % (unexplained, "" if ok else ", perturbation not sensitive")))
    return 0 if ok and unexplained == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
