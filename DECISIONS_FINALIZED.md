# Implementation Decisions - FINALIZED

## Date: 2025-12-19

---

## ✅ All Decisions Made

### 1. Date Year Handling
**Decision:** Always use 2025

**Rationale:** Simplest approach, matches current year of budget

**Implementation:**
```go
func ParseDate(dateStr string) (time.Time, error) {
    parts := strings.Split(dateStr, "/")
    day, _ := strconv.Atoi(parts[0])
    month, _ := strconv.Atoi(parts[1])

    return time.Date(2025, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}
```

---

### 2. Deployment Approach
**Decision:** Pure CLI tool

**Rationale:**
- Simpler to develop and maintain
- Can add VBA wrapper later if desired
- User can run from terminal or batch file

**Usage:**
```bash
# Single expense
expense-reporter.exe "Uber Centro;15/04;35,50;Uber/Taxi"

# Interactive mode
expense-reporter.exe --interactive

# Batch mode
expense-reporter.exe --batch expenses.txt
```

---

### 3. Project Location
**Decision:** `Z:\Meu Drive\controle\code\expense-reporter\`

**Structure:**
```
expense-reporter/
├── main.go
├── go.mod
├── go.sum
├── internal/
│   ├── parser/
│   ├── resolver/
│   ├── excel/
│   └── models/
├── pkg/
│   └── utils/
└── config/
    └── config.json
```

---

### 4. Ambiguity Resolution
**Decision:** Interactive prompts (Option 1)

**Rationale:**
- No Excel changes needed
- Clear UX when ambiguous
- Handles all 7 ambiguous cases gracefully

**Implementation:**
When user enters "Dentista", program shows:
```
⚠ Found 2 matches for 'Dentista':
  [1] Variáveis > Saúde
  [2] Extras > Saúde

Select option (1-2): _
```

---

### 5. Granularity Handling (Issue #4)
**Decision:** Smart matching with parent fallback

**Rationale:**
- No Excel restructuring needed
- User can be specific or generic
- Works with current sheet structure

**Implementation:**
```go
// User enters: "Orion - Consultas"
// 1. Try exact match: Not found
// 2. Extract parent: "Orion"
// 3. Find "Orion" in sheet: Success
// 4. Insert under "Orion" row
```

---

## 🎉 BONUS: Column Layout Standardization

### Issue
- Fixas/Extras/Adicionais started at column C
- Variáveis started at column D
- Required sheet-specific logic

### Solution Applied
**Fix #5:** Inserted blank column C in Fixas, Extras, Adicionais

### Result
**All sheets now uniform:**
- Column A: Category
- Column B: Subcategory
- Column C: (blank/spacing)
- Column D: Janeiro (all sheets)
- Column G: Fevereiro (all sheets)
- Pattern continues...

### Impact on Go Code
**Before:**
```go
var monthColumns = map[string]map[int]string{
    "Fixas":      {1: "C", 2: "F", ...},  // Different!
    "Variáveis":  {1: "D", 2: "G", ...},  // Different!
    "Extras":     {1: "C", 2: "F", ...},  // Different!
    "Adicionais": {1: "C", 2: "F", ...},  // Different!
}
```

**After:**
```go
// Single mapping for ALL sheets
var monthColumns = map[int]string{
    1: "D", 2: "G", 3: "J", 4: "M", 5: "P", 6: "S",
    7: "V", 8: "Y", 9: "AB", 10: "AE", 11: "AH", 12: "AK",
}
```

**Code Simplification:**
- No sheet-specific column logic needed
- -1,500 tokens in development
- Cleaner, more maintainable code
- Faster execution (no map-of-maps lookup)

---

## 📊 Updated Cost Estimate

### Original Estimate
- Total: 49,500 tokens

### With All Fixes Applied
- Fix #1 (Reference rebuild): -3,000 tokens
- Fix #3 (Unmerge cells): -2,000 tokens
- Fix #5 (Column standardization): -1,500 tokens
- Ambiguity handling: +2,500 tokens
- Smart matching: +800 tokens

**New Total: ~46,300 tokens** (6.5% reduction from original)

---

## 🚀 Ready to Proceed

All decisions finalized. No outstanding questions.

**Next Step:** Begin Go implementation Phase 1

### Phase 1 Tasks:
1. Initialize Go module
2. Install excelize dependency
3. Create project structure
4. Define data models
5. Create configuration file

**Estimated tokens for Phase 1:** ~4,000
**Estimated time:** 30-45 minutes

---

## Summary of All Applied Fixes

| Fix # | Description | Status |
|-------|-------------|--------|
| 1 | Rebuild reference sheet | ✅ Complete |
| 2 | Ambiguous subcategories | ✅ Configured (prompts) |
| 3 | Unmerge data columns | ✅ Complete |
| 4 | Granularity handling | ✅ Configured (smart match) |
| 5 | Standardize column layout | ✅ Complete |

**Backup:** `Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx`

**Excel File Status:** Optimized and ready for automation

---

## No Outstanding Questions

All decision points have been addressed:
- ✅ Date year handling
- ✅ Deployment approach
- ✅ Project location
- ✅ Ambiguity resolution
- ✅ Granularity handling
- ✅ Column layout (bonus fix)

**Status:** READY TO IMPLEMENT
