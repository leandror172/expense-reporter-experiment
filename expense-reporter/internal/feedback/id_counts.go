package feedback

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadExpenseIDCounts reads a JSONL file and returns a map of expense IDs to their counts.
func LoadExpenseIDCounts(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]int), nil
		}
		return nil, fmt.Errorf("opening expense log file: %w", err)
	}
	defer f.Close()

	counts := make(map[string]int)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing expense log line: %w", err)
		}

		counts[entry.ID]++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading expense log file: %w", err)
	}

	return counts, nil
}
