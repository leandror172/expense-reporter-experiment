package utils

import (
	"testing"
	"time"
)

// TDD RED: Write tests first, they will fail
func TestParseDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDay   int
		wantMonth time.Month
		wantYear  int
		wantErr   bool
	}{
		{
			name:      "valid date 15/04",
			input:     "15/04",
			wantDay:   15,
			wantMonth: time.April,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "valid date 01/01",
			input:     "01/01",
			wantDay:   1,
			wantMonth: time.January,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "valid date 31/12",
			input:     "31/12",
			wantDay:   31,
			wantMonth: time.December,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "single digit day",
			input:     "5/04",
			wantDay:   5,
			wantMonth: time.April,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:      "single digit month",
			input:     "15/4",
			wantDay:   15,
			wantMonth: time.April,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:    "invalid day 32",
			input:   "32/04",
			wantErr: true,
		},
		{
			name:    "invalid day 0",
			input:   "0/04",
			wantErr: true,
		},
		{
			name:    "invalid month 13",
			input:   "15/13",
			wantErr: true,
		},
		{
			name:    "invalid month 0",
			input:   "15/0",
			wantErr: true,
		},
		{
			name:    "invalid format no slash",
			input:   "1504",
			wantErr: true,
		},
		{
			name:    "invalid format multiple slashes",
			input:   "15/04/2025",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-numeric day",
			input:   "aa/04",
			wantErr: true,
		},
		{
			name:    "non-numeric month",
			input:   "15/bb",
			wantErr: true,
		},
		{
			name:      "february valid day",
			input:     "28/02",
			wantDay:   28,
			wantMonth: time.February,
			wantYear:  2025,
			wantErr:   false,
		},
		{
			name:    "february invalid day 30",
			input:   "30/02",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Day() != tt.wantDay {
					t.Errorf("ParseDate() day = %v, want %v", got.Day(), tt.wantDay)
				}
				if got.Month() != tt.wantMonth {
					t.Errorf("ParseDate() month = %v, want %v", got.Month(), tt.wantMonth)
				}
				if got.Year() != tt.wantYear {
					t.Errorf("ParseDate() year = %v, want %v", got.Year(), tt.wantYear)
				}
			}
		})
	}
}

func TestTimeToExcelDate(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
		want  float64
	}{
		{
			name:  "January 1, 2025",
			input: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  45658.0, // Excel serial number for 2025-01-01
		},
		{
			name:  "April 15, 2025",
			input: time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			want:  45762.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeToExcelDate(tt.input)
			if got != tt.want {
				t.Errorf("TimeToExcelDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
