package utils

import (
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
