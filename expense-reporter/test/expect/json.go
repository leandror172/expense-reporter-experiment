//go:build acceptance

package expect

import (
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// OutputJSONHasCategory asserts that the JSON output has the given category value.
func OutputJSONHasCategory(category string) func(*harness.Context) {
	return verify.OutputJSONHasValue("category", category)
}

// OutputJSONHasType asserts that the JSON output has the given top-level type value.
func OutputJSONHasType(typ string) func(*harness.Context) {
	return verify.OutputJSONHasValue("type", typ)
}

// OutputJSONHasAction asserts that the JSON output has the given action value.
func OutputJSONHasAction(action string) func(*harness.Context) {
	return verify.OutputJSONHasValue("action", action)
}
