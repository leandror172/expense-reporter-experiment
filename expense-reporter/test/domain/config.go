//go:build acceptance

package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/leandror172/acceptance-harness/harness"
)

// pendingConfig accumulates each scenario's config keys, keyed by its Context.
// The harness creates one Context per scenario, so entries never leak between
// scenarios; each is dropped in the scenario's own Cleanup.
var (
	pendingConfigMu sync.Mutex
	pendingConfig   = map[*harness.Context]map[string]interface{}{}
)

// SetupBinaryConfig contributes cfg to the config/config.json file alongside the binary.
// The binary resolves its config relative to os.Executable(), so this reaches it at test time.
//
// Keys MERGE across calls within a scenario, rather than the last call replacing the file.
// That is what lets a Given be composed from several independent events: one event can
// configure the taxonomy path and another the feedback log paths without either erasing
// the other's keys. Later calls still win on a key they both set.
//
// Registers a t.Cleanup to remove the config file and forget the accumulated keys.
func SetupBinaryConfig(ctx *harness.Context, cfg map[string]interface{}) error {
	merged := accumulateConfig(ctx, cfg)

	configDir := filepath.Join(filepath.Dir(ctx.BinaryPath), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return err
	}
	ctx.T.Cleanup(func() {
		forgetConfig(ctx)
		os.Remove(configPath)
	})
	return nil
}

// accumulateConfig folds cfg into this scenario's accumulated keys and returns a
// snapshot safe to serialize.
func accumulateConfig(ctx *harness.Context, cfg map[string]interface{}) map[string]interface{} {
	pendingConfigMu.Lock()
	defer pendingConfigMu.Unlock()

	merged, ok := pendingConfig[ctx]
	if !ok {
		merged = map[string]interface{}{}
		pendingConfig[ctx] = merged
	}
	for k, v := range cfg {
		merged[k] = v
	}

	snapshot := make(map[string]interface{}, len(merged))
	for k, v := range merged {
		snapshot[k] = v
	}
	return snapshot
}

// forgetConfig drops the scenario's accumulated keys once it finishes.
func forgetConfig(ctx *harness.Context) {
	pendingConfigMu.Lock()
	defer pendingConfigMu.Unlock()
	delete(pendingConfig, ctx)
}
