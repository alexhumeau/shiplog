package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// openaiCompatProvider works with any API that follows the OpenAI chat completions format.
// This covers: OpenAI, Mistral, DeepSeek, Groq, xAI (Grok), Together AI, Fireworks AI.
type openaiCompatProvider struct {
	name   string // for error messages
	apiURL string
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey, model string) Provider {
	return newOpenAICompat("OpenAI", "https://api.openai.com/v1/chat/completions", apiKey, model)
}

// NewMistral creates a new Mistral AI provider.
func NewMistral(apiKey, model string) Provider {
	return newOpenAICompat("Mistral", "https://api.mistral.ai/v1/chat/completions", apiKey, model)
}

// NewDeepSeek creates a new DeepSeek provider.
func NewDeepSeek(apiKey, model string) Provider {
	return newOpenAICompat("DeepSeek", "https://api.deepseek.com/v1/chat/completions", apiKey, model)
}

// NewGroq creates a new Groq provider.
func NewGroq(apiKey, model string) Provider {
	return newOpenAICompat("Groq", "https://api.groq.com/openai/v1/chat/completions", apiKey, model)
}

// NewXAI creates a new xAI (Grok) provider.
func NewXAI(apiKey, model string) Provider {
	return newOpenAICompat("xAI", "https://api.x.ai/v1/chat/completions", apiKey, model)
}

// NewTogether creates a new Together AI provider.
func NewTogether(apiKey, model string) Provider {
	return newOpenAICompat("Together", "https://api.together.xyz/v1/chat/completions", apiKey, model)
}

// NewFireworks creates a new Fireworks AI provider.
func NewFireworks(apiKey, model string) Provider {
	return newOpenAICompat("Fireworks", "https://api.fireworks.ai/inference/v1/chat/completions", apiKey, model)
}

func newOpenAICompat(name, apiURL, apiKey, model string) Provider {
	return &openaiCompatProvider{
		name:   name,
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *openaiCompatProvider) Complete(prompt string) (string, error) {
	reqBody := openaiRequest{
		Model: o.model,
		Messages: []openaiMessage{
			{Role: "user", Content: prompt},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", o.apiURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling %s API: %w", o.name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s API error (status %d): %s", o.name, resp.StatusCode, string(body))
	}

	var result openaiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("%s API error: %s", o.name, result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from %s API", o.name)
	}

	return result.Choices[0].Message.Content, nil
}
