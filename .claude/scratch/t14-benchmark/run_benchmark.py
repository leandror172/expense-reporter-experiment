#!/usr/bin/env python3
"""Run the T-14 classifier benchmark: call the expense-reporter binary once per sample item, append results to JSONL (resumable)."""
import subprocess
import json
import time
import argparse
from pathlib import Path
import logging
import sys

# Configure logging with lazy formatting
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

REPO_ROOT = Path(__file__).resolve().parents[3]
BINARY = REPO_ROOT / "expense-reporter" / "expense-reporter"

def read_jsonl(path: Path) -> list[dict]:
    """Parse each non-empty line as JSON. Return [] if the file does not exist."""
    results = []
    try:
        if path.exists():
            with path.open('r', encoding='utf-8') as f:
                for line in f:
                    stripped_line = line.strip()
                    if stripped_line:
                        try:
                            results.append(json.loads(stripped_line))
                        except json.JSONDecodeError as e:
                            logging.warning("Failed to decode JSON: %s", e)
    except Exception as e:
        logging.error("Error reading JSONL file: %s", e)
    return results

def done_ids(results_path: Path) -> set:
    """Set of rec['id'] already present in the results file."""
    done = set()
    try:
        if results_path.exists():
            with results_path.open('r', encoding='utf-8') as f:
                for line in f:
                    stripped_line = line.strip()
                    if stripped_line:
                        try:
                            record = json.loads(stripped_line)
                            done.add(record['id'])
                        except json.JSONDecodeError as e:
                            logging.warning("Failed to decode JSON: %s", e)
    except Exception as e:
        logging.error("Error reading results file for existing IDs: %s", e)
    return done

def br_value(value: float) -> str:
    """54.5 -> '54,50' (two decimals, comma separator)."""
    return f"{value:.2f}".replace('.', ',')

def classify_once(entry: dict, model: str, timeout_s: int, extra_args: list = ()) -> dict:
    """Run [BINARY, 'classify', entry['item'], br_value(entry['value']), entry['date'],
    '--model', model, '--json', '--top', '3'] with cwd=REPO_ROOT, capturing stdout+stderr as text.
    Measure wall-clock latency with time.monotonic().
    Return {'id': entry['id'], 'model': model, 'latency_s': round(latency, 2), 'ok': bool, ...}:
      - on exit code 0 and stdout parses as JSON: ok=True, plus 'top' = first element of the
        parsed object's 'candidates' list (or None if empty) and 'candidates' = the full list.
      - on non-zero exit: ok=False, 'error' = last 300 chars of stderr.
      - on subprocess.TimeoutExpired: ok=False, 'error' = 'timeout'.
      - on json.JSONDecodeError: ok=False, 'error' = 'bad json: ' + first 200 chars of stdout."""
    try:
        start_time = time.monotonic()
        process = subprocess.Popen(
            [
                str(BINARY), 'classify',
                entry['item'], br_value(entry['value']), entry['date'],
                '--model', model, '--json', '--top', '3', *extra_args
            ],
            cwd=str(REPO_ROOT),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        stdout, stderr = process.communicate(timeout=timeout_s)
    except subprocess.TimeoutExpired:
        process.kill()
        process.communicate()
        logging.warning("Timeout for entry %s", entry['id'])
        return {
            'id': entry['id'],
            'model': model,
            'latency_s': round(time.monotonic() - start_time, 2),
            'ok': False,
            'error': 'timeout'
        }
    except Exception as e:
        logging.error("Error during classification for entry %s: %s", entry['id'], e)
        return {
            'id': entry['id'],
            'model': model,
            'latency_s': round(time.monotonic() - start_time, 2),
            'ok': False,
            'error': str(e)
        }

    exit_code = process.returncode

    if exit_code != 0:
        logging.warning("Non-zero exit code for entry %s: %d", entry['id'], exit_code)
        return {
            'id': entry['id'],
            'model': model,
            'latency_s': round(time.monotonic() - start_time, 2),
            'ok': False,
            'error': stderr[-300:] if len(stderr) > 300 else stderr
        }

    try:
        parsed = json.loads(stdout)
        top_candidate = parsed.get('candidates', [{}])[0] if parsed.get('candidates') else None
        return {
            'id': entry['id'],
            'model': model,
            'latency_s': round(time.monotonic() - start_time, 2),
            'ok': True,
            'top': top_candidate,
            'candidates': parsed.get('candidates', [])
        }
    except json.JSONDecodeError as e:
        logging.warning("Failed to decode JSON output for entry %s: %s", entry['id'], e)
        return {
            'id': entry['id'],
            'model': model,
            'latency_s': round(time.monotonic() - start_time, 2),
            'ok': False,
            'error': f'bad json: {stdout[:200]}'
        }

def append_result(results_path: Path, rec: dict) -> None:
    """Append one json.dumps(ensure_ascii=False) line; open in 'a' mode and flush per line."""
    try:
        with results_path.open('a', encoding='utf-8') as f:
            f.write(json.dumps(rec, ensure_ascii=False))
            f.write('\n')
            f.flush()
    except Exception as e:
        logging.error("Error appending result to file: %s", e)

def main() -> None:
    """argparse: --model (required), --sample (Path, default Path(__file__).parent/'sample.jsonl'),
    --outdir (Path, default Path(__file__).parent), --limit (int, default 0 = no limit),
    --timeout (int seconds, default 240).
    Results file: outdir / f'results-{model}.jsonl'. Load sample, skip entries whose id is in
    done_ids (print how many skipped). Respect --limit as the max number of NEW calls this run.
    For each remaining entry: classify_once, append_result immediately, and print a one-line
    progress like '17/300 id=1567 ok=True 11.8s'. At the end print counts: attempted, ok, failed."""

    parser = argparse.ArgumentParser(description="Run T-14 classifier benchmark")
    parser.add_argument('--model', required=True, help='Model name')
    parser.add_argument('--sample', type=Path, default=Path(__file__).parent / 'sample.jsonl',
                        help='Sample JSONL file path (default: sample.jsonl in the same directory)')
    parser.add_argument('--outdir', type=Path, default=Path(__file__).parent,
                        help='Output directory for results (default: current directory)')
    parser.add_argument('--limit', type=int, default=0, help='Maximum number of new calls to perform')
    parser.add_argument('--timeout', type=int, default=240, help='Timeout in seconds for each classification call')
    parser.add_argument('--prefix', type=str, default='results', help='Results filename prefix (use "ood-results" for OOD runs)')
    parser.add_argument('--extra-args', type=str, default='', help='Extra CLI args appended to every classify call (space-separated)')

    args = parser.parse_args()

    results_path = args.outdir / f'{args.prefix}-{args.model}.jsonl'
    sample_path = args.sample
    limit = args.limit
    timeout_s = args.timeout

    try:
        sample_entries = read_jsonl(sample_path)
    except Exception as e:
        logging.error("Error reading sample file: %s", e)
        sys.exit(1)

    done_ids_set = done_ids(results_path)
    skipped_count = 0
    remaining_entries = []

    for entry in sample_entries:
        if entry['id'] in done_ids_set:
            skipped_count += 1
        else:
            remaining_entries.append(entry)

    logging.info("Skipped %d entries already processed", skipped_count)

    attempted = 0
    ok_count = 0
    failed_count = 0

    for idx, entry in enumerate(remaining_entries):
        if limit > 0 and idx >= limit:
            break
        result = classify_once(entry, args.model, timeout_s, args.extra_args.split())
        append_result(results_path, result)

        attempted += 1
        if result['ok']:
            ok_count += 1
        else:
            failed_count += 1

        progress_msg = f"{idx + 1}/{len(remaining_entries)} id={entry['id']} ok={result['ok']} {result['latency_s']:.1f}s"
        logging.info(progress_msg)

    logging.info("Total attempted: %d, OK: %d, Failed: %d", attempted, ok_count, failed_count)

if __name__ == '__main__':
    main()
