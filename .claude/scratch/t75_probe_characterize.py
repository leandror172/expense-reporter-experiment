#!/usr/bin/env python3
"""Characterization test for t75_oracle_probe.py — the `characterize-first` step of the
T-75 plan (tracker step 4). The probe is committed code with no tests of its own; before
its conversion source changes, every number it prints is pinned here, BLACK-BOX (the
probe is run as a subprocess and its stdout parsed), so a change in behaviour shows up
as a diff against the numbers `.claude/t75-oracle-viability.md` quotes.

Usage: python3 t75_probe_characterize.py [--source naive|converter] [probe args...]
  --source picks the expectation set; the remaining args are passed to the probe.
  Exit 0 when every pinned number matches, 1 otherwise (each mismatch is printed).
"""
import os, re, subprocess, sys
from decimal import Decimal

HERE = os.path.dirname(os.path.abspath(__file__))
PROBE = os.path.join(HERE, "t75_oracle_probe.py")

# ---- the pinned numbers -----------------------------------------------------------
# naive: the probe's own throwaway conversion of the 352 conforming messages, exactly
# as measured s75 and written into the report. These must NEVER drift silently.
EXPECT = {
    "naive": {
        "converted": 352,
        "rows": 383,
        "skipped": {"empty (photo/file/sticker)": 20, "field count 2": 12,
                    "bad date field": 4, "field count 1": 2, "bad value field": 2,
                    "field count 4": 1},
        "matched": 346, "a_unmatched": 37, "b_unmatched": 153,
        "residue": {"installment tail absent from the log": 24,
                    "expanded outside the comparison window": 9,
                    "message never entered in the workbook": 3,
                    "count mismatch on the same (date,value)": 1},
        "series": {"PARTIAL": 14},
        "missing": Decimal("7309.43"),
        "perturbation": [("shift ONE value by 1 centavo", 346, 345, "OK"),
                         ("shift ONE date by 1 day", 346, 345, "OK")],
        "exit": 0,
    },
}


def run_probe(args):
    p = subprocess.run([sys.executable, PROBE, *args], capture_output=True, text=True)
    return p.stdout + p.stderr, p.returncode


def parse(out):
    got = {"skipped": {}, "residue": {}, "series": {}, "perturbation": []}
    section = None
    for line in out.splitlines():
        if "side A residue" in line:
            section = "residue"; continue
        if line.startswith("INSTALLMENT SERIES"):
            section = "series"; continue
        if line.startswith("PERTURBATION"):
            section = "perturbation"; continue
        m = re.match(r"\s+(conforming messages|converted lines)\s*: (\d+)", line)
        if m: got["converted"] = int(m.group(2)); continue
        m = re.match(r"\s+rows after expand\s*: (\d+)", line)
        if m: got["rows"] = int(m.group(1)); continue
        m = re.match(r"\s+(\d+) skipped: (.+)$", line)
        if m: got["skipped"][m.group(2).strip()] = int(m.group(1)); continue
        m = re.match(r"\s+(matched|A unmatched|B unmatched)\s*: (\d+)", line)
        if m: got[m.group(1).lower().replace(" ", "_")] = int(m.group(2)); continue
        if section == "residue":
            m = re.match(r"^\s+(\d+)\s\s(\D.*?)\s*$", line)
            if m: got["residue"][m.group(2)] = int(m.group(1)); continue
        m = re.match(r"\s+(\d+) series (\w+)$", line)
        if m: got["series"][m.group(2)] = int(m.group(1)); continue
        m = re.match(r"\s+value of the missing installment rows : R\$ ([\d.]+)", line)
        if m: got["missing"] = Decimal(m.group(1)); continue
        m = re.match(r"\s+(shift ONE \w+ by 1 \w+)\s+(\d+) -> (\d+)\s+(.+?)\s*$", line)
        if m: got["perturbation"].append((m.group(1), int(m.group(2)), int(m.group(3)), m.group(4)))
    return got


def main():
    args = sys.argv[1:]
    source = "naive"
    if "--source" in args:
        i = args.index("--source"); source = args[i + 1]
    want = EXPECT.get(source)
    if want is None:
        print("no expectations pinned for --source", source); return 1
    out, code = run_probe(args)
    got = parse(out); got["exit"] = code
    bad = [(k, want[k], got.get(k)) for k in want if got.get(k) != want[k]]
    for k, w, g in bad:
        print("MISMATCH %-13s want %r\n                   got  %r" % (k, w, g))
    print("characterization (%s): %s — %d field(s) compared, %d mismatch(es)"
          % (source, "GREEN" if not bad else "RED", len(want), len(bad)))
    return 0 if not bad else 1


if __name__ == "__main__":
    sys.exit(main())
