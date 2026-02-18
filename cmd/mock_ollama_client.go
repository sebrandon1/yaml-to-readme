package cmd

import (
	"context"
	"strings"

	ollama "github.com/ollama/ollama/api"
)

// MockOllamaClient is a mock implementation of OllamaClient for testing.
type MockOllamaClient struct {
	// Map of file content snippets to mock summaries
	MockResponses map[string]string
	// Default response if no match is found
	DefaultResponse string
	// Available models to return from List()
	AvailableModels []string
}

// NewMockOllamaClient creates a new MockOllamaClient with default responses.
func NewMockOllamaClient() *MockOllamaClient {
	return &MockOllamaClient{
		MockResponses:   make(map[string]string),
		DefaultResponse: "This is a mock summary for testing purposes.",
		AvailableModels: []string{DefaultModelName},
	}
}

// Chat implements OllamaClient.Chat for the mock.
func (m *MockOllamaClient) Chat(ctx context.Context, req *ollama.ChatRequest, fn func(ollama.ChatResponse) error) error {
	// Extract the YAML content from the request
	var content string
	if len(req.Messages) > 0 {
		content = req.Messages[0].Content
	}

	// Find a matching mock response based on content
	summary := m.DefaultResponse
	for key, response := range m.MockResponses {
		if strings.Contains(content, key) {
			summary = response
			break
		}
	}

	// Call the callback function with the mock response
	resp := ollama.ChatResponse{
		Message: ollama.Message{
			Content: summary,
		},
	}
	return fn(resp)
}

// List implements OllamaClient.List for the mock.
func (m *MockOllamaClient) List(ctx context.Context) (*ollama.ListResponse, error) {
	models := make([]ollama.ListModelResponse, len(m.AvailableModels))
	for i, modelName := range m.AvailableModels {
		models[i] = ollama.ListModelResponse{
			Name: modelName,
		}
	}
	return &ollama.ListResponse{
		Models: models,
	}, nil
}
