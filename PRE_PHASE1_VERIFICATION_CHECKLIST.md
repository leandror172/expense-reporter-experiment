# Pre-Phase 1 Verification Checklist

## Before Starting Go Implementation

Run through this checklist to ensure everything is ready for smooth development.

---

## 1. Go Installation & Environment

### Check Go is Installed
```bash
go version
```
**Expected output:** `go version go1.21.x` (or higher)

**If not found:**
- Download from: https://golang.org/dl/
- Install Go 1.21 or higher
- Windows: Use installer, will auto-configure PATH
- Verify after install with `go version`

---

### Check Go Environment
```bash
go env GOPATH
go env GOROOT
```
**Expected:**
- GOPATH: Should show a valid directory (e.g., `C:\Users\YourName\go`)
- GOROOT: Should show Go installation path

---

### Check Go Modules Support
```bash
go env GO111MODULE
```
**Expected:** `on` or empty (defaults to on in Go 1.16+)

---

## 2. Directory Permissions & Access

### Check Write Access to Project Directory
```bash
cd "Z:\Meu Drive\controle\code"
mkdir test-dir
rmdir test-dir
```
**Expected:** No errors, directory created and removed successfully

---

### Check Excel File Access
```bash
ls -la "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx"
```
**Expected:**
- File exists
- Readable and writable permissions
- Not locked by Excel (close Excel if open)

---

### Check Backup Exists
```bash
ls -la "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx"
```
**Expected:** Backup file exists

---

## 3. Network Access (for go get)

### Test Internet Connectivity
```bash
ping -n 2 proxy.golang.org
```
**Expected:** Successful ping responses

**Alternative test:**
```bash
curl -I https://proxy.golang.org
```
**Expected:** HTTP 200 OK or similar response

---

## 4. Git Configuration (Optional but Recommended)

### Check Git is Available
```bash
git --version
```
**Expected:** `git version 2.x.x`

### Check if Directory is Git Repo
```bash
cd "Z:\Meu Drive\controle\code"
git status
```
**Expected:** Shows git status OR "not a git repository" (both OK)

**Recommendation:** Initialize git if not already:
```bash
git init
git add .
git commit -m "Initial commit before Go implementation"
```

---

## 5. Existing Files Verification

### Check All Fix Scripts Exist
```bash
ls -la fix_*.py
```
**Expected files:**
- fix_1_rebuild_reference_sheet.py
- fix_3_unmerge_data_columns.py
- fix_5_standardize_column_layout.py

---

### Check Documentation Files Exist
```bash
ls -la *.md
```
**Expected key files:**
- ARCHITECTURE_PLAN_Go_Implementation.md
- DECISIONS_FINALIZED.md
- FIXES_APPLIED_Report.md
- PRE_IMPLEMENTATION_CHECKLIST.md
- USAGE_CALCULATION.md

---

## 6. Excel File Structure Verification

### Verify Reference Sheet
```bash
python << 'EOF'
import openpyxl
wb = openpyxl.load_workbook('Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx')
if 'Referência de Categorias' in wb.sheetnames:
    ws = wb['Referência de Categorias']
    print(f"Reference sheet exists: {ws.max_row - 4} subcategories")
else:
    print("ERROR: Reference sheet not found!")
wb.close()
EOF
```
**Expected:** "Reference sheet exists: 76 subcategories"

---

### Verify Column Standardization
```bash
python << 'EOF'
import openpyxl
wb = openpyxl.load_workbook('Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx')
for sheet in ['Fixas', 'Variáveis', 'Extras', 'Adicionais']:
    ws = wb[sheet]
    for col in range(1, 10):
        if ws.cell(1, col).value == 'Janeiro':
            print(f"{sheet}: Janeiro at column {openpyxl.utils.get_column_letter(col)} (should be D)")
            break
wb.close()
EOF
```
**Expected:** All 4 sheets show "Janeiro at column D"

---

### Verify No Merged Cells in Data Columns
```bash
python << 'EOF'
import openpyxl
wb = openpyxl.load_workbook('Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx')
for sheet in ['Fixas', 'Variáveis', 'Extras', 'Adicionais']:
    ws = wb[sheet]
    data_merges = sum(1 for mr in ws.merged_cells.ranges if mr.max_col >= 4)
    print(f"{sheet}: {data_merges} merged ranges in data columns (should be 0)")
wb.close()
EOF
```
**Expected:** All show "0 merged ranges in data columns"

---

## 7. Disk Space Check

### Check Available Disk Space
```bash
df -h "Z:\Meu Drive\controle\code" 2>/dev/null || echo "Run 'wmic logicaldisk get size,freespace,caption' on Windows"
```
**Expected:** At least 500MB free space

---

## 8. Terminal/Shell Configuration

### Check Current Directory
```bash
pwd
```
**Expected:** Should be in `Z:\Meu Drive\controle\code` or easily navigable

---

### Test Long Path Support (Windows)
```bash
# This creates a nested directory structure to test path length
mkdir -p test/very/long/path/that/goes/deep
rmdir -p test/very/long/path/that/goes/deep 2>/dev/null || rm -rf test
```
**Expected:** No errors (Windows long path support enabled)

---

## 9. Dependencies Availability

### Check Python (for verification scripts)
```bash
python --version
```
**Expected:** Python 3.x

---

### Check openpyxl is Installed
```bash
python -c "import openpyxl; print('openpyxl version:', openpyxl.__version__)"
```
**Expected:** Shows version (e.g., "openpyxl version: 3.1.5")

---

## 10. TDD Prerequisites

### Check if Go Testing Tools Work
After Go is verified, test:
```bash
go help test
```
**Expected:** Shows help for `go test` command

---

### Verify Go Test File Creation
```bash
# Test if we can create a test file
cat > test_check.go << 'EOF'
package main
import "testing"
func TestDummy(t *testing.T) {
    if 1+1 != 2 {
        t.Error("Math is broken")
    }
}
EOF

go test test_check.go
rm test_check.go
```
**Expected:** "PASS" output

---

## Complete Verification Script

Run this all-in-one verification:

```bash
#!/bin/bash

echo "=================================="
echo "PRE-PHASE 1 VERIFICATION"
echo "=================================="

errors=0

# 1. Go
echo -n "1. Go installed: "
if go version >/dev/null 2>&1; then
    go version
else
    echo "FAIL - Go not found"
    ((errors++))
fi

# 2. Directory access
echo -n "2. Directory writable: "
if mkdir test-verify 2>/dev/null && rmdir test-verify 2>/dev/null; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 3. Excel file
echo -n "3. Excel file exists: "
if [ -f "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx" ]; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 4. Backup exists
echo -n "4. Backup exists: "
if [ -f "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx" ]; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 5. Python
echo -n "5. Python available: "
if python --version >/dev/null 2>&1; then
    python --version
else
    echo "FAIL"
    ((errors++))
fi

# 6. openpyxl
echo -n "6. openpyxl installed: "
if python -c "import openpyxl" 2>/dev/null; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 7. Internet
echo -n "7. Internet access: "
if curl -s -I https://proxy.golang.org >/dev/null 2>&1; then
    echo "OK"
else
    echo "FAIL (needed for 'go get')"
    ((errors++))
fi

echo "=================================="
if [ $errors -eq 0 ]; then
    echo "ALL CHECKS PASSED ✓"
    echo "Ready to start Phase 1"
else
    echo "FAILED: $errors checks"
    echo "Fix issues above before proceeding"
fi
echo "=================================="
```

Save as `verify_ready.sh` and run:
```bash
bash verify_ready.sh
```

---

## Windows-Specific Verification (PowerShell)

If bash script doesn't work, use this PowerShell version:

```powershell
Write-Host "=================================="
Write-Host "PRE-PHASE 1 VERIFICATION"
Write-Host "=================================="

$errors = 0

# 1. Go
Write-Host -NoNewline "1. Go installed: "
try {
    $goVersion = go version 2>&1
    Write-Host $goVersion
} catch {
    Write-Host "FAIL - Go not found"
    $errors++
}

# 2. Directory
Write-Host -NoNewline "2. Directory writable: "
try {
    New-Item -ItemType Directory -Name "test-verify" -ErrorAction Stop | Out-Null
    Remove-Item "test-verify"
    Write-Host "OK"
} catch {
    Write-Host "FAIL"
    $errors++
}

# 3. Excel file
Write-Host -NoNewline "3. Excel file exists: "
if (Test-Path "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx") {
    Write-Host "OK"
} else {
    Write-Host "FAIL"
    $errors++
}

# 4. Backup
Write-Host -NoNewline "4. Backup exists: "
if (Test-Path "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx") {
    Write-Host "OK"
} else {
    Write-Host "FAIL"
    $errors++
}

# 5. Python
Write-Host -NoNewline "5. Python available: "
try {
    $pyVersion = python --version 2>&1
    Write-Host $pyVersion
} catch {
    Write-Host "FAIL"
    $errors++
}

# 6. Internet
Write-Host -NoNewline "6. Internet access: "
try {
    $response = Invoke-WebRequest -Uri "https://proxy.golang.org" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "OK"
} catch {
    Write-Host "FAIL"
    $errors++
}

Write-Host "=================================="
if ($errors -eq 0) {
    Write-Host "ALL CHECKS PASSED" -ForegroundColor Green
    Write-Host "Ready to start Phase 1"
} else {
    Write-Host "FAILED: $errors checks" -ForegroundColor Red
    Write-Host "Fix issues above before proceeding"
}
Write-Host "=================================="
```

Save as `verify_ready.ps1` and run:
```powershell
powershell -ExecutionPolicy Bypass -File verify_ready.ps1
```

---

## TDD Implementation Plan Update

Since you want to use TDD, here's the updated Phase 1 plan:

### Phase 1 with TDD: Project Setup + First Tests

**Modified approach:**

1. **Setup project structure**
   - Create directories
   - Initialize go.mod
   - Install excelize

2. **Write failing tests FIRST** (TDD Red phase)
   - `models/expense_test.go` - test Expense struct validation
   - `parser/parser_test.go` - test input string parsing
   - `utils/date_test.go` - test date conversion
   - `utils/currency_test.go` - test currency parsing

3. **Implement minimal code to pass tests** (TDD Green phase)
   - `models/expense.go`
   - `parser/parser.go`
   - `utils/date.go`
   - `utils/currency.go`

4. **Refactor** (TDD Refactor phase)
   - Clean up implementations
   - Add error handling
   - Optimize

### TDD Benefits for This Project

✅ **Catch edge cases early:** Date/currency parsing has many edge cases
✅ **Refactoring confidence:** Can change implementation safely
✅ **Documentation:** Tests show how to use each function
✅ **Regression prevention:** Won't break existing functionality

### Updated Token Estimate with TDD

**Phase 1 (with TDD):**
- Original: 4,000 tokens
- With tests: +1,500 tokens
- **New estimate: 5,500 tokens**

**Overall project:**
- Original: 46,300 tokens
- Adding TDD to all phases: +8,000 tokens
- **New total: ~54,300 tokens**

**Still within budget:** Uses 35% of your remaining 68% (well within limits)

---

## Summary Checklist

Before starting Phase 1, verify:

- [ ] Go 1.21+ installed and in PATH
- [ ] Go environment variables set (GOPATH, GOROOT)
- [ ] Directory writable
- [ ] Excel file accessible and not locked
- [ ] Backup file exists
- [ ] Python + openpyxl available (for verification)
- [ ] Internet access (for go get excelize)
- [ ] All fix scripts present
- [ ] Reference sheet has 76 subcategories
- [ ] All sheets start at column D
- [ ] No merged cells in data columns
- [ ] Sufficient disk space (500MB+)
- [ ] Git initialized (optional but recommended)

**Run the verification script above to check all items automatically.**

---

## Ready to Proceed?

Once all checks pass, reply with:
- "all checks passed" - I'll start Phase 1 with TDD
- "issue with X" - I'll help you fix it before starting

---

**Next:** Phase 1 with TDD (Setup + First Tests) - 5,500 tokens, 20-25 messages
