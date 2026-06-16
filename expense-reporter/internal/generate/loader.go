package generate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadTaxonomy loads taxonomy and entries from JSON files.
func LoadTaxonomy(taxonomyPath, entriesPath string) ([]ExpenseSheet, []RevenueBlock, error) {
	sheets, incomeBlocks, err := loadTaxonomyFile(taxonomyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading taxonomy: %w", err)
	}

	// Duplicate-subcategory validation applies to the taxonomy itself,
	// even when no entries are loaded.
	if _, err := buildSubcategoryMap(sheets, incomeBlocks); err != nil {
		return nil, nil, fmt.Errorf("loading taxonomy: %w", err)
	}

	if entriesPath == "" {
		return sheets, incomeBlocks, nil
	}

	err = loadEntries(entriesPath, &sheets, &incomeBlocks)
	if err != nil {
		return nil, nil, fmt.Errorf("loading entries: %w", err)
	}

	return sheets, incomeBlocks, nil
}

// rawSheet mirrors one element of the taxonomy file's "sheets" array.
type rawSheet struct {
	Name       string `json:"name"`
	Categories []struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"categories"`
}

// loadTaxonomyFile parses the taxonomy JSON file.
func loadTaxonomyFile(path string) ([]ExpenseSheet, []RevenueBlock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading taxonomy file: %w", err)
	}

	var raw struct {
		Sheets           []rawSheet `json:"sheets"`
		IncomeCategories []struct {
			Name  string   `json:"name"`
			Blocks []string `json:"blocks"`
		} `json:"incomeCategories"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing taxonomy JSON: %w", err)
	}

	sheets := rawSheetsToExpenseSheets(raw.Sheets)
	incomeBlocks := incomeCatsToRevenueBlocks(raw.IncomeCategories)

	return sheets, incomeBlocks, nil
}

// rawSheetsToExpenseSheets builds the ExpenseSheet tree from the raw taxonomy sheets.
func rawSheetsToExpenseSheets(raw []rawSheet) []ExpenseSheet {
	sheets := make([]ExpenseSheet, len(raw))
	for i, rs := range raw {
		sheets[i] = ExpenseSheet{Name: rs.Name}
		cats := make([]Category, len(rs.Categories))
		for j, rc := range rs.Categories {
			cats[j] = Category{Name: rc.Name}
			subs := make([]Subcat, len(rc.Subcategories))
			for k, name := range rc.Subcategories {
				subs[k] = Subcat{Name: name}
			}
			cats[j].Subs = subs
		}
		sheets[i].Cats = cats
	}
	return sheets
}

// incomeCatsToRevenueBlocks flattens income categories into a []RevenueBlock slice.
func incomeCatsToRevenueBlocks(raw []struct {
	Name   string   `json:"name"`
	Blocks []string `json:"blocks"`
}) []RevenueBlock {
	incomeBlocks := make([]RevenueBlock, 0)
	for _, ic := range raw {
		for _, blockName := range ic.Blocks {
			incomeBlocks = append(incomeBlocks, RevenueBlock{
				Category: ic.Name,
				Label:    blockName,
			})
		}
	}
	return incomeBlocks
}

// loadEntries reads entries from a JSONL file and populates the taxonomy.
func loadEntries(path string, sheets *[]ExpenseSheet, incomeBlocks *[]RevenueBlock) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening entries file: %w", err)
	}
	defer file.Close()

	subcatMap, err := buildSubcategoryMap(*sheets, *incomeBlocks)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	return scanEntries(scanner, subcatMap)
}

// scanEntries reads each non-blank JSONL line, parses it, looks up its subcategory,
// and attaches the entry to the appropriate month slice.
func scanEntries(scanner *bufio.Scanner, subcatMap map[string]subcatTarget) error {
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			Item        string  `json:"item"`
			Date        string  `json:"date"`
			Value       float64 `json:"value"`
			Subcategory string  `json:"subcategory"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("parsing entry line: %w", err)
		}

		subcat, exists := subcatMap[entry.Subcategory]
		if !exists {
			fmt.Fprintf(os.Stderr, "skipping entry %q: subcategory %q not in taxonomy\n", entry.Item, entry.Subcategory)
			continue
		}

		day, month, err := parseDate(entry.Date)
		if err != nil {
			return fmt.Errorf("parsing date for item %q: %w", entry.Item, err)
		}

		subcat.attachEntry(Entry{Item: entry.Item, Day: day, Value: entry.Value}, month-1)
	}

	return scanner.Err()
}

// buildSubcategoryMap creates a map from subcategory names to their targets.
func buildSubcategoryMap(sheets []ExpenseSheet, incomeBlocks []RevenueBlock) (map[string]subcatTarget, error) {
	result := make(map[string]subcatTarget)

	// Index into the backing slices — pointers to range copies would lose appends.
	for i := range sheets {
		for j := range sheets[i].Cats {
			for k := range sheets[i].Cats[j].Subs {
				sub := &sheets[i].Cats[j].Subs[k]
				if _, exists := result[sub.Name]; exists {
					return nil, fmt.Errorf("subcategory %q appears more than once in taxonomy", sub.Name)
				}
				result[sub.Name] = subcatTarget{kind: "expense", expense: sub}
			}
		}
	}

	for i := range incomeBlocks {
		block := &incomeBlocks[i]
		if _, exists := result[block.Label]; exists {
			return nil, fmt.Errorf("subcategory %q (income block label) appears more than once in taxonomy", block.Label)
		}
		result[block.Label] = subcatTarget{kind: "income", income: block}
	}

	return result, nil
}

// parseDate converts DD/MM to day and month integers.
func parseDate(dateStr string) (int, int, error) {
	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("malformed date %q", dateStr)
	}

	day, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	month, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

	if err1 != nil || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day in date %q", dateStr)
	}
	if err2 != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("invalid month in date %q", dateStr)
	}

	return day, month, nil
}

// subcatTarget holds a reference to either an expense subcategory or income block.
type subcatTarget struct {
	kind    string // "expense" or "income"
	expense *Subcat
	income  *RevenueBlock
}

// attachEntry appends entry to the appropriate month slice based on kind.
func (t subcatTarget) attachEntry(entry Entry, month int) {
	if t.kind == "expense" {
		t.expense.Months[month] = append(t.expense.Months[month], entry)
	} else {
		t.income.Months[month] = append(t.income.Months[month], entry)
	}
}
