package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ExpenseEntry is one line in expenses_log.jsonl — a slim record of what was inserted.
type ExpenseEntry struct {
	ID          string  `json:"id"`
	Item        string  `json:"item"`
	Date        string  `json:"date"`
	Value       float64 `json:"value"`
	Subcategory string  `json:"subcategory"`
	Category    string  `json:"category"`
	Timestamp   string  `json:"timestamp"`
}

// NewExpenseEntry builds an ExpenseEntry using the shared GenerateID hash.
func NewExpenseEntry(item, date string, value float64, subcategory, category string) ExpenseEntry {
	return ExpenseEntry{
		ID:          GenerateID(item, date, value),
		Item:        item,
		Date:        date,
		Value:       value,
		Subcategory: subcategory,
		Category:    category,
		Timestamp:   Now().UTC().Format(time.RFC3339),
	}
}

// AppendExpense marshals entry as a single JSON line and appends it to path (creates if absent).
func AppendExpense(path string, entry ExpenseEntry) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening expense log file: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling expense entry: %w", err)
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing expense entry: %w", err)
	}
	return nil
}
