package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// ── Anthropic (Claude) adapter ───────────────────────────────────────────────

// anthropicRequest is the /v1/messages wire format.
type anthropicRequest struct {
	Model         string            `json:"model"`
	MaxTokens     int               `json:"max_tokens"`
	Messages      []anthropicOutMsg `json:"messages"`
	System        string            `json:"system,omitempty"`
	Stream        bool              `json:"stream,omitempty"`
	Temperature   *float64          `json:"temperature,omitempty"`
	TopP          *float64          `json:"top_p,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
}

type anthropicOutMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

const anthropicDefaultMaxTokens = 4096

// AnthropicProvider talks to the Anthropic Messages API.
type AnthropicProvider struct {
	httpProvider
}

// NewAnthropic creates an Anthropic adapter.
func NewAnthropic(apiKey, baseURL string) *AnthropicProvider {
	return &AnthropicProvider{newHTTPProvider("anthropic", apiKey, baseURL)}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) authHeaders() map[string]string {
	return map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}
}

// toAnthropic converts the unified request into the Anthropic wire format.
func toAnthropic(req *dto.ChatRequest) *anthropicRequest {
	out := &anthropicRequest{
		Model:         req.Model,
		Stream:        req.Stream,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		StopSequences: req.Stop,
		MaxTokens:     anthropicDefaultMaxTokens,
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		out.MaxTokens = *req.MaxTokens
	}
	for _, m := range req.Messages {
		if m.Role == "system" {
			if out.System != "" {
				out.System += "\n"
			}
			out.System += m.Content
			continue
		}
		out.Messages = append(out.Messages, anthropicOutMsg{Role: m.Role, Content: m.Content})
	}
	return out
}

// fromAnthropic converts an Anthropic response into the unified format.
func fromAnthropic(r *anthropicResponse) *dto.ChatResponse {
	text := ""
	for _, b := range r.Content {
		if b.Type == "text" {
			if text != "" {
				text += "\n"
			}
			text += b.Text
		}
	}
	finish := "stop"
	switch r.StopReason {
	case "max_tokens":
		finish = "length"
	case "end_turn", "":
		finish = "stop"
	default:
		finish = r.StopReason
	}
	return &dto.ChatResponse{
		ID:     r.ID,
		Object: "chat.completion",
		Model:  r.Model,
		Choices: []dto.Choice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: text},
			FinishReason: finish,
		}},
		Usage: dto.Usage{
			PromptTokens:     r.Usage.InputTokens,
			CompletionTokens: r.Usage.OutputTokens,
			TotalTokens:      r.Usage.InputTokens + r.Usage.OutputTokens,
		},
	}
}

func (p *AnthropicProvider) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	ar := toAnthropic(req)
	ar.Stream = false
	body, err := json.Marshal(ar)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.baseURL+"/v1/messages", body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ar2 anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar2); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}
	return fromAnthropic(&ar2), nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req *dto.ChatRequest) (*StreamResult, error) {
	// Cross-format stream translation is not implemented yet: fall back to a
	// buffered (non-stream) upstream call. The handler converts the result
	// into the client's inbound SSE format.
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	return bufferedResult(p.Name(), resp)
}

func (p *AnthropicProvider) Models(ctx context.Context) ([]dto.Model, error) {
	resp, err := p.do(ctx, "GET", p.baseURL+"/v1/models", nil, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("anthropic: decode models: %w", err)
	}
	out := make([]dto.Model, 0, len(list.Data))
	for _, m := range list.Data {
		out = append(out, dto.Model{ID: m.ID, Object: "model", OwnedBy: "anthropic"})
	}
	return out, nil
}

var _ Provider = (*AnthropicProvider)(nil)
