package collector

import (
	"fmt"
	"os"
	"sort"

	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/model"
)

// CollectContext reads README and doc files, truncating to fit within the budget.
func CollectContext(cfg config.ContextConfig) (model.ProjectContext, error) {
	ctx := model.ProjectContext{
		Docs: make(map[string]string),
	}

	if cfg.Readme {
		data, err := os.ReadFile("README.md")
		if err == nil {
			ctx.Readme = string(data)
		}
		// Not an error if README doesn't exist
	}

	for _, path := range cfg.Docs {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read doc %s: %v\n", path, err)
			continue
		}
		ctx.Docs[path] = string(data)
	}

	truncateContext(&ctx, cfg.MaxContextChars)
	return ctx, nil
}

// truncateContext ensures total context fits within maxChars.
// Truncates largest files first.
func truncateContext(ctx *model.ProjectContext, maxChars int) {
	if maxChars <= 0 {
		return
	}

	total := len(ctx.Readme)
	for _, content := range ctx.Docs {
		total += len(content)
	}

	if total <= maxChars {
		return
	}

	// Build list of all content entries sorted by size descending
	type entry struct {
		key    string // "readme" or doc path
		length int
	}
	var entries []entry
	if ctx.Readme != "" {
		entries = append(entries, entry{key: "readme", length: len(ctx.Readme)})
	}
	for path, content := range ctx.Docs {
		entries = append(entries, entry{key: path, length: len(content)})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].length > entries[j].length
	})

	// Truncate largest entries until within budget
	for total > maxChars && len(entries) > 0 {
		e := entries[0]
		excess := total - maxChars
		newLen := e.length - excess
		if newLen < 200 {
			newLen = 200 // keep at least 200 chars
		}

		if e.key == "readme" {
			if newLen < len(ctx.Readme) {
				ctx.Readme = ctx.Readme[:newLen] + "\n... (truncated)"
				total -= (e.length - newLen)
			}
		} else {
			content := ctx.Docs[e.key]
			if newLen < len(content) {
				ctx.Docs[e.key] = content[:newLen] + "\n... (truncated)"
				total -= (e.length - newLen)
			}
		}

		entries = entries[1:]
	}
}
