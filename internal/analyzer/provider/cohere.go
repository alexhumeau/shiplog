package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type cohereProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewCohere creates a new Cohere provider.
func NewCohere(apiKey, model string) Provider {
	return &cohereProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type cohereRequest struct {
	Model   string          `json:"model"`
	Messages []cohereMessage `json:"messages"`
}

type cohereMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type cohereResponse struct {
	Message *struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	Error *string `json:"error,omitempty"`
}

func (c *cohereProvider) Complete(prompt string) (string, error) {
	reqBody := cohereRequest{
		Model: c.model,
		Messages: []cohereMessage{
			{Role: "user", Content: prompt},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.cohere.com/v2/chat", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Cohere API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cohere API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result cohereResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("Cohere API error: %s", *result.Error)
	}

	if result.Message == nil || len(result.Message.Content) == 0 {
		return "", fmt.Errorf("empty response from Cohere API")
	}

	return result.Message.Content[0].Text, nil
}
