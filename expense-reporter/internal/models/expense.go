package models

import (
	"errors"
	"time"
)

// Expense represents a single expense entry
type Expense struct {
	Item        string
	Date        time.Time
	Value       float64
	Subcategory string
}

// Validate checks if the Expense has valid data
func (e *Expense) Validate() error {
	if e.Item == "" {
		return errors.New("item description cannot be empty")
	}

	if e.Date.IsZero() {
		return errors.New("date cannot be zero")
	}

	if e.Value < 0 {
		return errors.New("value cannot be negative")
	}

	if e.Subcategory == "" {
		return errors.New("subcategory cannot be empty")
	}

	return nil
}

// SheetLocation represents where an expense should be inserted in the Excel file
type SheetLocation struct {
	SheetName   string // e.g., "Variáveis", "Fixas", etc.
	Category    string // e.g., "Transporte", "Saúde", etc.
	SubcatRow   int    // Row number where the subcategory is located
	TargetRow   int    // Row number where the expense will be inserted
	MonthColumn string // Column letter for the month (e.g., "M" for April)
}
