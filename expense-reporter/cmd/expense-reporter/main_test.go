package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TDD RED: Test main CLI execution flow
func TestMainExecution(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name           string
		args           []string
		wantOutputContains string
		wantErr        bool
	}{
		{
			name:               "help command",
			args:               []string{"expense-reporter", "help"},
			wantOutputContains: "Usage:",
			wantErr:            false,
		},
		{
			name:               "version command",
			args:               []string{"expense-reporter", "version"},
			wantOutputContains: "version",
			wantErr:            false,
		},
		{
			name:               "no arguments shows help",
			args:               []string{"expense-reporter"},
			wantOutputContains: "Usage:",
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test args
			os.Args = tt.args

			// Capture output
			output := &bytes.Buffer{}

			// Run main logic (we'll implement runCLI function for testing)
			err := runCLI(output)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutputContains != "" {
				outputStr := output.String()
				if !strings.Contains(outputStr, tt.wantOutputContains) {
					t.Errorf("runCLI() output = %v, want to contain %v", outputStr, tt.wantOutputContains)
				}
			}
		})
	}
}

// Test workbook path resolution
func TestGetWorkbookPath(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantErr bool
	}{
		{
			name:    "valid path from environment",
			envVar:  "Z:\\Meu Drive\\controle\\code\\test.xlsx",
			wantErr: false,
		},
		{
			name:    "empty environment variable",
			envVar:  "",
			wantErr: false, // Should use default path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("EXPENSE_WORKBOOK_PATH", tt.envVar)
				defer os.Unsetenv("EXPENSE_WORKBOOK_PATH")
			}

			path, err := GetWorkbookPath()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkbookPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && path == "" {
				t.Error("GetWorkbookPath() returned empty path")
			}
		})
	}
}
