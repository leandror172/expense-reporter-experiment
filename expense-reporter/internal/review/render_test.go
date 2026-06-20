package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	t.Run("placeholder replaced", func(t *testing.T) {
		tmpl := `<html><body><script>var data = __REVIEW_DATA__;</script></body></html>`
		data := ReviewData{
			Source:      "test-source",
			GeneratedAt: "2023-01-01T00:00:00Z",
			Queue: []QueueEntry{
				{
					ID:           "1",
					Item:         "item1",
					Date:         "2023-01-01",
					RawValue:     "$100.00",
					Value:        100.0,
					Confidence:   0.95,
					AutoInserted: false,
					Predicted: Predicted{
						Type:        "sheet1",
						Category:    "category1",
						Subcategory: "subcategory1",
					},
				},
			},
			Taxonomy: Taxonomy{
				Types: []Type{
					{
						Name: "sheet1",
						Categories: []Category{
							{
								Name:          "category1",
								Subcategories: []string{"subcategory1", "subcategory2"},
							},
						},
					},
				},
			},
		}

		result, err := Render(tmpl, data)
		require.NoError(t, err)

		assert.NotContains(t, result, "__REVIEW_DATA__")
		assert.Contains(t, result, `"source":"test-source"`)

		// JSON key contract consumed by the review template's JS: the taxonomy
		// list must serialize as "types" and a predicted entry's type as "type"
		// (domain terms). The legacy "sheets"/"sheet" keys must NOT appear — the
		// template reads DATA.taxonomy.types and entry.predicted.type.
		assert.Contains(t, result, `"types":`)
		assert.Contains(t, result, `"type":"sheet1"`)
		assert.NotContains(t, result, `"sheets":`)
		assert.NotContains(t, result, `"sheet":`)
	})

	t.Run("missing placeholder", func(t *testing.T) {
		tmpl := `<html><body>No placeholder here</body></html>`
		result, err := Render(tmpl, ReviewData{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
		assert.Empty(t, result)
	})

	t.Run("multiple placeholders", func(t *testing.T) {
		tmpl := `<html><body><script>var data = __REVIEW_DATA__;</script><p>__REVIEW_DATA__</p></body></html>`
		result, err := Render(tmpl, ReviewData{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "occurrences")
		assert.Empty(t, result)
	})

	t.Run("script tag injection safe", func(t *testing.T) {
		tmpl := `<html><body><script>var data = __REVIEW_DATA__;</script></body></html>`
		data := ReviewData{
			Source: `</script><script>alert(1)</script>`,
		}

		result, err := Render(tmpl, data)
		require.NoError(t, err)
		assert.NotContains(t, result, `</script><script>`)
	})
}
