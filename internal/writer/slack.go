package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexandrehumeau/shiplog/internal/model"
)

// SlackWriter posts changelog entries to a Slack channel via webhook.
type SlackWriter struct {
	WebhookURL  string
	NotionDBID  string // optional, for link in footer
}

func (w *SlackWriter) Name() string { return "Slack" }

func (w *SlackWriter) Write(entries []model.ChangeEntry) error {
	if w.WebhookURL == "" {
		return fmt.Errorf("slack webhook_url not configured")
	}

	blocks := buildSlackBlocks(entries, w.NotionDBID)
	payload := map[string]interface{}{"blocks": blocks}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	resp, err := http.Post(w.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("posting to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func buildSlackBlocks(entries []model.ChangeEntry, notionDBID string) []map[string]interface{} {
	var blocks []map[string]interface{}

	// Header
	blocks = append(blocks, map[string]interface{}{
		"type": "header",
		"text": map[string]interface{}{
			"type": "plain_text",
			"text": fmt.Sprintf("🚢 Shiplog — %d new entries", len(entries)),
		},
	})

	// Entries (max 10)
	shown := entries
	if len(shown) > 10 {
		shown = shown[:10]
	}

	for _, e := range shown {
		emoji := TypeEmoji(e.Type)
		commitCount := len(e.Commits)
		fileCount := len(e.Files)

		text := fmt.Sprintf("%s *%s* — %s", emoji, e.Type, e.Title)
		if e.Description != "" && e.Description != e.Title {
			// Truncate description for Slack
			desc := e.Description
			if len(desc) > 150 {
				desc = desc[:147] + "..."
			}
			text += fmt.Sprintf("\n> %s", desc)
		}
		text += fmt.Sprintf("\n_%d commits · %d files_", commitCount, fileCount)

		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": text,
			},
		})
	}

	if len(entries) > 10 {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": fmt.Sprintf("_...and %d more entries_", len(entries)-10),
			},
		})
	}

	// Footer
	var contextParts []string
	if len(entries) > 0 {
		contextParts = append(contextParts, fmt.Sprintf("Branch: `%s`", entries[0].Branch))
	}
	if notionDBID != "" {
		contextParts = append(contextParts, fmt.Sprintf("<https://notion.so/%s|View in Notion>", strings.ReplaceAll(notionDBID, "-", "")))
	}

	if len(contextParts) > 0 {
		blocks = append(blocks, map[string]interface{}{
			"type": "context",
			"elements": []map[string]interface{}{
				{"type": "mrkdwn", "text": strings.Join(contextParts, " · ")},
			},
		})
	}

	return blocks
}
