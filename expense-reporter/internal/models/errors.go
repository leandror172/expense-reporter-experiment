package models

import "fmt"

// ErrorCategory represents the type of error that occurred
type ErrorCategory string

const (
	ErrorCategoryParse      ErrorCategory = "Parse"
	ErrorCategoryResolution ErrorCategory = "Resolution"
	ErrorCategoryCapacity   ErrorCategory = "Capacity"
	ErrorCategoryAmbiguous  ErrorCategory = "Ambiguous"
	ErrorCategoryIO         ErrorCategory = "IO"
)

// OutputFile represents where failed expenses should be written
type OutputFile string

const (
	OutputFileFailed    OutputFile = "failed"
	OutputFileAmbiguous OutputFile = "ambiguous"
)

// BatchError is a rich error type with metadata for categorization
type BatchError struct {
	Category      ErrorCategory // Error type category
	Message       string        // Human-readable error message
	OutputFile    OutputFile    // Which CSV file to write to
	GroupLabel    string        // Label for console grouping
	Retriable     bool          // Can user retry after fixing?
	OriginalError error         // Wrapped original error (if any)
}

// Error implements the error interface
func (e *BatchError) Error() string {
	return e.Message
}

// Unwrap allows error unwrapping for errors.Is/As
func (e *BatchError) Unwrap() error {
	return e.OriginalError
}

// NewParseError creates a parse error (failed CSV line parsing)
func NewParseError(message string, originalErr error) *BatchError {
	return &BatchError{
		Category:      ErrorCategoryParse,
		Message:       fmt.Sprintf("failed to parse expense: %s", message),
		OutputFile:    OutputFileFailed,
		GroupLabel:    "Parse Errors",
		Retriable:     true,
		OriginalError: originalErr,
	}
}

// NewResolutionError creates a resolution error (subcategory not found)
func NewResolutionError(subcategory string) *BatchError {
	return &BatchError{
		Category:      ErrorCategoryResolution,
		Message:       fmt.Sprintf("subcategory not found: %s", subcategory),
		OutputFile:    OutputFileFailed,
		GroupLabel:    "Subcategory Not Found",
		Retriable:     true,
		OriginalError: nil,
	}
}

// NewAmbiguousError creates an ambiguous error (multiple sheets)
func NewAmbiguousError(subcategory string, sheetCount int) *BatchError {
	return &BatchError{
		Category:      ErrorCategoryAmbiguous,
		Message:       fmt.Sprintf("subcategory '%s' is ambiguous, found in %d sheets: please specify which one to use", subcategory, sheetCount),
		OutputFile:    OutputFileAmbiguous,
		GroupLabel:    "Ambiguous Subcategories",
		Retriable:     true,
		OriginalError: nil,
	}
}

// NewCapacityError creates a capacity error (subcategory section full)
func NewCapacityError(subcategory, sheetName string, availableRows int) *BatchError {
	return &BatchError{
		Category:      ErrorCategoryCapacity,
		Message:       fmt.Sprintf("subcategory '%s' in sheet '%s' is full (%d rows available)", subcategory, sheetName, availableRows),
		OutputFile:    OutputFileFailed,
		GroupLabel:    "Capacity Full",
		Retriable:     false, // Requires manual sheet expansion
		OriginalError: nil,
	}
}

// NewIOError creates an IO error (file operations)
func NewIOError(operation string, originalErr error) *BatchError {
	return &BatchError{
		Category:      ErrorCategoryIO,
		Message:       fmt.Sprintf("failed to %s: %v", operation, originalErr),
		OutputFile:    OutputFileFailed,
		GroupLabel:    "File I/O Errors",
		Retriable:     false,
		OriginalError: originalErr,
	}
}
