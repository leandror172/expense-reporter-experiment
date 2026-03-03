package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application-wide settings loaded from config/config.json.
type Config struct {
	WorkbookPath       string   `json:"workbook_path"`
	ReferenceSheet     string   `json:"reference_sheet"`
	DateYear           int      `json:"date_year"`
	Verbose            bool     `json:"verbose"`
	AutoInsertExcluded []string `json:"auto_insert_excluded"`
}

// Load reads config/config.json relative to the source tree root.
// Falls back to an empty Config (no exclusions) if the file is not found.
func Load() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// configPath returns the absolute path to config/config.json.
// Resolves relative to the running binary so it works regardless of working directory.
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config/config.json"
	}
	// exe is .../bin/expense-reporter (or wherever it was installed)
	// config.json is expected alongside the binary's parent: .../config/config.json
	root := filepath.Dir(exe)
	return filepath.Join(root, "config", "config.json")
}
