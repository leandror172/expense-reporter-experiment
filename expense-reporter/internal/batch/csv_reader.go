package batch

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CSVReader reads expense strings from a semicolon-delimited CSV file
type CSVReader struct {
	filePath string
}

// NewCSVReader creates a new CSV reader for the given file path
func NewCSVReader(filePath string) *CSVReader {
	return &CSVReader{filePath: filePath}
}

// Read reads all non-empty, non-comment lines from the CSV file
// Returns a slice of expense strings (one per line)
// Empty lines and lines starting with # are skipped
func (r *CSVReader) Read() ([]string, error) {
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Trim trailing whitespace (but preserve leading/internal whitespace)
		line = strings.TrimRight(line, " \t\r\n")

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comment lines (starting with #)
		if strings.HasPrefix(line, "#") {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV file: %w", err)
	}

	return lines, nil
}
