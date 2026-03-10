package parser

import (
	"expense-reporter/internal/models"
	"fmt"
	"strings"
)

// ParseExpenseString parses an expense string in the format:
// "item_description;DD/MM;##,##;subcategory"
func ParseExpenseString(input string) (*models.Expense, error) {
	if input == "" {
		return nil, fmt.Errorf("input string cannot be empty, expected 4 fields separated by semicolons")
	}

	parts := strings.Split(input, ";")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid format: expected 4 fields separated by semicolons, got %d fields", len(parts))
	}

	// Trim whitespace from each part
	item := strings.TrimSpace(parts[0])
	dateStr := strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	subcategory := strings.TrimSpace(parts[3])

	return models.NewExpense(item, subcategory, dateStr, valueStr)
}

// ParseExpense parses a classified expense
func ParseExpense(input models.ClassifiedExpense) (*models.Expense, error) {

	// Trim whitespace from each part
	item := strings.TrimSpace(input.Item)
	dateStr := strings.TrimSpace(input.Date)
	valueStr := strings.TrimSpace(input.RawValue)
	subcategory := strings.TrimSpace(input.Subcategory)

	return models.NewExpense(item, subcategory, dateStr, valueStr)
}
