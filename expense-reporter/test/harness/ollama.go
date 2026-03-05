//go:build acceptance

package harness

import (
	"net/http"
	"testing"
	"time"
)

// RequireOllama checks that Ollama is reachable at the given URL.
// Calls t.Skipf if not reachable (3s timeout GET /api/tags returns non-200).
func RequireOllama(t *testing.T, url string) {
	t.Helper()
	if url == "" {
		url = "http://localhost:11434"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/api/tags")
	if err != nil {
		t.Skipf("Ollama not reachable at %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("Ollama not reachable at %s: status %d", url, resp.StatusCode)
	}
}
