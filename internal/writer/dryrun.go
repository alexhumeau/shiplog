package writer

import (
	"fmt"
	"strings"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

var typeEmoji = map[string]string{
	"feat":     "✨",
	"fix":      "🐛",
	"refactor": "♻️",
	"docs":     "📝",
	"chore":    "🔧",
	"perf":     "⚡",
	"unknown":  "❓",
}

// DryRun prints changelog entries to stdout without writing to Notion.
func DryRun(entries []model.ChangeEntry) {
	if len(entries) == 0 {
		fmt.Println("No changelog entries to create.")
		return
	}

	fmt.Printf("=== Shiplog Dry Run — %d entries ===\n\n", len(entries))

	for i, entry := range entries {
		emoji := typeEmoji[entry.Type]
		if emoji == "" {
			emoji = "📋"
		}

		fmt.Printf("%s [%s] %s\n", emoji, entry.Type, entry.Title)
		if entry.Description != "" && entry.Description != entry.Title {
			fmt.Printf("  %s\n", entry.Description)
		}

		// Commits
		shas := make([]string, len(entry.Commits))
		for j, c := range entry.Commits {
			shas[j] = c.SHA[:8]
		}
		fmt.Printf("  Commits: %s\n", strings.Join(shas, ", "))

		// Files
		if len(entry.Files) > 0 {
			fmt.Printf("  Files: %s\n", strings.Join(entry.Files, ", "))
		}

		fmt.Printf("  Branch: %s | Date: %s\n", entry.Branch, entry.Date.Format("2006-01-02"))

		if i < len(entries)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\n=== End Dry Run ===")
}
