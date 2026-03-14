package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application-wide settings loaded from config/config.json.
type Config struct {
	WorkbookPath        string   `json:"workbook_path"`
	ReferenceSheet      string   `json:"reference_sheet"`
	DateYear            int      `json:"date_year"`
	Verbose             bool     `json:"verbose"`
	AutoInsertExcluded  []string `json:"auto_insert_excluded"`
	ClassificationsPath string   `json:"classifications_path"`
	ExpensesLogPath     string   `json:"expenses_log_path"`
}

// ExpensesLogFilePath returns the absolute path to expenses_log.jsonl.
// Same resolution logic as ClassificationsFilePath.
func (c *Config) ExpensesLogFilePath() string {
	if c.ExpensesLogPath == "" {
		return ""
	}
	if filepath.IsAbs(c.ExpensesLogPath) {
		return c.ExpensesLogPath
	}
	exe, err := os.Executable()
	if err != nil {
		return c.ExpensesLogPath
	}
	return filepath.Join(filepath.Dir(exe), c.ExpensesLogPath)
}

// ClassificationsFilePath returns the absolute path to classifications.jsonl.
// If ClassificationsPath is absolute, it is returned as-is.
// If relative, it is resolved relative to the running binary's directory.
func (c *Config) ClassificationsFilePath() string {
	if c.ClassificationsPath == "" {
		return ""
	}
	if filepath.IsAbs(c.ClassificationsPath) {
		return c.ClassificationsPath
	}
	exe, err := os.Executable()
	if err != nil {
		return c.ClassificationsPath
	}
	return filepath.Join(filepath.Dir(exe), c.ClassificationsPath)
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
