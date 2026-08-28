package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// ── Google (Gemini) adapter ──────────────────────────────────────────────────

type googleGenRequest struct {
	Contents          []googleContent `json:"contents"`
	SystemInstruction *googleContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  *googleGenConfig `json:"generationConfig,omitempty"`
}

type googleContent struct {
	Role  string      `json:"role,omitempty"`
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type googleGenResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []googlePart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// GoogleProvider talks to the Google Generative Language API (Gemini).
type GoogleProvider struct {
	httpProvider
}

// NewGoogle creates a Google Gemini adapter.
func NewGoogle(apiKey, baseURL string) *GoogleProvider {
	return &GoogleProvider{newHTTPProvider("google", apiKey, baseURL)}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) authHeaders() map[string]string {
	return map[string]string{
		"Content-Type":   "application/json",
		"x-goog-api-key": p.apiKey,
	}
}

// toGoogle converts the unified request into the Gemini wire format.
func toGoogle(req *dto.ChatRequest) *googleGenRequest {
	out := &googleGenRequest{}
	if req.MaxTokens != nil || req.Temperature != nil || req.TopP != nil || len(req.Stop) > 0 {
		out.GenerationConfig = &googleGenConfig{
			Temperature:   req.Temperature,
			TopP:          req.TopP,
			StopSequences: req.Stop,
		}
		if req.MaxTokens != nil {
			out.GenerationConfig.MaxOutputTokens = *req.MaxTokens
		}
	}
	for _, m := range req.Messages {
		if m.Role == "system" {
			if out.SystemInstruction == nil {
				out.SystemInstruction = &googleContent{}
			}
			out.SystemInstruction.Parts = append(out.SystemInstruction.Parts, googlePart{Text: m.Content})
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		out.Contents = append(out.Contents, googleContent{
			Role:  role,
			Parts: []googlePart{{Text: m.Content}},
		})
	}
	return out
}

// fromGoogle converts a Gemini response into the unified format.
func fromGoogle(r *googleGenResponse, requestedModel string) *dto.ChatResponse {
	text := ""
	finish := "stop"
	if len(r.Candidates) > 0 {
		var sb strings.Builder
		for _, part := range r.Candidates[0].Content.Parts {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(part.Text)
		}
		text = sb.String()
		switch r.Candidates[0].FinishReason {
		case "MAX_TOKENS":
			finish = "length"
		case "STOP", "":
			finish = "stop"
		default:
			finish = strings.ToLower(r.Candidates[0].FinishReason)
		}
	}
	return &dto.ChatResponse{
		ID:      "",
		Object:  "chat.completion",
		Model:   requestedModel,
		Choices: []dto.Choice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: text},
			FinishReason: finish,
		}},
		Usage: dto.Usage{
			PromptTokens:     r.UsageMetadata.PromptTokenCount,
			CompletionTokens: r.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      r.UsageMetadata.PromptTokenCount + r.UsageMetadata.CandidatesTokenCount,
		},
	}
}

func (p *GoogleProvider) generateURL(model, method string) string {
	return p.baseURL + "/v1beta/models/" + model + ":" + method
}

func (p *GoogleProvider) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	gr := toGoogle(req)
	body, err := json.Marshal(gr)
	if err != nil {
		return nil, err
	}
	resp, err := p.do(ctx, "POST", p.generateURL(req.Model, "generateContent"), body, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out googleGenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("google: decode response: %w", err)
	}
	return fromGoogle(&out, req.Model), nil
}

func (p *GoogleProvider) ChatStream(ctx context.Context, req *dto.ChatRequest) (*StreamResult, error) {
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	return bufferedResult(p.Name(), resp)
}

func (p *GoogleProvider) Models(ctx context.Context) ([]dto.Model, error) {
	resp, err := p.do(ctx, "GET", p.baseURL+"/v1beta/models", nil, p.authHeaders())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var list struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("google: decode models: %w", err)
	}
	out := make([]dto.Model, 0, len(list.Models))
	for _, m := range list.Models {
		id := strings.TrimPrefix(m.Name, "models/")
		out = append(out, dto.Model{ID: id, Object: "model", OwnedBy: "google"})
	}
	return out, nil
}

var _ Provider = (*GoogleProvider)(nil)
