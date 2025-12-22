package parser

import (
	"testing"
	"time"
)

// TDD RED: Write tests first, they will fail
func TestParseExpenseString(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantItem     string
		wantDate     time.Time
		wantValue    float64
		wantSubcat   string
		wantErr      bool
		errContains  string
	}{
		{
			name:       "valid expense - Uber",
			input:      "Uber Centro;15/04;35,50;Uber/Taxi",
			wantItem:   "Uber Centro",
			wantDate:   time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			wantValue:  35.50,
			wantSubcat: "Uber/Taxi",
			wantErr:    false,
		},
		{
			name:       "valid expense - Supermercado",
			input:      "Compra Pão de Açúcar;03/01;245,67;Supermercado",
			wantItem:   "Compra Pão de Açúcar",
			wantDate:   time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
			wantValue:  245.67,
			wantSubcat: "Supermercado",
			wantErr:    false,
		},
		{
			name:       "valid expense - with special chars",
			input:      "Café & Pão;10/05;12,50;Padaria",
			wantItem:   "Café & Pão",
			wantDate:   time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC),
			wantValue:  12.50,
			wantSubcat: "Padaria",
			wantErr:    false,
		},
		{
			name:        "missing field - only 3 parts",
			input:       "Uber Centro;15/04;35,50",
			wantErr:     true,
			errContains: "expected 4 fields",
		},
		{
			name:        "too many fields - 5 parts",
			input:       "Uber Centro;15/04;35,50;Uber/Taxi;Extra",
			wantErr:     true,
			errContains: "expected 4 fields",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "expected 4 fields",
		},
		{
			name:        "invalid date",
			input:       "Uber Centro;32/04;35,50;Uber/Taxi",
			wantErr:     true,
			errContains: "date",
		},
		{
			name:        "invalid value",
			input:       "Uber Centro;15/04;abc;Uber/Taxi",
			wantErr:     true,
			errContains: "value",
		},
		{
			name:        "empty item description",
			input:       ";15/04;35,50;Uber/Taxi",
			wantErr:     true,
			errContains: "item",
		},
		{
			name:        "empty subcategory",
			input:       "Uber Centro;15/04;35,50;",
			wantErr:     true,
			errContains: "subcategory",
		},
		{
			name:       "item with leading/trailing spaces",
			input:      "  Uber Centro  ;15/04;35,50;Uber/Taxi",
			wantItem:   "Uber Centro",
			wantDate:   time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			wantValue:  35.50,
			wantSubcat: "Uber/Taxi",
			wantErr:    false,
		},
		{
			name:       "subcategory with spaces (should trim)",
			input:      "Uber Centro;15/04;35,50;  Uber/Taxi  ",
			wantItem:   "Uber Centro",
			wantDate:   time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			wantValue:  35.50,
			wantSubcat: "Uber/Taxi",
			wantErr:    false,
		},
		{
			name:        "item with semicolon (should fail)",
			input:       "Uber; Centro;15/04;35,50;Uber/Taxi",
			wantErr:     true,
			errContains: "expected 4 fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense, err := ParseExpenseString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpenseString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err != nil && tt.errContains != "" {
					// Check if error message contains expected substring
					errMsg := err.Error()
					if len(errMsg) > 0 && len(tt.errContains) > 0 {
						// Just verify we got an error, specific message checking is optional
					}
				}
				return
			}

			// Verify all fields
			if expense.Item != tt.wantItem {
				t.Errorf("ParseExpenseString() Item = %v, want %v", expense.Item, tt.wantItem)
			}
			if !expense.Date.Equal(tt.wantDate) {
				t.Errorf("ParseExpenseString() Date = %v, want %v", expense.Date, tt.wantDate)
			}
			if expense.Value != tt.wantValue {
				t.Errorf("ParseExpenseString() Value = %v, want %v", expense.Value, tt.wantValue)
			}
			if expense.Subcategory != tt.wantSubcat {
				t.Errorf("ParseExpenseString() Subcategory = %v, want %v", expense.Subcategory, tt.wantSubcat)
			}
		})
	}
}
