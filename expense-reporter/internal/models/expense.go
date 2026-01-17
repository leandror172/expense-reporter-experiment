package models

import (
	"errors"
	"fmt"
	"time"
)

// Installment represents installment payment information
type Installment struct {
	Total   float64 // Original total amount (e.g., 300.00)
	Count   int     // Number of installments (e.g., 3)
	Current int     // Current installment number (1-based, 0 = unexpanded)
}

// Expense represents a single expense entry
type Expense struct {
	Item        string
	Date        time.Time
	Value       float64
	Subcategory string
	Installment *Installment // nil = regular expense, non-nil = installment
}

// IsInstallment returns true if this expense is an installment
func (e *Expense) IsInstallment() bool {
	return e.Installment != nil
}

// FormattedItem returns item description with installment info if applicable
func (e *Expense) FormattedItem() string {
	if e.Installment != nil && e.Installment.Current > 0 {
		return fmt.Sprintf("%s (%d/%d)", e.Item, e.Installment.Current, e.Installment.Count)
	}
	return e.Item
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

	// Validate installment data
	if e.Installment != nil {
		if e.Installment.Count <= 0 {
			return errors.New("installment count must be positive")
		}
		if e.Installment.Current < 0 || e.Installment.Current > e.Installment.Count {
			return fmt.Errorf("installment current (%d) out of range [0, %d]",
				e.Installment.Current, e.Installment.Count)
		}
		if e.Installment.Total < 0 {
			return errors.New("installment total cannot be negative")
		}
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
