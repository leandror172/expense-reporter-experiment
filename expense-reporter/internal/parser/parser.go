package parser

import (
	"expense-reporter/internal/models"
	"expense-reporter/pkg/utils"
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

	// Validate item description
	if item == "" {
		return nil, fmt.Errorf("item description cannot be empty")
	}

	// Validate subcategory
	if subcategory == "" {
		return nil, fmt.Errorf("subcategory cannot be empty")
	}

	// Parse date
	date, err := utils.ParseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}

	// Parse value
	value, err := utils.ParseCurrency(valueStr)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	expense := &models.Expense{
		Item:        item,
		Date:        date,
		Value:       value,
		Subcategory: subcategory,
	}

	return expense, nil
}
