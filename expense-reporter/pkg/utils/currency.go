package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParseCurrency parses a currency string in ##,## format and returns a float64
// Accepts both comma (,) and period (.) as decimal separator
func ParseCurrency(valueStr string) (float64, error) {
	if valueStr == "" {
		return 0, errors.New("value string cannot be empty")
	}

	// Trim spaces
	valueStr = strings.TrimSpace(valueStr)

	// Replace comma with period for standard float parsing
	// This handles both "50,00" and "50.00"
	normalized := strings.ReplaceAll(valueStr, ",", ".")

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value format: %s", valueStr)
	}

	if value < 0 {
		return 0, errors.New("negative values are not allowed")
	}

	return value, nil
}
