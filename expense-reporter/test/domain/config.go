//go:build acceptance

package domain

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"

	"github.com/leandror172/acceptance-harness/harness"
)

// configStateKey names this package's slot in harness.Context.State.
const configStateKey = "domain.binaryConfig"

// SetupBinaryConfig contributes cfg to the config/config.json file alongside the binary.
// The binary resolves its config relative to os.Executable(), so this reaches it at test time.
//
// Keys ACCUMULATE across calls within a scenario and the file is written exactly once,
// from a ctx.BeforeWhen hook that runs after every Given event. That is what lets a Given
// be composed from several independent events: one event can configure the taxonomy path
// and another the feedback log paths without either erasing the other's keys. Later calls
// still win on a key they both set.
func SetupBinaryConfig(ctx *harness.Context, cfg map[string]interface{}) error {
	pending, ok := ctx.State[configStateKey].(map[string]interface{})
	if !ok {
		pending = map[string]interface{}{}
		ctx.State[configStateKey] = pending
		ctx.BeforeWhen(func() { writeBinaryConfig(ctx, pending) })
	}
	maps.Copy(pending, cfg)
	return nil
}

// writeBinaryConfig serializes the scenario's accumulated keys next to the binary and
// registers the cleanup. Runs once per scenario, after all Given events have contributed.
func writeBinaryConfig(ctx *harness.Context, cfg map[string]interface{}) {
	configDir := filepath.Join(filepath.Dir(ctx.BinaryPath), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: create config dir: %v", err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: marshal: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: write: %v", err)
	}
	ctx.T.Cleanup(func() { os.Remove(configPath) })
}
