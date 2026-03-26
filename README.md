# 🚢 Shiplog

Auto-generate a structured Notion changelog from your git history. Push to main → changelog appears in Notion.

## Features

- **Zero friction** — GitHub Action triggers on push, no manual work
- **Smart analysis** — parses conventional commits, groups related changes
- **AI-powered** (optional) — uses Claude or GPT to rewrite entries in plain language
- **Context-aware** — reads your README and docs to write business-oriented descriptions
- **Deduplication** — re-runs are safe, already-processed commits are skipped

## Quick Start

### 1. Install

```bash
go install github.com/alexandrehumeau/shiplog@latest
```

### 2. Setup

```bash
shiplog init
```

This creates a Notion database and generates `.shiplog.yml`.

### 3. Test

```bash
shiplog run --dry-run --last 10
```

### 4. GitHub Action

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

## CLI

```bash
shiplog run                    # analyze since last run, push to Notion
shiplog run --dry-run          # preview without writing
shiplog run --since abc123     # from specific commit
shiplog run --last 5           # last 5 commits
shiplog init                   # interactive setup
shiplog init --notion-setup    # guided Notion integration creation
```

## Configuration

Create `.shiplog.yml` in your repo root (or use `shiplog init`):

```yaml
notion:
  database_id: "your-database-id"

context:
  readme: true
  docs:
    - "docs/ARCHITECTURE.md"
  max_context_chars: 16000
  max_diff_lines: 500

ai:
  provider: "anthropic"        # anthropic | openai
  model: "claude-sonnet-4-6"  # optional, has smart defaults
  language: "en"               # output language

filters:
  branches: ["main"]
  ignore_paths: ["*.lock", "node_modules/**"]
  ignore_types: ["chore"]
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `NOTION_TOKEN` | Notion integration token (required) |
| `NOTION_DATABASE_ID` | Target database ID |
| `SHIPLOG_AI_PROVIDER` | `anthropic` or `openai` |
| `SHIPLOG_AI_API_KEY` | Generic LLM API key |
| `ANTHROPIC_API_KEY` | Anthropic-specific (fallback) |
| `OPENAI_API_KEY` | OpenAI-specific (fallback) |
| `SHIPLOG_LANGUAGE` | Output language |
| `SHIPLOG_DRY_RUN` | Set to `true` for dry-run |
| `SHIPLOG_CONFIG` | Config file path |

## How It Works

```
git push → GitHub Action triggers
  │
  ├─ Collector: git log + diff + README + docs
  │
  ├─ Analyzer: conventional commits + optional LLM
  │   ├─ Groups related commits
  │   ├─ Categorizes (feat/fix/refactor/docs/chore/perf)
  │   └─ Rewrites titles in plain language
  │
  └─ Writer: creates Notion database entries
      ├─ Deduplicates by commit SHA
      └─ Retries on rate limits
```

## Without AI

Shiplog works without an AI provider. It parses conventional commit messages (`feat:`, `fix:`, etc.) and groups by scope. AI adds smarter grouping, better titles, and context-aware descriptions.

## License

MIT
