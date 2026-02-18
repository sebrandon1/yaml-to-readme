package cmd

import (
	"context"

	ollama "github.com/ollama/ollama/api"
)

// OllamaProvider implements LLMProvider using the Ollama API.
type OllamaProvider struct {
	client OllamaClient
}

// NewOllamaProvider creates a new OllamaProvider from the environment.
func NewOllamaProvider() (*OllamaProvider, error) {
	client, err := NewRealOllamaClient()
	if err != nil {
		return nil, err
	}
	return &OllamaProvider{client: client}, nil
}

// NewOllamaProviderFromClient creates an OllamaProvider from an existing OllamaClient.
// Used for testing with MockOllamaClient.
func NewOllamaProviderFromClient(client OllamaClient) *OllamaProvider {
	return &OllamaProvider{client: client}
}

// Summarize implements LLMProvider.Summarize using the Ollama Chat API.
func (o *OllamaProvider) Summarize(ctx context.Context, content string, prompt string) (string, error) {
	falseVar := false
	chatReq := &ollama.ChatRequest{
		Model: ModelName,
		Messages: []ollama.Message{
			{
				Role:    "user",
				Content: prompt + content,
			},
		},
		Options: map[string]interface{}{
			"seed": 42,
		},
		Stream: &falseVar,
	}

	var summary string
	err := o.client.Chat(ctx, chatReq, func(resp ollama.ChatResponse) error {
		summary += resp.Message.Content
		return nil
	})
	if err != nil {
		return "", err
	}
	return summary, nil
}

// Available implements LLMProvider.Available by checking the Ollama model list.
func (o *OllamaProvider) Available(ctx context.Context) (bool, error) {
	response, err := o.client.List(ctx)
	if err != nil {
		return false, err
	}
	for _, model := range response.Models {
		if model.Name == ModelName {
			return true, nil
		}
	}
	return false, nil
}

// Name implements LLMProvider.Name.
func (o *OllamaProvider) Name() string {
	return "ollama"
}
