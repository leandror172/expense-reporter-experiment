# Expense Reporter

CLI tool for automating expense entry into Excel budget spreadsheet.

## Status: Production Ready ✅

**Current Version**: 1.0.0 with Batch Import
**Total Tests**: 179 tests (all passing)
**Development Phases**: 9 phases complete

## Features

### ✅ Single Expense Entry
Add individual expenses quickly:
```bash
expense-reporter add "Uber Centro;15/04;35,50;Uber/Taxi"
```

### ✅ Batch Import (CSV)
Import multiple expenses from a CSV file:
```bash
expense-reporter batch expenses.csv --backup --report=import_report.txt
```

### ✅ Smart Subcategory Matching
Automatically handles variations:
- "Orion - Consultas" → "Orion" (parent matching)
- Detects ambiguous subcategories across multiple sheets

### ✅ Professional CLI
- Cobra framework (kubectl/docker pattern)
- Built-in help and command structure
- Global and command-specific flags
- Environment variable support

### ✅ Robust Error Handling
- Clear error messages
- Ambiguous subcategory detection
- Continue-on-error batch processing
- Detailed error reporting

## Installation

### Prerequisites
- Go 1.21+
- Excel workbook with proper structure (see documentation)

### Build
```bash
git clone <repository>
cd expense-reporter
go build -o expense-reporter.exe ./cmd/expense-reporter
```

### Install to PATH
```bash
go install ./cmd/expense-reporter
```

## Quick Start

### 1. Set Workbook Path
```bash
# Option 1: Environment variable
export EXPENSE_WORKBOOK_PATH="/path/to/your/workbook.xlsx"

# Option 2: Global flag
expense-reporter --workbook="/path/to/workbook.xlsx" ...
```

### 2. Add Single Expense
```bash
expense-reporter add "Pão francês;22/12;8,50;Padaria"
# ✓ Expense added successfully!
```

### 3. Import from CSV
```bash
# Create CSV file (expenses.csv):
# Uber Centro;15/04;35,50;Uber/Taxi
# Compras Carrefour;03/01;150,00;Supermercado
# Pão francês;22/12;8,50;Padaria

expense-reporter batch expenses.csv
```

## Input Format

### Expense String Format
```
<item_description>;<DD/MM>;<value_##,##>;<subcategory>
```

**Fields:**
- `item_description`: Free text (no semicolons)
- `DD/MM`: Day and month (year is always 2025)
- `value_##,##`: Value with comma as decimal separator (Brazilian format)
- `subcategory`: Must match subcategory in Excel reference sheet

**Examples:**
```
Uber para o centro;15/04;35,50;Uber/Taxi
Compra Pão de Açúcar;03/01;245,67;Supermercado
Consulta vet;22/03;180,00;Orion - Consultas
```

### CSV File Format
```csv
# Comments start with #
# Empty lines are ignored

# Transportation
Uber Centro;15/04;35,50;Uber/Taxi
Combustível posto;10/04;250,00;Combustível

# Groceries
Compras Carrefour;03/01;150,00;Supermercado
Feira orgânica;05/01;80,00;Feira
```

## Commands

### Add Single Expense
```bash
expense-reporter add "<expense_string>"

# Examples:
expense-reporter add "Uber;15/04;35,50;Uber/Taxi"
expense-reporter add "Pão;22/12;8,50;Padaria"
```

### Batch Import
```bash
expense-reporter batch <csv_file> [flags]

Flags:
  --backup          Create timestamped backup before processing
  --report string   Report output path (default "batch_report.txt")
  --silent          Suppress progress bar output

# Examples:
expense-reporter batch expenses.csv
expense-reporter batch expenses.csv --backup
expense-reporter batch expenses.csv --silent --report=""
```

### Help
```bash
expense-reporter --help              # Show all commands
expense-reporter add --help          # Help for add command
expense-reporter batch --help        # Help for batch command
```

### Version
```bash
expense-reporter version
# Output: expense-reporter version 1.0.0
```

## Batch Import Features

### Progress Bar
Real-time visual feedback during batch processing:
```
Processing 15 expenses...
[████████████████████████████████] 15/15 (100%)
```

### Detailed Report
Generates comprehensive report with:
- Summary statistics (success/error/ambiguous counts)
- List of successful insertions
- Detailed error messages
- Ambiguous entries with sheet options

### Ambiguous Handling
When a subcategory appears in multiple sheets:
1. Skips insertion (doesn't guess)
2. Saves to `ambiguous_expenses.csv` with sheet options
3. User reviews and chooses correct sheet
4. Re-import after editing

### Backup Creation
Optional timestamped backup:
```bash
expense-reporter batch expenses.csv --backup
# Creates: workbook_backup_20250415_143022.xlsx
```

## Project Structure

```
expense-reporter/
├── cmd/
│   └── expense-reporter/
│       ├── main.go                 # CLI entry point (7 lines)
│       ├── main_test.go            # Main tests
│       └── cmd/
│           ├── root.go             # Root command & global flags
│           ├── add.go              # Single expense command
│           ├── batch.go            # Batch import command
│           ├── version.go          # Version command
│           └── *_test.go           # Command tests
├── internal/
│   ├── batch/                      # Batch import package
│   │   ├── models.go               # Batch data structures
│   │   ├── csv_reader.go           # CSV file reading
│   │   ├── processor.go            # Batch processing logic
│   │   ├── progress.go             # Progress bar wrapper
│   │   ├── report.go               # Report generation
│   │   ├── backup.go               # Backup creation
│   │   ├── ambiguous_writer.go     # Ambiguous CSV writer
│   │   └── *_test.go               # 23 test functions
│   ├── cli/                        # CLI interface utilities
│   ├── excel/                      # Excel operations
│   │   ├── columns.go              # Month column mapping
│   │   ├── reader.go               # Reference sheet reader
│   │   ├── writer.go               # Excel writing with formatting
│   │   └── *_test.go               # Excel tests
│   ├── models/                     # Data models
│   │   ├── expense.go              # Expense & SheetLocation
│   │   └── expense_test.go         # Model tests
│   ├── parser/                     # Expense string parser
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── resolver/                   # Subcategory resolver
│   │   ├── resolver.go             # Smart matching logic
│   │   └── resolver_test.go
│   └── workflow/                   # End-to-end workflow
│       ├── workflow.go             # Complete insertion logic
│       └── workflow_test.go
└── pkg/
    └── utils/                      # Utility functions
        ├── date.go                 # Date parsing (DD/MM)
        ├── currency.go             # Currency parsing (##,##)
        └── *_test.go               # Utility tests
```

## Development Phases

### Original Implementation (Phases 1-4)
1. **Phase 1**: Foundation - Parser, models, utils (50 tests)
2. **Phase 2**: Excel Integration - Resolver, reader (85 tests)
3. **Phase 3**: Business Logic - Writer, workflow (95 tests)
4. **Phase 4**: CLI Interface - Cobra migration (131 tests)

### Batch Import Feature (Phases 5-9)
5. **Batch Phase 1**: Cobra Framework - Professional CLI (132 tests)
6. **Batch Phase 2**: CSV + Ambiguous - Reader and handlers (151 tests)
7. **Batch Phase 3**: Processor + Progress - Core batch logic (166 tests)
8. **Batch Phase 4**: Report + Backup - Output and safety (175 tests)
9. **Batch Phase 5**: CLI Integration - Complete feature (179 tests)

## Dependencies

```go
require (
    github.com/xuri/excelize/v2 v2.10.0          // Excel operations
    github.com/spf13/cobra v1.10.2                // CLI framework
    github.com/schollz/progressbar/v3 v3.18.0     // Progress bar
)
```

## Testing

### Run All Tests
```bash
go test ./...
# ok  	expense-reporter/cmd/expense-reporter	0.310s
# ok  	expense-reporter/internal/batch	      1.318s
# ok  	expense-reporter/internal/excel	      0.535s
# ... (all passing)
```

### Run Specific Package
```bash
go test ./internal/batch/... -v      # Batch tests
go test ./internal/parser/... -v     # Parser tests
go test ./cmd/expense-reporter -v    # Main tests
```

### Test Coverage
```bash
go test ./... -cover
# Total: 179 tests across 50 test functions
```

## TDD Methodology

This project strictly follows Test-Driven Development:

1. **RED**: Write failing tests first
2. **GREEN**: Implement minimum code to pass
3. **REFACTOR**: Improve while keeping tests green

**Quality Standards:**
- ✅ No tautological tests (user's hard rule)
- ✅ Tests verify behavior, not field assignments
- ✅ Comprehensive edge case coverage
- ✅ Integration tests with real scenarios
- ✅ Table-driven test patterns

## Documentation

- **README.md** - This file (quick start & reference)
- **PHASE1_COMPLETE.md** - Foundation phase details
- **PHASE2_COMPLETE.md** - Excel integration details
- **PHASE3_COMPLETE.md** - Business logic details
- **PHASE4_COMPLETE.md** - CLI interface details
- **BATCH_IMPORT_COMPLETE.md** - Batch feature complete documentation
- **PROJECT_COMPLETE.md** - Full project summary

## Known Limitations

1. **Year Hardcoded**: Always uses 2025 (per specification)
2. **Brazilian Format**: Uses comma as decimal separator
3. **Sequential Processing**: Not parallel (Excel file locking)
4. **Windows Tested**: Primarily tested on Windows

## Troubleshooting

### "Workbook not found"
```bash
# Check path is correct
export EXPENSE_WORKBOOK_PATH="/full/path/to/workbook.xlsx"

# Or use flag
expense-reporter --workbook="/path/to/workbook.xlsx" add "..."
```

### "Subcategory not found"
Check Excel reference sheet - subcategory must exist exactly as entered.
Use parent matching: "Orion - Consultas" will match "Orion"

### "Ambiguous subcategory"
Subcategory appears in multiple sheets. Use batch import to generate options list,
or check reference sheet to see which sheets contain the subcategory.

### Batch Import Errors
Check `batch_report.txt` for detailed error list with line numbers.

## Contributing

This is a personal project but improvements are welcome:
1. Maintain TDD approach
2. Follow existing code patterns
3. No tautological tests
4. Update documentation

## License

Personal use project.

## Author

Built with Test-Driven Development following strict quality standards.

---

**Status**: ✅ Production Ready
**Tests**: 179 passing
**Quality**: High (TDD, comprehensive coverage)
