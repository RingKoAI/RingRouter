package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// ── OpenAI Responses API adapter ─────────────────────────────────────────────

// openaiResponsesRequest is the upstream wire format for POST /v1/responses.
type openaiResponsesRequest struct {
	Model       string          `json:"model"`
	Input       json.RawMessage `json:"input"`
	Instructions string         `json:"instructions,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	MaxTokens   *int            `json:"max_output_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

// openaiResponsesResponse is the upstream wire format returned by POST /v1/responses.
type openaiResponsesResponse struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Output []struct {
		Type    string `json:"type"`
		Content string `json:"content,omitempty"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIResponsesProvider talks to the OpenAI Responses API (POST /v1/responses).
type OpenAIResponsesProvider struct {
	httpProvider
}

// NewOpenAIResponses creates a Responses API adapter.
func NewOpenAIResponses(apiKey, baseURL string) *OpenAIResponsesProvider {
	return &OpenAIResponsesProvider{newHTTPProvider("openai-responses", apiKey, baseURL)}
}

func (p *OpenAIResponsesProvider) Name() string { return "openai-responses" }

func (p *OpenAIResponsesProvider) authHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + p.apiKey,
		"Content-Type":  "application/json",
	}
}

// toResponses converts a unified ChatRequest into the Responses wire format.
func toResponses(req *dto.ChatRequest) *openaiResponsesRequest {
	out := &openaiResponsesRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
	}

	// Build instructions from system messages; everything else becomes input blocks.
	var instructions []string
	var inputBlocks []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			instructions = append(instructions, m.Content)
		default:
			inputBlocks = append(inputBlocks, struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{Role: m.Role, Content: m.Content})
		}
	}

	if len(instructions) > 0 {
		out.Instructions = stringsJoin(instructions, "\n")
	}

	input, _ := json.Marshal(inputBlocks)
	out.Input = input

	return out
}

// fromResponses converts a Responses API result into the unified ChatResponse.
func fromResponses(r *openaiResponsesResponse, requestedModel string) *dto.ChatResponse {
	text := ""
	finish := "stop"
	for _, o := range r.Output {
		if o.Type == "output_text" {
			text += o.Content
		}
	}
	if text == "" && len(r.Output) > 0 {
		text = r.Output[0].Content
	}

	return &dto.ChatResponse{
		ID:      r.ID,
		Object:  "response",
		Model:   requestedModel,
		Choices: []dto.Choice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: text},
			FinishReason: finish,
		}},
		Usage: dto.Usage{
			PromptTokens:     r.Usage.InputTokens,
			CompletionTokens: r.Usage.OutputTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}
}

func (p *OpenAIResponsesProvider) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	rr := toResponses(req)
	rr.Stream = false
	body, err := json.Marshal(rr)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.baseURL+"/responses", body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out openaiResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("openai-responses: decode response: %w", err)
	}
	return fromResponses(&out, req.Model), nil
}

func (p *OpenAIResponsesProvider) ChatStream(ctx context.Context, req *dto.ChatRequest) (*StreamResult, error) {
	rr := toResponses(req)
	rr.Stream = true
	body, err := json.Marshal(rr)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.baseURL+"/responses", body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	return &StreamResult{
		Body:        resp.Body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func (p *OpenAIResponsesProvider) Models(ctx context.Context) ([]dto.Model, error) {
	resp, err := p.do(ctx, "GET", p.baseURL+"/models", nil, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list dto.ModelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("openai-responses: decode models: %w", err)
	}
	return list.Data, nil
}

var _ Provider = (*OpenAIResponsesProvider)(nil)

// stringsJoin is a helper to avoid importing strings here.
func stringsJoin(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
