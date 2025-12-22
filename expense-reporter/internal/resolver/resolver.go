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
	RowNumber   int
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
