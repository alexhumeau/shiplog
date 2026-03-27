package provider

import "fmt"

// Provider is the interface for LLM API clients.
type Provider interface {
	Complete(prompt string) (string, error)
}

// Supported lists all supported provider names.
var Supported = []string{
	"anthropic", "openai", "gemini", "mistral", "deepseek",
	"groq", "xai", "cohere", "together", "fireworks",
}

// New creates a provider based on the provider name.
func New(providerName, apiKey, model string) (Provider, error) {
	switch providerName {
	case "anthropic":
		return NewAnthropic(apiKey, model), nil
	case "openai":
		return NewOpenAI(apiKey, model), nil
	case "gemini":
		return NewGemini(apiKey, model), nil
	case "mistral":
		return NewMistral(apiKey, model), nil
	case "deepseek":
		return NewDeepSeek(apiKey, model), nil
	case "groq":
		return NewGroq(apiKey, model), nil
	case "xai":
		return NewXAI(apiKey, model), nil
	case "cohere":
		return NewCohere(apiKey, model), nil
	case "together":
		return NewTogether(apiKey, model), nil
	case "fireworks":
		return NewFireworks(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unknown AI provider: %q\nSupported: anthropic, openai, gemini, mistral, deepseek, groq, xai, cohere, together, fireworks", providerName)
	}
}
