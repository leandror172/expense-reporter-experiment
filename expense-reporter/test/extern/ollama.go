//go:build acceptance

// Package extern holds liveness gates for external services the acceptance suite
// depends on. It lives here rather than in the acceptance-harness module because
// that module's core is deliberately LLM-free and network-free — a generic CLI-test
// library must not assume an LLM. If a second LLM-backed consumer ever appears, this
// is the shape that would be promoted into an optional extern/llm package there.
package extern

import (
	"net/http"
	"testing"
	"time"
)

// RequireOllama checks that Ollama is reachable at the given URL.
// Calls t.Skipf if not reachable (3s timeout GET /api/tags returns non-200).
// Under `go test -short` it skips unconditionally, partitioning the suite into
// a deterministic group (runs everywhere, no Ollama) and an Ollama-gated group.
func RequireOllama(t *testing.T, url string) {
	t.Helper()
	if testing.Short() {
		t.Skip("Ollama-gated test skipped in -short mode")
	}
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
