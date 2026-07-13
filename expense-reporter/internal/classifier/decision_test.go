package classifier

import "testing"

func TestIsAutoInsertable(t *testing.T) {
	excl := []string{"Diversos"}
	tests := []struct {
		name        string
		subcategory string // result.Subcategory (the model's prediction)
		signal      MatchSignal
		excluded    []string
		want        bool
	}{
		{
			name:        "agreement at max specificity, not excluded",
			subcategory: "Diarista",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista"},
			excluded:    excl,
			want:        true,
		},
		{
			name:        "model disagrees with keyword top1",
			subcategory: "Uber/Taxi",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista"},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "specificity below max (0.7)",
			subcategory: "Diarista",
			signal:      MatchSignal{Matched: true, TopScore: 0.7, TopSubcategory: "Diarista"},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "specificity just below max (0.99)",
			subcategory: "Diarista",
			signal:      MatchSignal{Matched: true, TopScore: 0.99, TopSubcategory: "Diarista"},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "ambiguous top keyword (tie) never auto-inserts",
			subcategory: "Diarista",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista", Ambiguous: true},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "keyword miss (no match) routes to review",
			subcategory: "Diarista",
			signal:      MatchSignal{Matched: false},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "agreement but subcategory excluded",
			subcategory: "Diversos",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diversos"},
			excluded:    excl,
			want:        false,
		},
		{
			name:        "excluded list empty — Diversos allowed on agreement",
			subcategory: "Diversos",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diversos"},
			excluded:    []string{},
			want:        true,
		},
		{
			name:        "agreement with an unrelated exclusion present",
			subcategory: "Supermercado",
			signal:      MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Supermercado"},
			excluded:    excl,
			want:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{Subcategory: tt.subcategory}
			got := IsAutoInsertable(r, tt.signal, tt.excluded)
			if got != tt.want {
				t.Errorf("IsAutoInsertable(subcat=%q, signal=%+v, excluded=%v) = %v, want %v",
					tt.subcategory, tt.signal, tt.excluded, got, tt.want)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name        string
		subcategory string
		excluded    []string
		want        bool
	}{
		{"present", "Diversos", []string{"Diversos"}, true},
		{"absent", "Diarista", []string{"Diversos"}, false},
		{"empty list", "Diversos", []string{}, false},
		{"multiple, present", "Outros", []string{"Diversos", "Outros"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExcluded(tt.subcategory, tt.excluded); got != tt.want {
				t.Errorf("IsExcluded(%q, %v) = %v, want %v", tt.subcategory, tt.excluded, got, tt.want)
			}
		})
	}
}
