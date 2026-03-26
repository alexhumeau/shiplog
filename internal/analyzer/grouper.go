package analyzer

import (
	"github.com/alexandrehumeau/shiplog/internal/model"
)

// GroupByScope groups change entries with the same type + scope.
// Entries with type "unknown" are never grouped.
func GroupByScope(entries []model.ChangeEntry, parsed []ParsedCommit) []model.ChangeEntry {
	// Build scope lookup from parsed commits
	scopeByTitle := make(map[string]string)
	for _, p := range parsed {
		if p.Scope != "" {
			scopeByTitle[p.Title] = p.Scope
		}
	}

	type groupKey struct {
		typ   string
		scope string
	}

	groups := make(map[groupKey]*model.ChangeEntry)
	var order []groupKey
	var ungrouped []model.ChangeEntry

	for _, e := range entries {
		if e.Type == "unknown" {
			ungrouped = append(ungrouped, e)
			continue
		}

		scope := scopeByTitle[e.Title]
		key := groupKey{typ: e.Type, scope: scope}

		if existing, ok := groups[key]; ok {
			existing.Commits = append(existing.Commits, e.Commits...)
			existing.Files = dedup(append(existing.Files, e.Files...))
			if e.Date.After(existing.Date) {
				existing.Date = e.Date
			}
		} else {
			entry := e
			groups[key] = &entry
			order = append(order, key)
		}
	}

	var result []model.ChangeEntry
	for _, key := range order {
		result = append(result, *groups[key])
	}
	result = append(result, ungrouped...)
	return result
}

func dedup(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
