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

		// ─── T-64: the multiplier notation ────────────────────────────────────────
		//
		// THE DISCRIMINATING PAIR. These two lines carry the same digits and mean
		// things four times apart, in opposite directions:
		//
		//   "405,25/4"   the written number is the TOTAL          → 4 × 101,3125
		//   "405,25 x4"  the written number is PER-INSTALLMENT    → 4 × 405,25
		//
		// They are deliberately adjacent. A future reader who conflates the two forms
		// has to edit two neighbouring lines that contradict each other, rather than
		// finding one of them alone and "fixing" it. `x4` is Brazilian retail usage
		// ("4x de R$ 405,25"); `total/N` is how the card statement prints it.
		{
			name:      "divisor form: the written number is the total",
			input:     "405,25/4",
			wantValue: 101.3125,
			wantCount: 4,
			wantErr:   false,
		},
		{
			name:      "multiplier form: the written number is per-installment",
			input:     "405,25 x4",
			wantValue: 405.25,
			wantCount: 4,
			wantErr:   false,
		},

		// The mode switch is the PRESENCE of x/X — not a positional grammar. Where the
		// value ends and the count begins is a separate question, answered by trying
		// three readings and requiring EXACTLY ONE to hold:
		//
		//   A  "value x count"   left is currency  AND right is a plausible count
		//   B  "value count x"   right is EMPTY, and left splits on whitespace into both
		//   C  "count x value"   left is a plausible count AND right is currency
		//
		// Two valid readings is an ambiguity, and an ambiguity is an ERROR — never a
		// guess. That asymmetry is the point of the whole task: rejecting costs the user
		// one edit in failed.csv, which T-63 built for exactly this, while guessing costs
		// a silently 4×-wrong budget row that reads as legitimate.
		{
			name:      "multiplier without a space",
			input:     "405,25x4",
			wantValue: 405.25,
			wantCount: 4,
			wantErr:   false,
		},
		{
			name:      "multiplier is case-insensitive",
			input:     "405,25 X4",
			wantValue: 405.25,
			wantCount: 4,
			wantErr:   false,
		},
		{
			name:      "count before x, reading B",
			input:     "646,25 4x",
			wantValue: 646.25,
			wantCount: 4,
			wantErr:   false,
		},
		{
			name:      "count before x, uppercase",
			input:     "646,25 4X",
			wantValue: 646.25,
			wantCount: 4,
			wantErr:   false,
		},
		{
			// Brazilian retail writes the count first ("4x de R$ 405,25"). Reading C
			// costs nothing to support: A cannot also hold here, because "405,25" is
			// not an integer, so exactly one reading survives.
			name:      "count first, reading C",
			input:     "4x405,25",
			wantValue: 405.25,
			wantCount: 4,
			wantErr:   false,
		},
		{
			// The token this function receives is already normalized — "1.234,56 x4"
			// has had its thousands dots stripped by internal/parse before arriving
			// (design Q3: the boundary owns ALL input normalization). This case pins
			// the POST-normalization shape; the dotted form is covered at the parse
			// layer and in the classified.csv → ReadQueue seam test.
			name:      "multiplier on a four-digit value",
			input:     "1234,56 x4",
			wantValue: 1234.56,
			wantCount: 4,
			wantErr:   false,
		},
		{
			name:      "multiplier of one",
			input:     "405,25 x1",
			wantValue: 405.25,
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiplier tolerates surrounding whitespace",
			input:     "  405,25 x4  ",
			wantValue: 405.25,
			wantCount: 4,
			wantErr:   false,
		},

		// ─── T-64 rejections ──────────────────────────────────────────────────────
		//
		// Every row below asserts WHY it failed, not merely that it failed. An error
		// for the right reason and an error for the wrong reason are indistinguishable
		// from a bare wantErr, so `646,254x` could start failing on a count bound
		// instead of on ambiguity and this table would stay green while the rule it
		// documents had quietly stopped holding.
		{
			// The ambiguous one, and the reason reading B requires whitespace: with
			// nothing marking where the value ends, "646,25"×4 and "646,2"×54 are
			// equally available and a regex would pick one by accident of greediness.
			name:        "no delimiter between value and count",
			input:       "646,254x",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			// Both A and C hold: "4 paid 5 times" and "5 paid 4 times" are both
			// readable. Writing the cents ("4x5,00") collapses it to C and parses.
			name:        "two bare integers are ambiguous",
			input:       "4x5",
			wantErr:     true,
			errContains: "ambiguous",
		},
		{
			name:        "two bare integers are ambiguous regardless of magnitude",
			input:       "2x10",
			wantErr:     true,
			errContains: "ambiguous",
		},
		{
			// Reading A matches the SHAPE here and fails only the count bound, which is
			// what lets the error name the real objection instead of degrading to a
			// generic parse failure.
			name:        "multiplier count of zero",
			input:       "405,25 x0",
			wantErr:     true,
			errContains: "must be positive",
		},
		{
			name:        "multiplier count above the cap",
			input:       "405,25 x61",
			wantErr:     true,
			errContains: "too large",
		},
		{
			name:        "multiplier count is not a number",
			input:       "405,25 xabc",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			// Both notations at once. The x branch is entered first (presence of x IS
			// the mode switch), and "405,25/4" is not currency, so no reading holds.
			name:        "both notations in one token",
			input:       "405,25/4 x2",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			name:        "multiplier with no value",
			input:       "x4",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			name:        "multiplier with no count",
			input:       "405,25 x",
			wantErr:     true,
			errContains: "invalid installment format",
		},
		{
			name:        "more than one multiplier marker",
			input:       "4x5x6",
			wantErr:     true,
			errContains: "invalid installment format",
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
