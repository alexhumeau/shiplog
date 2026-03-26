package provider

import "fmt"

// Provider is the interface for LLM API clients.
type Provider interface {
	Complete(prompt string) (string, error)
}

// New creates a provider based on the provider name.
func New(providerName, apiKey, model string) (Provider, error) {
	switch providerName {
	case "anthropic":
		return NewAnthropic(apiKey, model), nil
	case "openai":
		return NewOpenAI(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %q (supported: anthropic, openai)", providerName)
	}
}
