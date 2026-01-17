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

func TestExpense_IsInstallment(t *testing.T) {
	regularExpense := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: nil,
	}

	installmentExpense := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   3,
			Current: 1,
		},
	}

	if regularExpense.IsInstallment() {
		t.Error("regular expense should not be installment")
	}

	if !installmentExpense.IsInstallment() {
		t.Error("installment expense should be installment")
	}
}

func TestExpense_FormattedItem(t *testing.T) {
	tests := []struct {
		name     string
		expense  *Expense
		expected string
	}{
		{
			name: "regular expense",
			expense: &Expense{
				Item:        "Compra",
				Installment: nil,
			},
			expected: "Compra",
		},
		{
			name: "unexpanded installment",
			expense: &Expense{
				Item: "Compra",
				Installment: &Installment{
					Total:   300.00,
					Count:   3,
					Current: 0,
				},
			},
			expected: "Compra",
		},
		{
			name: "first installment",
			expense: &Expense{
				Item: "Compra",
				Installment: &Installment{
					Total:   300.00,
					Count:   3,
					Current: 1,
				},
			},
			expected: "Compra (1/3)",
		},
		{
			name: "middle installment",
			expense: &Expense{
				Item: "Compra",
				Installment: &Installment{
					Total:   300.00,
					Count:   12,
					Current: 6,
				},
			},
			expected: "Compra (6/12)",
		},
		{
			name: "last installment",
			expense: &Expense{
				Item: "Compra",
				Installment: &Installment{
					Total:   300.00,
					Count:   3,
					Current: 3,
				},
			},
			expected: "Compra (3/3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.expense.FormattedItem()
			if got != tt.expected {
				t.Errorf("FormattedItem() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExpense_Validate_Installment(t *testing.T) {
	// Test valid installment
	validInstallment := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   3,
			Current: 1,
		},
	}
	if err := validInstallment.Validate(); err != nil {
		t.Errorf("valid installment should pass validation: %v", err)
	}

	// Test valid unexpanded installment
	unexpanded := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   3,
			Current: 0,
		},
	}
	if err := unexpanded.Validate(); err != nil {
		t.Errorf("unexpanded installment should pass validation: %v", err)
	}

	// Test invalid: zero count
	invalidCount := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   0,
			Current: 0,
		},
	}
	if err := invalidCount.Validate(); err == nil {
		t.Error("installment with zero count should fail validation")
	}

	// Test invalid: current out of range (too high)
	invalidCurrent := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   3,
			Current: 5,
		},
	}
	if err := invalidCurrent.Validate(); err == nil {
		t.Error("installment with current > count should fail validation")
	}

	// Test invalid: current out of range (negative)
	invalidNegative := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   300.00,
			Count:   3,
			Current: -1,
		},
	}
	if err := invalidNegative.Validate(); err == nil {
		t.Error("installment with negative current should fail validation")
	}

	// Test invalid: negative total
	invalidTotal := &Expense{
		Item:        "Test",
		Date:        time.Now(),
		Value:       100.00,
		Subcategory: "Test",
		Installment: &Installment{
			Total:   -300.00,
			Count:   3,
			Current: 1,
		},
	}
	if err := invalidTotal.Validate(); err == nil {
		t.Error("installment with negative total should fail validation")
	}
}
