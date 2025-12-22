package excel

import (
	"testing"
	"time"
)

// TDD RED: Test month to column mapping logic
func TestGetMonthColumn(t *testing.T) {
	tests := []struct {
		name      string
		month     time.Month
		wantItem  string
		wantDate  string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "January",
			month:     time.January,
			wantItem:  "D",
			wantDate:  "E",
			wantValue: "F",
			wantErr:   false,
		},
		{
			name:      "February",
			month:     time.February,
			wantItem:  "G",
			wantDate:  "H",
			wantValue: "I",
			wantErr:   false,
		},
		{
			name:      "March",
			month:     time.March,
			wantItem:  "J",
			wantDate:  "K",
			wantValue: "L",
			wantErr:   false,
		},
		{
			name:      "April",
			month:     time.April,
			wantItem:  "M",
			wantDate:  "N",
			wantValue: "O",
			wantErr:   false,
		},
		{
			name:      "May",
			month:     time.May,
			wantItem:  "P",
			wantDate:  "Q",
			wantValue: "R",
			wantErr:   false,
		},
		{
			name:      "June",
			month:     time.June,
			wantItem:  "S",
			wantDate:  "T",
			wantValue: "U",
			wantErr:   false,
		},
		{
			name:      "July",
			month:     time.July,
			wantItem:  "V",
			wantDate:  "W",
			wantValue: "X",
			wantErr:   false,
		},
		{
			name:      "August",
			month:     time.August,
			wantItem:  "Y",
			wantDate:  "Z",
			wantValue: "AA",
			wantErr:   false,
		},
		{
			name:      "September",
			month:     time.September,
			wantItem:  "AB",
			wantDate:  "AC",
			wantValue: "AD",
			wantErr:   false,
		},
		{
			name:      "October",
			month:     time.October,
			wantItem:  "AE",
			wantDate:  "AF",
			wantValue: "AG",
			wantErr:   false,
		},
		{
			name:      "November",
			month:     time.November,
			wantItem:  "AH",
			wantDate:  "AI",
			wantValue: "AJ",
			wantErr:   false,
		},
		{
			name:      "December",
			month:     time.December,
			wantItem:  "AK",
			wantDate:  "AL",
			wantValue: "AM",
			wantErr:   false,
		},
		{
			name:    "Invalid month 0",
			month:   time.Month(0),
			wantErr: true,
		},
		{
			name:    "Invalid month 13",
			month:   time.Month(13),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			itemCol, dateCol, valueCol, err := GetMonthColumns(tt.month)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetMonthColumns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if itemCol != tt.wantItem {
				t.Errorf("GetMonthColumns() itemCol = %v, want %v", itemCol, tt.wantItem)
			}
			if dateCol != tt.wantDate {
				t.Errorf("GetMonthColumns() dateCol = %v, want %v", dateCol, tt.wantDate)
			}
			if valueCol != tt.wantValue {
				t.Errorf("GetMonthColumns() valueCol = %v, want %v", valueCol, tt.wantValue)
			}
		})
	}
}

// Test column arithmetic (incrementing column letters)
func TestIncrementColumn(t *testing.T) {
	tests := []struct {
		name   string
		column string
		n      int
		want   string
	}{
		{
			name:   "D + 1 = E",
			column: "D",
			n:      1,
			want:   "E",
		},
		{
			name:   "D + 2 = F",
			column: "D",
			n:      2,
			want:   "F",
		},
		{
			name:   "Z + 1 = AA",
			column: "Z",
			n:      1,
			want:   "AA",
		},
		{
			name:   "Y + 2 = AA",
			column: "Y",
			n:      2,
			want:   "AA",
		},
		{
			name:   "AA + 1 = AB",
			column: "AA",
			n:      1,
			want:   "AB",
		},
		{
			name:   "M + 0 = M",
			column: "M",
			n:      0,
			want:   "M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IncrementColumn(tt.column, tt.n)
			if got != tt.want {
				t.Errorf("IncrementColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}
