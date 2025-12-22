package models

import (
	"testing"
	"time"
)

// TDD RED: Write tests first, they will fail
func TestExpenseValidation(t *testing.T) {
	tests := []struct {
		name    string
		expense Expense
		wantErr bool
	}{
		{
			name: "valid expense",
			expense: Expense{
				Item:        "Uber Centro",
				Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
				Value:       35.50,
				Subcategory: "Uber/Taxi",
			},
			wantErr: false,
		},
		{
			name: "empty item",
			expense: Expense{
				Item:        "",
				Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
				Value:       35.50,
				Subcategory: "Uber/Taxi",
			},
			wantErr: true,
		},
		{
			name: "zero date",
			expense: Expense{
				Item:        "Uber Centro",
				Date:        time.Time{},
				Value:       35.50,
				Subcategory: "Uber/Taxi",
			},
			wantErr: true,
		},
		{
			name: "negative value",
			expense: Expense{
				Item:        "Uber Centro",
				Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
				Value:       -35.50,
				Subcategory: "Uber/Taxi",
			},
			wantErr: true,
		},
		{
			name: "zero value allowed",
			expense: Expense{
				Item:        "Uber Centro",
				Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
				Value:       0.0,
				Subcategory: "Uber/Taxi",
			},
			wantErr: false,
		},
		{
			name: "empty subcategory",
			expense: Expense{
				Item:        "Uber Centro",
				Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
				Value:       35.50,
				Subcategory: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.expense.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Expense.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
