# Shiplog v1.0 — Full Send Design Spec

**Date**: 2026-03-26
**Status**: Draft
**Author**: Alexandre Humeau
**Base**: Builds on v0.1.0 (already shipped — CLI, Notion writer, conventional commits + LLM analysis)

## Overview

Elevate Shiplog from a functional MVP to a polished, shareable open-source tool. This spec covers: multi-destination writers (Slack, Markdown, JSON), convention-based screenshot capture, beautiful terminal UI, a killer README with animated GIF, a static landing page at `shiplog.sh`, and Homebrew distribution.

## Goals

- **First impression**: someone lands on the repo or `shiplog.sh` and stars within 30 seconds
- **Multi-destination**: Notion + Slack + CHANGELOG.md + JSON stdout
- **Visual proof**: screenshot auto-capture in Notion entries
- **Beautiful DX**: terminal output that makes you want to show colleagues
- **Easy install**: `brew install`, `go install`, pre-built binaries

## Non-Goals

- Discord writer (community can contribute, webhook is trivial)
- npm wrapper (later if traction)
- Full documentation site (README + landing page is enough for v1)
- Bidirectional sync with any destination

---

## 1. Multi-Writer Architecture

### Config

```yaml
writers:
  - notion
  - slack
  - markdown
  # json is activated via --output json flag, not config

slack:
  webhook_url: "https://hooks.slack.com/..."  # or env SHIPLOG_SLACK_WEBHOOK

markdown:
  path: "CHANGELOG.md"  # default
  format: "keepachangelog"  # only format for v1
```

If `writers` is not specified, default to `["notion"]` for backward compatibility. Writers run sequentially — if one fails, log warning and continue to next.

### Writer Interface

```go
type Writer interface {
    Write(entries []model.ChangeEntry) error
    Name() string
}
```

**Refactoring existing writers**: The current `notion.go` and `dryrun.go` use free functions. They must be wrapped in structs that hold their config and implement the `Writer` interface:

```go
// NotionWriter holds token, dbID, props — implements Writer
type NotionWriter struct {
    Token      string
    DatabaseID string
    Props      config.PropertiesConfig
}

// DryRunWriter implements Writer, outputs to terminal
type DryRunWriter struct{}
```

`multi.go` orchestrates: iterates over configured writers, calls each, collects errors. If ALL writers fail, exit 1. If at least one succeeds, exit 0 with warnings for failures.

**Backward compatibility**: existing `.shiplog.yml` files without a `writers` field remain valid — defaults to `["notion"]`.

### Slack Writer (`internal/writer/slack.go`)

POST to webhook URL using Slack Block Kit format:

```json
{
  "blocks": [
    { "type": "header", "text": { "type": "plain_text", "text": "🚢 Shiplog — 3 new entries" } },
    { "type": "section", "text": { "type": "mrkdwn", "text": "✨ *feat* — Système de styles narratifs\n> Ajout d'un popover pour choisir le style d'écriture..." } },
    { "type": "section", "text": { "type": "mrkdwn", "text": "🐛 *fix* — Correction layout side panel\n> 2 commits · 4 files" } },
    { "type": "context", "elements": [{ "type": "mrkdwn", "text": "Branch: main · <https://notion.so/db|View in Notion>" }] }
  ]
}
```

- Single message per push (not one per entry)
- Max 10 entries per message (Slack Block Kit limit is 50 blocks). If more than 10, show first 10 + "and N more entries"
- Include link to Notion database if Notion writer is also active
- Env var: `SHIPLOG_SLACK_WEBHOOK` overrides config

### Markdown Writer (`internal/writer/markdown.go`)

Generates/updates `CHANGELOG.md` using Keep a Changelog format:

```markdown
# Changelog

## 2026-03-26

### ✨ Features

- **Système de styles narratifs** — Ajout d'un popover pour choisir le style d'écriture (a835a3f, cb376ef)

### 🐛 Bug Fixes

- **Correction layout side panel** — Fix du scroll et du sizing (1efc5f9, 767e46e)
```

- Prepend new entries at top (under `# Changelog` heading)
- Create file with `# Changelog` heading if it doesn't exist
- If file exists but has no `# Changelog` heading, prepend the heading
- **Concurrency**: not safe for parallel runs writing to the same file. In CI, ensure only one workflow run writes CHANGELOG.md at a time (use GitHub concurrency groups)
- Group entries by type within each date section
- Include short SHAs in parentheses

### JSON Writer (`internal/writer/json.go`)

Activated via `--output json` flag (mutually exclusive with table output). Outputs to stdout:

```json
{
  "version": "1.0.0",
  "branch": "main",
  "date": "2026-03-26",
  "entries": [
    {
      "title": "Système de styles narratifs",
      "type": "feat",
      "description": "Ajout d'un popover...",
      "commits": ["a835a3f", "cb376ef"],
      "files": ["src/components/NarrativeStylePopover.tsx"]
    }
  ]
}
```

When `--output json` is used, suppress all other stdout output (spinners, tables, etc.).

---

## 2. Convention-Based Screenshots

### Config

```yaml
# .shiplog-screenshots.yml (separate file, optional)
base_url: "http://localhost:3000"
viewport:
  width: 1280
  height: 800
routes:
  "src/pages/Book*": "/books/demo"
  "src/pages/Dashboard*": "/dashboard"
  "src/components/v2/interview/*": "/interview/demo"
```

### Flow

1. After analysis, check if `.shiplog-screenshots.yml` exists
2. For each `ChangeEntry`, match changed files against route patterns (glob match)
3. If match found, launch headless Chromium via `chromedp` (Go library, no external Playwright dependency)
4. Navigate to `base_url + route`, wait for network idle, capture screenshot
5. Save screenshot as PNG to a temp file
6. Upload to Notion as a page content block (image block with external URL is not viable — instead, append the screenshot as a child block using the Notion `image` block type with `file` upload via multipart). **Fallback**: if direct upload is not supported by Notion API version, save screenshot locally to `.shiplog/screenshots/` and add the local path as a rich_text annotation in the Commits property. The Notion `files` property type only supports external URLs, so we embed as an image block in the page body instead.
7. If no match, no `.shiplog-screenshots.yml`, or any error → skip silently, log at debug level

### Why `chromedp` over Playwright

- Pure Go, no npm/node dependency — keeps the tool as a single binary
- `chromedp` uses Chrome DevTools Protocol directly
- Trade-off: requires Chrome/Chromium installed. In CI, use `browser-actions/setup-chrome@v1` step.
- For local dev, most devs have Chrome already

### Limitations (v1)

- No authentication support (screenshots of public/unauthenticated pages only)
- No JavaScript interaction (just load and capture)
- Single viewport size per config
- CI requires a running app server (separate workflow step)

---

## 3. Terminal UI

### Library: `charmbracelet/lipgloss` + `charmbracelet/bubbles`

No full TUI (bubbletea) — just styled output and a spinner component.

### Components

**Header** (shown at start):
```
🚢 Shiplog v1.0.0

  Branch   feat/narrative-style-system
  Commits  a835a3f..cb376ef (5 commits, 12 files)
```

**Spinner** (during LLM call):
```
⣾ Analyzing with Claude claude-sonnet-4-6...
```

Using `bubbles/spinner` with the Dot style. Shown only when LLM is active.

**Results Table** (after analysis):
```
┌──────────┬──────────────────────────────────────────────┐
│ ✨ feat  │ Système de styles narratifs                  │
│          │ Ajout d'un popover permettant de choisir le  │
│          │ style d'écriture pour la génération d'IA     │
│          │ 3 commits · 8 files                          │
├──────────┼──────────────────────────────────────────────┤
│ 🐛 fix   │ Correction du layout side panel              │
│          │ 2 commits · 4 files                          │
└──────────┴──────────────────────────────────────────────┘
```

Using `lipgloss` for borders, colors, padding.

**Footer** (after write):
```
✓ 2 entries written
  → Notion     https://notion.so/abc123
  → Slack      #changelog
  → CHANGELOG  CHANGELOG.md updated
```

**Color scheme:**
- feat: green
- fix: red
- refactor: blue
- docs: yellow
- chore: gray
- perf: orange
- Borders: subtle gray
- Success: green checkmark
- Warning: yellow
- Error: red

**`--output json` mode**: all UI output suppressed, only JSON to stdout.

**`--quiet` flag**: suppress everything except errors. For CI where you don't need pretty output.

### Complete CLI flags for `shiplog run`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | false | Preview without writing |
| `--since` | string | | From specific commit SHA |
| `--last` | int | 0 | Last N commits |
| `--config` | string | `.shiplog.yml` | Config file path |
| `--output` | string | `table` | Output format: `table` or `json` |
| `--quiet` | bool | false | Suppress non-error output |

---

## 4. README Rewrite

### Structure

1. **Hero**: centered project name + one-line tagline + GIF
2. **Badges**: Go version, license, release version, CI status, stars
3. **What it does**: 3-bullet explanation
4. **Before/After**: two-column comparison — raw commits vs Notion screenshot
5. **Quick Start**: 4 steps (install, init, test, deploy)
6. **Demo GIF**: full terminal session showing `shiplog run --dry-run`
7. **Writers**: subsection for each (Notion, Slack, Markdown, JSON) with config examples
8. **Screenshots**: how to set up auto-capture
9. **Configuration**: full reference with all options
10. **GitHub Action**: complete workflow example
11. **CLI Reference**: all commands and flags
12. **Architecture**: ASCII pipeline diagram
13. **Contributing**: brief guide
14. **License**: MIT

### GIF Generation

VHS tape file (`demo.tape`):

```tape
Output docs/site/demo.gif
Set FontSize 14
Set Width 1000
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "shiplog run --dry-run --last 5"
Enter
Sleep 4s
```

Run with `vhs demo.tape` to generate. GIF committed to repo and referenced in README + landing page.

---

## 5. Landing Page (`shiplog.sh`)

### Hosting

- GitHub Pages from `docs/site/` directory
- CNAME file: `shiplog.sh`
- Static HTML/CSS, no framework, no build step

### Content (single page)

1. **Hero**: logo/emoji + "Shiplog" + tagline "Push to main. Changelog appears everywhere." + GIF demo + install command
2. **Features**: 3 cards
   - "Smart Analysis" — groups commits, rewrites in plain language
   - "Multi-Destination" — Notion, Slack, CHANGELOG.md, JSON
   - "Zero Config" — works with conventional commits, AI optional
3. **Before/After**: side-by-side — terminal git log vs formatted Notion database screenshot
4. **Quick Start**: code block with 3 commands
5. **Footer**: GitHub link, MIT license, "Made by Alexandre Humeau"

### Design

- Dark background (#0d1117, GitHub dark)
- Accent color: ocean blue (#58a6ff)
- Monospace font for code blocks
- Max-width 800px, centered
- Responsive (works on mobile)
- No JavaScript required (pure CSS animations for subtle effects)
- Minimal — loads in <100KB total

---

## 6. Distribution

### Homebrew Tap

Create repo `alexhumeau/homebrew-tap`. goreleaser auto-publishes the formula on release.

`.goreleaser.yml` addition:

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

Install: `brew install alexhumeau/tap/shiplog`

### Release workflow

GitHub Actions workflow (`.github/workflows/release.yml`):
- Trigger on tag push (`v*`)
- goreleaser builds + releases + updates Homebrew tap
- VHS generates demo GIF (for consistency)

```yaml
name: Release
on:
  push:
    tags: ["v*"]
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: charmbracelet/vhs-action@v2
        with:
          path: demo.tape
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 7. New/Modified Files

| File | Action | Description |
|------|--------|-------------|
| `internal/writer/slack.go` | Create | Slack Block Kit webhook writer |
| `internal/writer/markdown.go` | Create | CHANGELOG.md Keep a Changelog writer |
| `internal/writer/json.go` | Create | JSON stdout writer |
| `internal/writer/multi.go` | Create | Multi-writer orchestrator |
| `internal/writer/writer.go` | Create | Writer interface definition |
| `internal/writer/notion.go` | Modify | Implement Writer interface |
| `internal/writer/dryrun.go` | Modify | Implement Writer interface |
| `internal/screenshot/screenshot.go` | Create | chromedp-based screenshot capture |
| `internal/ui/header.go` | Create | Styled header output |
| `internal/ui/table.go` | Create | lipgloss results table |
| `internal/ui/spinner.go` | Create | LLM progress spinner |
| `internal/ui/colors.go` | Create | Color constants and type→color mapping |
| `internal/config/config.go` | Modify | Add writers, slack, markdown, screenshot fields |
| `internal/pipeline/pipeline.go` | Modify | Multi-writer orchestration + screenshot step |
| `cmd/shiplog/main.go` | Modify | `--output` and `--quiet` flags |
| `docs/site/index.html` | Create | Landing page |
| `docs/site/CNAME` | Create | `shiplog.sh` |
| `demo.tape` | Create | VHS demo script |
| `.goreleaser.yml` | Modify | Add homebrew tap config |
| `.github/workflows/release.yml` | Create | Automated release workflow |
| `README.md` | Rewrite | Full rewrite with GIF, badges, writers docs |

## 8. Error Handling Additions

| Scenario | Behavior |
|----------|----------|
| Slack webhook fails | Warning logged, continue to next writer |
| CHANGELOG.md write permission denied | Warning logged, continue |
| chromedp: Chrome not found | Warning: "Chrome/Chromium required for screenshots", skip |
| chromedp: page load timeout | Warning logged, skip screenshot for this entry |
| chromedp: no `.shiplog-screenshots.yml` | Silent skip |
| `--output json` + `--dry-run` | JSON output to stdout, no Notion write |
| Multiple writers, one fails | Log failure, continue others, exit 0 if at least one succeeded |

## 9. Config Reference (v1.0 complete)

```yaml
# Core
notion:
  database_id: "abc123"

# Writers (default: ["notion"])
writers:
  - notion
  - slack
  - markdown

# Slack
slack:
  webhook_url: "https://hooks.slack.com/..."

# Markdown
markdown:
  path: "CHANGELOG.md"
  format: "keepachangelog"

# Context
context:
  readme: true
  docs: ["CLAUDE.md", "docs/ARCHITECTURE.md"]
  max_context_chars: 16000
  max_diff_lines: 500

# AI (optional)
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-6"
  language: "fr"

# Categories
categories:
  feat: "✨ Feature"
  fix: "🐛 Bug Fix"
  refactor: "♻️ Refactor"
  docs: "📝 Documentation"
  chore: "🔧 Maintenance"
  perf: "⚡ Performance"

# Filters
filters:
  branches: ["main"]
  ignore_paths: ["*.lock", "node_modules/**"]
  ignore_types: ["chore"]

# Property mapping (for existing Notion DBs)
properties:
  title: "Title"
  type: "Type"
  date: "Date"
  branch: "Branch"
  commits: "Commits"
  files: "Files Changed"
  screenshot: "Screenshot"
```
