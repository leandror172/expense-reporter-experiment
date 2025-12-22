package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TDD RED: Test CLI argument parsing
func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
		wantInput   string
		wantErr     bool
	}{
		{
			name:        "valid add command",
			args:        []string{"add", "Test expense;15/04;35,50;Uber/Taxi"},
			wantCommand: "add",
			wantInput:   "Test expense;15/04;35,50;Uber/Taxi",
			wantErr:     false,
		},
		{
			name:        "help command",
			args:        []string{"help"},
			wantCommand: "help",
			wantInput:   "",
			wantErr:     false,
		},
		{
			name:        "version command",
			args:        []string{"version"},
			wantCommand: "version",
			wantInput:   "",
			wantErr:     false,
		},
		{
			name:        "no arguments",
			args:        []string{},
			wantCommand: "",
			wantInput:   "",
			wantErr:     true,
		},
		{
			name:        "add without expense string",
			args:        []string{"add"},
			wantCommand: "",
			wantInput:   "",
			wantErr:     true,
		},
		{
			name:        "unknown command",
			args:        []string{"unknown"},
			wantCommand: "",
			wantInput:   "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, input, err := ParseArgs(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if cmd != tt.wantCommand {
				t.Errorf("ParseArgs() command = %v, want %v", cmd, tt.wantCommand)
			}

			if input != tt.wantInput {
				t.Errorf("ParseArgs() input = %v, want %v", input, tt.wantInput)
			}
		})
	}
}

// Test interactive selection from ambiguous options
func TestPromptForSelection(t *testing.T) {
	tests := []struct {
		name         string
		options      []string
		userInput    string
		wantSelected int
		wantErr      bool
	}{
		{
			name:         "valid selection - first option",
			options:      []string{"Variáveis - Dentista", "Extras - Dentista"},
			userInput:    "1\n",
			wantSelected: 0,
			wantErr:      false,
		},
		{
			name:         "valid selection - second option",
			options:      []string{"Variáveis - Orion", "Extras - Orion"},
			userInput:    "2\n",
			wantSelected: 1,
			wantErr:      false,
		},
		{
			name:         "invalid selection - out of range",
			options:      []string{"Option 1", "Option 2"},
			userInput:    "3\n",
			wantSelected: -1,
			wantErr:      true,
		},
		{
			name:         "invalid selection - zero",
			options:      []string{"Option 1", "Option 2"},
			userInput:    "0\n",
			wantSelected: -1,
			wantErr:      true,
		},
		{
			name:         "invalid selection - non-numeric",
			options:      []string{"Option 1", "Option 2"},
			userInput:    "abc\n",
			wantSelected: -1,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock input reader
			input := strings.NewReader(tt.userInput)
			output := &bytes.Buffer{}

			selected, err := PromptForSelection(tt.options, input, output)

			if (err != nil) != tt.wantErr {
				t.Errorf("PromptForSelection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if selected != tt.wantSelected {
				t.Errorf("PromptForSelection() selected = %v, want %v", selected, tt.wantSelected)
			}

			// Verify output contains the options
			if !tt.wantErr {
				outputStr := output.String()
				for i, opt := range tt.options {
					expectedLine := strings.Contains(outputStr, opt)
					expectedNum := strings.Contains(outputStr, string(rune('1'+i)))
					if !expectedLine || !expectedNum {
						t.Errorf("PromptForSelection() output missing option %d: %s", i+1, opt)
					}
				}
			}
		})
	}
}

// Test help text output
func TestPrintHelp(t *testing.T) {
	output := &bytes.Buffer{}
	PrintHelp(output)

	helpText := output.String()

	// Verify help contains key information
	requiredStrings := []string{
		"expense-reporter",
		"add",
		"help",
		"version",
		"<item_description>;<DD/MM>;<value>;<sub_category>",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(helpText, required) {
			t.Errorf("PrintHelp() output missing required string: %s", required)
		}
	}
}

// Test version output
func TestPrintVersion(t *testing.T) {
	output := &bytes.Buffer{}
	PrintVersion(output)

	versionText := output.String()

	if versionText == "" {
		t.Error("PrintVersion() output is empty")
	}

	// Should contain version number
	if !strings.Contains(versionText, "1.0.0") && !strings.Contains(versionText, "version") {
		t.Errorf("PrintVersion() output doesn't look like version info: %s", versionText)
	}
}
