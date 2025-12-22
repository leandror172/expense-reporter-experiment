# Expense Reporter

CLI tool for automating expense entry into Excel budget spreadsheet.

## Status: Phase 1 Complete ✓

**TDD Approach:**
- ✅ Tests written first (RED)
- ✅ Implementation to pass tests (GREEN)
- ✅ All tests passing

**Completed Components:**
- ✅ Date parsing (DD/MM → time.Time)
- ✅ Currency parsing (##,## → float64)
- ✅ Expense string parsing (semicolon-delimited)
- ✅ Data models (Expense, SheetLocation)
- ✅ Input validation
- ✅ Error handling

**Test Coverage:**
- `pkg/utils/date_test.go`: 16 test cases
- `pkg/utils/currency_test.go`: 14 test cases
- `internal/parser/parser_test.go`: 13 test cases
- `internal/models/expense_test.go`: 7 test cases
- **Total:** 50 test cases, all passing

## Project Structure

```
expense-reporter/
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── config/
│   └── config.json             # Configuration file
├── internal/
│   ├── models/
│   │   ├── expense.go          # Expense & SheetLocation structs
│   │   └── expense_test.go     # Model tests
│   ├── parser/
│   │   ├── parser.go           # Expense string parser
│   │   └── parser_test.go      # Parser tests
│   ├── resolver/               # (Phase 2)
│   └── excel/                  # (Phase 3)
└── pkg/
    └── utils/
        ├── date.go             # Date parsing utilities
        ├── date_test.go        # Date tests
        ├── currency.go         # Currency parsing utilities
        └── currency_test.go    # Currency tests
```

## Usage (After Full Implementation)

### Single Expense
```bash
expense-reporter "Uber Centro;15/04;35,50;Uber/Taxi"
```

### Interactive Mode
```bash
expense-reporter --interactive
```

### Batch Mode
```bash
expense-reporter --batch expenses.txt
```

## Input Format

```
<item_description>;<DD/MM>;<value_as_##,##>;<sub_category>
```

**Fields:**
- `item_description`: Free text (no semicolons)
- `DD/MM`: Day and month (year is 2025)
- `value_as_##,##`: Decimal value with comma separator
- `sub_category`: Must match subcategory in reference sheet

**Examples:**
```
Uber para o centro;15/04;35,50;Uber/Taxi
Compra Pão de Açúcar;03/01;245,67;Supermercado
Consulta vet Orion;22/03;180,00;Orion
```

## Development

### Run Tests
```bash
go test ./... -v
```

### Run Specific Package Tests
```bash
go test ./pkg/utils/... -v
go test ./internal/parser/... -v
go test ./internal/models/... -v
```

### Build
```bash
go build -o expense-reporter.exe
```

## Next Phases

- **Phase 2:** Excel integration (resolver, column calculator)
- **Phase 3:** Business logic (row finder, data insertion)
- **Phase 4:** CLI interface
- **Phase 5:** Integration testing

## Dependencies

- Go 1.21+
- github.com/xuri/excelize/v2 - Excel file manipulation

## TDD Methodology

This project follows Test-Driven Development:
1. **RED:** Write failing tests first
2. **GREEN:** Write minimal code to pass tests
3. **REFACTOR:** Improve code quality

All code is test-driven with comprehensive edge case coverage.
