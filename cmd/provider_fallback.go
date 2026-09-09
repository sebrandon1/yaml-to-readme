package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// FallbackProvider tries a list of providers in order, returning the first successful result.
type FallbackProvider struct {
	providers []LLMProvider
}

// Summarize implements LLMProvider.Summarize by trying each provider in order.
func (f *FallbackProvider) Summarize(ctx context.Context, content string, prompt string) (string, error) {
	var lastErr error
	for _, p := range f.providers {
		summary, err := p.Summarize(ctx, content, prompt)
		if err == nil {
			return summary, nil
		}
		slog.Warn("provider failed, trying next in chain", "provider", p.Name(), "error", err)
		lastErr = err
	}
	return "", fmt.Errorf("all providers in chain failed; last error: %w", lastErr)
}

// Available implements LLMProvider.Available by returning true if any provider has the model.
func (f *FallbackProvider) Available(ctx context.Context) (bool, error) {
	for _, p := range f.providers {
		ok, err := p.Available(ctx)
		if err == nil && ok {
			return true, nil
		}
	}
	return false, nil
}

// Name implements LLMProvider.Name.
func (f *FallbackProvider) Name() string {
	names := make([]string, len(f.providers))
	for i, p := range f.providers {
		names[i] = p.Name()
	}
	return strings.Join(names, ",")
}
