package config

import (
	"path/filepath"
	"testing"
)

func TestClassificationsFilePath_Empty(t *testing.T) {
	c := &Config{}
	if got := c.ClassificationsFilePath(); got != "" {
		t.Errorf("ClassificationsFilePath() = %q, want empty string when path is unset", got)
	}
}

func TestClassificationsFilePath_Absolute(t *testing.T) {
	abs := "/tmp/classifications.jsonl"
	c := &Config{ClassificationsPath: abs}
	if got := c.ClassificationsFilePath(); got != abs {
		t.Errorf("ClassificationsFilePath() = %q, want %q", got, abs)
	}
}

func TestClassificationsFilePath_Relative(t *testing.T) {
	c := &Config{ClassificationsPath: "classifications.jsonl"}
	got := c.ClassificationsFilePath()
	// Must be absolute after resolution
	if !filepath.IsAbs(got) {
		t.Errorf("ClassificationsFilePath() = %q, want an absolute path for relative input", got)
	}
	// Must end with the filename
	if filepath.Base(got) != "classifications.jsonl" {
		t.Errorf("ClassificationsFilePath() base = %q, want classifications.jsonl", filepath.Base(got))
	}
}
