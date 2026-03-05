package classifier

import "testing"

func TestIsAutoInsertable(t *testing.T) {
	tests := []struct {
		name        string
		subcategory string
		confidence  float64
		threshold   float64
		excluded    []string
		want        bool
	}{
		{"high confidence, specific subcategory", "Uber/Taxi", 0.92, 0.85, []string{"Diversos"}, true},
		{"high confidence, Diarista", "Diarista", 0.95, 0.85, []string{"Diversos"}, true},
		{"exactly at threshold", "Supermercado", 0.85, 0.85, []string{"Diversos"}, true},
		{"high confidence, Diversos excluded", "Diversos", 0.95, 0.85, []string{"Diversos"}, false},
		{"high confidence, Diversos at 100%", "Diversos", 1.0, 0.85, []string{"Diversos"}, false},
		{"below threshold", "Uber/Taxi", 0.84, 0.85, []string{"Diversos"}, false},
		{"below threshold, Diversos", "Diversos", 0.60, 0.85, []string{"Diversos"}, false},
		{"zero confidence", "Diarista", 0.0, 0.85, []string{"Diversos"}, false},
		{"empty exclusion list — Diversos allowed", "Diversos", 0.95, 0.85, []string{}, true},
		{"custom threshold 0.70 passes at 0.75", "Uber/Taxi", 0.75, 0.70, []string{}, true},
		{"custom threshold 0.70 fails at 0.69", "Uber/Taxi", 0.69, 0.70, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{Subcategory: tt.subcategory, Confidence: tt.confidence}
			got := IsAutoInsertable(r, tt.threshold, tt.excluded)
			if got != tt.want {
				t.Errorf("IsAutoInsertable(%q, %.2f, threshold=%.2f, %v) = %v, want %v",
					tt.subcategory, tt.confidence, tt.threshold, tt.excluded, got, tt.want)
			}
		})
	}
}
