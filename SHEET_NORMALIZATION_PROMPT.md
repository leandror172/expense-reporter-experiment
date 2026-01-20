# Excel Budget Sheet Normalization for Go Program Compatibility

## Context and Objective

You are tasked with normalizing an Excel budget workbook to ensure it has a perfectly consistent structure required by a Go program (`expense-reporter`) that automatically inserts expenses.

**Critical Requirement**: After normalization, the sheet structure must allow the Go program to insert ANY expense with ANY valid category/subcategory from the reference sheet without errors.

**You will NOT run the Go program** - you must ensure the Excel structure adheres to the program's requirements through analysis and normalization alone.

## Go Program Requirements (Critical Context)

The Go expense-reporter program works as follows:

### 1. Expense Format
```
<item_description>;<DD/MM>;<value_##,##>;<subcategory>
Example: Uber Centro;15/04;35,50;Uber/Taxi
```

### 2. Program Workflow
```
1. Parse expense string (item, date, value, subcategory)
2. Load "lista de itens" (reference sheet) to find subcategory mappings
3. Look up: which sheet contains this subcategory, at which row
4. Navigate to that sheet and row
5. Find the next empty row in that subcategory section
6. Write: item (column D/G/J/etc.), date (column E/H/K/etc.), value (column F/I/L/etc.)
```

### 3. What the Program Expects

**Reference Sheet ("lista de itens")**:
- Column A: Sheet name ("Fixas", "Variáveis", "Extras", "Investimentos")
- Column B: Category name
- Column C: Subcategory name
- Column D: Row number where subcategory starts in the expense sheet

**Expense Sheets (Fixas, Variáveis, Extras, Investimentos)**:
- Column A: Empty/formatting
- Column B: Category names (merged cells spanning multiple rows)
- Column C: Subcategory names (merged cells spanning multiple rows for each subcategory section)
- Columns D-AM: 12 months × 3 columns each (Item, Date, Value)
  - Jan: D (Item), E (Date), F (Value)
  - Feb: G (Item), H (Date), I (Value)
  - Mar: J (Item), K (Date), L (Value)
  - Apr: M (Item), N (Date), O (Value)
  - May: P (Item), Q (Date), R (Value)
  - Jun: S (Item), T (Date), U (Value)
  - Jul: V (Item), W (Date), X (Value)
  - Aug: Y (Item), Z (Date), AA (Value)
  - Sep: AB (Item), AC (Date), AD (Value)
  - Oct: AE (Item), AF (Date), AG (Value)
  - Nov: AH (Item), AI (Date), AJ (Value)
  - Dec: AK (Item), AL (Date), AM (Value)

**Subcategory Section Structure**:
Each subcategory must have:
1. **Header row**: Subcategory name in column C (merged cell)
2. **Data rows**: Multiple rows for individual expenses (can be empty initially)
3. **TOTAL row**: Row with SUM formulas in value columns (F, I, L, O, R, U, X, AA, AD, AG, AJ, AM)
   - This row sums all values for that subcategory across the month
   - The Go program scans for the TOTAL row to know where the subcategory section ends

### 4. Critical Details for Finding Empty Rows

The Go program uses this logic:
```go
// Pseudocode from internal/excel/reader.go
func FindNextEmptyRow(sheet, itemColumn, startRow, subcategoryName):
    currentRow = startRow
    loop up to 100 rows:
        itemValue = sheet.Cell(currentRow, itemColumn)
        subcatValue = sheet.Cell(currentRow, "C")

        // If item column is empty AND we're still in same subcategory section
        if itemValue == "" AND (subcatValue == "" OR subcatValue == subcategoryName):
            return currentRow  // Found empty row!

        // If we hit a different subcategory, we've gone too far
        if subcatValue != "" AND subcatValue != subcategoryName:
            return error

        currentRow++

    return error "no empty rows found"
```

**Key Point**: The TOTAL row must NOT have an item value, so the program can skip over it and continue scanning for empty rows if needed.

### 5. Merged Cell Handling

The Excel library (excelize) reads merged cells specially:
- A merged cell value appears in the TOP-LEFT cell
- Other cells in the merge return empty string
- Category in column B: merged across multiple rows, value appears in first row only
- Subcategory in column C: merged across the subcategory section rows, value appears in first row only

## Current Workbook Structure

### Sheets
1. **"lista de itens"** - Reference sheet (SOURCE OF TRUTH for which subcategories must exist)
2. **"Fixas"** - Fixed monthly expenses
3. **"Variáveis"** - Variable expenses
4. **"Extras"** - Extra/occasional expenses
5. **"Investimentos"** - Investment expenses

### Known Issues (To Fix)

**Issue 1: Missing Subcategories**
- Some subcategories listed in "lista de itens" don't have sections in their expense sheets
- Result: Go program fails with "subcategory not found" error

**Issue 2: Missing TOTAL Rows**
- Some subcategory sections have data but no TOTAL row with formulas
- Result: Go program may insert beyond the intended section or fail boundary detection

**Issue 3: Incorrect Row References**
- "lista de itens" may reference wrong row numbers
- Result: Go program navigates to wrong location and fails or corrupts data

**Issue 4: Inconsistent Section Boundaries**
- Subcategory sections may not have clear boundaries
- Result: Go program may overflow into next section

## Normalization Tasks

### Task 1: Inventory Current State

**Substeps**:
1. Read "lista de itens" completely - this defines ALL subcategories that MUST exist
2. For each expense sheet (Fixas, Variáveis, Extras, Investimentos):
   - Scan column C to find all subcategory headers
   - For each subcategory section:
     - Identify first row (where subcategory name appears in column C)
     - Identify data rows (rows below header until TOTAL or next subcategory)
     - Check if TOTAL row exists (row with formulas in value columns)
     - Record actual row number
3. Compare "lista de itens" vs. actual sheets:
   - Missing subcategories (in lista but not in sheet)
   - Orphan subcategories (in sheet but not in lista)
   - Row number mismatches
   - Missing TOTAL rows

### Task 2: Add Missing Subcategory Sections

For each subcategory in "lista de itens" that doesn't exist in its target sheet:

1. Identify correct insertion point (under appropriate category)
2. Insert rows for the new section:
   - Header row: Subcategory name in column C (merged across ~10 rows initially)
   - 5-10 blank data rows for future expense entries
   - TOTAL row with SUM formulas
3. Update category merged cell in column B if needed
4. Ensure proper row spacing

**Structure Template**:
```
Row N:   [Category in B (merged)] | [Subcategory in C (merged)] | [Empty D-AM]
Row N+1: [Category merge cont.]   | [Subcat merge cont.]        | [Empty D-AM]
Row N+2: [Category merge cont.]   | [Subcat merge cont.]        | [Empty D-AM]
...
Row N+8: [Category merge cont.]   | [Subcat merge cont.]        | [Empty D-AM]
Row N+9: [Category merge cont.]   | TOTAL                       | [Formulas: =SUM(...)]
```

### Task 3: Add Missing TOTAL Rows

For each subcategory section that lacks a TOTAL row:

1. Identify the last data row of the section
2. Insert a new row below it
3. Add "TOTAL" or the subcategory name in column C
4. Add SUM formulas in ALL 12 month value columns:
   - Column F (Jan): `=SUM(F[start]:F[end])`
   - Column I (Feb): `=SUM(I[start]:I[end])`
   - Column L (Mar): `=SUM(L[start]:L[end])`
   - Column O (Apr): `=SUM(O[start]:O[end])`
   - Column R (May): `=SUM(R[start]:R[end])`
   - Column U (Jun): `=SUM(U[start]:U[end])`
   - Column X (Jul): `=SUM(X[start]:X[end])`
   - Column AA (Aug): `=SUM(AA[start]:AA[end])`
   - Column AD (Sep): `=SUM(AD[start]:AD[end])`
   - Column AG (Oct): `=SUM(AG[start]:AG[end])`
   - Column AJ (Nov): `=SUM(AJ[start]:AJ[end])`
   - Column AM (Dec): `=SUM(AM[start]:AM[end])`

Where [start] = first data row after header, [end] = last data row before TOTAL.

**Important**: Leave item columns (D, G, J, M, P, S, V, Y, AB, AE, AH, AK) EMPTY in TOTAL row so the Go program knows to skip it.

### Task 4: Regenerate "lista de itens"

After normalizing the expense sheets:

1. Clear existing "lista de itens" content (except headers)
2. Scan each expense sheet in order:
   - For each subcategory found in column C:
     - Record: Sheet name, Category (from column B), Subcategory (from column C), Row number
     - Add entry to "lista de itens"
3. Sort entries logically (by sheet, then by row number)
4. Verify: every subcategory now has an accurate row reference

### Task 5: Validate Structure

Check all validation rules:

1. ✅ Every subcategory in "lista de itens" exists in its designated sheet
2. ✅ Every subcategory section has a TOTAL row with 12 formulas
3. ✅ Every row number in "lista de itens" column D is accurate
4. ✅ No subcategories exist in sheets without being in "lista de itens"
5. ✅ All TOTAL rows have empty item columns (D, G, J, etc.)
6. ✅ All formulas are error-free (no #REF!, #VALUE!, etc.)
7. ✅ Merged cells in columns B and C are consistent
8. ✅ All sheets follow the same column layout (D-AM pattern)

## Required Artifacts

### Artifact 1: Analysis Report
**File**: `normalization_analysis.md`

```markdown
# Sheet Normalization Analysis

## Summary Statistics
- Total subcategories in "lista de itens": [count]
- Breakdown by sheet:
  - Fixas: [count] subcategories
  - Variáveis: [count] subcategories
  - Extras: [count] subcategories
  - Investimentos: [count] subcategories

## Problems Identified

### Missing Subcategory Sections
[List subcategories from "lista de itens" that don't exist in sheets]

Example:
- Sheet "Variáveis", Subcategory "Uber/Taxi" - NOT FOUND in sheet
- Sheet "Fixas", Subcategory "Condomínio" - NOT FOUND in sheet

Total missing: [count]

### Missing TOTAL Rows
[List subcategory sections without TOTAL rows]

Example:
- Sheet "Variáveis", Row 15-20, Subcategory "Supermercado" - NO TOTAL ROW
- Sheet "Extras", Row 8-12, Subcategory "Dentista" - NO TOTAL ROW

Total missing TOTAL rows: [count]

### Incorrect Row References
[List entries in "lista de itens" with wrong row numbers]

Example:
- "Aluguel" referenced at row 5, actually at row 8
- "Internet" referenced at row 35, actually at row 42

Total incorrect references: [count]

### Orphan Subcategories
[List subcategories in sheets but not in "lista de itens"]

Example:
- Sheet "Variáveis", Row 25, Subcategory "Streaming" - NOT IN LISTA DE ITENS

Total orphans: [count]

## Recommended Actions
1. Add [count] missing subcategory sections
2. Add [count] TOTAL rows
3. Update [count] row references in "lista de itens"
4. Add [count] orphan subcategories to "lista de itens"
```

### Artifact 2: Subcategory Inventory
**File**: `subcategory_inventory.json`

```json
{
  "timestamp": "2025-12-27T00:00:00Z",
  "source_of_truth": {
    "sheet": "lista de itens",
    "total_entries": 145,
    "entries": [
      {
        "sheet": "Variáveis",
        "category": "Transporte",
        "subcategory": "Uber/Taxi",
        "referenced_row": 10,
        "actual_row": null,
        "exists_in_sheet": false,
        "has_total_row": false,
        "status": "MISSING_SECTION",
        "action_required": "Add complete subcategory section"
      },
      {
        "sheet": "Variáveis",
        "category": "Alimentação",
        "subcategory": "Supermercado",
        "referenced_row": 20,
        "actual_row": 20,
        "exists_in_sheet": true,
        "has_total_row": false,
        "status": "MISSING_TOTAL",
        "action_required": "Add TOTAL row with formulas"
      },
      {
        "sheet": "Fixas",
        "category": "Moradia",
        "subcategory": "Aluguel",
        "referenced_row": 5,
        "actual_row": 8,
        "exists_in_sheet": true,
        "has_total_row": true,
        "status": "WRONG_ROW_REFERENCE",
        "action_required": "Update row reference in lista de itens"
      },
      {
        "sheet": "Fixas",
        "category": "Moradia",
        "subcategory": "Condomínio",
        "referenced_row": 12,
        "actual_row": 12,
        "exists_in_sheet": true,
        "has_total_row": true,
        "status": "OK",
        "action_required": null
      }
    ]
  },
  "summary": {
    "status_counts": {
      "OK": 98,
      "MISSING_SECTION": 12,
      "MISSING_TOTAL": 23,
      "WRONG_ROW_REFERENCE": 15,
      "ORPHAN": 3
    }
  }
}
```

### Artifact 3: Detailed Change Log
**File**: `normalization_changes.md`

```markdown
# Normalization Changes Applied

## Fixas Sheet

### Added Subcategory Sections
1. **Row 25-32: "Seguro Residencial" under "Moradia"**
   - Added header row 25: Category "Moradia" in B, Subcategory "Seguro Residencial" in C
   - Merged B25:B32, C25:C31
   - Added 6 blank data rows (26-31)
   - Added TOTAL row 32:
     - F32: =SUM(F26:F31)
     - I32: =SUM(I26:I31)
     - [... all 12 months]

### Added TOTAL Rows
1. **Row 42: TOTAL for "Internet" (rows 38-41)**
   - Added formulas:
     - F42: =SUM(F38:F41)
     - I42: =SUM(I38:I41)
     - L42: =SUM(L38:L41)
     - O42: =SUM(O38:O41)
     - R42: =SUM(R38:R41)
     - U42: =SUM(U38:U41)
     - X42: =SUM(X38:X41)
     - AA42: =SUM(AA38:AA41)
     - AD42: =SUM(AD38:AD41)
     - AG42: =SUM(AG38:AG41)
     - AJ42: =SUM(AJ38:AJ41)
     - AM42: =SUM(AM38:AM41)

## Variáveis Sheet

### Added Subcategory Sections
[Similar detailed format for each addition]

### Added TOTAL Rows
[Similar detailed format for each TOTAL row]

## Extras Sheet

[Same structure]

## Investimentos Sheet

[Same structure]

## "lista de itens" Sheet

### Updated Row References
1. Row 5: "Aluguel" - changed from row 5 to row 8
2. Row 12: "Internet" - changed from row 35 to row 42
[... complete list]

### Added Entries (Orphan Subcategories)
1. Added: Variáveis | Assinaturas | Streaming | Row 67
[... complete list]

### Removed Obsolete Entries
[If any subcategories were removed from sheets]

## Summary
- Subcategory sections added: 12
- TOTAL rows added: 23
- Row references updated: 15
- Orphan entries added to lista: 3
- Total changes: 53
```

### Artifact 4: Validation Report
**File**: `validation_report.md`

```markdown
# Validation Report

## Rule 1: All Subcategories from "lista de itens" Exist in Sheets
✅ PASS - All 145 subcategories from "lista de itens" now exist in their designated sheets

Details:
- Fixas: 35/35 ✅
- Variáveis: 58/58 ✅
- Extras: 32/32 ✅
- Investimentos: 20/20 ✅

## Rule 2: All Subcategory Sections Have TOTAL Rows
✅ PASS - All 145 subcategory sections have TOTAL rows with formulas

Spot checks:
- Variáveis | Supermercado | Row 20 | TOTAL at row 27 ✅
- Fixas | Aluguel | Row 8 | TOTAL at row 11 ✅
- Extras | Dentista | Row 15 | TOTAL at row 22 ✅

## Rule 3: All Row References in "lista de itens" Are Accurate
✅ PASS - All 145 row references verified correct

Sample verification:
- "Uber/Taxi" → listed at row 45 → verified subcategory name at Variáveis!C45 ✅
- "Condomínio" → listed at row 12 → verified subcategory name at Fixas!C12 ✅

## Rule 4: No Orphan Subcategories
✅ PASS - All subcategories in expense sheets are now in "lista de itens"

Previously found 3 orphans, all added to "lista de itens".

## Rule 5: TOTAL Rows Have Empty Item Columns
✅ PASS - All TOTAL rows have empty item columns (D, G, J, M, P, S, V, Y, AB, AE, AH, AK)

This ensures the Go program will correctly skip TOTAL rows when finding empty cells.

## Rule 6: No Formula Errors
✅ PASS - Scanned entire workbook, found 0 formula errors

No instances of: #REF!, #VALUE!, #DIV/0!, #N/A, #NAME?, #NUM!

## Rule 7: Consistent Column Layout
✅ PASS - All expense sheets follow D-AM pattern

Verified:
- All sheets start month columns at D
- All sheets use 3-column pattern per month
- All sheets have 12 months (D-AM = 40 columns)

## Rule 8: Merged Cells Are Consistent
✅ PASS - All category and subcategory cells properly merged

Pattern verified:
- Column B: Category names merged vertically across their sections
- Column C: Subcategory names merged vertically across their data rows (excluding TOTAL)

## Final Verdict
✅ ALL VALIDATION RULES PASS

The normalized workbook is ready for use with the Go expense-reporter program.
Any expense with any valid subcategory from "lista de itens" can now be inserted.
```

### Artifact 5: Formula Reference
**File**: `formula_reference.md`

```markdown
# Formula Reference Guide

## TOTAL Row Formulas

All TOTAL rows use this pattern across 12 months:

| Month    | Item Col | Date Col | Value Col | Formula Pattern |
|----------|----------|----------|-----------|-----------------|
| January  | D        | E        | F         | =SUM(F[s]:F[e]) |
| February | G        | H        | I         | =SUM(I[s]:I[e]) |
| March    | J        | K        | L         | =SUM(L[s]:L[e]) |
| April    | M        | N        | O         | =SUM(O[s]:O[e]) |
| May      | P        | Q        | R         | =SUM(R[s]:R[e]) |
| June     | S        | T        | U         | =SUM(U[s]:U[e]) |
| July     | V        | W        | X         | =SUM(X[s]:X[e]) |
| August   | Y        | Z        | AA        | =SUM(AA[s]:AA[e]) |
| September| AB       | AC       | AD        | =SUM(AD[s]:AD[e]) |
| October  | AE       | AF       | AG        | =SUM(AG[s]:AG[e]) |
| November | AH       | AI       | AJ        | =SUM(AJ[s]:AJ[e]) |
| December | AK       | AL       | AM        | =SUM(AM[s]:AM[e]) |

Where:
- [s] = first data row (row after subcategory header)
- [e] = last data row (row before TOTAL)

## Example: Subcategory at Rows 10-16

Row 10: Header (Subcategory name in C10)
Rows 11-15: Data rows
Row 16: TOTAL row

Formulas in row 16:
- F16: =SUM(F11:F15)  [January]
- I16: =SUM(I11:I15)  [February]
- L16: =SUM(L11:L15)  [March]
- O16: =SUM(O11:O15)  [April]
- R16: =SUM(R11:R15)  [May]
- U16: =SUM(U11:U15)  [June]
- X16: =SUM(X11:X15)  [July]
- AA16: =SUM(AA11:AA15)  [August]
- AD16: =SUM(AD11:AD15)  [September]
- AG16: =SUM(AG11:AG15)  [October]
- AJ16: =SUM(AJ11:AJ15)  [November]
- AM16: =SUM(AM11:AM15)  [December]

## Important Notes

1. **Absolute vs Relative References**: Use relative references (F11:F15) not absolute ($F$11:$F$15) so formulas are easier to maintain

2. **Empty Item Columns**: NEVER put values in item columns (D, G, J, M, P, S, V, Y, AB, AE, AH, AK) of TOTAL rows

3. **Formula Consistency**: All 12 formulas must follow the same pattern within a TOTAL row

4. **Range Accuracy**: Ensure the SUM range captures all data rows but excludes the header and TOTAL itself
```

### Artifact 6: Go Program Compatibility Test Matrix
**File**: `go_compatibility_test.csv`

This shows which operations the Go program must be able to perform after normalization:

```csv
Test_ID,Test_Type,Subcategory,Expected_Behavior,Status
T001,Find_Subcategory,Uber/Taxi,Program finds row reference in lista de itens,PASS
T002,Navigate_To_Row,Uber/Taxi,Program navigates to correct row in Variáveis sheet,PASS
T003,Find_Empty_Row,Uber/Taxi,Program finds empty row in subcategory section,PASS
T004,Insert_Expense,Uber/Taxi,Program successfully inserts expense data,PASS
T005,Respect_TOTAL,Uber/Taxi,Program skips TOTAL row (no item value) when scanning,PASS
T006,Boundary_Detection,Uber/Taxi,Program stops at next subcategory boundary,PASS
T007,Find_Subcategory,Supermercado,Program finds row reference in lista de itens,PASS
...
[One test per subcategory for key operations]
```

## Execution Instructions

### Phase 1: Analysis (Generate Artifacts 1-2)
1. Load the Excel workbook
2. Read "lista de itens" completely
3. Scan all expense sheets (Fixas, Variáveis, Extras, Investimentos)
4. Build complete inventory of current vs. expected state
5. Generate Artifact 1 (Analysis Report)
6. Generate Artifact 2 (Subcategory Inventory JSON)

### Phase 2: Normalization (Modify Excel)
1. Add missing subcategory sections (Task 2)
2. Add missing TOTAL rows (Task 3)
3. Fix any structural inconsistencies
4. Regenerate "lista de itens" with accurate row numbers (Task 4)

### Phase 3: Validation (Generate Artifacts 3-6)
1. Verify all validation rules pass (Task 5)
2. Generate Artifact 3 (Change Log)
3. Generate Artifact 4 (Validation Report)
4. Generate Artifact 5 (Formula Reference)
5. Generate Artifact 6 (Test Matrix)

### Phase 4: Delivery
1. Provide normalized Excel workbook
2. Provide all 6 artifacts
3. Confirm all validation rules pass
4. Provide summary of changes

## Critical Success Criteria

✅ **Completeness**: Every subcategory in "lista de itens" exists with proper structure
✅ **Accuracy**: All row references in "lista de itens" are correct
✅ **Consistency**: All TOTAL rows follow the same formula pattern
✅ **Compatibility**: Structure matches exactly what Go program expects
✅ **Validation**: All 8 validation rules pass
✅ **Documentation**: All 6 artifacts generated

## Special Instructions

### 1. Preserve Existing Data
- NEVER delete or modify existing expense entries
- Only add structure (headers, TOTAL rows, missing sections)
- If adjusting row positions, ensure data moves correctly

### 2. Formula Precision
- All TOTAL formulas must be exact
- Use correct column letters for each month
- Ensure SUM ranges capture all and only data rows

### 3. Merged Cell Handling
- Category cells in column B: merge vertically across entire category section
- Subcategory cells in column C: merge vertically from header to last data row (exclude TOTAL)
- Follow existing merge patterns in the workbook

### 4. Source of Truth
- "lista de itens" defines what MUST exist
- If conflicts arise, trust "lista de itens"
- Add any orphans found in sheets to "lista de itens"

### 5. Think Like the Go Program
Remember the program will:
- Read "lista de itens" to find subcategories
- Navigate using row numbers
- Scan for empty cells in item columns
- Stop at subcategory boundaries or TOTAL rows
- Your structure must support these operations

## Quality Checklist

Before delivering, verify:

- [ ] All 6 artifacts generated
- [ ] All validation rules pass (8/8)
- [ ] No Excel formula errors exist
- [ ] "lista de itens" is 100% accurate
- [ ] Every subcategory has proper structure
- [ ] All TOTAL rows have 12 formulas
- [ ] All changes are documented
- [ ] Normalized workbook is provided

## Final Note

This is NOT about running the Go program - it's about ensuring the Excel structure is **exactly** what the Go program needs to function correctly. Your expertise in Excel analysis and normalization is what's required here.

The Go program has already been built and tested. Your job is to make the Excel workbook match its requirements perfectly.

Success = The workbook structure is compatible with the Go program's expectations, verified through structural analysis and validation rules.
