<p align="center">
  <img src="docs/site/demo.gif" alt="Shiplog Demo" width="700">
</p>

<h1 align="center">🚢 Shiplog</h1>

<p align="center">
  <strong>Auto-generate a structured changelog from git history.</strong><br>
  Push to main → changelog appears in Notion, Slack, Markdown, and more.
</p>

<p align="center">
  <a href="https://github.com/alexandrehumeau/shiplog/releases"><img src="https://img.shields.io/github/v/release/alexandrehumeau/shiplog?style=flat-square" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/alexandrehumeau/shiplog"><img src="https://goreportcard.com/badge/github.com/alexandrehumeau/shiplog?style=flat-square" alt="Go Report"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://github.com/alexandrehumeau/shiplog/actions"><img src="https://img.shields.io/github/actions/workflow/status/alexandrehumeau/shiplog/release.yml?style=flat-square&label=CI" alt="CI"></a>
</p>

---

## What it does

- **Collects** commits, diffs, and project context from your git history
- **Analyzes** changes using conventional commits + optional AI (Claude / GPT)
- **Writes** structured changelog entries to **Notion**, **Slack**, **Markdown**, and **JSON**

## Quick Start

### Install

```bash
# Homebrew
brew install alexhumeau/tap/shiplog

# Go
go install github.com/alexandrehumeau/shiplog@latest

# Binary (Linux/macOS/Windows)
# → https://github.com/alexandrehumeau/shiplog/releases
```

### Setup

```bash
shiplog init                   # interactive setup, creates .shiplog.yml
shiplog init --notion-setup    # guided Notion integration creation
```

### Test

```bash
shiplog run --dry-run --last 10
```

### Deploy

```yaml
# .github/workflows/shiplog.yml
name: Shiplog
on:
  push:
    branches: [main]
jobs:
  changelog:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: alexandrehumeau/shiplog@v1
        with:
          notion_token: ${{ secrets.NOTION_TOKEN }}
          notion_database_id: ${{ secrets.NOTION_DATABASE_ID }}
          ai_provider: anthropic
          ai_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Writers

Shiplog supports multiple output destinations. Enable them in `.shiplog.yml`:

```yaml
writers:
  - notion
  - slack
  - markdown
```

### Notion

Creates rich database entries with type, title, description, commits, files, and date.

```yaml
notion:
  database_id: "your-database-id"
# Set NOTION_TOKEN env var
```

### Slack

Posts Block Kit messages to a webhook.

```yaml
slack:
  webhook_url: "https://hooks.slack.com/services/..."
# Or set SHIPLOG_SLACK_WEBHOOK env var
```

### Markdown

Appends entries to a CHANGELOG.md in [Keep a Changelog](https://keepachangelog.com) format.

```yaml
markdown:
  path: "CHANGELOG.md"
```

### JSON

Output structured JSON to stdout for piping:

```bash
shiplog run --output json --last 5 | jq '.entries'
```

## Screenshots

Automatically capture screenshots of changed pages and attach them to Notion entries.

Create `.shiplog-screenshots.yml`:

```yaml
base_url: "http://localhost:3000"
viewport:
  width: 1280
  height: 800
routes:
  "src/pages/*.tsx": "/dashboard"
  "src/components/auth/*": "/login"
```

When a changelog entry's files match a route pattern, Shiplog opens a headless Chrome, navigates to the URL, and captures a screenshot.

## Configuration

Full `.shiplog.yml` reference:

```yaml
# Writers (default: notion)
writers:
  - notion
  - slack
  - markdown

# Notion
notion:
  database_id: "your-database-id"

# Slack
slack:
  webhook_url: "https://hooks.slack.com/services/..."

# Markdown
markdown:
  path: "CHANGELOG.md"

# Project context for AI
context:
  readme: true
  docs:
    - "docs/ARCHITECTURE.md"
  max_context_chars: 16000
  max_diff_lines: 500

# AI provider (optional)
ai:
  provider: "anthropic"        # anthropic | openai
  model: "claude-sonnet-4-6"  # optional, has smart defaults
  language: "en"               # output language

# Filters
filters:
  branches: ["main"]
  ignore_paths: ["*.lock", "node_modules/**"]
  ignore_types: ["chore"]

# Notion property names (customize to match your DB)
properties:
  title: "Title"
  type: "Type"
  date: "Date"
  branch: "Branch"
  commits: "Commits"
  files: "Files Changed"
  screenshot: "Screenshot"
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `NOTION_TOKEN` | Notion integration token |
| `NOTION_DATABASE_ID` | Target database ID |
| `SHIPLOG_AI_PROVIDER` | `anthropic` or `openai` |
| `SHIPLOG_AI_API_KEY` | Generic LLM API key |
| `ANTHROPIC_API_KEY` | Anthropic-specific fallback |
| `OPENAI_API_KEY` | OpenAI-specific fallback |
| `SHIPLOG_SLACK_WEBHOOK` | Slack webhook URL |
| `SHIPLOG_LANGUAGE` | Output language |
| `SHIPLOG_DRY_RUN` | Set to `true` for dry-run |
| `SHIPLOG_CONFIG` | Config file path |

## CLI Reference

```
shiplog run                        # analyze since last run, push to writers
shiplog run --dry-run              # preview without writing
shiplog run --since abc123         # from specific commit
shiplog run --last 5               # last 5 commits
shiplog run --output json          # JSON to stdout
shiplog run --quiet                # suppress non-error output
shiplog init                       # interactive setup
shiplog init --notion-setup        # guided Notion integration
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `false` | Preview without writing |
| `--since` | string | | Start from specific commit SHA |
| `--last` | int | `0` | Analyze last N commits |
| `--output` | string | `table` | Output format: `table` or `json` |
| `--quiet` | bool | `false` | Suppress non-error output |
| `--config` | string | `.shiplog.yml` | Config file path |

## GitHub Action

```yaml
- uses: alexandrehumeau/shiplog@v1
  with:
    notion_token: ${{ secrets.NOTION_TOKEN }}
    notion_database_id: ${{ secrets.NOTION_DATABASE_ID }}
    # Optional
    ai_provider: anthropic
    ai_api_key: ${{ secrets.ANTHROPIC_API_KEY }}
    writers: "notion,slack"
    slack_webhook: ${{ secrets.SLACK_WEBHOOK }}
    config: ".shiplog.yml"
```

## Architecture

```
git push → Shiplog triggers
  │
  ├─ Collector: git log + diff + README + docs
  │
  ├─ Analyzer: conventional commits + optional LLM
  │   ├─ Groups related commits
  │   ├─ Categorizes (feat/fix/refactor/docs/chore/perf)
  │   └─ Rewrites titles in plain language
  │
  ├─ Screenshot: headless Chrome capture (optional)
  │
  └─ Writers: multi-destination output
      ├─ Notion: rich database entries with dedup
      ├─ Slack: Block Kit webhook messages
      ├─ Markdown: CHANGELOG.md (Keep a Changelog)
      └─ JSON: structured stdout for piping
```

## Without AI

Shiplog works without an AI provider. It parses conventional commit messages (`feat:`, `fix:`, etc.) and groups by scope. AI adds smarter grouping, better titles, and context-aware descriptions.

## Contributing

```bash
git clone https://github.com/alexandrehumeau/shiplog
cd shiplog
go build ./cmd/shiplog/
./shiplog run --dry-run --last 5
```

## License

MIT
