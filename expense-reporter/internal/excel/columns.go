package excel

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// Month column mapping - all sheets now start at D (after Fix #5)
// Janeiro=D, Fevereiro=G, Março=J, etc. (3-column spacing)
var monthColumns = map[time.Month]string{
	time.January:   "D",
	time.February:  "G",
	time.March:     "J",
	time.April:     "M",
	time.May:       "P",
	time.June:      "S",
	time.July:      "V",
	time.August:    "Y",
	time.September: "AB",
	time.October:   "AE",
	time.November:  "AH",
	time.December:  "AK",
}

// GetMonthColumns returns the Item, Date, and Value column letters for a given month
// All sheets use the same mapping after column standardization
func GetMonthColumns(month time.Month) (itemCol, dateCol, valueCol string, err error) {
	itemCol, exists := monthColumns[month]
	if !exists {
		return "", "", "", fmt.Errorf("invalid month: %d", month)
	}

	// Date and Value are always +1 and +2 from Item column
	dateCol = IncrementColumn(itemCol, 1)
	valueCol = IncrementColumn(itemCol, 2)

	return itemCol, dateCol, valueCol, nil
}

// IncrementColumn increments a column letter by n positions
// Examples: "D" + 1 = "E", "Z" + 1 = "AA", "AA" + 1 = "AB"
func IncrementColumn(col string, n int) string {
	// Convert column letter to number
	colNum, err := excelize.ColumnNameToNumber(col)
	if err != nil {
		return col // Return original if conversion fails
	}

	// Increment
	newColNum := colNum + n

	// Convert back to letter
	newCol, err := excelize.ColumnNumberToName(newColNum)
	if err != nil {
		return col // Return original if conversion fails
	}

	return newCol
}
