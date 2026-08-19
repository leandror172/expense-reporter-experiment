package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeywordHint(t *testing.T) {
	tests := []struct {
		name             string
		signal           MatchSignal
		modelSubcategory string
		want             string
	}{
		{
			name:             "an unambiguous maximum-specificity keyword that disagrees is offered as the hint",
			signal:           MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista"},
			modelSubcategory: "Uber/Taxi",
			want:             "Diarista",
		},
		{
			name:             "agreement offers nothing — there is no second opinion to show",
			signal:           MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista"},
			modelSubcategory: "Diarista",
			want:             "",
		},
		{
			name:             "a tie at the top is not an answer, so an ambiguous match stays silent",
			signal:           MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista", Ambiguous: true},
			modelSubcategory: "Uber/Taxi",
			want:             "",
		},
		{
			name:             "sub-maximal specificity stays silent even when clearly below the bar",
			signal:           MatchSignal{Matched: true, TopScore: 0.7, TopSubcategory: "Diarista"},
			modelSubcategory: "Uber/Taxi",
			want:             "",
		},
		{
			name:             "sub-maximal specificity stays silent just below the bar too",
			signal:           MatchSignal{Matched: true, TopScore: 0.99, TopSubcategory: "Diarista"},
			modelSubcategory: "Uber/Taxi",
			want:             "",
		},
		{
			name:             "a keyword miss has no opinion to offer",
			signal:           MatchSignal{},
			modelSubcategory: "Uber/Taxi",
			want:             "",
		},
		{
			name:             "an absent model answer is not agreement, so the hint is still offered",
			signal:           MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista"},
			modelSubcategory: "",
			want:             "Diarista",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KeywordHint(tt.signal, tt.modelSubcategory)
			assert.Equalf(t, tt.want, got, "KeywordHint(%+v, %q)", tt.signal, tt.modelSubcategory)
		})
	}
}
