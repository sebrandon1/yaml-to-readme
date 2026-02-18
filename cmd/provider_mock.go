package cmd

import (
	"context"
	"strings"
)

// MockLLMProvider is a mock implementation of LLMProvider for testing.
type MockLLMProvider struct {
	// MockResponses maps content snippets to mock summaries.
	MockResponses map[string]string
	// DefaultResponse is returned when no matching snippet is found.
	DefaultResponse string
	// ModelAvailable controls the return value of Available().
	ModelAvailable bool
}

// NewMockLLMProvider creates a new MockLLMProvider with default settings.
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		MockResponses:   make(map[string]string),
		DefaultResponse: "This is a mock summary for testing purposes.",
		ModelAvailable:  true,
	}
}

// Summarize implements LLMProvider.Summarize for the mock.
func (m *MockLLMProvider) Summarize(ctx context.Context, content string, prompt string) (string, error) {
	for key, response := range m.MockResponses {
		if strings.Contains(content, key) {
			return response, nil
		}
	}
	return m.DefaultResponse, nil
}

// Available implements LLMProvider.Available for the mock.
func (m *MockLLMProvider) Available(ctx context.Context) (bool, error) {
	return m.ModelAvailable, nil
}

// Name implements LLMProvider.Name for the mock.
func (m *MockLLMProvider) Name() string {
	return "mock"
}
