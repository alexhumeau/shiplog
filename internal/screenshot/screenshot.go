package screenshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/model"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

// Config holds screenshot capture settings.
type Config struct {
	BaseURL  string            `yaml:"base_url"`
	Viewport Viewport          `yaml:"viewport"`
	Routes   map[string]string `yaml:"routes"` // glob pattern → route path
}

// Viewport defines the browser window size.
type Viewport struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// LoadConfig reads .shiplog-screenshots.yml. Returns nil if not found.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading screenshot config: %w", err)
	}

	cfg := &Config{
		Viewport: Viewport{Width: 1280, Height: 800},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing screenshot config: %w", err)
	}

	if cfg.BaseURL == "" {
		return nil, nil
	}

	return cfg, nil
}

// CaptureForEntry takes a screenshot if any of the entry's files match a route pattern.
// Returns PNG bytes or nil if no match.
func CaptureForEntry(cfg *Config, entry model.ChangeEntry) ([]byte, error) {
	if cfg == nil || len(cfg.Routes) == 0 {
		return nil, nil
	}

	// Find first matching route
	route := matchRoute(cfg.Routes, entry.Files)
	if route == "" {
		return nil, nil
	}

	url := cfg.BaseURL + route
	width := cfg.Viewport.Width
	height := cfg.Viewport.Height
	if width == 0 {
		width = 1280
	}
	if height == 0 {
		height = 800
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(width), int64(height)),
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second), // wait for rendering
		chromedp.FullScreenshot(&buf, 90),
	)
	if err != nil {
		return nil, fmt.Errorf("capturing screenshot for %s: %w", url, err)
	}

	return buf, nil
}

// matchRoute finds the first route whose glob pattern matches any file in the list.
func matchRoute(routes map[string]string, files []string) string {
	for pattern, route := range routes {
		for _, f := range files {
			if matched, _ := filepath.Match(pattern, f); matched {
				return route
			}
			// Also try matching just the filename
			if matched, _ := filepath.Match(pattern, filepath.Base(f)); matched {
				return route
			}
		}
	}
	return ""
}
