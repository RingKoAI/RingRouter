package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// httpProvider is the shared HTTP plumbing for vendor adapters.
type httpProvider struct {
	name       string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func newHTTPProvider(name, apiKey, baseURL string) httpProvider {
	return httpProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// do sends a request and returns the raw response body. The caller owns
// closing resp.Body; on non-2xx it returns an error carrying the status.
func (p httpProvider) do(ctx context.Context, method, url string, body []byte, hdr map[string]string) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", p.name, err)
	}
	if hdr != nil {
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", p.name, err)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, &UpstreamError{Provider: p.name, Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	return resp, nil
}

// UpstreamError reports a non-2xx upstream response.
type UpstreamError struct {
	Provider string
	Status   int
	Body     string
}

func (e *UpstreamError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s upstream returned %d", e.Provider, e.Status)
	}
	return fmt.Sprintf("%s upstream returned %d: %s", e.Provider, e.Status, e.Body)
}

// ── OpenAI adapter ───────────────────────────────────────────────────────────

// OpenAIProvider talks to any OpenAI-compatible endpoint.
type OpenAIProvider struct {
	httpProvider
}

// NewOpenAI creates an OpenAI-compatible adapter.
func NewOpenAI(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{newHTTPProvider("openai", apiKey, baseURL)}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) authHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + p.apiKey,
		"Content-Type":  "application/json",
	}
}

func (p *OpenAIProvider) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.baseURL+"/chat/completions", body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out dto.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}
	return &out, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req *dto.ChatRequest) (*StreamResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.baseURL+"/chat/completions", body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	return &StreamResult{
		Body:        resp.Body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (p *OpenAIProvider) Models(ctx context.Context) ([]dto.Model, error) {
	resp, err := p.do(ctx, "GET", p.baseURL+"/models", nil, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list dto.ModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("openai: decode models: %w", err)
	}
	return list.Data, nil
}

var _ Provider = (*OpenAIProvider)(nil)