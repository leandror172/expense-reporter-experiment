package resolver

import (
	"errors"
	"fmt"
	"strings"
)

// SubcategoryMapping represents a subcategory's location information
type SubcategoryMapping struct {
	Subcategory string
	SheetName   string
	Category    string
	RowNumber   int // Subcategory header row (from column D)
	TotalRow    int // TOTAL row number (from column F "Total Linha")
}

// PathIndex provides hierarchical lookup capabilities for subcategories
type PathIndex struct {
	// Flat map for backward compatibility
	BySubcategory map[string][]SubcategoryMapping

	// Hierarchical indexes
	ByFullPath map[string]*SubcategoryMapping  // "fixas,habitação,diarista" → single mapping
	By2Level   map[string][]SubcategoryMapping // "habitação,diarista" → possible mappings
	By1Level   map[string][]SubcategoryMapping // "diarista" → possible mappings
}

// NormalizePath converts path to lowercase and trims spaces
// "Fixas, Habitação,  Diarista" → "fixas,habitação,diarista"
func NormalizePath(path string) string {
	path = strings.ToLower(strings.TrimSpace(path))
	parts := strings.Split(path, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ",")
}

// SplitPath splits and normalizes a hierarchical path
// Returns: [sheet, category, subcategory] or fewer levels
func SplitPath(path string) []string {
	normalized := NormalizePath(path)
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, ",")
}

// ResolveSubcategoryWithPath implements hierarchical path resolution
// Tries deepest level first: "Diarista" → "Habitação,Diarista" → "Fixas,Habitação,Diarista"
// Returns (mapping, isAmbiguous, error)
func ResolveSubcategoryWithPath(index *PathIndex, subcategory string) (*SubcategoryMapping, bool, error) {
	if subcategory == "" {
		return nil, false, errors.New("subcategory cannot be empty")
	}

	// Hierarchical path (contains comma)
	if strings.Contains(subcategory, ",") {
		return resolveHierarchical(index, subcategory)
	}

	// Single-level lookup
	return resolveSingleLevel(index, subcategory)
}

// resolveHierarchical handles comma-separated hierarchical paths
func resolveHierarchical(index *PathIndex, input string) (*SubcategoryMapping, bool, error) {
	parts := SplitPath(input)
	numLevels := len(parts)

	if numLevels == 0 {
		return nil, false, errors.New("empty path")
	}
	if numLevels > 3 {
		return nil, false, fmt.Errorf("path too deep: max 3 levels (sheet,category,subcategory)")
	}

	// Try from deepest to shallowest
	for depth := 1; depth <= numLevels; depth++ {
		pathParts := parts[numLevels-depth:]
		testPath := strings.Join(pathParts, ",")

		var options []SubcategoryMapping

		if depth == 1 {
			options = index.By1Level[testPath]
		} else if depth == 2 {
			options = index.By2Level[testPath]
		} else {
			// depth == 3: full path
			if mapping, exists := index.ByFullPath[testPath]; exists {
				return mapping, false, nil
			}
			continue
		}

		if len(options) == 1 {
			return &options[0], false, nil // Unambiguous!
		}
		if len(options) > 1 {
			continue // Try next level
		}
	}

	return nil, false, fmt.Errorf("subcategory path not found: %s", input)
}

// resolveSingleLevel handles single-level subcategory lookups
func resolveSingleLevel(index *PathIndex, subcategory string) (*SubcategoryMapping, bool, error) {
	normalized := NormalizePath(subcategory)
	options := index.By1Level[normalized]

	if len(options) == 1 {
		return &options[0], false, nil
	}
	if len(options) > 1 {
		return nil, true, nil // Ambiguous
	}

	return nil, false, fmt.Errorf("subcategory not found: %s", subcategory)
}

// ResolveSubcategory finds the sheet and location for a subcategory
// Returns (mapping, isAmbiguous, error)
// If isAmbiguous is true, use GetAmbiguousOptions to present choices to user
func ResolveSubcategory(mappings map[string][]SubcategoryMapping, subcategory string) (*SubcategoryMapping, bool, error) {
	if subcategory == "" {
		return nil, false, errors.New("subcategory cannot be empty")
	}

	// Try exact match first
	if options, exists := mappings[subcategory]; exists {
		if len(options) == 1 {
			return &options[0], false, nil
		}
		if len(options) > 1 {
			// Ambiguous - needs user choice
			return nil, true, nil
		}
	}

	// Try smart matching - extract parent if detailed subcategory
	parent := ExtractParentSubcategory(subcategory)
	if parent != subcategory {
		// Try again with parent
		if options, exists := mappings[parent]; exists {
			if len(options) == 1 {
				return &options[0], false, nil
			}
			if len(options) > 1 {
				return nil, true, nil
			}
		}
	}

	return nil, false, fmt.Errorf("subcategory not found: %s", subcategory)
}

// ExtractParentSubcategory extracts parent from detailed subcategory
// "Orion - Consultas" → "Orion"
// "Uber/Taxi" → "Uber/Taxi" (unchanged)
func ExtractParentSubcategory(subcategory string) string {
	// Look for " - " pattern (space-dash-space)
	if before, _, ok := strings.Cut(subcategory, " - "); ok {
		return strings.TrimSpace(before)
	}
	return subcategory
}

// GetAmbiguousOptions returns all options for an ambiguous subcategory
func GetAmbiguousOptions(mappings map[string][]SubcategoryMapping, subcategory string) []SubcategoryMapping {
	if options, exists := mappings[subcategory]; exists {
		return options
	}
	return []SubcategoryMapping{}
}

// SelectOption allows user to pick from ambiguous options (1-indexed)
func SelectOption(options []SubcategoryMapping, choice int) (*SubcategoryMapping, error) {
	if choice < 1 || choice > len(options) {
		return nil, fmt.Errorf("invalid choice %d, must be between 1 and %d", choice, len(options))
	}
	return &options[choice-1], nil
}
