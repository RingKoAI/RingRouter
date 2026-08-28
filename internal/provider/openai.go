package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// ── OpenAI Chat Completions adapter ──────────────────────────────────────────

// OpenAIProvider talks to any OpenAI-compatible chat completions endpoint.
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
