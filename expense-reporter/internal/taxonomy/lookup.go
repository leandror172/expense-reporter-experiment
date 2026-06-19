package taxonomy

import (
	"errors"
	"strings"
)

// ErrTypeNotFound is returned by TypeIndex.LookupType when the (category, subcategory)
// pair is not present in any expense type.
var ErrTypeNotFound = errors.New("type not found")

// ErrTypeAmbiguous is returned by TypeIndex.LookupType when the same (category,
// subcategory) pair appears under more than one expense type.
var ErrTypeAmbiguous = errors.New("type ambiguous")

// TypeIndex holds a reverse index from (category, subcategory) pairs to expense types.
// It is built by BuildTypeIndex and queried by LookupType.
type TypeIndex struct {
	byCategorySubcat map[string]string // normalized key → typeName
	ambiguous        map[string]bool   // normalized key → is ambiguous
}

// BuildTypeIndex constructs a reverse lookup index from expense type trees.
// It records each (category, sub) pair's owning type name. If a pair appears under
// more than one type, it becomes permanently ambiguous (sticky): a later occurrence
// can neither resolve nor un-ambiguate it. Income blocks are not indexed.
func BuildTypeIndex(sheets []ExpenseType) TypeIndex {
	byCategorySubcat := make(map[string]string)
	ambiguous := make(map[string]bool)

	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				registerPair(byCategorySubcat, ambiguous, cat.Name, sub.Name, sheet.Name)
			}
		}
	}

	return TypeIndex{byCategorySubcat: byCategorySubcat, ambiguous: ambiguous}
}

// registerPair records typeName under the (category, sub) key. Finding the key already
// present means a second type claims it: the key is deleted and marked permanently
// ambiguous, which also stops a third occurrence from re-adding it.
func registerPair(byCategorySubcat map[string]string, ambiguous map[string]bool, category, sub, typeName string) {
	key := categorySubcategoryKey(category, sub)
	if ambiguous[key] {
		return
	}
	if _, exists := byCategorySubcat[key]; exists {
		delete(byCategorySubcat, key)
		ambiguous[key] = true
		return
	}
	byCategorySubcat[key] = typeName
}

// LookupType returns the expense type name that owns the given (category, subcategory)
// pair. It returns ErrTypeAmbiguous if the pair is claimed by multiple types, or
// ErrTypeNotFound if the pair is absent from the taxonomy.
func (idx TypeIndex) LookupType(category, subcategory string) (string, error) {
	key := categorySubcategoryKey(category, subcategory)
	if idx.ambiguous[key] {
		return "", ErrTypeAmbiguous
	}
	typeName, exists := idx.byCategorySubcat[key]
	if !exists {
		return "", ErrTypeNotFound
	}
	return typeName, nil
}

// categorySubcategoryKey builds the NFC-normalized composite key, joining the two
// segments with a null byte — a separator that cannot occur in human-typed names.
func categorySubcategoryKey(category, subcategory string) string {
	return strings.Join([]string{normalizeKey(category), normalizeKey(subcategory)}, "\x00")
}
