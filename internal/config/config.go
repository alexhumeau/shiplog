package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for Shiplog.
type Config struct {
	Notion     NotionConfig      `yaml:"notion"`
	Writers    []string          `yaml:"writers"`
	Slack      SlackConfig       `yaml:"slack"`
	Markdown   MarkdownConfig    `yaml:"markdown"`
	Context    ContextConfig     `yaml:"context"`
	AI         AIConfig          `yaml:"ai"`
	Categories map[string]string `yaml:"categories"`
	Filters    FiltersConfig     `yaml:"filters"`
	Properties PropertiesConfig  `yaml:"properties"`
	DryRun     bool              `yaml:"-"`
}

type NotionConfig struct {
	DatabaseID string `yaml:"database_id"`
	Token      string `yaml:"-"` // from env only
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

type MarkdownConfig struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
}

type ContextConfig struct {
	Readme          bool     `yaml:"readme"`
	Docs            []string `yaml:"docs"`
	MaxContextChars int      `yaml:"max_context_chars"`
	MaxDiffLines    int      `yaml:"max_diff_lines"`
}

type AIConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Language string `yaml:"language"`
	APIKey   string `yaml:"-"` // from env only
}

type FiltersConfig struct {
	Branches    []string `yaml:"branches"`
	IgnorePaths []string `yaml:"ignore_paths"`
	IgnoreTypes []string `yaml:"ignore_types"`
}

type PropertiesConfig struct {
	Title      string `yaml:"title"`
	Type       string `yaml:"type"`
	Date       string `yaml:"date"`
	Branch     string `yaml:"branch"`
	Commits    string `yaml:"commits"`
	Files      string `yaml:"files"`
	Screenshot string `yaml:"screenshot"`
}

// Load reads a .shiplog.yml file and applies env var overrides.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}

	applyEnvOverrides(cfg)
	applyModelDefaults(cfg)
	applyWriterDefaults(cfg)

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Context: ContextConfig{
			Readme:          true,
			MaxContextChars: 16000,
			MaxDiffLines:    500,
		},
		Markdown: MarkdownConfig{
			Path:   "CHANGELOG.md",
			Format: "keepachangelog",
		},
		Categories: map[string]string{
			"feat":     "✨ Feature",
			"fix":      "🐛 Bug Fix",
			"refactor": "♻️ Refactor",
			"docs":     "📝 Documentation",
			"chore":    "🔧 Maintenance",
			"perf":     "⚡ Performance",
		},
		Properties: PropertiesConfig{
			Title:      "Title",
			Type:       "Type",
			Date:       "Date",
			Branch:     "Branch",
			Commits:    "Commits",
			Files:      "Files Changed",
			Screenshot: "Screenshot",
		},
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("NOTION_TOKEN"); v != "" {
		cfg.Notion.Token = v
	}
	if v := os.Getenv("NOTION_DATABASE_ID"); v != "" {
		cfg.Notion.DatabaseID = v
	}
	if v := os.Getenv("SHIPLOG_AI_PROVIDER"); v != "" {
		cfg.AI.Provider = v
	}
	// Generic key first, then provider-specific fallback
	if v := os.Getenv("SHIPLOG_AI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	} else if cfg.AI.Provider == "anthropic" {
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			cfg.AI.APIKey = v
		}
	} else if cfg.AI.Provider == "openai" {
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			cfg.AI.APIKey = v
		}
	}
	if v := os.Getenv("SHIPLOG_LANGUAGE"); v != "" {
		cfg.AI.Language = v
	}
	if v := os.Getenv("SHIPLOG_DRY_RUN"); strings.EqualFold(v, "true") {
		cfg.DryRun = true
	}
	if v := os.Getenv("SHIPLOG_SLACK_WEBHOOK"); v != "" {
		cfg.Slack.WebhookURL = v
	}
}

func applyModelDefaults(cfg *Config) {
	if cfg.AI.Provider != "" && cfg.AI.Model == "" {
		switch cfg.AI.Provider {
		case "anthropic":
			cfg.AI.Model = "claude-sonnet-4-6"
		case "openai":
			cfg.AI.Model = "gpt-4o-mini"
		}
	}
	if cfg.AI.Language == "" {
		cfg.AI.Language = "en"
	}
}

func applyWriterDefaults(cfg *Config) {
	if len(cfg.Writers) == 0 {
		cfg.Writers = []string{"notion"}
	}
}

// HasWriter checks if a specific writer is enabled.
func (c *Config) HasWriter(name string) bool {
	for _, w := range c.Writers {
		if w == name {
			return true
		}
	}
	return false
}

// Validate checks that required config values are present for enabled writers.
func (c *Config) Validate() error {
	if c.HasWriter("notion") {
		if c.Notion.Token == "" {
			return fmt.Errorf("NOTION_TOKEN env var is required when using Notion writer. Create an integration at https://www.notion.so/my-integrations")
		}
		if c.Notion.DatabaseID == "" {
			return fmt.Errorf("notion.database_id is required in .shiplog.yml or set NOTION_DATABASE_ID env var")
		}
	}
	if c.HasWriter("slack") {
		if c.Slack.WebhookURL == "" {
			return fmt.Errorf("slack.webhook_url is required in .shiplog.yml or set SHIPLOG_SLACK_WEBHOOK env var")
		}
	}
	return nil
}
