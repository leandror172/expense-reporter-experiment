//go:build acceptance

package verify

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/harness"
)

// FeedbackFileExists asserts the artifact key maps to an existing file.
func FeedbackFileExists(artifactKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		path, ok := ctx.Artifacts[artifactKey]
		if !assert.True(ctx.T, ok, "artifact %q not registered", artifactKey) {
			return
		}
		_, err := os.Stat(path)
		assert.NoError(ctx.T, err, "feedback file %q should exist at %s", artifactKey, path)
	}
}

// FeedbackFileNotExists asserts the artifact key's file does NOT exist.
func FeedbackFileNotExists(artifactKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		path, ok := ctx.Artifacts[artifactKey]
		if !ok {
			// Artifact not registered means file was never created — pass
			return
		}
		_, err := os.Stat(path)
		assert.True(ctx.T, os.IsNotExist(err),
			"feedback file %q should NOT exist at %s", artifactKey, path)
	}
}

// FeedbackEntryCount asserts the JSONL artifact has exactly n lines.
func FeedbackEntryCount(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		lines := readFeedbackLines(ctx, artifactKey)
		if lines == nil {
			return
		}
		assert.Equal(ctx.T, n, len(lines),
			"%q should have exactly %d feedback entries, got %d", artifactKey, n, len(lines))
	}
}

// FeedbackContainsStatus asserts at least one entry has the given status value.
func FeedbackContainsStatus(artifactKey, status string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		lines := readFeedbackLines(ctx, artifactKey)
		if lines == nil {
			return
		}
		for _, line := range lines {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			if s, ok := entry["status"].(string); ok && s == status {
				return
			}
		}
		ctx.T.Errorf("%q: no entry with status=%q found\nstdout: %s\nstderr: %s",
			artifactKey, status, ctx.Stdout, ctx.Stderr)
	}
}

// FeedbackContainsItem asserts at least one entry has the given item value.
func FeedbackContainsItem(artifactKey, item string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		lines := readFeedbackLines(ctx, artifactKey)
		if lines == nil {
			return
		}
		for _, line := range lines {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}
			if v, ok := entry["item"].(string); ok && v == item {
				return
			}
		}
		ctx.T.Errorf("%q: no entry with item=%q found\nstdout: %s\nstderr: %s",
			artifactKey, item, ctx.Stdout, ctx.Stderr)
	}
}

// FeedbackAllConfirmed asserts every entry in the JSONL file has status="confirmed".
func FeedbackAllConfirmed(artifactKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		lines := readFeedbackLines(ctx, artifactKey)
		if lines == nil {
			return
		}
		for i, line := range lines {
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				ctx.T.Errorf("%q line %d: invalid JSON: %v", artifactKey, i+1, err)
				continue
			}
			s, _ := entry["status"].(string)
			assert.Equal(ctx.T, "confirmed", s,
				"%q line %d: expected status=confirmed", artifactKey, i+1)
		}
	}
}

// FeedbackMatchesExpected compares the actual JSONL artifact against an expected JSONL file.
// The expected file is a specification: only the fields present in each expected entry are checked.
// id and timestamp are always skipped (they are implementation details, not business data).
// Line count must match exactly.
func FeedbackMatchesExpected(artifactKey, expectedPath string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		expectedLines := readJSONLFile(ctx.T, expectedPath)
		if expectedLines == nil {
			return
		}
		actualLines := readFeedbackLines(ctx, artifactKey)
		if actualLines == nil {
			return
		}
		if !assert.Equal(ctx.T, len(expectedLines), len(actualLines),
			"%q: expected %d entries, got %d", artifactKey, len(expectedLines), len(actualLines)) {
			return
		}
		for i := range expectedLines {
			var exp, act map[string]interface{}
			if err := json.Unmarshal([]byte(expectedLines[i]), &exp); err != nil {
				ctx.T.Errorf("expected line %d: invalid JSON: %v", i+1, err)
				continue
			}
			if err := json.Unmarshal([]byte(actualLines[i]), &act); err != nil {
				ctx.T.Errorf("actual line %d: invalid JSON: %v", i+1, err)
				continue
			}
			for field, expVal := range exp {
				if field == "id" || field == "timestamp" {
					continue
				}
				actVal, ok := act[field]
				if !assert.True(ctx.T, ok, "line %d: field %q missing from actual entry", i+1, field) {
					continue
				}
				assert.Equal(ctx.T, expVal, actVal, "line %d field %q mismatch", i+1, field)
			}
		}
	}
}

// readJSONLFile reads a file and returns non-empty lines.
func readJSONLFile(t interface{ Helper(); Errorf(string, ...interface{}) }, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("opening expected file %s: %v", path, err)
		return nil
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// readFeedbackLines reads the JSONL artifact and returns non-empty lines.
// Returns nil if the artifact is not registered or the file can't be read.
func readFeedbackLines(ctx *harness.Context, key string) []string {
	ctx.T.Helper()
	path, ok := ctx.Artifacts[key]
	if !assert.True(ctx.T, ok, "artifact %q not registered in ctx.Artifacts", key) {
		return nil
	}
	f, err := os.Open(path)
	if !assert.NoError(ctx.T, err, "opening feedback file %q at %s", key, path) {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
