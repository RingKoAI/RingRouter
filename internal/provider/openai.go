package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements Provider for OpenAI-compatible APIs.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, body io.Reader, headers map[string]string) (*Response, error) {
	url := p.baseURL + "/chat/completions"
	return p.doRequest(ctx, "POST", url, body, headers)
}

func (p *OpenAIProvider) ListModels(ctx context.Context, headers map[string]string) (*Response, error) {
	url := p.baseURL + "/models"
	return p.doRequest(ctx, "GET", url, nil, headers)
}

func (p *OpenAIProvider) doRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Forward relevant headers from client
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	isStream := false
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/event-stream") {
		isStream = true
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       resp.Body,
		IsStream:   isStream,
	}, nil
}

// Ensure OpenAIProvider implements Provider.
var _ Provider = (*OpenAIProvider)(nil)