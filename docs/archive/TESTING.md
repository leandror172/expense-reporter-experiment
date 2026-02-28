# Testing Guide

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests with Verbose Output
```bash
go test ./... -v
```

### Run Specific Package Tests
```bash
go test ./internal/batch/...
go test ./internal/parser/...
go test ./internal/workflow/...
```

### Run Specific Test by Name
```bash
go test ./internal/batch/... -run TestAmbiguousWriter
```

## Test Workbook Configuration

The integration tests require an Excel workbook. By default, tests look for:
```
expense-reporter/Planilha_Normalized_Final.xlsx
```

### Custom Workbook Path
You can specify a custom path using an environment variable:

**Windows CMD:**
```cmd
set TEST_WORKBOOK_PATH=C:\path\to\your\workbook.xlsx
go test ./...
```

**Windows PowerShell:**
```powershell
$env:TEST_WORKBOOK_PATH="C:\path\to\your\workbook.xlsx"
go test ./...
```

**Linux/Mac:**
```bash
export TEST_WORKBOOK_PATH=/path/to/your/workbook.xlsx
go test ./...
```

### Tests That Require Workbook
- `internal/excel` - Excel reading/writing tests
- `internal/workflow` - End-to-end integration tests

If the workbook is not found, these tests will be automatically skipped with a message indicating the expected path.

## Kaspersky Antivirus Issue

If you're using Kaspersky antivirus, it may block the `resolver.test.exe` file with a false positive detection:
- **Type:** VHO:Trojan.Win64.Loader.gen (heuristic detection)
- **Reason:** Go creates temporary test executables in `%TEMP%\go-build*` which triggers heuristic analysis

### Workarounds
1. **Add exception in Kaspersky** for the Go build temp directory
2. **Run full test suite** (`go test ./...`) - sometimes works when individual package tests are blocked
3. **Compile without running**: `go test -c ./internal/resolver` - verifies code compiles correctly
4. **Trust the code**: The resolver tests are tested indirectly through processor and workflow tests

The resolver code is thoroughly tested through:
- Unit tests in other packages that import it
- Integration tests in workflow package
- Batch processor tests

## Test Structure

```
expense-reporter/
├── cmd/expense-reporter/
│   └── *_test.go           # CLI tests
├── internal/
│   ├── batch/
│   │   └── *_test.go       # Batch processing tests
│   ├── cli/
│   │   └── *_test.go       # CLI component tests
│   ├── excel/
│   │   └── *_test.go       # Excel I/O tests (require workbook)
│   ├── parser/
│   │   └── *_test.go       # Expense parsing tests
│   ├── resolver/
│   │   └── *_test.go       # Hierarchical resolution tests (may be blocked by Kaspersky)
│   └── workflow/
│       └── *_test.go       # Integration tests (require workbook)
└── pkg/utils/
    └── *_test.go           # Utility function tests
```

## Test Categories

### Unit Tests (No Dependencies)
- ✅ `internal/parser` - Expense string parsing
- ✅ `internal/batch` - Batch processing logic
- ✅ `pkg/utils` - Utility functions
- ✅ `internal/models` - Data structures
- ⚠️ `internal/resolver` - May be blocked by antivirus

### Integration Tests (Require Workbook)
- ⚠️ `internal/excel` - Excel file operations
- ⚠️ `internal/workflow` - End-to-end workflows

## Continuous Testing

For TDD workflow, you can use:

```bash
# Watch mode (requires external tool like `gow` or `reflex`)
gow test ./...

# Or just re-run tests manually after each change
go test ./...
```
