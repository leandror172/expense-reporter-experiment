//go:build acceptance

package domain

import (
	"encoding/json"
	"os"
	"path/filepath"

	"expense-reporter/test/harness"
)

// SetupBinaryConfig writes cfg as JSON to the config/config.json file alongside the binary.
// The binary resolves its config relative to os.Executable(), so this reaches it at test time.
// Registers a t.Cleanup to remove the config file after the test runs.
func SetupBinaryConfig(ctx *harness.Context, cfg map[string]interface{}) error {
	configDir := filepath.Join(filepath.Dir(ctx.BinaryPath), "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return err
	}
	ctx.T.Cleanup(func() { os.Remove(configPath) })
	return nil
}
