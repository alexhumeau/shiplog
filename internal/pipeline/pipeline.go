package pipeline

import (
	"fmt"
	"os"

	"github.com/alexandrehumeau/shiplog/internal/analyzer"
	"github.com/alexandrehumeau/shiplog/internal/collector"
	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/model"
	"github.com/alexandrehumeau/shiplog/internal/writer"
)

// RunOptions holds CLI-specific options that override config.
type RunOptions struct {
	DryRun bool
	Since  string
	Last   int
}

// Run executes the full Collector → Analyzer → Writer pipeline.
func Run(cfg *config.Config, opts RunOptions) error {
	dryRun := cfg.DryRun || opts.DryRun

	// 1. Detect context
	branch, beforeSHA, afterSHA, err := collector.DetectContext(opts.Since, opts.Last)
	if err != nil {
		return fmt.Errorf("detecting context: %w", err)
	}

	fmt.Printf("Analyzing commits on %s (%s..%s)\n", branch, shortenSHA(beforeSHA), shortenSHA(afterSHA))

	// 2. Collect commits
	commits, err := collector.CollectCommits(beforeSHA, afterSHA)
	if err != nil {
		return fmt.Errorf("collecting commits: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println("No new commits found.")
		return nil
	}
	fmt.Printf("Found %d commits\n", len(commits))

	// 3. Collect diffs
	diffs, err := collector.CollectDiffs(commits, cfg.Context.MaxDiffLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not collect diffs: %v\n", err)
	}

	// 4. Collect project context
	ctx, err := collector.CollectContext(cfg.Context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not collect project context: %v\n", err)
	}

	// 5. Build PushData
	pushData := model.PushData{
		Commits:   commits,
		Diffs:     diffs,
		Context:   ctx,
		Branch:    branch,
		BeforeSHA: beforeSHA,
		AfterSHA:  afterSHA,
	}

	// 6. Analyze
	entries, err := analyzer.Analyze(pushData, cfg)
	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No changelog entries after analysis.")
		return nil
	}

	// 7. Write
	if dryRun {
		writer.DryRun(entries)
	} else {
		if err := cfg.Validate(); err != nil {
			return err
		}
		fmt.Printf("Writing %d entries to Notion...\n", len(entries))
		if err := writer.WriteAll(cfg.Notion.Token, cfg.Notion.DatabaseID, entries, cfg.Properties); err != nil {
			return fmt.Errorf("writing to Notion: %w", err)
		}

		// Save state for CLI mode
		if os.Getenv("GITHUB_ACTIONS") != "true" {
			if err := collector.SaveState(".shiplog-state.json", afterSHA); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save state: %v\n", err)
			}
		}
	}

	return nil
}

func shortenSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
