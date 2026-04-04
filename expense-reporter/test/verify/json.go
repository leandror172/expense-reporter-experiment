//go:build acceptance

package verify

import (
	"encoding/json"
	"strings"

	"expense-reporter/test/harness"
)

// OutputIsValidJSON asserts that stdout contains valid JSON.
func OutputIsValidJSON() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		if ctx.Stdout == "" {
			ctx.T.Errorf("expected non-empty stdout, got empty string")
			return
		}

		var result interface{}
		if err := json.Unmarshal([]byte(ctx.Stdout), &result); err != nil {
			ctx.T.Errorf("stdout is not valid JSON: %v\nstdout: %s", err, ctx.Stdout)
		}
	}
}

// OutputJSONHasValue asserts that stdout JSON has the given value at the given top-level key.
func OutputJSONHasValue(key string, expected interface{}) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(ctx.Stdout), &result); err != nil {
			ctx.T.Errorf("stdout is not valid JSON: %v\nstdout: %s", err, ctx.Stdout)
			return
		}

		actual, exists := result[key]
		if !exists {
			ctx.T.Errorf("JSON missing key %q\nstdout: %s", key, ctx.Stdout)
			return
		}
		if actual != expected {
			ctx.T.Errorf("JSON key %q: expected %v, got %v\nstdout: %s", key, expected, actual, ctx.Stdout)
		}
	}
}

// OutputJSONHasCategory asserts that the JSON output has the given category value.
func OutputJSONHasCategory(category string) func(*harness.Context) {
	return OutputJSONHasValue("category", category)
}

// OutputJSONHasAction asserts that the JSON output has the given action value.
func OutputJSONHasAction(action string) func(*harness.Context) {
	return OutputJSONHasValue("action", action)
}

// OutputJSONHasKey asserts that stdout contains valid JSON with the given top-level key.
func OutputJSONHasKey(key string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(ctx.Stdout), &result); err != nil {
			ctx.T.Errorf("stdout is not valid JSON: %v\nstdout: %s", err, ctx.Stdout)
			return
		}

		if _, exists := result[key]; !exists {
			keys := make([]string, 0, len(result))
			for k := range result {
				keys = append(keys, k)
			}
			ctx.T.Errorf("JSON missing key %q. Available: %s\nstdout: %s",
				key, strings.Join(keys, ", "), ctx.Stdout)
		}
	}
}
