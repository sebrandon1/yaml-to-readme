package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

// OpenAIProvider implements LLMProvider using the OpenAI-compatible chat completions API.
// Works with OpenAI, Azure OpenAI, vLLM, llama.cpp server, and other compatible endpoints.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAIProvider from environment variables.
// Requires OPENAI_API_KEY. OPENAI_BASE_URL defaults to https://api.openai.com.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required for the openai provider")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIError struct {
	Message string `json:"message"`
}

type openAIModelList struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID string `json:"id"`
}

const maxRetries = 3

// retryDelay returns the duration to wait before the next attempt.
// It respects the Retry-After header when present, falling back to exponential backoff.
func retryDelay(attempt int, retryAfter string) time.Duration {
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s
}

// Summarize implements LLMProvider.Summarize using the OpenAI chat completions API.
// Retries up to maxRetries times on HTTP 429 with exponential backoff or Retry-After.
func (o *OpenAIProvider) Summarize(ctx context.Context, content string, prompt string) (string, error) {
	reqBody := openAIChatRequest{
		Model: ModelName,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt + content,
			},
		},
		Temperature: 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := o.baseURL + "/v1/chat/completions"

	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+o.apiKey)

		resp, err := o.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("openai API request failed: %w", err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("failed to read response: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries-1 {
			delay := retryDelay(attempt, resp.Header.Get("Retry-After"))
			slog.Warn("openai rate limit hit, retrying", "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("openai API returned status %d: %s", resp.StatusCode, string(respBody))
		}

		var chatResp openAIChatResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		if chatResp.Error != nil {
			return "", fmt.Errorf("openai API error: %s", chatResp.Error.Message)
		}

		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("openai API returned no choices")
		}

		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("openai API rate limit exceeded after %d retries", maxRetries)
}

// Available implements LLMProvider.Available by checking the OpenAI models endpoint.
func (o *OpenAIProvider) Available(ctx context.Context) (bool, error) {
	url := o.baseURL + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("openai API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	var modelList openAIModelList
	if err := json.Unmarshal(respBody, &modelList); err != nil {
		return false, fmt.Errorf("failed to parse models response: %w", err)
	}

	for _, model := range modelList.Data {
		if model.ID == ModelName {
			return true, nil
		}
	}
	return false, nil
}

// Name implements LLMProvider.Name.
func (o *OpenAIProvider) Name() string {
	return "openai"
}
