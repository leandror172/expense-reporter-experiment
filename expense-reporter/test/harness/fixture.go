//go:build acceptance

package harness

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// FixtureConfig holds settings loaded from a fixture's config.json.
type FixtureConfig struct {
	Command       string   `json:"command"`        // e.g. "classify", "batch-auto"
	Model         string   `json:"model"`          // e.g. "my-classifier-q3"
	Threshold     float64  `json:"threshold"`      // default 0.85
	AssertionType string   `json:"assertion_type"` // "hard" or "soft"
	AccuracyFloor float64  `json:"accuracy_floor"` // default 0.0
	TopN          int      `json:"top_n"`          // default 3
	ExtraArgs     []string `json:"extra_args"`     // additional CLI flags
}

// LoadFixtureConfig reads config.json from dir and applies defaults.
func LoadFixtureConfig(dir string) (FixtureConfig, error) {
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return FixtureConfig{}, err
	}
	var cfg FixtureConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FixtureConfig{}, err
	}
	if cfg.Threshold == 0 {
		cfg.Threshold = 0.85
	}
	if cfg.TopN == 0 {
		cfg.TopN = 3
	}
	if cfg.AssertionType == "" {
		cfg.AssertionType = "hard"
	}
	return cfg, nil
}

// CopyFixtureToWorkDir copies all files from fixtureDir into ctx.WorkDir (shallow copy).
func CopyFixtureToWorkDir(ctx *Context, fixtureDir string) error {
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(fixtureDir, entry.Name())
		dst := filepath.Join(ctx.WorkDir, entry.Name())
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// DiscoverFixtures returns all subdirectories under baseDir that contain a config.json.
func DiscoverFixtures(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	var fixtures []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(baseDir, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, "config.json")); err == nil {
			fixtures = append(fixtures, dir)
		}
	}
	return fixtures, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
