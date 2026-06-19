package taxonomy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// normalizeKey canonicalizes a routing-key segment to Unicode NFC so that taxonomy
// strings (from config/taxonomy.json) and entry strings (from the review→apply→log
// path, which reads the workbook) compare equal even if one source emits decomposed
// accents (NFD: a+◌́) and the other composed (NFC: á). Applied to map KEYS only —
// stored display names (Subcat.Name etc.) are left exactly as authored.
func normalizeKey(s string) string {
	return norm.NFC.String(s)
}

// LoadTaxonomy loads taxonomy and entries from JSON files.
func LoadTaxonomy(taxonomyPath, entriesPath string) ([]ExpenseType, []RevenueBlock, error) {
	sheets, incomeBlocks, err := loadTaxonomyFile(taxonomyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading taxonomy: %w", err)
	}

	// Duplicate-subcategory validation applies to the taxonomy itself,
	// even when no entries are loaded. Both routing maps + the ambiguity set are
	// reused below when entries are present.
	byPath, byName, ambiguous, err := buildSubcategoryMap(sheets, incomeBlocks)
	if err != nil {
		return nil, nil, fmt.Errorf("loading taxonomy: %w", err)
	}

	if entriesPath == "" {
		return sheets, incomeBlocks, nil
	}

	if err := loadEntries(entriesPath, byPath, byName, ambiguous); err != nil {
		return nil, nil, fmt.Errorf("loading entries: %w", err)
	}

	return sheets, incomeBlocks, nil
}

// rawType mirrors one element of the taxonomy file's "sheets" array.
type rawType struct {
	Name       string `json:"name"`
	Categories []struct {
		Name          string   `json:"name"`
		Subcategories []string `json:"subcategories"`
	} `json:"categories"`
}

// loadTaxonomyFile parses the taxonomy JSON file.
func loadTaxonomyFile(path string) ([]ExpenseType, []RevenueBlock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading taxonomy file: %w", err)
	}

	var raw struct {
		Sheets           []rawType `json:"types"`
		IncomeCategories []struct {
			Name   string   `json:"name"`
			Blocks []string `json:"blocks"`
		} `json:"incomeCategories"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing taxonomy JSON: %w", err)
	}

	sheets := rawTypesToExpenseTypes(raw.Sheets)
	incomeBlocks := incomeCatsToRevenueBlocks(raw.IncomeCategories)

	return sheets, incomeBlocks, nil
}

// rawTypesToExpenseTypes builds the ExpenseType tree from the raw taxonomy sheets.
func rawTypesToExpenseTypes(raw []rawType) []ExpenseType {
	sheets := make([]ExpenseType, len(raw))
	for i, rs := range raw {
		sheets[i] = ExpenseType{Name: rs.Name}
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

// loadEntries reads entries from a JSONL file and routes them using the two
// pre-built routing maps and the ambiguity set.
func loadEntries(path string, byPath, byName map[string]subcatTarget, ambiguous map[string]bool) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening entries file: %w", err)
	}
	defer file.Close()

	return scanEntries(bufio.NewScanner(file), byPath, byName, ambiguous)
}

// scanEntries reads each non-blank JSONL line, parses it, routes it to a target via
// two-tier lookup, and attaches the entry to the appropriate month slice.
//
// Tier 1 (typed entry): if the entry carries a type, route on the full-path key —
// this resolves ambiguous leaf names to exactly one block.
// Tier 2 (type-less entry): fall back to the bare-name map (today's behavior), which
// still skips genuinely-ambiguous names. Legacy/auto/batch-auto lines take this path.
func scanEntries(scanner *bufio.Scanner, byPath, byName map[string]subcatTarget, ambiguous map[string]bool) error {
	fallbackCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			Item        string  `json:"item"`
			Date        string  `json:"date"`
			Value       float64 `json:"value"`
			Type        string  `json:"type"` // expense type (Plan A); "" for legacy/auto entries
			Category    string  `json:"category"`
			Subcategory string  `json:"subcategory"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("parsing entry line: %w", err)
		}

		// NFC-normalize the routing fields so they match the (also-NFC) map keys
		// regardless of how the source encoded accents. Item is display-only — left as-is.
		entry.Type = normalizeKey(entry.Type)
		entry.Category = normalizeKey(entry.Category)
		entry.Subcategory = normalizeKey(entry.Subcategory)

		subcat, exists := routeEntry(entry.Type, entry.Category, entry.Subcategory, byPath, byName)
		if !exists {
			warnUnroutable(entry.Item, entry.Subcategory, entry.Type == "" && ambiguous[entry.Subcategory])
			continue
		}
		if entry.Type == "" {
			fallbackCount++
		}

		day, month, err := parseDate(entry.Date)
		if err != nil {
			return fmt.Errorf("parsing date for item %q: %w", entry.Item, err)
		}

		subcat.attachEntry(Entry{Item: entry.Item, Day: day, Value: entry.Value}, month-1)
	}

	if fallbackCount > 0 {
		// Transitional: surfaces how many entries still lack a type (bare-name routed).
		// Drops to zero once the classifier emits a type for every entry.
		fmt.Fprintf(os.Stderr, "note: %d entr%s routed via the type-less bare-name fallback\n",
			fallbackCount, plural(fallbackCount, "y", "ies"))
	}

	return scanner.Err()
}

// routeEntry resolves an entry to a target using two-tier lookup: full-path key when a
// type is present, bare-name fallback otherwise.
func routeEntry(typ, category, subcategory string, byPath, byName map[string]subcatTarget) (subcatTarget, bool) {
	if typ != "" {
		target, ok := byPath[expensePath(typ, category, subcategory)]
		return target, ok
	}
	target, ok := byName[subcategory]
	return target, ok
}

// plural picks the singular or plural suffix for n.
func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}

// warnUnroutable reports a skipped entry, distinguishing an ambiguous subcategory
// (a name shared by multiple full paths, skipped pending the full-path routing
// redesign) from one that is simply absent from the taxonomy. Both exit 0.
func warnUnroutable(item, subcategory string, isAmbiguous bool) {
	if isAmbiguous {
		fmt.Fprintf(os.Stderr,
			"skipping entry %q: subcategory %q is ambiguous across multiple blocks (pending full-path routing)\n",
			item, subcategory)
		return
	}
	fmt.Fprintf(os.Stderr, "skipping entry %q: subcategory %q not in taxonomy\n", item, subcategory)
}

// buildSubcategoryMap builds TWO routing maps plus the set of ambiguous bare names:
//
//   - byPath: full-path key (expensePath/incomePath) → target. Used to route entries
//     that carry a type (Plan A field). A full path resolves to exactly one target, so
//     ambiguity never applies here — this is what lets ambiguous leaf names route.
//   - byName: bare subcategory/label → target, with ambiguous names dropped (see
//     registerTarget). RETAINED as the fallback for type-less entries (legacy logs,
//     auto/batch-auto output). This is a transitional bridge, kept until the classifier
//     emits a type for every entry; do not treat it as permanent.
//
// A subcategory's identity is its full path; only an exact repeat of a full path is a
// validation error (detected via byPath presence).
func buildSubcategoryMap(sheets []ExpenseType, incomeBlocks []RevenueBlock) (byPath, byName map[string]subcatTarget, ambiguous map[string]bool, err error) {
	byPath = make(map[string]subcatTarget)
	byName = make(map[string]subcatTarget)
	ambiguous = make(map[string]bool)

	// Index into the backing slices — pointers to range copies would lose appends.
	for i := range sheets {
		for j := range sheets[i].Cats {
			for k := range sheets[i].Cats[j].Subs {
				sub := &sheets[i].Cats[j].Subs[k]
				path := expensePath(sheets[i].Name, sheets[i].Cats[j].Name, sub.Name)
				if _, dup := byPath[path]; dup {
					return nil, nil, nil, fmt.Errorf("subcategory %q appears more than once in %s/%s", sub.Name, sheets[i].Name, sheets[i].Cats[j].Name)
				}
				target := subcatTarget{kind: "expense", expense: sub}
				byPath[path] = target
				registerTarget(byName, ambiguous, sub.Name, target)
			}
		}
	}

	for i := range incomeBlocks {
		block := &incomeBlocks[i]
		path := incomePath(block.Category, block.Label)
		if _, dup := byPath[path]; dup {
			return nil, nil, nil, fmt.Errorf("income block %q appears more than once in %s", block.Label, block.Category)
		}
		target := subcatTarget{kind: "income", income: block}
		byPath[path] = target
		registerTarget(byName, ambiguous, block.Label, target)
	}

	return byPath, byName, ambiguous, nil
}

// registerTarget records target under name for bare-name routing. Each call is a
// distinct full path (callers reject exact-path repeats first), so finding the
// name already present means a second full path claims it: the name becomes
// ambiguous, is removed from the map, and stays ambiguous permanently — which
// also stops a third occurrence from re-adding it.
func registerTarget(result map[string]subcatTarget, ambiguous map[string]bool, name string, target subcatTarget) {
	name = normalizeKey(name) // keys are NFC so type-less lookups match regardless of source encoding
	if ambiguous[name] {
		return
	}
	if _, exists := result[name]; exists {
		delete(result, name)
		ambiguous[name] = true
		return
	}
	result[name] = target
}

// expensePath / incomePath join taxonomy segments with a null byte — a separator
// that cannot occur in human-typed names (which DO contain '/', e.g. "Uber/Taxi").
// The kind prefix keeps a 3-segment expense path from ever equalling a 2-segment
// income path.
func expensePath(sheet, category, sub string) string {
	return "expense\x00" + normalizeKey(sheet) + "\x00" + normalizeKey(category) + "\x00" + normalizeKey(sub)
}

func incomePath(category, label string) string {
	return "income\x00" + normalizeKey(category) + "\x00" + normalizeKey(label)
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
