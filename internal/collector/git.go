package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

const maxPatchLines = 20

// GitHubPushEvent is the relevant subset of the GitHub push event payload.
type GitHubPushEvent struct {
	Before string `json:"before"`
	After  string `json:"after"`
	Ref    string `json:"ref"`
}

// DetectContext determines the commit range and branch based on the environment.
// In GitHub Actions, it reads the push event. In CLI mode, it uses state or flags.
func DetectContext(sinceSHA string, lastN int) (branch, beforeSHA, afterSHA string, err error) {
	branch, err = gitCurrentBranch()
	if err != nil {
		return "", "", "", fmt.Errorf("detecting branch: %w", err)
	}

	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return detectGitHubActions(branch)
	}

	// CLI mode
	afterSHA, err = gitOutput("rev-parse", "HEAD")
	if err != nil {
		return "", "", "", fmt.Errorf("getting HEAD: %w", err)
	}

	if sinceSHA != "" {
		return branch, sinceSHA, afterSHA, nil
	}

	if lastN > 0 {
		// Use empty beforeSHA to signal "last N" mode
		return branch, fmt.Sprintf("~%d", lastN), afterSHA, nil
	}

	// Try loading state
	state, err := LoadState(".shiplog-state.json")
	if err != nil {
		return "", "", "", fmt.Errorf("loading state: %w", err)
	}
	if state != nil {
		return branch, state.LastSHA, afterSHA, nil
	}

	return "", "", "", fmt.Errorf("no previous state found. Use --since <sha> or --last <n> for first run")
}

func detectGitHubActions(branch string) (string, string, string, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return "", "", "", fmt.Errorf("GITHUB_EVENT_PATH not set")
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return "", "", "", fmt.Errorf("reading event file: %w", err)
	}

	var event GitHubPushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return "", "", "", fmt.Errorf("parsing event: %w", err)
	}

	// Extract branch from ref (refs/heads/main → main)
	if strings.HasPrefix(event.Ref, "refs/heads/") {
		branch = strings.TrimPrefix(event.Ref, "refs/heads/")
	}

	return branch, event.Before, event.After, nil
}

// CollectCommits gathers commits in the given range.
// beforeSHA can be a real SHA, "~N" for last N commits, or an all-zeros SHA (force push).
func CollectCommits(beforeSHA, afterSHA string) ([]model.Commit, error) {
	if err := checkShallowClone(); err != nil {
		return nil, err
	}

	var args []string
	if strings.HasPrefix(beforeSHA, "~") {
		n, _ := strconv.Atoi(beforeSHA[1:])
		args = []string{"log", fmt.Sprintf("--max-count=%d", n), afterSHA,
			"--format=%H\x1f%s\x1f%an\x1f%aI", "--reverse"}
	} else if isNullSHA(beforeSHA) {
		// Force push or initial push — fallback to last 20
		args = []string{"log", "--max-count=20", afterSHA,
			"--format=%H\x1f%s\x1f%an\x1f%aI", "--reverse"}
	} else {
		// Verify beforeSHA exists
		if _, err := gitOutput("cat-file", "-t", beforeSHA); err != nil {
			// Force push — before SHA doesn't exist
			fmt.Fprintf(os.Stderr, "warning: before SHA %s not found (force push?), using last 20 commits\n", beforeSHA)
			args = []string{"log", "--max-count=20", afterSHA,
				"--format=%H\x1f%s\x1f%an\x1f%aI", "--reverse"}
		} else {
			args = []string{"log", fmt.Sprintf("%s..%s", beforeSHA, afterSHA),
				"--format=%H\x1f%s\x1f%an\x1f%aI", "--reverse"}
		}
	}

	output, err := gitOutput(args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	if output == "" {
		return nil, nil
	}

	var commits []model.Commit
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "\x1f", 4)
		if len(parts) != 4 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[3])
		files, _ := gitCommitFiles(parts[0])

		commits = append(commits, model.Commit{
			SHA:     parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    date,
			Files:   files,
		})
	}

	return commits, nil
}

// CollectDiffs gathers file diffs for the given commits, respecting maxLines budget.
func CollectDiffs(commits []model.Commit, maxLines int) ([]model.FileDiff, error) {
	if len(commits) == 0 {
		return nil, nil
	}

	firstSHA := commits[0].SHA
	lastSHA := commits[len(commits)-1].SHA

	// Get diffstat
	statOutput, err := gitOutput("diff", "--numstat", firstSHA+"~1", lastSHA)
	if err != nil {
		// Fallback: first commit might not have a parent
		statOutput, err = gitOutput("diff", "--numstat", firstSHA, lastSHA)
		if err != nil {
			return nil, fmt.Errorf("git diff --numstat: %w", err)
		}
	}

	if statOutput == "" {
		return nil, nil
	}

	var diffs []model.FileDiff
	totalLines := 0

	for _, line := range strings.Split(statOutput, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		adds, _ := strconv.Atoi(parts[0])
		dels, _ := strconv.Atoi(parts[1])
		path := parts[2]

		diff := model.FileDiff{
			Path:      path,
			Additions: adds,
			Deletions: dels,
		}

		// Get truncated patch if within budget
		if totalLines < maxLines {
			patch, _ := gitFilePatch(firstSHA, lastSHA, path)
			lines := strings.Split(patch, "\n")
			if len(lines) > maxPatchLines {
				lines = lines[:maxPatchLines]
				lines = append(lines, "... (truncated)")
			}
			diff.Patch = strings.Join(lines, "\n")
			totalLines += len(lines)
		} else {
			diff.Patch = fmt.Sprintf("+%d -%d %s", adds, dels, path)
		}

		diffs = append(diffs, diff)
	}

	return diffs, nil
}

func checkShallowClone() error {
	output, err := gitOutput("rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil // can't detect, proceed anyway
	}
	if strings.TrimSpace(output) == "true" {
		return fmt.Errorf("shallow clone detected. Use `fetch-depth: 0` in your checkout step")
	}
	return nil
}

func isNullSHA(sha string) bool {
	return sha == "" || sha == strings.Repeat("0", 40)
}

func gitCurrentBranch() (string, error) {
	return gitOutput("rev-parse", "--abbrev-ref", "HEAD")
}

func gitCommitFiles(sha string) ([]string, error) {
	output, err := gitOutput("diff-tree", "--no-commit-id", "-r", "--name-only", sha)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}

func gitFilePatch(fromSHA, toSHA, path string) (string, error) {
	return gitOutput("diff", fromSHA+"~1", toSHA, "--", path)
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
