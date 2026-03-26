package analyzer

import (
	"regexp"
	"strings"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

var conventionalRe = regexp.MustCompile(`^(feat|fix|refactor|docs|chore|perf)(\(([^)]+)\))?!?:\s*(.+)`)

// ParsedCommit holds the result of parsing a conventional commit message.
type ParsedCommit struct {
	Type    string
	Scope   string
	Title   string
	Commit  model.Commit
	Matched bool
}

// ParseConventional parses conventional commit messages from commits.
// Non-matching commits get type "unknown".
func ParseConventional(commits []model.Commit) []ParsedCommit {
	var parsed []ParsedCommit
	for _, c := range commits {
		firstLine := strings.SplitN(c.Message, "\n", 2)[0]
		matches := conventionalRe.FindStringSubmatch(firstLine)

		if matches != nil {
			parsed = append(parsed, ParsedCommit{
				Type:    matches[1],
				Scope:   matches[3],
				Title:   matches[4],
				Commit:  c,
				Matched: true,
			})
		} else {
			parsed = append(parsed, ParsedCommit{
				Type:    "unknown",
				Title:   firstLine,
				Commit:  c,
				Matched: false,
			})
		}
	}
	return parsed
}

// ToChangeEntries converts parsed commits into ChangeEntry list (one per commit).
func ToChangeEntries(parsed []ParsedCommit, branch string) []model.ChangeEntry {
	var entries []model.ChangeEntry
	for _, p := range parsed {
		entries = append(entries, model.ChangeEntry{
			Title:       p.Title,
			Type:        p.Type,
			Description: p.Commit.Message,
			Commits:     []model.Commit{p.Commit},
			Files:       p.Commit.Files,
			Branch:      branch,
			Date:        p.Commit.Date,
		})
	}
	return entries
}
