package taxonomy

import (
	"errors"
	"fmt"
)

var (
	// ErrLeafNotFound means the bare subcategory name does not exist anywhere in
	// the taxonomy.
	ErrLeafNotFound = errors.New("leaf not found")
	// ErrLeafAmbiguous means the bare subcategory name exists under more than one
	// type and the supplied hint did not resolve it to exactly one.
	ErrLeafAmbiguous = errors.New("ambiguous leaf name")
)

// PathMap is a bidirectional mapping between an expense taxonomy and a flat list
// of full-path "enum" strings of the form "Type/Category/Subcategory". The reverse
// direction is a lookup, never a parse: type/category/subcategory names may
// themselves contain "/", so the display string cannot be split back into parts.
type PathMap struct {
	displayStrings []string
	reverseMap     map[string][3]string
}

// BuildPathMap constructs a PathMap by walking every type→category→subcategory
// leaf in taxonomy order. It errors if two distinct leaves collapse to the same
// display string, which would make the enum ambiguous to a model.
func BuildPathMap(sheets []ExpenseType) (PathMap, error) {
	pm := PathMap{reverseMap: make(map[string][3]string)}
	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				display := buildDisplayString(sheet.Name, cat.Name, sub.Name)
				if _, dup := pm.reverseMap[display]; dup {
					return PathMap{}, fmt.Errorf("ambiguous path: %s", display)
				}
				pm.displayStrings = append(pm.displayStrings, display)
				pm.reverseMap[display] = [3]string{sheet.Name, cat.Name, sub.Name}
			}
		}
	}
	return pm, nil
}

// Enum returns the ordered list of display strings — the enum members.
func (pm PathMap) Enum() []string {
	return pm.displayStrings
}

// Split returns the type, category, and subcategory for a display string via a
// pure reverse-map lookup. ok is false when the path is not in the taxonomy.
// It never splits on "/".
func (pm PathMap) Split(path string) (typ, cat, sub string, ok bool) {
	parts, exists := pm.reverseMap[path]
	if !exists {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// PathFor returns the canonical display string for a (type, category, subcategory)
// triple, or ok=false if that exact leaf is not in the taxonomy. The result is
// validated against the reverse map, so callers never fabricate an off-enum path.
func (pm PathMap) PathFor(typ, cat, sub string) (path string, ok bool) {
	display := buildDisplayString(typ, cat, sub)
	if _, exists := pm.reverseMap[display]; !exists {
		return "", false
	}
	return display, true
}

// PathEnum builds a PathMap and returns its Enum list.
func PathEnum(sheets []ExpenseType) ([]string, error) {
	pm, err := BuildPathMap(sheets)
	if err != nil {
		return nil, err
	}
	return pm.Enum(), nil
}

// ResolveLeaf resolves a bare subcategory name to its owning (type, category).
// Matching is NFC-insensitive but the verbatim taxonomy names are returned.
//   - 0 matches → ErrLeafNotFound.
//   - exactly 1 match → returned; typeHint is ignored.
//   - >1 matches → typeHint must resolve to exactly one owning type, else
//     ErrLeafAmbiguous (the name exists, so it is never "not found" here).
func ResolveLeaf(sheets []ExpenseType, sub, typeHint string) (typ, cat string, err error) {
	matches := collectLeafMatches(sheets, sub)

	switch len(matches) {
	case 0:
		return "", "", ErrLeafNotFound
	case 1:
		return matches[0][0], matches[0][1], nil
	default:
		if typeHint == "" {
			return "", "", ErrLeafAmbiguous
		}
		filtered := filterByTypeHint(matches, typeHint)
		if len(filtered) == 1 {
			return filtered[0][0], filtered[0][1], nil
		}
		// Zero or >1 after filtering: the hint did not pin exactly one owning
		// type, so the leaf remains ambiguous (never "not found").
		return "", "", ErrLeafAmbiguous
	}
}

// TypesForLeaf returns the distinct expense type names a subcategory appears under,
// in taxonomy order. It powers disambiguation prompts and error messages for the
// model-less `add` path, where a bare leaf name may belong to several types.
func TypesForLeaf(sheets []ExpenseType, sub string) []string {
	seen := make(map[string]bool)
	var types []string
	for _, match := range collectLeafMatches(sheets, sub) {
		if !seen[match[0]] {
			seen[match[0]] = true
			types = append(types, match[0])
		}
	}
	return types
}

// CategoryForLeaf returns the category owning a subcategory when that category is the
// same for every type the leaf appears under — true for all real leaves, including
// the few that repeat across types but share one category (e.g. Dentista is always
// under Saúde, Estacionamento always under Transporte). Returns ok=false when the
// leaf is absent or, defensively, appears under differing categories. Useful where
// only the category is needed and the type is irrelevant (e.g. the feedback log).
func CategoryForLeaf(sheets []ExpenseType, sub string) (category string, ok bool) {
	matches := collectLeafMatches(sheets, sub)
	if len(matches) == 0 {
		return "", false
	}
	cat := matches[0][1]
	for _, m := range matches[1:] {
		if m[1] != cat {
			return "", false
		}
	}
	return cat, true
}

// buildDisplayString joins the three taxonomy names into one enum display string.
func buildDisplayString(typ, cat, sub string) string {
	return fmt.Sprintf("%s/%s/%s", typ, cat, sub)
}

// collectLeafMatches returns every (type, category) pair whose subcategory name
// matches sub (compared NFC-insensitively), preserving verbatim names.
func collectLeafMatches(sheets []ExpenseType, sub string) [][2]string {
	want := normalizeKey(sub)
	var matches [][2]string
	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, subcat := range cat.Subs {
				if normalizeKey(subcat.Name) == want {
					matches = append(matches, [2]string{sheet.Name, cat.Name})
				}
			}
		}
	}
	return matches
}

// filterByTypeHint keeps only matches whose type equals typeHint (NFC-insensitive).
func filterByTypeHint(matches [][2]string, typeHint string) [][2]string {
	want := normalizeKey(typeHint)
	var filtered [][2]string
	for _, match := range matches {
		if normalizeKey(match[0]) == want {
			filtered = append(filtered, match)
		}
	}
	return filtered
}
