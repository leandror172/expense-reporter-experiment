package utils

import (
	"strings"
	"testing"
)

// TDD RED: Write tests first, they will fail
func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "standard format 35,50",
			input:   "35,50",
			want:    35.50,
			wantErr: false,
		},
		{
			name:    "whole number 100,00",
			input:   "100,00",
			want:    100.00,
			wantErr: false,
		},
		{
			name:    "large amount 1234,56",
			input:   "1234,56",
			want:    1234.56,
			wantErr: false,
		},
		{
			name:    "single decimal 50,5",
			input:   "50,5",
			want:    50.5,
			wantErr: false,
		},
		{
			name:    "no decimals 100",
			input:   "100",
			want:    100.0,
			wantErr: false,
		},
		{
			name:    "zero value 0,00",
			input:   "0,00",
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "small value 0,01",
			input:   "0,01",
			want:    0.01,
			wantErr: false,
		},
		{
			name:    "negative value not allowed",
			input:   "-50,00",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid format letters",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "multiple commas",
			input:   "50,,00",
			want:    0,
			wantErr: true,
		},
		{
			name:    "dot instead of comma (should fail)",
			input:   "50.00",
			want:    50.00,
			wantErr: false, // Should still work, convert dot to comma
		},
		{
			name:    "mixed format with spaces",
			input:   " 35,50 ",
			want:    35.50,
			wantErr: false, // Should trim spaces
		},
		{
			name:    "very large amount",
			input:   "9999999,99",
			want:    9999999.99,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurrency(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCurrency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseCurrency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCurrencyWithInstallments(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   float64
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "regular value",
			input:     "100,00",
			wantValue: 100.00,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "3 installments",
			input:     "300,00/3",
			wantValue: 100.00,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "12 installments",
			input:     "1200,00/12",
			wantValue: 100.00,
			wantCount: 12,
			wantErr:   false,
		},
		{
			name:      "24 installments",
			input:     "2400,00/24",
			wantValue: 100.00,
			wantCount: 24,
			wantErr:   false,
		},
		{
			name:      "single installment treated as regular",
			input:     "100,00/1",
			wantValue: 100.00,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:        "zero divisor",
			input:       "300,00/0",
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:        "non-numeric count",
			input:       "300,00/abc",
			wantErr:     true,
			errContains: "must be a number",
		},
		{
			name:        "too many slashes",
			input:       "300,00/3/2",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			name:        "invalid total value",
			input:       "abc,00/3",
			wantErr:     true,
			errContains: "invalid total value",
		},
		{
			name:        "count too large",
			input:       "300,00/100",
			wantErr:     true,
			errContains: "too large",
		},
		{
			name:        "negative count",
			input:       "300,00/-3",
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:      "with spaces",
			input:     " 300,00 / 3 ",
			wantValue: 100.00,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "division with remainder",
			input:     "100,00/3",
			wantValue: 33.333333333333336,
			wantCount: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, count, err := ParseCurrencyWithInstallments(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if value != tt.wantValue {
				t.Errorf("value = %v, want %v", value, tt.wantValue)
			}
			if count != tt.wantCount {
				t.Errorf("count = %v, want %v", count, tt.wantCount)
			}
		})
	}
}
