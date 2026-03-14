# Expense Classifier: External Data Inventory

**Date:** 2026-02-26
**Source location:** `I:\workspaces\expenses\` (copy from `Z:\Meu Drive\controle\`)

This document maps the auto-categorization analysis artifacts produced by Claude on claude.ai
(Dec 2025 - Jan 2026) and the existing Go expense-reporter tool.

---

## Auto-Category Analysis (`I:\workspaces\expenses\auto-category\`)

### Must-Read (for building the classifier)
| File | Content | Priority |
|------|---------|----------|
| `feature_dictionary_enhanced.json` | 229 keywords + TF-IDF scores + value ranges for 68 subcategories + correction rules. **This is the core knowledge base.** | Critical |
| `training_data_complete.json` | 694 historical expenses with full labels — the few-shot source | Critical |
| `algorithm_parameters.json` | Exact weights, thresholds, preprocessing settings | Critical |
| `classification_algorithm.md` | Full algorithm spec with formulas and flowchart | High |
| `reproducibility_guide.md` | Step-by-step implementation instructions | High |
| `confusion_analysis.json` | Ambiguous cases and edge cases — test cases for the classifier | High |

### Reference (already summarized in FINAL_SUMMARY.md)
| File | Content | Priority |
|------|---------|----------|
| `FINAL_SUMMARY.md` | Round 2 results: 261 expenses, 90% HIGH, correction impacts | Read |
| `README.md` | Package overview, all 14 artifacts listed | Read |
| `classification_reasoning.md` | Per-expense reasoning examples | Reference |
| `research_insights.md` | Challenges, discriminative features, transfer learning | Reference |
| `llm_reasoning_meta.md` | Claude's self-analysis of classification process | Reference |

### Data Files (use during build, don't need to read upfront)
| File | Content |
|------|---------|
| `training_data.json` | Round 1: 160 expenses |
| `final_classifications.json` | Round 1: 87 detailed classification results |
| `new_expenses_classified_final.json` | Round 2: 261 detailed results |
| `new_expenses_review_final.json` | 26 cases that need manual review |
| `new_expenses_categorized_final.csv` | Round 2 output CSV |
| `categorized_expenses.csv` | Round 1 output CSV |
| `classification_statistics_final.json` | Round 2 statistics |
| `statistical_summary.json` | Round 1 statistics |
| `similarity_matrix.json` | Pairwise similarity scores (~95KB) |
| `vector_representations.json` | TF-IDF vectors |
| `decision_matrix.csv` | Score matrix |
| `feature_dictionary.json` | Original (pre-correction) feature dict |

### Archive
| File | Content |
|------|---------|
| `auto-categorization-analysis.zip` | Round 1 archive |
| `auto-category.zip` | Archive |
| `auto-category-2.zip` | Archive |

---

## Expense Reporter Tool (`I:\workspaces\expenses\code\expense-reporter\`)

### Key Facts
- **Version:** 2.1.0, production ready, 190+ tests
- **Language:** Go 1.25.5
- **Dependencies:** excelize v2.10.0, cobra v1.10.2, progressbar v3.18.0
- **Commands:** `add`, `batch`, `version`
- **Input format:** `item;DD/MM;value;subcategory` (4 fields, semicolon-separated)
- **Year:** Hardcoded to 2025
- **Workbook:** `Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx`

### Architecture (relevant for Layer 5 integration)
```
cmd/expense-reporter/cmd/
  root.go       — Cobra root, workbook path resolution
  add.go        — Single expense insertion
  batch.go      — CSV batch import with progress bar

internal/
  parser/       — Parse "item;DD/MM;value;subcategory" strings
  resolver/     — Subcategory → sheet/row resolution (hierarchical paths)
  excel/        — Read reference sheet, find rows, write expenses
  models/       — Expense, SheetLocation, Installment, BatchError
  batch/        — CSV reader, processor, progress, report, backup
  workflow/     — InsertExpense(), InsertBatchExpenses()

config/
  config.json   — workbook_path, reference_sheet, date_year
```

### What needs adding for classification
- New `internal/classifier/` package
- New `cmd/.../cmd/classify.go`, `auto.go`, `batch_auto.go`
- New `data/` directory for training data + feature dictionary
- HTTP client for Ollama API (no new Go dependencies — stdlib net/http)

---

## Design Decisions (from existing code)
- Subcategory is always resolved against the Excel reference sheet
- Ambiguous subcategories are detected and reported (not guessed)
- Hierarchical paths disambiguate: `Habitação,Diarista` vs `Casa,Diarista`
- All 4 expense sheets now have uniform column layout (starts at column D)
- Installments auto-expand across months with year rollover handling
