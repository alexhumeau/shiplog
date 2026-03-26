# Shiplog v1.0 "Full Send" Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Elevate Shiplog from MVP to polished open-source tool with multi-writer support, beautiful terminal UI, screenshots, landing page, and Homebrew distribution.

**Architecture:** Refactor writers behind a `Writer` interface, add Slack/Markdown/JSON writers, add lipgloss terminal UI, add chromedp screenshot capture, rewrite README with VHS GIF, create static landing page, configure goreleaser for Homebrew tap.

**Tech Stack:** Go 1.22+, charmbracelet/lipgloss, charmbracelet/bubbles, chromedp, VHS

**Spec:** `docs/specs/2026-03-26-shiplog-v1-sexy-design.md`

**Repo:** `~/Dev/shiplog/`

---

## File Map

```
Changes to existing:
  internal/config/config.go          — add writers, slack, markdown, screenshot config
  internal/writer/notion.go          — refactor to Writer interface
  internal/writer/dryrun.go          — refactor to Writer interface
  internal/pipeline/pipeline.go      — multi-writer orchestration + screenshot + UI
  cmd/shiplog/main.go                — --output, --quiet flags
  .goreleaser.yml                    — homebrew tap
  README.md                          — full rewrite

New files:
  internal/writer/writer.go          — Writer interface + multi-writer orchestrator (spec's multi.go merged here)
  internal/writer/slack.go           — Slack webhook writer
  internal/writer/markdown.go        — CHANGELOG.md writer
  internal/writer/json.go            — JSON stdout writer
  internal/screenshot/screenshot.go  — chromedp screenshot capture
  internal/ui/ui.go                  — header, table, footer, colors
  internal/ui/spinner.go             — LLM spinner (bubbles)
  docs/site/index.html               — landing page
  docs/site/CNAME                    — shiplog.sh domain
  demo.tape                          — VHS demo script
  .github/workflows/release.yml      — automated release
```

---

### Task 1: Writer Interface + Refactor Existing Writers

**Files:**
- Create: `internal/writer/writer.go`
- Modify: `internal/writer/notion.go`
- Modify: `internal/writer/dryrun.go`

- [ ] **Step 1: Create Writer interface and multi-writer orchestrator** (`internal/writer/writer.go`)

```go
package writer

import (
    "fmt"
    "github.com/alexandrehumeau/shiplog/internal/model"
)

type Writer interface {
    Write(entries []model.ChangeEntry) error
    Name() string
}

func WriteAll(writers []Writer, entries []model.ChangeEntry) error {
    var lastErr error
    succeeded := 0
    for _, w := range writers {
        if err := w.Write(entries); err != nil {
            fmt.Printf("  ⚠ %s failed: %v\n", w.Name(), err)
            lastErr = err
        } else {
            succeeded++
        }
    }
    if succeeded == 0 && lastErr != nil {
        return fmt.Errorf("all writers failed, last error: %w", lastErr)
    }
    return nil
}
```

- [ ] **Step 2: Refactor NotionWriter to struct implementing Writer interface**

Wrap existing `WriteAll` logic into a `NotionWriter` struct holding `Token`, `DatabaseID`, `Props`. Rename old `WriteAll` free function to avoid conflict. Implement `Write(entries)` and `Name()`.

- [ ] **Step 3: Refactor DryRunWriter to struct implementing Writer interface**

Wrap `DryRun` function into `DryRunWriter` struct. Implement `Write(entries)` and `Name()`.

- [ ] **Step 4: Verify compilation**

```bash
cd ~/Dev/shiplog && go build ./cmd/shiplog/
```

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "refactor: Writer interface, wrap Notion and DryRun as structs"
```

---

### Task 2: Config Updates

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add new config fields**

Add to `Config`:
```go
Writers    []string        `yaml:"writers"`     // default: ["notion"]
Slack      SlackConfig     `yaml:"slack"`
Markdown   MarkdownConfig  `yaml:"markdown"`
```

New types:
```go
type SlackConfig struct {
    WebhookURL string `yaml:"webhook_url"`
}

type MarkdownConfig struct {
    Path   string `yaml:"path"`
    Format string `yaml:"format"`
}
```

Add defaults: `Writers: []string{"notion"}`, `Markdown.Path: "CHANGELOG.md"`, `Markdown.Format: "keepachangelog"`.

Add env overrides: `SHIPLOG_SLACK_WEBHOOK` → `Slack.WebhookURL`.

- [ ] **Step 2: Verify compilation + existing behavior unchanged**

```bash
go build ./cmd/shiplog/ && cd ~/Dev/raconteo && ~/Dev/shiplog/shiplog run --dry-run --last 3
```

- [ ] **Step 3: Commit**

```bash
cd ~/Dev/shiplog && git add -A && git commit -m "feat: config — add writers, slack, markdown fields"
```

---

### Task 3: Slack Writer

**Files:**
- Create: `internal/writer/slack.go`

- [ ] **Step 1: Implement Slack writer**

`SlackWriter` struct with `WebhookURL` and optional `NotionDBID` (for link in footer).

`Write(entries)`: build Slack Block Kit payload — header block with count, section block per entry (max 10, then "and N more"), context block with branch + Notion link. POST to webhook URL.

`Name()` returns `"Slack"`.

- [ ] **Step 2: Verify compilation**

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: Slack webhook writer with Block Kit formatting"
```

---

### Task 4: Markdown Writer

**Files:**
- Create: `internal/writer/markdown.go`

- [ ] **Step 1: Implement Markdown writer**

`MarkdownWriter` struct with `Path` string.

`Write(entries)`:
1. Read existing file (or empty string if not exists)
2. If no `# Changelog` heading, prepend it
3. Build new section: `## YYYY-MM-DD`, sub-headings by type (`### ✨ Features`, `### 🐛 Bug Fixes`, etc.), bulleted entries with title, description snippet, short SHAs
4. Insert after `# Changelog` line, before previous content
5. Write file

`Name()` returns `"Markdown"`.

- [ ] **Step 2: Verify compilation**

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: Markdown writer — CHANGELOG.md in Keep a Changelog format"
```

---

### Task 5: JSON Writer

**Files:**
- Create: `internal/writer/json.go`

- [ ] **Step 1: Implement JSON writer**

`JSONWriter` struct (no config needed).

`Write(entries)`: marshal entries to JSON with `version`, `branch`, `date`, `entries` fields. Print to stdout.

`Name()` returns `"JSON"`.

- [ ] **Step 2: Commit**

```bash
git add -A && git commit -m "feat: JSON stdout writer for piping and scripting"
```

---

### Task 6: Terminal UI — Colors + Table + Header + Footer

**Files:**
- Create: `internal/ui/ui.go`

- [ ] **Step 1: Implement UI package**

Using `charmbracelet/lipgloss`:

`TypeColor(typ string) lipgloss.Color` — green for feat, red for fix, blue for refactor, yellow for docs, gray for chore, orange for perf.

`TypeEmoji(typ string) string` — same mapping as existing dryrun.go.

`RenderHeader(version, branch, commitRange string, commitCount, fileCount int) string` — styled header block.

`RenderTable(entries []model.ChangeEntry) string` — bordered table with type+emoji column and title+description+stats column.

`RenderFooter(results map[string]string) string` — checkmark + destination list.

- [ ] **Step 2: Add lipgloss dependency**

```bash
go get github.com/charmbracelet/lipgloss
```

- [ ] **Step 3: Verify compilation**

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: terminal UI — lipgloss header, table, footer with colors"
```

---

### Task 7: Terminal UI — LLM Spinner

**Files:**
- Create: `internal/ui/spinner.go`

- [ ] **Step 1: Implement spinner**

Using `charmbracelet/bubbles/spinner`. Wrap in a simple API:

```go
func StartSpinner(message string) *Spinner
func (s *Spinner) Stop()
```

The spinner runs in a goroutine, prints to stderr (so it doesn't interfere with JSON output). On `Stop()`, clears the line and prints the final message.

- [ ] **Step 2: Add bubbles dependency**

```bash
go get github.com/charmbracelet/bubbles
```

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: LLM progress spinner using charmbracelet/bubbles"
```

---

### Task 8: Wire Multi-Writer + UI into Pipeline

**Files:**
- Modify: `internal/pipeline/pipeline.go`
- Modify: `cmd/shiplog/main.go`

- [ ] **Step 1: Update pipeline to use Writer interface and UI**

Replace direct writer calls with:
1. Build writer list from `cfg.Writers` config
2. Show UI header at start
3. Show spinner during LLM call (wrap analyzer.Analyze)
4. Show UI table after analysis
5. Call `writer.WriteAll(writers, entries)`
6. Show UI footer with results

Add `--output` flag: if `json`, use only JSONWriter and suppress all UI. Add `--quiet` flag: suppress all non-error output.

Pass `Output` and `Quiet` through `RunOptions`.

- [ ] **Step 2: Test dry-run still works**

```bash
cd ~/Dev/raconteo && ~/Dev/shiplog/shiplog run --dry-run --last 5
```

- [ ] **Step 3: Test JSON output**

```bash
cd ~/Dev/raconteo && ~/Dev/shiplog/shiplog run --dry-run --last 5 --output json
```

- [ ] **Step 4: Commit**

```bash
cd ~/Dev/shiplog && git add -A && git commit -m "feat: multi-writer pipeline + terminal UI integration"
```

---

### Task 9: Screenshot Capture

**Files:**
- Create: `internal/screenshot/screenshot.go`
- Modify: `internal/pipeline/pipeline.go`

- [ ] **Step 1: Implement screenshot module**

```go
type ScreenshotConfig struct {
    BaseURL  string            `yaml:"base_url"`
    Viewport Viewport          `yaml:"viewport"`
    Routes   map[string]string `yaml:"routes"`  // glob pattern → route
}

type Viewport struct {
    Width  int `yaml:"width"`
    Height int `yaml:"height"`
}
```

`LoadConfig(path string) (*ScreenshotConfig, error)` — reads `.shiplog-screenshots.yml`, returns nil if not found.

`CaptureForEntry(cfg *ScreenshotConfig, entry model.ChangeEntry) ([]byte, error)`:
1. Match entry.Files against cfg.Routes patterns (filepath.Match)
2. If match, use `chromedp` to navigate to `baseURL + route`, wait for load, screenshot
3. Return PNG bytes or nil if no match

Viewport defaults: 1280x800.

- [ ] **Step 2: Add chromedp dependency**

```bash
go get github.com/chromedp/chromedp
```

- [ ] **Step 3: Wire into pipeline** — after analysis, before write. For each entry, attempt capture. If screenshot bytes returned, attach to entry (add `Screenshot []byte` field to model or pass separately to Notion writer).

- [ ] **Step 4: Commit**

```bash
cd ~/Dev/shiplog && git add -A && git commit -m "feat: convention-based screenshot capture with chromedp"
```

---

### Task 10: README Rewrite

**Files:**
- Rewrite: `README.md`

- [ ] **Step 1: Full README rewrite**

Structure:
1. Hero: `# 🚢 Shiplog` + tagline + `[demo GIF placeholder]`
2. Badges: Go report, license, release, CI
3. What it does: 3 bullets
4. Quick Start: install → init → test → deploy (4 steps)
5. Writers: Notion, Slack, Markdown, JSON — each with config example
6. Screenshots: `.shiplog-screenshots.yml` example
7. Configuration: full `.shiplog.yml` reference
8. GitHub Action: complete workflow
9. CLI Reference: all commands + flags table
10. Architecture: ASCII pipeline diagram
11. Without AI: how it works without LLM
12. Contributing: brief
13. License: MIT

- [ ] **Step 2: Commit**

```bash
git add -A && git commit -m "docs: complete README rewrite — writers, screenshots, config reference"
```

---

### Task 11: VHS Demo GIF

**Files:**
- Create: `demo.tape`

- [ ] **Step 1: Write VHS tape file**

```tape
Output docs/site/demo.gif
Set FontSize 14
Set Width 1000
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "shiplog run --dry-run --last 5"
Enter
Sleep 5s
```

- [ ] **Step 2: Install VHS and generate GIF**

```bash
brew install vhs
cd ~/Dev/shiplog && vhs demo.tape
```

- [ ] **Step 3: Update README to reference GIF**

Replace placeholder with `![Shiplog Demo](docs/site/demo.gif)`.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: VHS demo GIF for README and landing page"
```

---

### Task 12: Landing Page

**Files:**
- Create: `docs/site/index.html`
- Create: `docs/site/CNAME`

- [ ] **Step 1: Create landing page**

Single HTML file, dark theme (#0d1117), accent blue (#58a6ff), responsive, <100KB.

Sections: hero with GIF + tagline + install command, 3 feature cards, before/after visual, quick start code block, footer with GitHub link.

Pure HTML/CSS, no JS required.

- [ ] **Step 2: Create CNAME**

```
shiplog.sh
```

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: landing page for shiplog.sh — dark theme, responsive"
```

---

### Task 13: Homebrew Tap + Release Workflow

**Files:**
- Modify: `.goreleaser.yml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Update goreleaser for Homebrew**

Add `brews` section:
```yaml
brews:
  - repository:
      owner: alexhumeau
      name: homebrew-tap
    homepage: "https://shiplog.sh"
    description: "Auto-generate a structured changelog from git history"
    license: "MIT"
    install: |
      bin.install "shiplog"
```

- [ ] **Step 2: Create Homebrew tap repo on GitHub**

```bash
gh repo create homebrew-tap --public --description "Homebrew tap for Shiplog"
```

- [ ] **Step 3: Create release workflow**

`.github/workflows/release.yml` — trigger on `v*` tags, setup Go, run goreleaser.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: Homebrew tap + automated release workflow"
```

---

### Task 14: Final Integration Test + v1.0 Release

- [ ] **Step 1: Build and test full pipeline**

```bash
cd ~/Dev/shiplog && go build ./cmd/shiplog/
cd ~/Dev/raconteo && ~/Dev/shiplog/shiplog run --dry-run --last 10
cd ~/Dev/raconteo && ~/Dev/shiplog/shiplog run --dry-run --last 10 --output json
```

- [ ] **Step 2: Push all commits**

```bash
cd ~/Dev/shiplog && git push origin main
```

- [ ] **Step 3: Tag and release v1.0.0**

```bash
git tag v1.0.0
git push origin v1.0.0
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
```

- [ ] **Step 4: Verify release**

```bash
gh release view v1.0.0
brew install alexhumeau/tap/shiplog
shiplog --version
```
