# ARCHITECTURAL PLAN: Expense Automation with Go
## Claude Code Architecture Document - Go Implementation

---

## EXECUTIVE SUMMARY

**Goal:** Automate expense entry using Go CLI tool with input format:
`<item_description>;<DD/MM>;<value_as_##,##>;<sub_category>`

**Approach:** Standalone Go executable (no Excel button dependency)

**Complexity Level:** Medium (Go has excellent Excel libraries)

**Estimated Total Cost:** 35,000-45,000 tokens (lower than Python due to Go's simplicity)

**Why Go is Better for This:**
- `excelize` library: Production-grade Excel manipulation (better than openpyxl)
- Single compiled binary: No runtime dependencies
- Fast execution: ~10-50ms vs Python's ~500ms
- Type safety: Catch errors at compile time
- Concurrent operations: Can batch multiple expenses easily
- Cross-platform: Single codebase for Windows/Mac/Linux

---

## 1. GO ECOSYSTEM ANALYSIS

### Primary Library: excelize

**GitHub:** https://github.com/qax-os/excelize (14k+ stars)

**Capabilities:**
- Read/write Excel 2007+ (.xlsx) files
- Cell manipulation with proper type handling
- Formula preservation
- Date/time handling with Excel serial numbers
- Streaming mode for large files
- No Excel installation required

**Installation:**
```bash
go get github.com/qax-os/excelize/v2
```

**Advantages over openpyxl:**
- 10x faster for large files
- Better memory efficiency
- Native datetime handling
- Simpler API for cell operations
- Built-in formula evaluation

### Alternative Libraries (Not Recommended)

1. **tealeg/xlsx** - Older, less maintained
2. **360EntSecGroup-Skylar/excelize** - Original, now deprecated
3. **go-xlsx** - Limited features

---

## 2. SOLUTION ARCHITECTURE (GO-BASED)

### Approach: Standalone CLI Tool

**Why Not VBA Integration:**
- VBA calling Go executables works, but adds complexity
- User can run Go CLI directly (simpler workflow)
- Can still add VBA wrapper later if needed

**Workflow:**

```
User Input (3 options):
├─ Option 1: Interactive Mode
│  └─ Run: expense-report.exe
│  └─ Prompts for each field
│
├─ Option 2: Single Command
│  └─ Run: expense-report.exe "Uber Centro;15/04;35,50;Uber/Taxi"
│  └─ One-line entry
│
└─ Option 3: Batch Mode
   └─ Run: expense-report.exe --batch expenses.txt
   └─ Multiple entries from file
```

**Architecture Layers:**

```
┌─────────────────────────────────────────────┐
│         CLI Interface (cobra/flag)          │
│  - Parse arguments                          │
│  - Interactive prompts                      │
│  - Batch file reading                       │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│         Input Validator                     │
│  - Parse semicolon-delimited string         │
│  - Validate date format (DD/MM)             │
│  - Validate value format (##,##)            │
│  - Sanitize item description                │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│      Subcategory Resolver                   │
│  - Load "Referência de Categorias" sheet    │
│  - Match subcategory → (sheet, category)    │
│  - Handle ambiguous cases                   │
│  - Cache mappings for performance           │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│      Column Calculator                      │
│  - Parse month from DD/MM                   │
│  - Get sheet-specific column mapping        │
│  - Calculate Item/Date/Value column indices │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│      Row Finder                             │
│  - Open expense sheet (Fixas/Variáveis/etc)│
│  - Locate subcategory row                   │
│  - Scan for next empty cell in month column │
│  - Validate no overflow into next subcategory│
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│      Excel Writer                           │
│  - Open workbook with excelize              │
│  - Write Item (string)                      │
│  - Write Date (Excel datetime)              │
│  - Write Value (float64)                    │
│  - Save workbook                            │
│  - Generate success report                  │
└─────────────────────────────────────────────┘
```

---

## 3. PROJECT STRUCTURE

```
controle/
├── code/
│   ├── expense-reporter/              # Go project root
│   │   ├── main.go                    # Entry point & CLI
│   │   ├── go.mod                     # Dependencies
│   │   ├── go.sum                     # Dependency checksums
│   │   │
│   │   ├── internal/
│   │   │   ├── parser/
│   │   │   │   └── parser.go          # Input parsing & validation
│   │   │   │
│   │   │   ├── resolver/
│   │   │   │   ├── resolver.go        # Subcategory resolution
│   │   │   │   └── cache.go           # In-memory mapping cache
│   │   │   │
│   │   │   ├── excel/
│   │   │   │   ├── reader.go          # Excel reading operations
│   │   │   │   ├── writer.go          # Excel writing operations
│   │   │   │   └── columns.go         # Column calculation logic
│   │   │   │
│   │   │   └── models/
│   │   │       └── expense.go         # Data structures
│   │   │
│   │   ├── pkg/
│   │   │   └── utils/
│   │   │       ├── date.go            # Date conversion utilities
│   │   │       └── currency.go        # Currency parsing (##,##)
│   │   │
│   │   └── config/
│   │       └── config.json            # Configuration (file path, etc)
│   │
│   ├── dist/
│   │   └── expense-reporter.exe       # Compiled binary (Windows)
│   │
│   ├── Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx
│   └── [other existing files...]
```

---

## 4. DETAILED IMPLEMENTATION PLAN

### Phase 1: Project Setup & Scaffolding (Tokens: ~4,000)

**Task 1.1:** Initialize Go module
```bash
cd "Z:\Meu Drive\controle\code"
mkdir expense-reporter
cd expense-reporter
go mod init expense-reporter
go get github.com/qax-os/excelize/v2
```
- **Tokens:** ~800

**Task 1.2:** Create project structure
- Create all directories and initial files
- Set up `go.mod` with dependencies
- **Tokens:** ~1,000

**Task 1.3:** Define data models
```go
type Expense struct {
    Item        string
    Date        time.Time
    Value       float64
    Subcategory string
}

type SheetLocation struct {
    SheetName   string
    Category    string
    SubcatRow   int
    TargetRow   int
    MonthColumn string  // e.g., "M" for Abril in Variáveis
}
```
- **Tokens:** ~1,200

**Task 1.4:** Create configuration file
```json
{
  "workbook_path": "../Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx",
  "reference_sheet": "Referência de Categorias",
  "date_year": 2025,
  "verbose": false
}
```
- **Tokens:** ~1,000

---

### Phase 2: Input Parsing & Validation (Tokens: ~6,000)

**Task 2.1:** Implement semicolon parser
```go
func ParseExpenseString(input string) (*Expense, error) {
    parts := strings.Split(input, ";")
    if len(parts) != 4 {
        return nil, errors.New("expected format: item;DD/MM;##,##;subcategory")
    }

    item := strings.TrimSpace(parts[0])
    date := strings.TrimSpace(parts[1])
    value := strings.TrimSpace(parts[2])
    subcat := strings.TrimSpace(parts[3])

    // Validation...
    return &Expense{...}, nil
}
```
- **Tokens:** ~2,500

**Task 2.2:** Date parsing with validation
```go
func ParseDate(dateStr string, year int) (time.Time, error) {
    // Parse "DD/MM" format
    parts := strings.Split(dateStr, "/")
    if len(parts) != 2 {
        return time.Time{}, errors.New("invalid date format, use DD/MM")
    }

    day, _ := strconv.Atoi(parts[0])
    month, _ := strconv.Atoi(parts[1])

    // Validate ranges
    if day < 1 || day > 31 || month < 1 || month > 12 {
        return time.Time{}, errors.New("invalid day or month")
    }

    return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
```
- **Tokens:** ~2,000

**Task 2.3:** Currency parsing (##,## → float64)
```go
func ParseCurrency(valueStr string) (float64, error) {
    // Replace comma with period: "123,45" → "123.45"
    normalized := strings.ReplaceAll(valueStr, ",", ".")

    value, err := strconv.ParseFloat(normalized, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid value format: %w", err)
    }

    if value < 0 {
        return 0, errors.New("negative values not allowed")
    }

    return value, nil
}
```
- **Tokens:** ~1,500

---

### Phase 3: Excel Integration (Tokens: ~10,000)

**Task 3.1:** Load and cache reference sheet
```go
type SubcategoryMapping struct {
    Subcategory string
    SheetName   string
    Category    string
    RowNumber   int  // Cached row number in target sheet
}

func LoadReferenceSheet(f *excelize.File) (map[string][]SubcategoryMapping, error) {
    cache := make(map[string][]SubcategoryMapping)

    rows, err := f.GetRows("Referência de Categorias")
    if err != nil {
        return nil, err
    }

    for i, row := range rows {
        if i < 4 { continue } // Skip header rows

        if len(row) >= 3 {
            mainType := row[0]  // Fixas, Variáveis, etc.
            category := row[1]
            subcat := row[2]

            // Build lookup map
            cache[subcat] = append(cache[subcat], SubcategoryMapping{
                Subcategory: subcat,
                SheetName:   mainType,
                Category:    category,
            })
        }
    }

    return cache, nil
}
```
- **Tokens:** ~3,500

**Task 3.2:** Column calculation logic
```go
var monthColumns = map[string]map[int]string{
    "Fixas":      {1: "C", 2: "F", 3: "I", 4: "L", 5: "O", 6: "R", 7: "U", 8: "X", 9: "AA", 10: "AD", 11: "AG", 12: "AJ"},
    "Variáveis":  {1: "D", 2: "G", 3: "J", 4: "M", 5: "P", 6: "S", 7: "V", 8: "Y", 9: "AB", 10: "AE", 11: "AH", 12: "AK"},
    "Extras":     {1: "C", 2: "F", 3: "I", 4: "L", 5: "O", 6: "R", 7: "U", 8: "X", 9: "AA", 10: "AD", 11: "AG", 12: "AJ"},
    "Adicionais": {1: "C", 2: "F", 3: "I", 4: "L", 5: "O", 6: "R", 7: "U", 8: "X", 9: "AA", 10: "AD", 11: "AG", 12: "AJ"},
}

func GetMonthColumns(sheetName string, month int) (item, date, value string, err error) {
    colMap, ok := monthColumns[sheetName]
    if !ok {
        return "", "", "", fmt.Errorf("unknown sheet: %s", sheetName)
    }

    itemCol, ok := colMap[month]
    if !ok {
        return "", "", "", fmt.Errorf("invalid month: %d", month)
    }

    // Calculate adjacent columns
    dateCol := incrementColumn(itemCol, 1)
    valueCol := incrementColumn(itemCol, 2)

    return itemCol, dateCol, valueCol, nil
}

func incrementColumn(col string, n int) string {
    // "M" + 1 = "N", "Z" + 1 = "AA", etc.
    // (excelize has utility for this: excelize.ColumnNumberToName)
    colNum, _ := excelize.ColumnNameToNumber(col)
    return excelize.ColumnNumberToName(colNum + n)
}
```
- **Tokens:** ~3,000

**Task 3.3:** Row finder with subcategory scanning
```go
func FindTargetRow(f *excelize.File, sheetName, subcategory, itemColumn string, startRow int) (int, error) {
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return 0, err
    }

    // Find subcategory row in column B
    subcatRow := -1
    for i, row := range rows {
        if len(row) > 1 && strings.TrimSpace(row[1]) == subcategory {
            subcatRow = i + 1  // Excel is 1-indexed
            break
        }
    }

    if subcatRow == -1 {
        return 0, fmt.Errorf("subcategory not found: %s", subcategory)
    }

    // Scan down from subcategory row to find next empty cell in itemColumn
    for row := subcatRow; row < len(rows)+1; row++ {
        cellRef := fmt.Sprintf("%s%d", itemColumn, row)
        cellValue, _ := f.GetCellValue(sheetName, cellRef)

        if cellValue == "" {
            // Check if we've crossed into next subcategory
            nextSubcat, _ := f.GetCellValue(sheetName, fmt.Sprintf("B%d", row))
            if nextSubcat != "" && nextSubcat != subcategory {
                return 0, errors.New("no empty cells available for this subcategory")
            }

            return row, nil
        }
    }

    return 0, errors.New("no available rows found")
}
```
- **Tokens:** ~3,500

---

### Phase 4: Core Business Logic (Tokens: ~8,000)

**Task 4.1:** Main expense insertion function
```go
func InsertExpense(expense *Expense, workbookPath string) error {
    // 1. Open workbook
    f, err := excelize.OpenFile(workbookPath)
    if err != nil {
        return fmt.Errorf("failed to open workbook: %w", err)
    }
    defer f.Close()

    // 2. Load reference mappings
    mappings, err := LoadReferenceSheet(f)
    if err != nil {
        return fmt.Errorf("failed to load reference sheet: %w", err)
    }

    // 3. Resolve subcategory
    location, err := ResolveSubcategory(mappings, expense.Subcategory)
    if err != nil {
        return err
    }

    // 4. Get month columns
    month := int(expense.Date.Month())
    itemCol, dateCol, valueCol, err := GetMonthColumns(location.SheetName, month)
    if err != nil {
        return err
    }

    // 5. Find target row
    targetRow, err := FindTargetRow(f, location.SheetName, expense.Subcategory, itemCol, location.SubcatRow)
    if err != nil {
        return err
    }

    // 6. Write data
    if err := f.SetCellValue(location.SheetName, fmt.Sprintf("%s%d", itemCol, targetRow), expense.Item); err != nil {
        return err
    }

    // Excel date serial number
    excelDate := TimeToExcelDate(expense.Date)
    if err := f.SetCellValue(location.SheetName, fmt.Sprintf("%s%d", dateCol, targetRow), excelDate); err != nil {
        return err
    }

    if err := f.SetCellValue(location.SheetName, fmt.Sprintf("%s%d", valueCol, targetRow), expense.Value); err != nil {
        return err
    }

    // 7. Save
    if err := f.Save(); err != nil {
        return fmt.Errorf("failed to save workbook: %w", err)
    }

    fmt.Printf("✓ Expense inserted: %s | %s | R$ %.2f | %s (row %d)\n",
        expense.Item, expense.Date.Format("02/01"), expense.Value, location.SheetName, targetRow)

    return nil
}
```
- **Tokens:** ~4,000

**Task 4.2:** Ambiguity resolution
```go
func ResolveSubcategory(mappings map[string][]SubcategoryMapping, subcategory string) (*SheetLocation, error) {
    matches := mappings[subcategory]

    if len(matches) == 0 {
        return nil, fmt.Errorf("subcategory not found: %s", subcategory)
    }

    if len(matches) == 1 {
        return &SheetLocation{
            SheetName: matches[0].SheetName,
            Category:  matches[0].Category,
        }, nil
    }

    // Multiple matches - prompt user
    fmt.Printf("\n⚠ Found %d matches for '%s':\n", len(matches), subcategory)
    for i, match := range matches {
        fmt.Printf("  [%d] %s > %s\n", i+1, match.SheetName, match.Category)
    }

    fmt.Print("\nSelect option (1-N): ")
    var choice int
    fmt.Scanf("%d", &choice)

    if choice < 1 || choice > len(matches) {
        return nil, errors.New("invalid choice")
    }

    selected := matches[choice-1]
    return &SheetLocation{
        SheetName: selected.SheetName,
        Category:  selected.Category,
    }, nil
}
```
- **Tokens:** ~2,500

**Task 4.3:** Excel datetime conversion
```go
func TimeToExcelDate(t time.Time) float64 {
    // Excel epoch: December 30, 1899
    epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
    duration := t.Sub(epoch)
    days := duration.Hours() / 24
    return days
}
```
- **Tokens:** ~500

**Task 4.4:** Cache row numbers for performance
```go
// Cache subcategory row numbers after first lookup
type RowCache struct {
    cache map[string]int  // "SheetName:Subcategory" → row number
    mu    sync.RWMutex
}

func (rc *RowCache) Get(sheetName, subcategory string) (int, bool) {
    rc.mu.RLock()
    defer rc.mu.RUnlock()

    key := fmt.Sprintf("%s:%s", sheetName, subcategory)
    row, ok := rc.cache[key]
    return row, ok
}

func (rc *RowCache) Set(sheetName, subcategory string, row int) {
    rc.mu.Lock()
    defer rc.mu.Unlock()

    key := fmt.Sprintf("%s:%s", sheetName, subcategory)
    rc.cache[key] = row
}
```
- **Tokens:** ~1,000

---

### Phase 5: CLI Interface (Tokens: ~5,000)

**Task 5.1:** Main CLI with flags
```go
func main() {
    var (
        input      string
        batchFile  string
        interactive bool
        configPath string
    )

    flag.StringVar(&input, "input", "", "Expense string: 'item;DD/MM;##,##;subcategory'")
    flag.StringVar(&batchFile, "batch", "", "Batch file with one expense per line")
    flag.BoolVar(&interactive, "interactive", false, "Interactive mode with prompts")
    flag.StringVar(&configPath, "config", "config/config.json", "Config file path")
    flag.Parse()

    // Load config
    config, err := LoadConfig(configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Mode selection
    if interactive {
        runInteractive(config)
    } else if batchFile != "" {
        runBatch(batchFile, config)
    } else if input != "" {
        runSingle(input, config)
    } else {
        // Default to positional argument
        if len(os.Args) > 1 {
            runSingle(os.Args[1], config)
        } else {
            runInteractive(config)
        }
    }
}
```
- **Tokens:** ~2,500

**Task 5.2:** Interactive mode
```go
func runInteractive(config *Config) {
    reader := bufio.NewReader(os.Stdin)

    fmt.Println("=== Expense Reporter - Interactive Mode ===")
    fmt.Println()

    for {
        fmt.Print("Item description: ")
        item, _ := reader.ReadString('\n')
        item = strings.TrimSpace(item)

        if item == "" || item == "exit" {
            break
        }

        fmt.Print("Date (DD/MM): ")
        date, _ := reader.ReadString('\n')
        date = strings.TrimSpace(date)

        fmt.Print("Value (##,##): ")
        value, _ := reader.ReadString('\n')
        value = strings.TrimSpace(value)

        fmt.Print("Subcategory: ")
        subcat, _ := reader.ReadString('\n')
        subcat = strings.TrimSpace(subcat)

        // Build expense string
        expenseStr := fmt.Sprintf("%s;%s;%s;%s", item, date, value, subcat)

        // Process
        if err := processExpense(expenseStr, config); err != nil {
            fmt.Printf("✗ Error: %v\n\n", err)
        } else {
            fmt.Println()
        }

        fmt.Print("Add another? (y/n): ")
        cont, _ := reader.ReadString('\n')
        if strings.ToLower(strings.TrimSpace(cont)) != "y" {
            break
        }
        fmt.Println()
    }
}
```
- **Tokens:** ~1,500

**Task 5.3:** Batch mode
```go
func runBatch(filename string, config *Config) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatalf("Failed to open batch file: %v", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0
    successes := 0
    failures := 0

    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        if line == "" || strings.HasPrefix(line, "#") {
            continue  // Skip empty lines and comments
        }

        fmt.Printf("[%d] Processing: %s\n", lineNum, line)

        if err := processExpense(line, config); err != nil {
            fmt.Printf("    ✗ Error: %v\n", err)
            failures++
        } else {
            successes++
        }
    }

    fmt.Printf("\n=== Batch Complete ===\n")
    fmt.Printf("Successes: %d\n", successes)
    fmt.Printf("Failures:  %d\n", failures)
}
```
- **Tokens:** ~1,000

---

### Phase 6: Testing (Tokens: ~7,000)

**Task 6.1:** Unit tests for parser
```go
func TestParseExpenseString(t *testing.T) {
    tests := []struct {
        input   string
        wantErr bool
    }{
        {"Uber Centro;15/04;35,50;Uber/Taxi", false},
        {"Item;32/01;10,00;Cat", true},  // Invalid day
        {"Item;15/13;10,00;Cat", true},  // Invalid month
        {"Item;15/04;-10,00;Cat", true}, // Negative value
        {"Item;15/04;10,00", true},      // Missing field
    }

    for _, tt := range tests {
        _, err := ParseExpenseString(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("ParseExpenseString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
        }
    }
}
```
- **Tokens:** ~2,000

**Task 6.2:** Integration tests with test workbook
```go
func TestInsertExpense_Integration(t *testing.T) {
    // Create a copy of the workbook for testing
    testWorkbook := "test_workbook.xlsx"
    copyFile("../Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx", testWorkbook)
    defer os.Remove(testWorkbook)

    expense := &Expense{
        Item:        "Test Expense",
        Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
        Value:       35.50,
        Subcategory: "Uber/Taxi",
    }

    err := InsertExpense(expense, testWorkbook)
    if err != nil {
        t.Fatalf("InsertExpense failed: %v", err)
    }

    // Verify insertion
    f, _ := excelize.OpenFile(testWorkbook)
    defer f.Close()

    // Check that the value was written
    // (Need to know expected row/column from test data)
    // This is a simplified check
    rows, _ := f.GetRows("Variáveis")
    found := false
    for _, row := range rows {
        for _, cell := range row {
            if cell == "Test Expense" {
                found = true
                break
            }
        }
    }

    if !found {
        t.Error("Expense not found in workbook after insertion")
    }
}
```
- **Tokens:** ~3,000

**Task 6.3:** Manual testing with real data
- Test all 4 sheets
- Test all 12 months
- Test ambiguous subcategories
- Test edge cases
- **Tokens:** ~2,000

---

### Phase 7: Documentation & Build (Tokens: ~3,000)

**Task 7.1:** Create README.md
```markdown
# Expense Reporter

CLI tool for automating expense entry into budget spreadsheet.

## Installation

1. Install Go 1.21+
2. Clone/download this directory
3. Run: `go build -o dist/expense-reporter.exe main.go`

## Usage

### Single Expense
```bash
expense-reporter.exe "Uber Centro;15/04;35,50;Uber/Taxi"
```

### Interactive Mode
```bash
expense-reporter.exe --interactive
```

### Batch Mode
```bash
expense-reporter.exe --batch expenses.txt
```

## Input Format

`<item_description>;<DD/MM>;<value_as_##,##>;<sub_category>`

- **item_description**: Text description (no semicolons)
- **DD/MM**: Day and month (year is 2025)
- **value_as_##,##**: Decimal value with comma separator
- **sub_category**: Must match exactly from reference sheet

## Examples

```
Uber para o centro;15/04;35,50;Uber/Taxi
Compra Pão de Açúcar;03/01;245,67;Supermercado
Consulta vet Orion;22/03;180,00;Orion - Consultas
```
```
- **Tokens:** ~1,500

**Task 7.2:** Build for Windows
```bash
# Windows executable
GOOS=windows GOARCH=amd64 go build -o dist/expense-reporter.exe main.go

# Optional: Linux/Mac builds
GOOS=linux GOARCH=amd64 go build -o dist/expense-reporter-linux main.go
GOOS=darwin GOARCH=amd64 go build -o dist/expense-reporter-mac main.go
```
- **Tokens:** ~500

**Task 7.3:** User guide & troubleshooting
- Common errors and solutions
- Subcategory reference guide
- **Tokens:** ~1,000

---

## 5. COST BREAKDOWN (GO IMPLEMENTATION)

### Token Usage by Phase

| Phase | Description | Estimated Tokens | Cumulative |
|-------|-------------|------------------|------------|
| 1 | Project Setup & Scaffolding | 4,000 | 4,000 |
| 2 | Input Parsing & Validation | 6,000 | 10,000 |
| 3 | Excel Integration | 10,000 | 20,000 |
| 4 | Core Business Logic | 8,000 | 28,000 |
| 5 | CLI Interface | 5,000 | 33,000 |
| 6 | Testing | 7,000 | 40,000 |
| 7 | Documentation & Build | 3,000 | 43,000 |
| **BUFFER (15%)** | Go is more predictable | 6,450 | **49,450** |

**Total: ~49,500 tokens** (vs. 56,000 for Python)

**Why Lower Cost:**
- Go's type system catches errors at compile time (less debugging)
- excelize library is more straightforward than openpyxl
- No VBA integration complexity
- Better error messages = less trial-and-error

---

## 6. SESSION BREAKDOWN (CLAUDE.AI PRO)

### Session 1: COMPLETED ✓
- Analysis & Go Planning
- Tokens: ~7,000
- This document

### Session 2: Setup + Parsing + Excel (Part 1)
- Phases 1, 2, and partial 3
- Tokens: ~15,000
- Duration: 2-3 hours
- **No wait needed**

### Session 3: Excel + Business Logic
- Complete Phase 3 and Phase 4
- Tokens: ~13,000
- Duration: 2-3 hours
- **Possible wait: 2 hours** (if approaching limit)

### Session 4: CLI + Testing
- Phases 5 and 6
- Tokens: ~12,000
- Duration: 2-3 hours
- **Recommended wait: 5 hours or next day**

### Session 5: Documentation + Polish
- Phase 7 + final testing
- Tokens: ~3,000
- Duration: 1 hour
- **No wait needed**

**Timeline:**
- Best case: 1-2 days (8-10 hours active)
- Realistic: 2-3 days with managed breaks
- Worst case: 3-4 days (if hitting limits)

---

## 7. WORKFLOW COMPARISON

### Option A: Pure CLI (RECOMMENDED)

**User Workflow:**
1. Open terminal/cmd
2. Navigate to folder: `cd "Z:\Meu Drive\controle\code"`
3. Run: `expense-reporter.exe "Uber;15/04;35,50;Uber/Taxi"`
4. See confirmation message
5. Open Excel to verify (optional)

**Pros:**
- Simple, no Excel dependencies
- Can run from anywhere
- Can batch multiple expenses
- Fast (50ms execution)

**Cons:**
- Requires terminal access
- Not "in Excel"

### Option B: CLI + VBA Wrapper (Optional Add-on)

**User Workflow:**
1. Open Excel
2. Click "Reportar Gasto" button
3. Enter expense string in dialog
4. VBA calls CLI: `Shell("expense-reporter.exe '" & input & "'")`
5. Excel refreshes
6. See confirmation

**Pros:**
- Feels integrated
- No terminal needed
- Button click convenience

**Cons:**
- +5,000 tokens for VBA wrapper
- More complex error handling
- Excel must be closed during write

**Recommendation:** Start with Option A, add Option B later if needed

### Option C: Background Watcher (Advanced)

**User Workflow:**
1. Run: `expense-reporter.exe --watch`
2. Create file: `new_expense.txt` with expense string
3. Watcher detects file, processes, deletes file
4. Notification shown

**Pros:**
- Most automated
- Can integrate with other tools
- No manual CLI invocation

**Cons:**
- +8,000 tokens
- Requires background process
- Overkill for this use case

---

## 8. AMBIGUITY RESOLUTION STRATEGY (GO-SPECIFIC)

### Problem: "Gás" in Multiple Categories

**Go Advantage:** Can use interactive prompts efficiently

**Strategy:**

```go
// Option 1: Interactive Selection (when ambiguous)
⚠ Found 2 matches for 'Gás':
  [1] Variáveis > Alimentação / Limpeza
  [2] Variáveis > Habitação

Select option (1-2): _
```

**Option 2: Use qualified names in input**
```
Input format with category prefix:
"Botijão;15/04;120,00;Alimentação / Limpeza|Gás"
                        ^^^^^^^^^^^^^^^^^^^^^^^^^
                        Category|Subcategory
```

**Option 3: Smart keyword detection**
```go
var contextKeywords = map[string]map[string][]string{
    "Gás": {
        "Alimentação / Limpeza": []string{"botijão", "fogão", "cozinha"},
        "Habitação": []string{"conta", "copagaz", "compagas", "ultragaz"},
    },
}

// Check if item contains keywords
func detectCategory(item, subcategory string) string {
    keywords := contextKeywords[subcategory]
    itemLower := strings.ToLower(item)

    for category, words := range keywords {
        for _, word := range words {
            if strings.Contains(itemLower, word) {
                return category
            }
        }
    }

    return ""  // Couldn't determine, fallback to prompt
}
```

**Recommendation:** Start with Option 1 (interactive), add Option 3 (keywords) later

---

## 9. ADVANTAGES OF GO OVER PYTHON

| Aspect | Python | Go | Winner |
|--------|--------|-----|--------|
| **Performance** | ~500ms per insert | ~50ms per insert | ✓ Go (10x) |
| **Dependencies** | Requires Python runtime + pip | Single binary, no runtime | ✓ Go |
| **Distribution** | Install Python, openpyxl, etc. | Copy .exe file | ✓ Go |
| **Type Safety** | Runtime errors | Compile-time errors | ✓ Go |
| **Excel Library** | openpyxl (good) | excelize (excellent) | ✓ Go |
| **Concurrency** | GIL limits | Native goroutines | ✓ Go |
| **Memory** | ~50-100MB | ~10-20MB | ✓ Go |
| **Error Messages** | Stack traces | Structured errors | ✓ Go |
| **File Size** | N/A (interpreter) | ~8MB executable | ✓ Go |
| **Learning Curve** | Lower (if you know Python) | Higher | ✓ Python |
| **Your Experience** | N/A | 16 years Java → Go is natural | ✓ Go |

**Verdict:** Go is the superior choice for this project

---

## 10. NEXT STEPS & APPROVAL

### Pre-Implementation Checklist

- [ ] Go installed (1.21+)
- [ ] Confirm Go module path preference
- [ ] Decide on ambiguity resolution (Option 1/2/3)
- [ ] Confirm date year strategy (always 2025)
- [ ] Decide if VBA wrapper is needed (now or later)
- [ ] Review project structure

### Decision Points

**1. Ambiguity Resolution:**
- [ ] Option 1: Interactive prompt (recommended)
- [ ] Option 2: Qualified names (Category|Subcategory)
- [ ] Option 3: Keyword detection
- [ ] Combination: Keywords first, fallback to prompt

**2. Date Year Handling:**
- [ ] Always use 2025
- [ ] Smart detection (if month < current, use next year)
- [ ] Add optional year to format: `DD/MM/YYYY`

**3. Deployment:**
- [ ] Pure CLI (recommended for start)
- [ ] CLI + VBA wrapper (add later)
- [ ] Background watcher (advanced, later)

**4. Project Location:**
- Current: `Z:\Meu Drive\controle\code\expense-reporter\`
- Alternative: `_______________________`

### Ready to Start?

Reply with:
1. Your choices for the 4 decision points
2. Confirmation of Go installation (or need help installing)
3. Any modifications to the project structure
4. Preferred timeline (aggressive 1-2 days vs. relaxed 3-4 days)

---

**Document Version:** 2.0 (Go Implementation)
**Date:** 2025-12-19
**Author:** Claude Code Architect (Sonnet 4.5)
**Review Status:** Pending User Approval
