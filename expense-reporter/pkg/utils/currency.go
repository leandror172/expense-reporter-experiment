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

// ParseCurrencyWithInstallments parses PT-BR currency with optional installment syntax
// Examples:
//   "100,00"    → (100.00, 1, nil)       // Regular value
//   "300,00/3"  → (100.00, 3, nil)       // 3 installments of 100 each
//   "300,00/0"  → (0, 0, error)          // Invalid: zero divisor
//   "300,00/abc"→ (0, 0, error)          // Invalid: non-numeric divisor
func ParseCurrencyWithInstallments(s string) (perInstallment float64, count int, err error) {
	// Trim whitespace
	s = strings.TrimSpace(s)

	// Check for "/" (installment syntax)
	if strings.Contains(s, "/") {
		parts := strings.Split(s, "/")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid installment format: %s", s)
		}

		// Parse total value (PT-BR: "300,00" → 300.00)
		total, err := ParseCurrency(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid total value in installment: %w", err)
		}

		// Parse installment count
		countStr := strings.TrimSpace(parts[1])
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid installment count '%s': must be a number", countStr)
		}

		// Validate count
		if count <= 0 {
			return 0, 0, fmt.Errorf("installment count must be positive, got %d", count)
		}
		if count > 60 {
			return 0, 0, fmt.Errorf("installment count too large: %d (max 60)", count)
		}

		// Calculate per-installment value
		perInstallment := total / float64(count)
		return perInstallment, count, nil
	}

	// Regular value (no installments)
	value, err := ParseCurrency(s)
	if err != nil {
		return 0, 0, err
	}
	return value, 1, nil
}
