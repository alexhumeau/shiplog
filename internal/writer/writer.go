package writer

import (
	"fmt"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

// Writer is the interface for changelog destinations.
type Writer interface {
	Write(entries []model.ChangeEntry) error
	Name() string
}

// RunAll executes all writers, continuing on individual failures.
// Returns error only if ALL writers fail.
func RunAll(writers []Writer, entries []model.ChangeEntry) (map[string]string, error) {
	results := make(map[string]string)
	var lastErr error
	succeeded := 0

	for _, w := range writers {
		if err := w.Write(entries); err != nil {
			fmt.Printf("  ⚠ %s failed: %v\n", w.Name(), err)
			results[w.Name()] = fmt.Sprintf("failed: %v", err)
			lastErr = err
		} else {
			results[w.Name()] = "✓"
			succeeded++
		}
	}

	if succeeded == 0 && lastErr != nil {
		return results, fmt.Errorf("all writers failed, last error: %w", lastErr)
	}
	return results, nil
}
