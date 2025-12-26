package batch

import (
	"testing"
)

// TDD RED: Test progress reporting functionality

// Test SilentProgress does not panic (no-op implementation)
func TestSilentProgress(t *testing.T) {
	progress := NewSilentProgress()

	// Should not panic
	progress.Update(1, 10)
	progress.Update(5, 10)
	progress.Update(10, 10)
	progress.Finish()
}

// Test ConsoleProgress creation
func TestConsoleProgress_Create(t *testing.T) {
	// Should create without panicking
	progress := NewConsoleProgress(10)
	if progress == nil {
		t.Error("NewConsoleProgress returned nil")
	}
	progress.Finish()
}

// Test ConsoleProgress update sequence
func TestConsoleProgress_UpdateSequence(t *testing.T) {
	total := 5
	progress := NewConsoleProgress(total)

	// Update through the sequence - should not panic
	for i := 1; i <= total; i++ {
		progress.Update(i, total)
	}

	progress.Finish()
}

// Test ConsoleProgress handles zero total gracefully
func TestConsoleProgress_ZeroTotal(t *testing.T) {
	// Should handle edge case without panicking
	progress := NewConsoleProgress(0)
	progress.Update(0, 0)
	progress.Finish()
}

// Test NewProgressReporter factory function
func TestNewProgressReporter(t *testing.T) {
	tests := []struct {
		name   string
		silent bool
		total  int
	}{
		{
			name:   "silent mode - returns SilentProgress",
			silent: true,
			total:  10,
		},
		{
			name:   "console mode - returns ConsoleProgress",
			silent: false,
			total:  10,
		},
		{
			name:   "silent mode with zero total",
			silent: true,
			total:  0,
		},
		{
			name:   "console mode with zero total",
			silent: false,
			total:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := NewProgressReporter(tt.total, tt.silent)

			if progress == nil {
				t.Error("NewProgressReporter returned nil")
				return
			}

			// Should be able to call Update and Finish without panic
			progress.Update(1, tt.total)
			progress.Finish()
		})
	}
}

// Test that progress reporter can be used with processor callback
func TestProgressReporter_WithProcessor(t *testing.T) {
	expenseStrings := []string{
		"Uber Centro;15/04;35,50;Uber/Taxi",
		"Compras;03/01;150,00;Supermercado",
		"Pão;22/12;8,50;Padaria",
	}

	mockInsert := &mockInsertFunc{}
	processor := NewProcessor("test.xlsx")
	processor.SetMappings(createTestMappings())

	// Test with silent progress
	silentProgress := NewSilentProgress()
	_, err := processor.Process(expenseStrings, mockInsert.Insert, silentProgress.Update)
	if err != nil {
		t.Errorf("Processor with SilentProgress failed: %v", err)
	}
	silentProgress.Finish()

	// Reset mock
	mockInsert.calls = nil

	// Test with console progress
	consoleProgress := NewConsoleProgress(len(expenseStrings))
	_, err = processor.Process(expenseStrings, mockInsert.Insert, consoleProgress.Update)
	if err != nil {
		t.Errorf("Processor with ConsoleProgress failed: %v", err)
	}
	consoleProgress.Finish()
}
