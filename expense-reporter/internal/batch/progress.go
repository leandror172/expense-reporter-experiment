package batch

import (
	"github.com/schollz/progressbar/v3"
)

// ProgressReporter is the interface for reporting batch processing progress
type ProgressReporter interface {
	Update(current, total int)
	Finish()
}

// SilentProgress is a no-op implementation for silent operation
type SilentProgress struct{}

// NewSilentProgress creates a new silent progress reporter
func NewSilentProgress() *SilentProgress {
	return &SilentProgress{}
}

// Update does nothing in silent mode
func (p *SilentProgress) Update(current, total int) {
	// No-op
}

// Finish does nothing in silent mode
func (p *SilentProgress) Finish() {
	// No-op
}

// ConsoleProgress wraps progressbar for console output
type ConsoleProgress struct {
	bar *progressbar.ProgressBar
}

// NewConsoleProgress creates a new console progress bar
func NewConsoleProgress(total int) *ConsoleProgress {
	bar := progressbar.NewOptions(total,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription("[cyan]Processing expenses...[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	return &ConsoleProgress{bar: bar}
}

// Update updates the progress bar to the current position
func (p *ConsoleProgress) Update(current, total int) {
	// Set to current position (progressbar is 0-indexed, but we use 1-indexed)
	p.bar.Set(current)
}

// Finish completes the progress bar
func (p *ConsoleProgress) Finish() {
	p.bar.Finish()
}

// NewProgressReporter creates the appropriate progress reporter based on silent flag
func NewProgressReporter(total int, silent bool) ProgressReporter {
	if silent {
		return NewSilentProgress()
	}
	return NewConsoleProgress(total)
}
