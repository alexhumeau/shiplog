package ui

import (
	"fmt"
	"strings"

	"github.com/alexandrehumeau/shiplog/internal/model"
	"github.com/charmbracelet/lipgloss"
)

var (
	subtle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	bold    = lipgloss.NewStyle().Bold(true)
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	blue    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	gray    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	orange  = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

var typeColors = map[string]lipgloss.Style{
	"feat":     green,
	"fix":      red,
	"refactor": blue,
	"docs":     yellow,
	"chore":    gray,
	"perf":     orange,
	"unknown":  gray,
}

var typeEmojis = map[string]string{
	"feat":     "✨",
	"fix":      "🐛",
	"refactor": "♻️",
	"docs":     "📝",
	"chore":    "🔧",
	"perf":     "⚡",
	"unknown":  "❓",
}

// RenderHeader prints the shiplog header with git context.
func RenderHeader(version, branch, commitRange string, commitCount, fileCount int) string {
	var sb strings.Builder
	sb.WriteString(bold.Render("🚢 Shiplog "+version) + "\n\n")
	sb.WriteString(fmt.Sprintf("  %s  %s\n", subtle.Render("Branch  "), branch))
	sb.WriteString(fmt.Sprintf("  %s  %s (%d commits, %d files)\n",
		subtle.Render("Range   "), commitRange, commitCount, fileCount))
	return sb.String()
}

// RenderTable renders changelog entries as a styled table.
func RenderTable(entries []model.ChangeEntry) string {
	if len(entries) == 0 {
		return subtle.Render("No entries.")
	}

	// Calculate column widths
	maxTypeWidth := 12
	maxContentWidth := 50

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1)

	var rows []string
	for _, e := range entries {
		emoji := typeEmojis[e.Type]
		if emoji == "" {
			emoji = "📋"
		}
		style := typeColors[e.Type]
		if style.GetForeground() == (lipgloss.NoColor{}) {
			style = gray
		}

		typeCell := style.Render(fmt.Sprintf("%s %s", emoji, e.Type))
		// Pad type cell
		typeCell = fmt.Sprintf("%-*s", maxTypeWidth, typeCell)

		// Content
		title := e.Title
		if len(title) > maxContentWidth {
			title = title[:maxContentWidth-3] + "..."
		}

		desc := ""
		if e.Description != "" && e.Description != e.Title {
			d := e.Description
			if len(d) > maxContentWidth {
				d = d[:maxContentWidth-3] + "..."
			}
			desc = "\n" + subtle.Render(d)
		}

		stats := subtle.Render(fmt.Sprintf("%d commits · %d files", len(e.Commits), len(e.Files)))

		content := bold.Render(title) + desc + "\n" + stats
		rows = append(rows, fmt.Sprintf("%s │ %s", typeCell, content))
	}

	table := strings.Join(rows, "\n"+subtle.Render(strings.Repeat("─", maxTypeWidth+maxContentWidth+5))+"\n")
	return border.Render(table) + "\n"
}

// RenderFooter renders the result summary.
func RenderFooter(entryCount int, results map[string]string) string {
	var sb strings.Builder
	sb.WriteString(success.Render(fmt.Sprintf("✓ %d entries written", entryCount)) + "\n")
	for name, status := range results {
		if status == "✓" {
			sb.WriteString(fmt.Sprintf("  → %s %s\n", subtle.Render(fmt.Sprintf("%-10s", name)), success.Render("✓")))
		} else {
			sb.WriteString(fmt.Sprintf("  → %s %s\n", subtle.Render(fmt.Sprintf("%-10s", name)), red.Render(status)))
		}
	}
	return sb.String()
}
