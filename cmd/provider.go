package cmd

import "context"

// LLMProvider defines a provider-agnostic interface for LLM operations.
// Implementations include Ollama (default) and OpenAI-compatible APIs.
type LLMProvider interface {
	// Summarize sends content with a prompt to the LLM and returns the generated summary.
	Summarize(ctx context.Context, content string, prompt string) (string, error)
	// Available checks if the configured model is accessible.
	Available(ctx context.Context) (bool, error)
	// Name returns the provider name for display purposes.
	Name() string
}
