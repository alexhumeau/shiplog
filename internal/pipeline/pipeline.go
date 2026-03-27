package pipeline

import (
	"fmt"
	"os"

	"github.com/alexandrehumeau/shiplog/internal/analyzer"
	"github.com/alexandrehumeau/shiplog/internal/collector"
	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/model"
	"github.com/alexandrehumeau/shiplog/internal/screenshot"
	"github.com/alexandrehumeau/shiplog/internal/ui"
	"github.com/alexandrehumeau/shiplog/internal/writer"
)

// RunOptions holds CLI-specific options that override config.
type RunOptions struct {
	DryRun bool
	Since  string
	Last   int
	Output string // "table" or "json"
	Quiet  bool
}

// Run executes the full Collector → Analyzer → Writer pipeline.
func Run(cfg *config.Config, opts RunOptions) error {
	dryRun := cfg.DryRun || opts.DryRun
	jsonOutput := opts.Output == "json"
	quiet := opts.Quiet || jsonOutput

	// 1. Detect context
	branch, beforeSHA, afterSHA, err := collector.DetectContext(opts.Since, opts.Last)
	if err != nil {
		return fmt.Errorf("detecting context: %w", err)
	}

	// 2. Collect commits
	commits, err := collector.CollectCommits(beforeSHA, afterSHA)
	if err != nil {
		return fmt.Errorf("collecting commits: %w", err)
	}

	if len(commits) == 0 {
		if !quiet {
			fmt.Println("No new commits found.")
		}
		return nil
	}

	// Count total files
	fileCount := countFiles(commits)

	// Show header
	if !quiet {
		header := ui.RenderHeader("v1.0.0", branch,
			shortenSHA(beforeSHA)+".."+shortenSHA(afterSHA),
			len(commits), fileCount)
		fmt.Print(header)
		fmt.Println()
	}

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

	// 6. Analyze (with spinner if LLM active)
	var spinner *ui.Spinner
	if !quiet && cfg.AI.Provider != "" && cfg.AI.APIKey != "" {
		spinner = ui.StartSpinner(fmt.Sprintf("Analyzing with %s %s...", cfg.AI.Provider, cfg.AI.Model))
	}

	entries, err := analyzer.Analyze(pushData, cfg)

	if spinner != nil {
		spinner.Stop()
	}

	if err != nil {
		return fmt.Errorf("analyzing commits: %w", err)
	}

	if len(entries) == 0 {
		if !quiet {
			fmt.Println("No changelog entries after analysis.")
		}
		return nil
	}

	// 6b. Capture screenshots (if configured)
	screenshotCfg, err := screenshot.LoadConfig(".shiplog-screenshots.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load screenshot config: %v\n", err)
	}
	if screenshotCfg != nil && !dryRun {
		for i := range entries {
			buf, err := screenshot.CaptureForEntry(screenshotCfg, entries[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: screenshot capture failed: %v\n", err)
				continue
			}
			entries[i].Screenshot = buf
		}
	}

	// 7. Show results
	if !quiet {
		table := ui.RenderTable(entries)
		fmt.Print(table)
		fmt.Println()
	}

	// 8. Build writer list
	writers := buildWriters(cfg, dryRun, jsonOutput)

	// 9. Validate config for enabled writers
	if !dryRun {
		if err := cfg.Validate(); err != nil {
			return err
		}
	}

	// 10. Write
	results, err := writer.RunAll(writers, entries)

	// 11. Show footer
	if !quiet && !jsonOutput {
		footer := ui.RenderFooter(len(entries), results)
		fmt.Print(footer)
	}

	// 12. Save state for CLI mode
	if !dryRun && os.Getenv("GITHUB_ACTIONS") != "true" {
		if err := collector.SaveState(".shiplog-state.json", afterSHA); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save state: %v\n", err)
		}
	}

	return err
}

func buildWriters(cfg *config.Config, dryRun, jsonOutput bool) []writer.Writer {
	if jsonOutput {
		return []writer.Writer{&writer.JSONWriter{}}
	}

	if dryRun {
		return []writer.Writer{&writer.DryRunWriter{}}
	}

	var writers []writer.Writer
	for _, name := range cfg.Writers {
		switch name {
		case "notion":
			writers = append(writers, &writer.NotionWriter{
				Token:      cfg.Notion.Token,
				DatabaseID: cfg.Notion.DatabaseID,
				Props:      cfg.Properties,
			})
		case "slack":
			writers = append(writers, &writer.SlackWriter{
				WebhookURL: cfg.Slack.WebhookURL,
				NotionDBID: cfg.Notion.DatabaseID,
			})
		case "markdown":
			writers = append(writers, &writer.MarkdownWriter{
				Path: cfg.Markdown.Path,
			})
		}
	}

	return writers
}

func shortenSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func countFiles(commits []model.Commit) int {
	seen := make(map[string]bool)
	for _, c := range commits {
		for _, f := range c.Files {
			seen[f] = true
		}
	}
	return len(seen)
}
