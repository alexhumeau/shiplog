package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

// JSONWriter outputs changelog entries as JSON to stdout.
type JSONWriter struct{}

func (w *JSONWriter) Name() string { return "JSON" }

func (w *JSONWriter) Write(entries []model.ChangeEntry) error {
	output := jsonOutput{
		Version: "1.0.0",
		Date:    time.Now().Format("2006-01-02"),
	}

	if len(entries) > 0 {
		output.Branch = entries[0].Branch
	}

	for _, e := range entries {
		shas := make([]string, len(e.Commits))
		for i, c := range e.Commits {
			shas[i] = c.SHA[:8]
		}
		output.Entries = append(output.Entries, jsonEntry{
			Title:       e.Title,
			Type:        e.Type,
			Description: e.Description,
			Commits:     shas,
			Files:       e.Files,
			Date:        e.Date.Format("2006-01-02"),
		})
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

type jsonOutput struct {
	Version string      `json:"version"`
	Branch  string      `json:"branch"`
	Date    string      `json:"date"`
	Entries []jsonEntry `json:"entries"`
}

type jsonEntry struct {
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Commits     []string `json:"commits"`
	Files       []string `json:"files"`
	Date        string   `json:"date"`
}
