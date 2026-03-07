package utils

import (
	"fmt"
	"strings"
)

// FormatBRValue formats a float64 as Brazilian decimal string with comma as separator.
func FormatBRValue(v float64) string {
	formatted := fmt.Sprintf("%.2f", v)
	return strings.Replace(formatted, ".", ",", 1)
}

// BuildInsertString returns a semicolon-delimited string with the given parameters.
// The value is formatted using FormatBRValue.
func BuildInsertString(item, date string, value float64, subcategory string) string {
	formattedValue := FormatBRValue(value)
	return fmt.Sprintf("%s;%s;%s;%s", item, date, formattedValue, subcategory)
}
