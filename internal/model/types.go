package model

import "time"

// Commit represents a single git commit.
type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
	Files   []string
}

// FileDiff represents the diff of a single file in a commit range.
type FileDiff struct {
	Path      string
	Additions int
	Deletions int
	Patch     string // truncated to first 20 lines
}

// ProjectContext holds documentation context for LLM enrichment.
type ProjectContext struct {
	Readme string
	Docs   map[string]string // path → content (truncated)
}

// PushData is the output of the Collector stage.
type PushData struct {
	Commits   []Commit
	Diffs     []FileDiff
	Context   ProjectContext
	Branch    string
	BeforeSHA string
	AfterSHA  string
}

// ChangeEntry is a single changelog entry, output of the Analyzer stage.
type ChangeEntry struct {
	Title       string
	Type        string // feat, fix, refactor, docs, chore, perf
	Description string // 2-3 sentences, business-oriented
	Commits     []Commit
	Files       []string
	Branch      string
	Date        time.Time
	Screenshot  []byte // PNG screenshot bytes, nil if none captured
}
