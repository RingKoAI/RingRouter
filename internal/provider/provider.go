package provider

import (
	"context"
	"io"
)

// Provider defines the interface for upstream LLM providers.
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "claude").
	Name() string

	// ChatCompletion proxies a chat completion request to the upstream provider.
	// The body is the raw JSON request body. Returns the response body and status code.
	ChatCompletion(ctx context.Context, body io.Reader, headers map[string]string) (*Response, error)

	// ListModels returns available models from the provider.
	ListModels(ctx context.Context, headers map[string]string) (*Response, error)
}

// Response represents an upstream provider response.
type Response struct {
	StatusCode int
	Header     map[string][]string
	Body       io.ReadCloser
	IsStream   bool
}