package models

import (
	"errors"
	"testing"
)

func TestNewParseError(t *testing.T) {
	originalErr := errors.New("invalid format")
	err := NewParseError("missing semicolon", originalErr)

	if err.Category != ErrorCategoryParse {
		t.Errorf("Expected category %s, got %s", ErrorCategoryParse, err.Category)
	}
	if err.OutputFile != OutputFileFailed {
		t.Errorf("Expected output file %s, got %s", OutputFileFailed, err.OutputFile)
	}
	if err.GroupLabel != "Parse Errors" {
		t.Errorf("Expected group label 'Parse Errors', got %s", err.GroupLabel)
	}
	if !err.Retriable {
		t.Error("Expected parse error to be retriable")
	}
	if err.Error() != "failed to parse expense: missing semicolon" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
	if err.Unwrap() != originalErr {
		t.Error("Unwrap should return original error")
	}
}

func TestNewResolutionError(t *testing.T) {
	err := NewResolutionError("Streaming")

	if err.Category != ErrorCategoryResolution {
		t.Errorf("Expected category %s, got %s", ErrorCategoryResolution, err.Category)
	}
	if err.OutputFile != OutputFileFailed {
		t.Errorf("Expected output file %s, got %s", OutputFileFailed, err.OutputFile)
	}
	if err.GroupLabel != "Subcategory Not Found" {
		t.Errorf("Expected group label 'Subcategory Not Found', got %s", err.GroupLabel)
	}
	if !err.Retriable {
		t.Error("Expected resolution error to be retriable")
	}
	if err.Error() != "subcategory not found: Streaming" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
	if err.Unwrap() != nil {
		t.Error("Unwrap should return nil for resolution error")
	}
}

func TestNewAmbiguousError(t *testing.T) {
	err := NewAmbiguousError("Leandro", 2)

	if err.Category != ErrorCategoryAmbiguous {
		t.Errorf("Expected category %s, got %s", ErrorCategoryAmbiguous, err.Category)
	}
	if err.OutputFile != OutputFileAmbiguous {
		t.Errorf("Expected output file %s, got %s", OutputFileAmbiguous, err.OutputFile)
	}
	if err.GroupLabel != "Ambiguous Subcategories" {
		t.Errorf("Expected group label 'Ambiguous Subcategories', got %s", err.GroupLabel)
	}
	if !err.Retriable {
		t.Error("Expected ambiguous error to be retriable")
	}
	expectedMsg := "subcategory 'Leandro' is ambiguous, found in 2 sheets: please specify which one to use"
	if err.Error() != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestNewCapacityError(t *testing.T) {
	err := NewCapacityError("Uber/Taxi", "Variáveis", 0)

	if err.Category != ErrorCategoryCapacity {
		t.Errorf("Expected category %s, got %s", ErrorCategoryCapacity, err.Category)
	}
	if err.OutputFile != OutputFileFailed {
		t.Errorf("Expected output file %s, got %s", OutputFileFailed, err.OutputFile)
	}
	if err.GroupLabel != "Capacity Full" {
		t.Errorf("Expected group label 'Capacity Full', got %s", err.GroupLabel)
	}
	if err.Retriable {
		t.Error("Expected capacity error to be non-retriable")
	}
	expectedMsg := "subcategory 'Uber/Taxi' in sheet 'Variáveis' is full (0 rows available)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestNewIOError(t *testing.T) {
	originalErr := errors.New("permission denied")
	err := NewIOError("save workbook", originalErr)

	if err.Category != ErrorCategoryIO {
		t.Errorf("Expected category %s, got %s", ErrorCategoryIO, err.Category)
	}
	if err.OutputFile != OutputFileFailed {
		t.Errorf("Expected output file %s, got %s", OutputFileFailed, err.OutputFile)
	}
	if err.GroupLabel != "File I/O Errors" {
		t.Errorf("Expected group label 'File I/O Errors', got %s", err.GroupLabel)
	}
	if err.Retriable {
		t.Error("Expected IO error to be non-retriable")
	}
	if err.Error() != "failed to save workbook: permission denied" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
	if err.Unwrap() != originalErr {
		t.Error("Unwrap should return original error")
	}
}
