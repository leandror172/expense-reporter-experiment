package apply

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadReviewed reads and validates the reviewed JSON file at path.
func ReadReviewed(path string) (ReviewedFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return ReviewedFile{}, fmt.Errorf("reading reviewed file: %w", err)
	}
	defer file.Close()

	var reviewed ReviewedFile
	if err = json.NewDecoder(file).Decode(&reviewed); err != nil {
		return ReviewedFile{}, fmt.Errorf("reading reviewed file: %w", err)
	}

	if err = validateEntries(reviewed.Entries); err != nil {
		return ReviewedFile{}, err
	}

	return reviewed, nil
}

// validateEntries checks that each entry has a valid Action and a usable installment
// count.
func validateEntries(entries []ReviewedEntry) error {
	validActions := map[string]bool{
		ActionConfirmed: true,
		ActionCorrected: true,
		ActionSkipped:   true,
		ActionPending:   true,
	}

	for i, entry := range entries {
		if !validActions[entry.Action] {
			return fmt.Errorf("entry %d: unknown action %q", i, entry.Action)
		}
		// Deliberately not defaulted to 1. An absent key decodes to 0, so defaulting
		// would silently accept a producer that stopped emitting the field, and the
		// only symptom would be installment purchases quietly recorded as one row.
		if entry.Installments < 1 {
			return fmt.Errorf(
				"entry %d: installments is %d; it must be at least 1 (a purchase that is not split into installments is 1, not 0)",
				i, entry.Installments)
		}
	}

	return nil
}
