package cmd

import (
	"context"

	ollama "github.com/ollama/ollama/api"
)

// OllamaClient defines the interface for interacting with Ollama.
// This allows us to mock the client for testing purposes.
type OllamaClient interface {
	Chat(ctx context.Context, req *ollama.ChatRequest, fn func(ollama.ChatResponse) error) error
	List(ctx context.Context) (*ollama.ListResponse, error)
}

// RealOllamaClient is a wrapper around the actual Ollama client that implements OllamaClient.
type RealOllamaClient struct {
	client *ollama.Client
}

// NewRealOllamaClient creates a new RealOllamaClient from the environment.
func NewRealOllamaClient() (*RealOllamaClient, error) {
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return &RealOllamaClient{client: client}, nil
}

// Chat implements OllamaClient.Chat
func (r *RealOllamaClient) Chat(ctx context.Context, req *ollama.ChatRequest, fn func(ollama.ChatResponse) error) error {
	return r.client.Chat(ctx, req, fn)
}

// List implements OllamaClient.List
func (r *RealOllamaClient) List(ctx context.Context) (*ollama.ListResponse, error) {
	return r.client.List(ctx)
}
