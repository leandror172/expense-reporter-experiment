package taxonomy

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadTypeDescriptions loads type descriptions from a JSON file.
// If path is empty, returns an empty map and nil error.
// If the file does not exist, returns an empty map and nil error (graceful no-sidecar case).
// On read or parse errors, returns nil map and wrapped error.
func LoadTypeDescriptions(path string) (map[string]string, error) {
	if path == "" {
		return make(map[string]string), nil
	}

	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("loading type descriptions: %w", err)
	}

	var byName map[string]string
	if err := json.Unmarshal(file, &byName); err != nil {
		return nil, fmt.Errorf("loading type descriptions: %w", err)
	}

	return byName, nil
}

// ApplyDescriptions mutates the ExpenseType.Description field in place using the provided map.
// It normalizes both keys and values with normalizeKey before matching.
func ApplyDescriptions(types []ExpenseType, byName map[string]string) {
	if len(byName) == 0 {
		return
	}

	// Build a normalized-key lookup map from byName
	normalizedByName := make(map[string]string)
	for k, v := range byName {
		normalizedKey := normalizeKey(k)
		normalizedByName[normalizedKey] = v
	}

	// Apply descriptions to each type in the slice
	for i := range types {
		typeName := normalizeKey(types[i].Name)
		if desc, ok := normalizedByName[typeName]; ok {
			types[i].Description = desc
		}
	}
}
