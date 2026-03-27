package writer

import (
	"fmt"
	"strings"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

// DryRunWriter prints changelog entries to stdout.
type DryRunWriter struct{}

func (w *DryRunWriter) Name() string { return "DryRun" }

func (w *DryRunWriter) Write(entries []model.ChangeEntry) error {
	if len(entries) == 0 {
		fmt.Println("No changelog entries to create.")
		return nil
	}

	fmt.Printf("=== Shiplog Dry Run — %d entries ===\n\n", len(entries))

	for i, entry := range entries {
		emoji := TypeEmoji(entry.Type)

		fmt.Printf("%s [%s] %s\n", emoji, entry.Type, entry.Title)
		if entry.Description != "" && entry.Description != entry.Title {
			fmt.Printf("  %s\n", entry.Description)
		}

		shas := make([]string, len(entry.Commits))
		for j, c := range entry.Commits {
			shas[j] = c.SHA[:8]
		}
		fmt.Printf("  Commits: %s\n", strings.Join(shas, ", "))

		if len(entry.Files) > 0 {
			fmt.Printf("  Files: %s\n", strings.Join(entry.Files, ", "))
		}

		fmt.Printf("  Branch: %s | Date: %s\n", entry.Branch, entry.Date.Format("2006-01-02"))

		if i < len(entries)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\n=== End Dry Run ===")
	return nil
}

// TypeEmoji returns the emoji for a change type.
func TypeEmoji(typ string) string {
	emojis := map[string]string{
		"feat":     "✨",
		"fix":      "🐛",
		"refactor": "♻️",
		"docs":     "📝",
		"chore":    "🔧",
		"perf":     "⚡",
		"unknown":  "❓",
	}
	if e, ok := emojis[typ]; ok {
		return e
	}
	return "📋"
}
