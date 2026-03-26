package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrehumeau/shiplog/internal/config"
	"github.com/alexandrehumeau/shiplog/internal/model"
)

const notionAPIBase = "https://api.notion.com/v1"
const notionVersion = "2022-06-28"

// NotionEntry is a simplified representation of a Notion database entry.
type NotionEntry struct {
	Title string
	Type  string
}

// WriteAll creates Notion pages for each entry, with deduplication.
func WriteAll(token, dbID string, entries []model.ChangeEntry, props config.PropertiesConfig) error {
	// Collect all SHAs for dedup check
	var allSHAs []string
	for _, e := range entries {
		for _, c := range e.Commits {
			allSHAs = append(allSHAs, c.SHA)
		}
	}

	existingSHAs, err := queryExistingSHAs(token, dbID, allSHAs)
	if err != nil {
		fmt.Printf("warning: could not check for duplicates: %v\n", err)
		existingSHAs = make(map[string]bool)
	}

	created := 0
	for _, entry := range entries {
		// Check if all commits in this entry are already tracked
		allExist := true
		for _, c := range entry.Commits {
			if !existingSHAs[c.SHA] {
				allExist = false
				break
			}
		}
		if allExist && len(entry.Commits) > 0 {
			fmt.Printf("  skip (already exists): %s\n", entry.Title)
			continue
		}

		if err := createPage(token, dbID, entry, props); err != nil {
			return fmt.Errorf("creating page for %q: %w", entry.Title, err)
		}
		created++
		fmt.Printf("  created: [%s] %s\n", entry.Type, entry.Title)
	}

	fmt.Printf("\n%d entries created in Notion\n", created)
	return nil
}

// QueryRecentEntries fetches the last N entries from the database (title + type only).
func QueryRecentEntries(token, dbID string, limit int) ([]NotionEntry, error) {
	body := map[string]interface{}{
		"page_size": limit,
		"sorts": []map[string]string{
			{"property": "Date", "direction": "descending"},
		},
	}

	data, _ := json.Marshal(body)
	resp, err := notionRequest(token, "POST", fmt.Sprintf("/databases/%s/query", dbID), data)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"results"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parsing query response: %w", err)
	}

	var entries []NotionEntry
	for _, r := range result.Results {
		entry := NotionEntry{}
		// Extract title
		if titleRaw, ok := r.Properties["Title"]; ok {
			entry.Title = extractTitle(titleRaw)
		}
		// Extract type
		if typeRaw, ok := r.Properties["Type"]; ok {
			entry.Type = extractSelect(typeRaw)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func queryExistingSHAs(token, dbID string, shas []string) (map[string]bool, error) {
	// Query entries that might contain our SHAs
	// We search the Commits property for each SHA
	// Notion filter: "Commits" rich_text contains any SHA
	if len(shas) == 0 {
		return make(map[string]bool), nil
	}

	// Build OR filter for first 10 SHAs (Notion has filter limits)
	searchSHAs := shas
	if len(searchSHAs) > 10 {
		searchSHAs = searchSHAs[:10]
	}

	var orFilters []map[string]interface{}
	for _, sha := range searchSHAs {
		orFilters = append(orFilters, map[string]interface{}{
			"property": "Commits",
			"rich_text": map[string]string{
				"contains": sha[:8], // use short SHA for matching
			},
		})
	}

	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"or": orFilters,
		},
	}

	data, _ := json.Marshal(body)
	resp, err := notionRequest(token, "POST", fmt.Sprintf("/databases/%s/query", dbID), data)
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"results"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	existing := make(map[string]bool)
	for _, r := range result.Results {
		if commitsRaw, ok := r.Properties["Commits"]; ok {
			text := extractRichText(commitsRaw)
			for _, sha := range shas {
				if strings.Contains(text, sha[:8]) {
					existing[sha] = true
				}
			}
		}
	}

	return existing, nil
}

func createPage(token, dbID string, entry model.ChangeEntry, props config.PropertiesConfig) error {
	commitSHAs := make([]string, len(entry.Commits))
	for i, c := range entry.Commits {
		commitSHAs[i] = c.SHA[:8]
	}

	page := map[string]interface{}{
		"parent": map[string]string{
			"database_id": dbID,
		},
		"properties": buildProperties(entry, props, commitSHAs),
		"children":   buildChildren(entry, commitSHAs),
	}

	data, _ := json.Marshal(page)

	// Retry with backoff on 429
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}

		_, err := notionRequest(token, "POST", "/pages", data)
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "429") {
			lastErr = err
			continue
		}
		return err
	}
	return fmt.Errorf("rate limited after 3 attempts: %w", lastErr)
}

func buildProperties(entry model.ChangeEntry, props config.PropertiesConfig, commitSHAs []string) map[string]interface{} {
	return map[string]interface{}{
		props.Title: map[string]interface{}{
			"title": []map[string]interface{}{
				{"text": map[string]string{"content": entry.Title}},
			},
		},
		props.Type: map[string]interface{}{
			"select": map[string]string{"name": entry.Type},
		},
		props.Date: map[string]interface{}{
			"date": map[string]string{"start": entry.Date.Format("2006-01-02")},
		},
		props.Branch: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"text": map[string]string{"content": entry.Branch}},
			},
		},
		props.Commits: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"text": map[string]string{"content": strings.Join(commitSHAs, ", ")}},
			},
		},
		props.Files: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"text": map[string]string{"content": strings.Join(entry.Files, ", ")}},
			},
		},
	}
}

func buildChildren(entry model.ChangeEntry, commitSHAs []string) []map[string]interface{} {
	var children []map[string]interface{}

	// Description
	if entry.Description != "" {
		children = append(children, paragraph(entry.Description))
	}

	// Commits section
	children = append(children, heading3("Commits"))
	for i, c := range entry.Commits {
		sha := commitSHAs[i]
		children = append(children, bulletItem(fmt.Sprintf("%s — %s", sha, c.Message)))
	}

	// Files section
	if len(entry.Files) > 0 {
		children = append(children, heading3("Files Changed"))
		for _, f := range entry.Files {
			children = append(children, bulletItem(f))
		}
	}

	return children
}

func heading3(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "heading_3",
		"heading_3": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": text}},
			},
		},
	}
}

func paragraph(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "paragraph",
		"paragraph": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": text}},
			},
		},
	}
}

func bulletItem(text string) map[string]interface{} {
	return map[string]interface{}{
		"object": "block",
		"type":   "bulleted_list_item",
		"bulleted_list_item": map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{"type": "text", "text": map[string]string{"content": text}},
			},
		},
	}
}

func notionRequest(token, method, path string, body []byte) ([]byte, error) {
	url := notionAPIBase + path
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Notion API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func extractTitle(raw json.RawMessage) string {
	var prop struct {
		Title []struct {
			PlainText string `json:"plain_text"`
		} `json:"title"`
	}
	json.Unmarshal(raw, &prop)
	if len(prop.Title) > 0 {
		return prop.Title[0].PlainText
	}
	return ""
}

func extractSelect(raw json.RawMessage) string {
	var prop struct {
		Select *struct {
			Name string `json:"name"`
		} `json:"select"`
	}
	json.Unmarshal(raw, &prop)
	if prop.Select != nil {
		return prop.Select.Name
	}
	return ""
}

func extractRichText(raw json.RawMessage) string {
	var prop struct {
		RichText []struct {
			PlainText string `json:"plain_text"`
		} `json:"rich_text"`
	}
	json.Unmarshal(raw, &prop)
	var parts []string
	for _, t := range prop.RichText {
		parts = append(parts, t.PlainText)
	}
	return strings.Join(parts, "")
}
