package writer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

// MarkdownWriter writes changelog entries to a CHANGELOG.md file.
type MarkdownWriter struct {
	Path string // default: CHANGELOG.md
}

func (w *MarkdownWriter) Name() string { return "Markdown" }

func (w *MarkdownWriter) Write(entries []model.ChangeEntry) error {
	path := w.Path
	if path == "" {
		path = "CHANGELOG.md"
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(existing)

	// Ensure # Changelog heading
	if !strings.Contains(content, "# Changelog") {
		content = "# Changelog\n\n" + content
	}

	// Build new section
	section := buildMarkdownSection(entries)

	// Insert after # Changelog heading
	parts := strings.SplitN(content, "# Changelog", 2)
	if len(parts) == 2 {
		content = parts[0] + "# Changelog\n\n" + section + strings.TrimLeft(parts[1], "\n")
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

func buildMarkdownSection(entries []model.ChangeEntry) string {
	var sb strings.Builder

	date := time.Now().Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("## %s\n\n", date))

	// Group by type
	typeOrder := []string{"feat", "fix", "refactor", "perf", "docs", "chore", "unknown"}
	typeHeadings := map[string]string{
		"feat":     "✨ Features",
		"fix":      "🐛 Bug Fixes",
		"refactor": "♻️ Refactors",
		"perf":     "⚡ Performance",
		"docs":     "📝 Documentation",
		"chore":    "🔧 Maintenance",
		"unknown":  "Other",
	}

	grouped := make(map[string][]model.ChangeEntry)
	for _, e := range entries {
		grouped[e.Type] = append(grouped[e.Type], e)
	}

	for _, typ := range typeOrder {
		group := grouped[typ]
		if len(group) == 0 {
			continue
		}

		heading := typeHeadings[typ]
		if heading == "" {
			heading = typ
		}
		sb.WriteString(fmt.Sprintf("### %s\n\n", heading))

		for _, e := range group {
			shas := make([]string, len(e.Commits))
			for i, c := range e.Commits {
				shas[i] = c.SHA[:8]
			}
			sb.WriteString(fmt.Sprintf("- **%s**", e.Title))
			if e.Description != "" && e.Description != e.Title {
				desc := e.Description
				if len(desc) > 200 {
					desc = desc[:197] + "..."
				}
				sb.WriteString(fmt.Sprintf(" — %s", desc))
			}
			sb.WriteString(fmt.Sprintf(" (%s)\n", strings.Join(shas, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
