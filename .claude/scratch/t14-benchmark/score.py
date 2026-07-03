#!/usr/bin/env python3
"""Score T-14 benchmark results: accuracy, latency, calibration, leakage split, OOD sentinel-decline rate."""
import json
import argparse
import statistics
from pathlib import Path
import logging
import re

# Configure logging with lazy formatting
logging.basicConfig(level=logging.INFO, format='%(message)s')

def read_jsonl(path: Path) -> list[dict]:
    """Parse each non-empty line as JSON. Return [] if the file does not exist."""
    if not path.exists():
        return []
    
    results = []
    try:
        with path.open('r', encoding='utf-8') as f:
            for line in f:
                stripped_line = line.strip()
                if stripped_line:
                    results.append(json.loads(stripped_line))
    except json.JSONDecodeError as e:
        logging.error(f"JSON decode error in {path}: {e}")
    return results

def join_results(sample: list[dict], results: list[dict]) -> list[dict]:
    """For each result whose id exists in the sample, return a merged dict {**sample_row, **result}.
    Results with ids not in the sample are ignored."""
    sample_id_set = {str(row['id']) for row in sample}
    joined_rows = []
    
    for result in results:
        if str(result['id']) in sample_id_set:
            sample_row = next((row for row in sample if str(row['id']) == str(result['id'])), None)
            if sample_row is not None:
                merged_row = {**sample_row, **result}
                joined_rows.append(merged_row)
    
    return joined_rows

def is_full_match(row: dict) -> bool:
    """ok, top is not None, and top's type/category/subcategory equal the expected_* fields."""
    if not row.get('ok') or row.get('top') is None:
        return False
    
    top = row['top']
    expected_type = row.get('expected_type')
    expected_category = row.get('expected_category')
    expected_subcategory = row.get('expected_subcategory')
    
    return (top['type'] == expected_type and 
            top['category'] == expected_category and 
            top['subcategory'] == expected_subcategory)

def is_catsub_match(row: dict) -> bool:
    """ok, top not None, top's category and subcategory equal expected."""
    if not row.get('ok') or row.get('top') is None:
        return False
    
    top = row['top']
    expected_category = row.get('expected_category')
    expected_subcategory = row.get('expected_subcategory')
    
    return (top['category'] == expected_category and 
            top['subcategory'] == expected_subcategory)

def is_sub_match(row: dict) -> bool:
    """ok, top not None, top's subcategory equals expected."""
    if not row.get('ok') or row.get('top') is None:
        return False
    
    top = row['top']
    expected_subcategory = row.get('expected_subcategory')
    
    return top['subcategory'] == expected_subcategory

def is_type_match(row: dict) -> bool:
    """ok, top not None, top's type equals expected."""
    if not row.get('ok') or row.get('top') is None:
        return False
    
    top = row['top']
    expected_type = row.get('expected_type')
    
    return top['type'] == expected_type

def pct(part: int, whole: int) -> str:
    """'87.3%' with one decimal; '-' when whole is 0."""
    if whole == 0:
        return '-'
    try:
        percentage = (part / whole) * 100
        return f"{percentage:.1f}%"
    except ZeroDivisionError:
        return '-'

def accuracy_line(rows: list[dict], label: str) -> str:
    """One markdown table row: | label | n | full-path% | cat+sub% | sub% | type% |"""
    total = len(rows)
    full_match_count = sum(1 for row in rows if is_full_match(row))
    catsub_match_count = sum(1 for row in rows if is_catsub_match(row))
    sub_match_count = sum(1 for row in rows if is_sub_match(row))
    type_match_count = sum(1 for row in rows if is_type_match(row))
    
    return f"| {label} | {total} | {pct(full_match_count, total)} | {pct(catsub_match_count, total)} | {pct(sub_match_count, total)} | {pct(type_match_count, total)} |"

def latency_stats(rows: list[dict]) -> str:
    """Over ALL rows (ok or not): 'mean 7.1s / median 6.8s / p90 12.3s' (p90 = quantiles n=10)."""
    latencies = [row.get('latency_s', 0) for row in rows]
    
    if not latencies:
        return "mean 0.0s / median 0.0s / p90 0.0s"
    
    mean_latency = statistics.mean(latencies)
    median_latency = statistics.median(latencies)
    p90_latency = statistics.quantiles(latencies, n=10)[-1] if len(latencies) >= 2 else latencies[0]
    
    return f"mean {mean_latency:.1f}s / median {median_latency:.1f}s / p90 {p90_latency:.1f}s"

def calibration_summary(rows: list[dict]) -> list[str]:
    """Over ok rows with top: lines reporting (a) mean confidence on full-path-correct vs wrong rows,
    (b) count and percent of WRONG (not full-path-correct) rows with confidence >= 0.85 — the
    would-have-auto-inserted-wrongly danger metric."""
    correct_rows = []
    incorrect_rows = []
    
    for row in rows:
        if not row.get('ok') or row.get('top') is None:
            continue
        
        top = row['top']
        expected_type = row.get('expected_type')
        expected_category = row.get('expected_category')
        expected_subcategory = row.get('expected_subcategory')
        
        full_match = (top['type'] == expected_type and 
                      top['category'] == expected_category and 
                      top['subcategory'] == expected_subcategory)
        
        confidence = top.get('confidence', 0.0)
        
        if full_match:
            correct_rows.append(confidence)
        else:
            incorrect_rows.append(confidence)
    
    summary = []
    
    # Mean confidence on correct vs incorrect rows
    if correct_rows and incorrect_rows:
        mean_correct_confidence = statistics.mean(correct_rows)
        mean_incorrect_confidence = statistics.mean(incorrect_rows)
        summary.append(f"Mean confidence: {mean_correct_confidence:.2f} (correct) / {mean_incorrect_confidence:.2f} (incorrect)")
    
    # Count and percent of incorrect rows with confidence >= 0.85
    high_confidence_incorrect = sum(1 for conf in incorrect_rows if conf >= 0.85)
    total_incorrect = len(incorrect_rows)
    
    if total_incorrect > 0:
        percent_high_confidence = (high_confidence_incorrect / total_incorrect) * 100
        summary.append(f"High confidence wrong: {high_confidence_incorrect} ({percent_high_confidence:.1f}%)")
    
    return summary

def is_sentinel_decline(row: dict) -> bool:
    """ok, top not None, top's category == 'Diversos' and top's confidence <= 0.30."""
    if not row.get('ok') or row.get('top') is None:
        return False
    
    top = row['top']
    subcategory = top.get('subcategory')
    confidence = top.get('confidence', 0.0)
    
    return (subcategory == 'Diversos' and confidence <= 0.30)

def score_model(sample: list[dict], results_path: Path) -> list[str]:
    """Return markdown lines for one model: header with model name (from filename), failure count,
    a table with accuracy_line for: all rows, leaky==True rows, leaky==False rows;
    then latency_stats and calibration_summary lines."""
    model_name = results_path.name.replace('results-', '').replace('.jsonl', '')
    
    results = read_jsonl(results_path)
    joined_rows = join_results(sample, results)
    
    total_rows = len(joined_rows)
    failure_count = sum(1 for row in joined_rows if not row.get('ok'))
    
    # Filter rows by leaky status
    all_rows = [row for row in joined_rows]
    leaky_true_rows = [row for row in joined_rows if row.get('leaky', False)]
    leaky_false_rows = [row for row in joined_rows if not row.get('leaky', False)]
    
    accuracy_lines = [
        "| subset | n | full-path | cat+sub | sub | type |",
        "|---|---|---|---|---|---|",
        accuracy_line(all_rows, "All"),
        accuracy_line(leaky_true_rows, "Leaky=True"),
        accuracy_line(leaky_false_rows, "Leaky=False")
    ]
    
    latency_line = latency_stats(joined_rows)
    calibration_lines = calibration_summary(joined_rows)
    
    markdown_lines = []
    markdown_lines.append(f"## {model_name}")
    markdown_lines.append(f"Failures: {failure_count}")
    markdown_lines.extend(accuracy_lines)
    markdown_lines.append(latency_line)
    markdown_lines.extend(calibration_lines)
    
    return markdown_lines

def score_ood(ood: list[dict], results_path: Path) -> list[str]:
    """Join by id; return lines: total OOD attempted, sentinel-decline count/percent, and for each
    NON-declined row a line 'item -> type/category/subcategory @ confidence'."""
    ood_rows = ood
    results = read_jsonl(results_path)
    
    joined_rows = []
    ood_id_set = {str(row['id']) for row in ood_rows}
    
    for result in results:
        if str(result['id']) in ood_id_set:
            ood_row = next((row for row in ood_rows if str(row['id']) == str(result['id'])), None)
            if ood_row is not None:
                merged_row = {**ood_row, **result}
                joined_rows.append(merged_row)
    
    total_attempted = len(joined_rows)
    sentinel_decline_count = sum(1 for row in joined_rows if is_sentinel_decline(row))
    
    sentinel_decline_percent = (sentinel_decline_count / total_attempted) * 100 if total_attempted > 0 else 0
    
    markdown_lines = []
    markdown_lines.append(f"Total OOD attempted: {total_attempted}")
    markdown_lines.append(f"Sentinel decline count: {sentinel_decline_count} ({sentinel_decline_percent:.1f}%)")
    
    for row in joined_rows:
        if not is_sentinel_decline(row):
            item = row.get('item', '')
            top = row.get('top')
            confidence = top.get('confidence', 0.0)
            
            if top and 'type' in top and 'category' in top and 'subcategory' in top:
                markdown_lines.append(f"{item} -> {top['type']}/{top['category']}/{top['subcategory']} @ {confidence:.2f}")
    
    return markdown_lines

def main() -> None:
    """argparse: --dir (Path, default Path(__file__).parent). Read sample.jsonl and ood.jsonl from dir.
    For every file matching results-*.jsonl in dir (sorted): print score_model lines; if ood.jsonl is
    non-empty and a matching ood-results-*.jsonl exists for the same model suffix, print score_ood lines.
    Print everything to stdout as markdown."""
    parser = argparse.ArgumentParser(description="Score T-14 benchmark results")
    parser.add_argument('--dir', type=Path, default=Path(__file__).parent)
    args = parser.parse_args()
    
    dir_path = args.dir
    sample_path = dir_path / "sample.jsonl"
    ood_path = dir_path / "ood.jsonl"
    
    sample_rows = read_jsonl(sample_path)
    ood_rows = read_jsonl(ood_path)
    
    results_files = sorted(dir_path.glob("results-*.jsonl"))
    
    all_output_lines = []
    
    for result_file in results_files:
        model_lines = score_model(sample_rows, result_file)
        all_output_lines.extend(model_lines)
        
        # Check if ood.jsonl is non-empty and a matching ood-results file exists
        if ood_rows and ood_path.exists():
            ood_result_file = dir_path / f"ood-{result_file.name}"
            if ood_result_file.exists():
                ood_model_lines = score_ood(ood_rows, ood_result_file)
                all_output_lines.extend(ood_model_lines)
    
    # Print everything to stdout
    for line in all_output_lines:
        print(line)

if __name__ == '__main__':
    main()
