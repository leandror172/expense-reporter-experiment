#!/usr/bin/env python3
"""
Reconstruct classified.csv and review.csv from batch-auto log + original CSV.
Line numbers in log match indices in original CSV (1-based).
"""
import re
import sys
from collections import defaultdict

def parse_log(log_path):
    """Parse batch-auto log and return dict: index → {status, subcategory, category, confidence}"""
    results = {}

    # Pattern: [123/373] STATUS item → Subcategory (95%)
    # Handles multiword subcategories like "Plano de saúde"
    pattern = r'^\[(\d+)/\d+\]\s+(\w+)\s+(.+?)\s+→\s+(.+?)\s+\((\d+)%\)$'

    with open(log_path, 'r', encoding='utf-8') as f:
        for line in f:
            line = line.rstrip('\n')
            match = re.match(pattern, line)
            if match:
                index = int(match.group(1))
                status = match.group(2)  # AUTO, REVIEW, SKIP
                # item = match.group(3)  # not needed; use CSV source
                subcategory = match.group(4).strip()
                confidence = int(match.group(5))

                results[index] = {
                    'status': status,
                    'subcategory': subcategory,
                    'confidence': confidence / 100.0,
                    'category': ''  # To be filled from taxonomy
                }
            elif 'SKIP' in line:
                # SKIP lines have parse errors; extract index before the error msg
                skip_match = re.match(r'^\[(\d+)/\d+\]\s+SKIP\s+', line)
                if skip_match:
                    index = int(skip_match.group(1))
                    results[index] = {
                        'status': 'SKIP',
                        'subcategory': '',
                        'confidence': 0.0,
                        'category': ''
                    }

    return results

def load_csv(csv_path):
    """Load original CSV. Returns list of (item, date, value) tuples."""
    rows = []
    with open(csv_path, 'r', encoding='utf-8') as f:
        for line in f:
            line = line.rstrip('\n')
            if not line.strip():
                continue
            parts = [p.strip() for p in line.split(';')]
            if len(parts) >= 3:
                rows.append((parts[0], parts[1], parts[2]))
            else:
                rows.append((line, '', ''))  # Malformed; preserve as-is
    return rows

def infer_category(subcategory):
    """Map subcategory to parent category. Expand as needed."""
    # Partial mapping from common categories
    category_map = {
        'Uber/Taxi': 'Transporte',
        'Plano de saúde': 'Saúde',
        'Supermercado': 'Alimentação',
        'Diarista': 'Habitação',
    }
    # Default: use subcategory as category if no mapping
    return category_map.get(subcategory, 'Diversos')

def write_csv(path, headers, rows):
    """Write CSV with semicolon delimiter."""
    with open(path, 'w', encoding='utf-8') as f:
        f.write(';'.join(headers) + '\n')
        for row in rows:
            f.write(';'.join(str(v) for v in row) + '\n')
    print(f"✓ {path} ({len(rows)} rows)")

def main(csv_path, log_path, output_dir):
    print(f"Loading original CSV: {csv_path}")
    csv_rows = load_csv(csv_path)

    print(f"Parsing log: {log_path}")
    log_results = parse_log(log_path)

    classified_rows = []
    review_rows = []
    skipped = 0

    headers = ['item', 'date', 'value', 'subcategory', 'category', 'confidence', 'auto_inserted']

    # Process each row: match by 1-based index
    for idx, (item, date, value) in enumerate(csv_rows, start=1):
        if idx not in log_results:
            print(f"⚠  Row {idx} not found in log (probably skipped)")
            skipped += 1
            continue

        result = log_results[idx]
        if result['status'] == 'SKIP':
            skipped += 1
            continue

        subcategory = result['subcategory']
        category = infer_category(subcategory)
        confidence = result['confidence']
        auto_inserted = 1 if result['status'] == 'AUTO' else 0

        row = (item, date, value, subcategory, category, f"{confidence:.2f}", auto_inserted)

        if result['status'] == 'AUTO':
            classified_rows.append(row)
        elif result['status'] == 'REVIEW':
            review_rows.append(row)

    print(f"\nResults:")
    print(f"  Classified (auto-inserted): {sum(1 for r in classified_rows if r[6] == 1)}")
    print(f"  Review (manual needed): {len(review_rows)}")
    print(f"  Skipped (parse errors): {skipped}")

    write_csv(f"{output_dir}/classified.csv", headers, classified_rows + review_rows)
    write_csv(f"{output_dir}/review.csv", headers, review_rows)

if __name__ == '__main__':
    csv_path = '/mnt/i/workspaces/expenses/extracted_entities.csv'
    log_path = '/mnt/i/workspaces/expenses/extracted_entities_log.txt'
    output_dir = '/mnt/i/workspaces/expenses'

    main(csv_path, log_path, output_dir)
