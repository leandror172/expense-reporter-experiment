//go:build acceptance

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/harness"
)

func HTMLFileEmbeddedJSON(artifactKey, scriptID string, target any) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		path, ok := ctx.Artifacts[artifactKey]
		if !assert.True(ctx.T, ok, "artifact %q not registered", artifactKey) {
			return
		}

		content, err := os.ReadFile(path)
		if !assert.NoError(ctx.T, err, "failed to read file %q at path %s", artifactKey, path) {
			return
		}

		re := regexp.MustCompile(fmt.Sprintf(`<script id="%s" type="application/json">(.+?)</script>`, regexp.QuoteMeta(scriptID)))
		matches := re.FindStringSubmatch(string(content))
		if !assert.Len(ctx.T, matches, 2, "no JSON found inside <script id=%q> in file %q", scriptID, artifactKey) {
			return
		}

		jsonContent := matches[1]
		err = json.Unmarshal([]byte(jsonContent), target)
		assert.NoError(ctx.T, err, "failed to unmarshal JSON from <script id=%q> in file %q: %s", scriptID, artifactKey, jsonContent)
	}
}

func HTMLFileContainsScript(artifactKey, scriptID string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		path, ok := ctx.Artifacts[artifactKey]
		if !assert.True(ctx.T, ok, "artifact %q not registered", artifactKey) {
			return
		}

		content, err := os.ReadFile(path)
		if !assert.NoError(ctx.T, err, "failed to read file %q at path %s", artifactKey, path) {
			return
		}

		assert.Contains(ctx.T, string(content), fmt.Sprintf(`<script id="%s"`, scriptID),
			"file %q does not contain <script id=%q> tag", artifactKey, scriptID)
	}
}
