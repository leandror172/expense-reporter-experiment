package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

const placeholder = "__REVIEW_DATA__"

// Render injects ReviewData as JSON into the HTML template.
// The template must contain exactly one occurrence of __REVIEW_DATA__.
// json.Marshal escapes <, >, & by default — safe for inline <script> injection.
func Render(template string, data ReviewData) (string, error) {
	count := strings.Count(template, placeholder)
	if count == 0 {
		return "", fmt.Errorf("template missing %q placeholder", placeholder)
	}
	if count > 1 {
		return "", fmt.Errorf("template contains %d occurrences of %q; expected exactly 1", count, placeholder)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal review data: %w", err)
	}

	return strings.Replace(template, placeholder, string(jsonBytes), 1), nil
}
