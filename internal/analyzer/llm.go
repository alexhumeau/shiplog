package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/analyzer/provider"
	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/model"
	"github.com/alexandrehumeau/shiplog/internal/writer"
)

// LLMResponse is the expected JSON structure from the LLM.
type LLMResponse struct {
	Entries []LLMEntry `json:"entries"`
}

type LLMEntry struct {
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	CommitSHAs  []string `json:"commit_shas"`
}

// Analyze runs the full analysis pipeline: conventional commits + optional LLM.
func Analyze(data model.PushData, cfg *config.Config) ([]model.ChangeEntry, error) {
	// Always parse conventional commits
	parsed := ParseConventional(data.Commits)

	// Try LLM if configured
	if cfg.AI.Provider != "" && cfg.AI.APIKey != "" {
		entries, err := analyzeWithLLM(data, cfg, parsed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: LLM analysis failed, falling back to conventional commits: %v\n", err)
		} else {
			return filterIgnoredTypes(entries, cfg.Filters.IgnoreTypes), nil
		}
	}

	// Fallback: conventional commits + heuristic grouping
	entries := ToChangeEntries(parsed, data.Branch)
	entries = GroupByScope(entries, parsed)
	return filterIgnoredTypes(entries, cfg.Filters.IgnoreTypes), nil
}

func analyzeWithLLM(data model.PushData, cfg *config.Config, parsed []ParsedCommit) ([]model.ChangeEntry, error) {
	// Fetch Notion history for tone consistency
	var notionHistory string
	if cfg.Notion.Token != "" && cfg.Notion.DatabaseID != "" {
		entries, err := writer.QueryRecentEntries(cfg.Notion.Token, cfg.Notion.DatabaseID, 5)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not fetch Notion history: %v\n", err)
		} else {
			notionHistory = formatNotionHistory(entries)
		}
	}

	prompt := buildPrompt(data, cfg, notionHistory)

	p, err := provider.New(cfg.AI.Provider, cfg.AI.APIKey, cfg.AI.Model)
	if err != nil {
		return nil, err
	}

	response, err := p.Complete(prompt)
	if err != nil {
		return nil, err
	}

	return parseLLMResponse(response, data)
}

func buildPrompt(data model.PushData, cfg *config.Config, notionHistory string) string {
	var sb strings.Builder

	sb.WriteString("You are a changelog assistant analyzing git commits.\n\n")

	// Project context
	if data.Context.Readme != "" {
		sb.WriteString("Project README:\n")
		sb.WriteString(data.Context.Readme)
		sb.WriteString("\n\n")
	}
	for path, content := range data.Context.Docs {
		sb.WriteString(fmt.Sprintf("Doc (%s):\n%s\n\n", path, content))
	}

	// Notion history
	if notionHistory != "" {
		sb.WriteString("Recent changelog entries (for tone consistency):\n")
		sb.WriteString(notionHistory)
		sb.WriteString("\n\n")
	}

	// Commits with diffs
	sb.WriteString("Commits since last push:\n")
	for _, c := range data.Commits {
		sb.WriteString(fmt.Sprintf("- %s: %s (by %s)\n", c.SHA[:8], c.Message, c.Author))
	}
	sb.WriteString("\n")

	if len(data.Diffs) > 0 {
		sb.WriteString("File changes:\n")
		for _, d := range data.Diffs {
			if d.Patch != "" && !strings.HasPrefix(d.Patch, "+") {
				sb.WriteString(fmt.Sprintf("--- %s (+%d -%d) ---\n%s\n", d.Path, d.Additions, d.Deletions, d.Patch))
			} else {
				sb.WriteString(fmt.Sprintf("- %s (+%d -%d)\n", d.Path, d.Additions, d.Deletions))
			}
		}
		sb.WriteString("\n")
	}

	// Instructions
	sb.WriteString(fmt.Sprintf(`Tasks:
1. Group commits that relate to the same feature/fix
2. For each group, generate:
   - title: clear title in %s, user-oriented not technical
   - type: feat|fix|refactor|docs|chore|perf
   - description: 2-3 sentences explaining what changed and why
   - commit_shas: array of SHA strings included in this group
3. Respond ONLY with JSON (no markdown, no explanation): { "entries": [...] }
`, cfg.AI.Language))

	return sb.String()
}

func parseLLMResponse(response string, data model.PushData) ([]model.ChangeEntry, error) {
	// Strip markdown code blocks if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
			response = strings.Join(lines, "\n")
		}
	}

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(response), &llmResp); err != nil {
		return nil, fmt.Errorf("parsing LLM JSON response: %w (response: %s)", err, response[:min(200, len(response))])
	}

	// Build SHA → Commit lookup
	commitBySHA := make(map[string]model.Commit)
	for _, c := range data.Commits {
		commitBySHA[c.SHA] = c
		commitBySHA[c.SHA[:8]] = c // also match short SHAs
	}

	var entries []model.ChangeEntry
	now := time.Now()

	for _, e := range llmResp.Entries {
		entry := model.ChangeEntry{
			Title:       e.Title,
			Type:        e.Type,
			Description: e.Description,
			Branch:      data.Branch,
			Date:        now,
		}

		for _, sha := range e.CommitSHAs {
			if c, ok := commitBySHA[sha]; ok {
				entry.Commits = append(entry.Commits, c)
				entry.Files = append(entry.Files, c.Files...)
				if c.Date.After(entry.Date) || entry.Date.Equal(now) {
					entry.Date = c.Date
				}
			}
		}

		// Dedup files
		entry.Files = dedup(entry.Files)
		entries = append(entries, entry)
	}

	return entries, nil
}

func formatNotionHistory(entries []writer.NotionEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Type, e.Title))
	}
	return sb.String()
}

func filterIgnoredTypes(entries []model.ChangeEntry, ignoreTypes []string) []model.ChangeEntry {
	if len(ignoreTypes) == 0 {
		return entries
	}
	ignore := make(map[string]bool)
	for _, t := range ignoreTypes {
		ignore[t] = true
	}
	var result []model.ChangeEntry
	for _, e := range entries {
		if !ignore[e.Type] {
			result = append(result, e)
		}
	}
	return result
}

// dedup is defined in grouper.go
