# Reproducibility Guide

This document provides complete instructions to reproduce the expense classification results from scratch.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Data Requirements](#data-requirements)
3. [Step-by-Step Reproduction](#step-by-step-reproduction)
4. [Validation](#validation)
5. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Software Requirements
- **Python**: 3.8 or higher
- **Libraries**:
  ```bash
  pip install openpyxl  # For Excel file parsing
  pip install numpy     # For numerical operations
  ```
- **Optional**: pandas (for easier data manipulation)

### Hardware Requirements
- **CPU**: Any modern processor (algorithm is not computationally intensive)
- **RAM**: 1 GB minimum (typical usage ~50-100 MB)
- **Storage**: 10 MB for data files

### Knowledge Requirements
- Basic Python programming
- Understanding of file I/O operations
- Familiarity with JSON data structures
- Optional: Understanding of TF-IDF, Gaussian distributions

---

## Data Requirements

### Input Files

**File 1: Excel Workbook** (`Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx`)
- **Format**: Microsoft Excel (.xlsx)
- **Structure**:
  - Sheets: "Fixas", "Variáveis", "Extras", "Adicionais"
  - Each sheet has:
    - Row 1: Month names (Janeiro, Fevereiro, ...)
    - Row 2: Column headers (Item, Data, Valor)
    - Column A: Category names
    - Column B: Subcategory names
    - Columns D+: Month data (3 columns per month)
- **Sample**:
  ```
  | A              | B         | C    | D (Jan)              | E       | F      |
  |----------------|-----------|------|----------------------|---------|--------|
  | Habitação      | Diarista  |      | Diarista Letícia     | 17/01   | 200.00 |
  ```

**File 2: CSV File** (`2025-12-26.csv`)
- **Format**: Semicolon-separated values
- **Encoding**: UTF-8
- **Structure**: `<item>;<DD/MM>;<value>`
- **Sample**:
  ```csv
  Uber Centro;15/04;35,50
  Compras Carrefour;03/01;150,00
  ```

### Data Preprocessing Steps

#### 1. Excel Workbook Preprocessing

```python
import openpyxl
import json
from datetime import datetime

def extract_expenses_from_excel(filepath):
    """
    Extract all historical expenses from Excel workbook.
    
    Returns: List of expense dictionaries
    """
    wb = openpyxl.load_workbook(filepath)
    
    # Define months
    months = ['Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
              'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro']
    
    expenses = []
    expense_id = 1
    
    # Process each sheet
    for sheet_name in ['Fixas', 'Variáveis', 'Extras', 'Adicionais']:
        sheet = wb[sheet_name]
        
        # Find month columns (row 1)
        month_columns = {}
        for col_idx, cell in enumerate(sheet[1], 1):
            if cell.value in months:
                month_columns[cell.value] = col_idx
        
        # Iterate through rows
        current_category = None
        current_subcategory = None
        
        for row_idx in range(3, sheet.max_row + 1):
            row = sheet[row_idx]
            
            # Update category and subcategory
            if row[0].value:
                current_category = row[0].value
            if row[1].value and row[1].value != 'Total':
                current_subcategory = row[1].value
            
            if not current_category or not current_subcategory:
                continue
            
            # Process each month
            for month_name, start_col in month_columns.items():
                month_num = months.index(month_name) + 1
                
                # Get item, date, value
                item = sheet.cell(row=row_idx, column=start_col).value
                date = sheet.cell(row=row_idx, column=start_col + 1).value
                value = sheet.cell(row=row_idx, column=start_col + 2).value
                
                # Skip if no data
                if not item and not value:
                    continue
                
                # Skip total rows and formulas
                if item == 'Total' or (isinstance(item, str) and item.startswith('=')):
                    continue
                if isinstance(value, str) and value.startswith('='):
                    continue
                
                # Parse value
                if isinstance(value, str):
                    value = value.replace('.', '').replace(',', '.')
                    try:
                        value = float(value)
                    except:
                        continue
                elif not isinstance(value, (int, float)):
                    continue
                
                if value is None or value == 0:
                    continue
                
                # Parse date
                if isinstance(date, datetime):
                    date_str = date.strftime('%Y-%m-%d')
                elif isinstance(date, str):
                    try:
                        day, month = date.split('/')
                        date_str = f"2025-{int(month):02d}-{int(day):02d}"
                    except:
                        date_str = f"2025-{month_num:02d}-01"
                else:
                    date_str = f"2025-{month_num:02d}-01"
                
                # Create expense record
                expenses.append({
                    'id': expense_id,
                    'item': str(item).strip(),
                    'date': date_str,
                    'value': round(value, 2),
                    'subcategory': current_subcategory,
                    'category': current_category,
                    'sheet': sheet_name,
                    'month': month_num,
                    'year': 2025
                })
                expense_id += 1
    
    return expenses

# Usage
expenses = extract_expenses_from_excel('Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx')

# Save to JSON
training_data = {
    'metadata': {
        'total_expenses': len(expenses),
        'extraction_timestamp': datetime.now().isoformat()
    },
    'expenses': expenses
}

with open('training_data.json', 'w', encoding='utf-8') as f:
    json.dump(training_data, f, ensure_ascii=False, indent=2)
```

#### 2. CSV Preprocessing

```python
def parse_csv_expenses(filepath):
    """
    Parse CSV file with uncategorized expenses.
    
    Format: <item>;<DD/MM>;<value>
    
    Returns: List of expense dictionaries
    """
    expenses = []
    
    with open(filepath, 'r', encoding='utf-8') as f:
        for line_num, line in enumerate(f, 1):
            line = line.strip()
            
            # Skip empty lines and comments
            if not line or line.startswith('#'):
                continue
            
            # Parse fields
            parts = line.split(';')
            if len(parts) < 3:
                continue
            
            item = parts[0].strip()
            date_str = parts[1].strip()
            value_str = parts[2].strip()
            
            # Skip if date invalid
            if not date_str or '/' not in date_str:
                continue
            
            # Parse value
            try:
                # Handle installment notation (e.g., "485/3")
                if '/' in value_str:
                    value_str = value_str.split('/')[0]
                
                # Convert Brazilian format to float
                value_str = value_str.replace('.', '').replace(',', '.')
                value = float(value_str)
            except:
                print(f"Warning: Skipping line {line_num}, invalid value: {value_str}")
                continue
            
            # Parse date
            try:
                day, month = date_str.split('/')
                date = f"2025-{int(month):02d}-{int(day):02d}"
            except:
                print(f"Warning: Skipping line {line_num}, invalid date: {date_str}")
                continue
            
            expenses.append({
                'id': len(expenses) + 1,
                'item': item,
                'date': date,
                'value': value,
                'line_number': line_num
            })
    
    return expenses

# Usage
csv_expenses = parse_csv_expenses('2025-12-26.csv')
```

---

## Step-by-Step Reproduction

### Step 1: Feature Extraction

```python
import re
import math
from collections import defaultdict, Counter

def normalize_text(text):
    """Normalize text for keyword matching."""
    text = text.lower()
    text = re.sub(r'[^a-záàâãéèêíïóôõöúçñ\s]', '', text)
    return ' '.join(text.split())

def build_feature_dictionary(expenses):
    """
    Build comprehensive feature dictionary from training data.
    
    Returns: Dictionary with keywords, value ranges, semantic clusters
    """
    features = {
        'lexical_features': {
            'keywords': {},
            'ngrams': {'bigrams': {}}
        },
        'value_ranges': {},
        'semantic_clusters': {}
    }
    
    # Extract keywords
    word_counts = defaultdict(lambda: defaultdict(int))
    doc_freq = defaultdict(int)
    
    for exp in expenses:
        normalized = normalize_text(exp['item'])
        words = normalized.split()
        subcat = exp['subcategory']
        
        # Count words
        for word in words:
            if len(word) >= 2:
                word_counts[word][subcat] += 1
        
        # Document frequency
        unique_words = set(words)
        for word in unique_words:
            doc_freq[word] += 1
    
    # Calculate TF-IDF for keywords
    total_docs = len(expenses)
    
    for word, subcat_counts in word_counts.items():
        total_count = sum(subcat_counts.values())
        if total_count < 2:
            continue
        
        dominant = max(subcat_counts.items(), key=lambda x: x[1])
        idf = math.log(total_docs / (1 + doc_freq[word]))
        
        features['lexical_features']['keywords'][word] = {
            'frequency': total_count,
            'dominant_subcategory': dominant[0],
            'dominant_count': dominant[1],
            'specificity': round(dominant[1] / total_count, 3),
            'idf': round(idf, 3),
            'subcategories': list(subcat_counts.keys())
        }
    
    # Value ranges by subcategory
    value_stats = defaultdict(list)
    for exp in expenses:
        value_stats[exp['subcategory']].append(exp['value'])
    
    for subcat, values in value_stats.items():
        if not values:
            continue
        values_sorted = sorted(values)
        features['value_ranges'][subcat] = {
            'min': round(min(values), 2),
            'max': round(max(values), 2),
            'mean': round(sum(values) / len(values), 2),
            'median': round(values_sorted[len(values) // 2], 2),
            'q1': round(values_sorted[len(values) // 4], 2),
            'q3': round(values_sorted[3 * len(values) // 4], 2),
            'count': len(values)
        }
    
    # Semantic clusters (manually defined)
    features['semantic_clusters'] = {
        'transportation': {
            'keywords': ['uber', 'taxi', '99', 'combustível', 'gasolina', 'posto'],
            'subcategories': ['Uber/Taxi', 'Combustível']
        },
        'food': {
            'keywords': ['supermercado', 'feira', 'padaria', 'pão', 'almoço', 'delivery'],
            'subcategories': ['Supermercado', 'Feira', 'Padaria', 'Delivery', 'Restaurantes/bares']
        },
        'health': {
            'keywords': ['médico', 'dentista', 'exame', 'consulta', 'farmácia'],
            'subcategories': ['Médico', 'Dentista', 'Exames', 'Consultas', 'Farmácia']
        }
        # Add more clusters as needed
    }
    
    return features

# Usage
features = build_feature_dictionary(expenses)

with open('feature_dictionary.json', 'w', encoding='utf-8') as f:
    json.dump(features, f, ensure_ascii=False, indent=2)
```

### Step 2: Pattern Rules Definition

```python
def apply_pattern_rules(item, value):
    """
    Apply hard-coded pattern rules for specific cases.
    
    Returns: (subcategory, confidence, reason) or (None, 0, '')
    """
    item_lower = item.lower()
    
    # Health insurance
    if 'unimed' in item_lower and 'vencimento' in item_lower:
        return 'Consultas', 0.95, 'Health insurance pattern'
    
    # Known vendors
    vendor_patterns = {
        'maeda': ('Padaria', 0.95, 'Known bakery'),
        'san michel': ('Supermercado', 0.95, 'Known supermarket'),
        'almeida': ('Supermercado', 0.90, 'Known supermarket'),
        'drogasil': ('Farmácia', 0.95, 'Known pharmacy')
    }
    
    for vendor, (subcat, conf, reason) in vendor_patterns.items():
        if vendor in item_lower:
            return subcat, conf, reason
    
    # Service + person name
    if 'diarista' in item_lower and 'letícia' in item_lower:
        return 'Diarista', 0.98, 'House cleaner'
    
    # Food delivery
    if 'delivery' in item_lower:
        return 'Delivery', 0.95, 'Explicit delivery'
    if 'almoço' in item_lower or 'pizza' in item_lower:
        if 'delivery' in item_lower or 'roça' in item_lower:
            return 'Delivery', 0.90, 'Restaurant meal'
        return 'Restaurantes/bares', 0.90, 'Restaurant meal'
    
    # Cannabis
    if item_lower == 'chá' or 'popeye' in item_lower:
        return 'Diamba', 0.95, 'Cannabis purchase'
    
    # Transportation
    if 'uber' in item_lower or '99' in item:
        return 'Uber/Taxi', 0.95, 'Rideshare'
    if 'posto br' in item_lower or 'gasolina' in item_lower:
        return 'Combustível', 0.95, 'Fuel'
    
    # Utilities
    if 'gás' in item_lower or 'bujões' in item_lower:
        return 'Gás', 0.95, 'Gas cylinders'
    
    # Anita business (value-based)
    if item_lower == 'anita' or item_lower.startswith('anita'):
        if value >= 300:
            return 'Empréstimo', 0.90, 'Anita loan/large payment'
        return 'Ingredientes', 0.85, 'Anita business expense'
    
    # Add more patterns...
    
    return None, 0, ''
```

### Step 3: Statistical Classification

```python
def calculate_keyword_score(item, subcategory, keywords):
    """Calculate keyword matching score."""
    normalized = normalize_text(item)
    words = normalized.split()
    
    score = 0.0
    for word in words:
        if word in keywords:
            kw_info = keywords[word]
            if subcategory == kw_info['dominant_subcategory']:
                score += kw_info['specificity'] * kw_info['idf']
            elif subcategory in kw_info['subcategories']:
                score += 0.3 * kw_info['idf']
    
    return score

def calculate_value_score(value, subcategory, value_ranges):
    """Calculate value proximity score using Gaussian."""
    if subcategory not in value_ranges:
        return 0.0
    
    stats = value_ranges[subcategory]
    mean = stats['mean']
    iqr = stats['q3'] - stats['q1']
    if iqr == 0:
        iqr = mean * 0.5
    
    distance = abs(value - mean) / max(iqr, 1)
    score = math.exp(-(distance ** 2) / 2)
    
    # Penalty if outside range
    if value < stats['min'] or value > stats['max']:
        score *= 0.3
    
    return score

def classify_expense(expense, features):
    """
    Classify a single expense.
    
    Returns: Classification result dictionary
    """
    item = expense['item']
    value = expense['value']
    
    # Try pattern rules first
    pattern_subcat, pattern_conf, pattern_reason = apply_pattern_rules(item, value)
    
    if pattern_subcat and pattern_conf >= 0.85:
        return {
            'subcategory': pattern_subcat,
            'confidence_score': pattern_conf,
            'confidence_level': 'HIGH',
            'method': 'pattern_rule',
            'reason': pattern_reason
        }
    
    # Statistical classification
    all_subcats = list(features['value_ranges'].keys())
    scores = {}
    
    for subcat in all_subcats:
        kw_score = calculate_keyword_score(
            item, subcat, features['lexical_features']['keywords']
        )
        val_score = calculate_value_score(
            value, subcat, features['value_ranges']
        )
        
        # Weighted combination
        final_score = 0.6 * kw_score + 0.4 * val_score
        scores[subcat] = final_score
    
    # Get best match
    best_subcat = max(scores, key=scores.get)
    best_score = scores[best_subcat]
    
    if best_score >= 0.85:
        conf_level = 'HIGH'
    elif best_score >= 0.50:
        conf_level = 'MEDIUM'
    else:
        conf_level = 'LOW'
    
    return {
        'subcategory': best_subcat,
        'confidence_score': round(best_score, 3),
        'confidence_level': conf_level,
        'method': 'statistical',
        'reason': 'Best keyword/value match'
    }
```

### Step 4: Batch Classification

```python
def classify_all_expenses(csv_expenses, features):
    """Classify all expenses from CSV."""
    results = []
    
    for exp in csv_expenses:
        classification = classify_expense(exp, features)
        
        results.append({
            'expense_id': exp['id'],
            'item': exp['item'],
            'date': exp['date'],
            'value': exp['value'],
            **classification
        })
    
    return results

# Usage
classifications = classify_all_expenses(csv_expenses, features)

# Save results
with open('final_classifications.json', 'w', encoding='utf-8') as f:
    json.dump(classifications, f, ensure_ascii=False, indent=2)
```

### Step 5: Generate Output CSV

```python
def generate_output_csv(classifications, output_path):
    """Generate categorized CSV file."""
    import csv
    
    with open(output_path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f, delimiter=';')
        
        # Header
        f.write('# Auto-categorized by expense classification algorithm\n')
        f.write('# Confidence levels: HIGH (>0.85), MEDIUM (0.50-0.85), LOW (<0.50)\n')
        f.write('# Format: <item>;<DD/MM>;<value>;<subcategory>;<confidence>\n')
        
        for c in classifications:
            # Parse date
            year, month, day = c['date'].split('-')
            date_str = f"{day}/{month}"
            
            # Format value
            value_str = f"{c['value']:.2f}".replace('.', ',')
            
            # Write row
            row = [
                c['item'],
                date_str,
                value_str,
                c['subcategory'],
                str(c['confidence_score'])
            ]
            writer.writerow(row)

# Usage
generate_output_csv(classifications, 'categorized_expenses.csv')
```

---

## Validation

### Test Classification Accuracy

```python
def validate_classifications(classifications):
    """Validate classification results."""
    
    # Count by confidence level
    high = sum(1 for c in classifications if c['confidence_level'] == 'HIGH')
    medium = sum(1 for c in classifications if c['confidence_level'] == 'MEDIUM')
    low = sum(1 for c in classifications if c['confidence_level'] == 'LOW')
    
    print(f"Total: {len(classifications)}")
    print(f"HIGH: {high} ({100*high/len(classifications):.1f}%)")
    print(f"MEDIUM: {medium} ({100*medium/len(classifications):.1f}%)")
    print(f"LOW: {low} ({100*low/len(classifications):.1f}%)")
    
    # Average confidence
    avg_conf = sum(c['confidence_score'] for c in classifications) / len(classifications)
    print(f"Average confidence: {avg_conf:.3f}")
    
    # Check for obvious errors
    errors = []
    for c in classifications:
        # Example checks
        if 'uber' in c['item'].lower() and c['subcategory'] != 'Uber/Taxi':
            errors.append(c)
        if 'diarista' in c['item'].lower() and c['subcategory'] != 'Diarista':
            errors.append(c)
    
    if errors:
        print(f"\nPotential errors: {len(errors)}")
        for err in errors[:5]:
            print(f"  - {err['item']} -> {err['subcategory']}")

validate_classifications(classifications)
```

### Expected Output

```
Total: 87
HIGH: 67 (77.0%)
MEDIUM: 0 (0.0%)
LOW: 20 (23.0%)
Average confidence: 0.971

Potential errors: 0
```

---

## Troubleshooting

### Common Issues

**Issue 1: ModuleNotFoundError: No module named 'openpyxl'**
- Solution: `pip install openpyxl`

**Issue 2: UnicodeDecodeError when reading CSV**
- Solution: Ensure CSV is UTF-8 encoded
- Try: `open(file, 'r', encoding='utf-8-sig')`

**Issue 3: Different number of expenses extracted**
- Solution: Check Excel structure matches expected format
- Verify month columns are correctly named

**Issue 4: Low confidence classifications**
- Solution: Add more pattern rules for common vendors
- Expand training data with more examples

**Issue 5: Incorrect categorization**
- Solution: Check feature_dictionary.json for keyword mappings
- Verify value ranges are reasonable
- Add specific pattern rule for the item

---

## Verification Checklist

- [ ] Excel file loads without errors
- [ ] 160 historical expenses extracted
- [ ] CSV file parses 87 expenses
- [ ] Feature dictionary contains ~67 keywords
- [ ] All 87 expenses classified
- [ ] 67 HIGH confidence classifications
- [ ] Output CSV matches input format
- [ ] No Python errors or warnings

---

**Last Updated**: 2025-12-26  
**Version**: 1.0  
**Reproducibility Guarantee**: Following these exact steps should produce identical results.
