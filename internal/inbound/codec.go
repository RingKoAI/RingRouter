// Package inbound defines the inbound protocol adapters. Each adapter decodes
// a client-side wire format into the unified dto.ChatRequest and encodes
// unified results back into that format, so any native client (OpenAI,
// Anthropic, Google) can talk to RingRouter directly.
package inbound

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// Format enumerates supported inbound wire protocols.
type Format string

const (
	FormatOpenAI      Format = "openai"
	FormatResponses   Format = "responses"
	FormatAnthropic   Format = "anthropic"
	FormatGoogle      Format = "google"
)

// DetectFormat infers the inbound format from the request path.
func DetectFormat(path string) Format {
	switch {
	case strings.Contains(path, "/v1/messages"):
		return FormatAnthropic
	case strings.Contains(path, "/v1beta/"), strings.Contains(path, "/gemini/"):
		return FormatGoogle
	case strings.Contains(path, "/v1/responses"):
		return FormatResponses
	default:
		return FormatOpenAI
	}
}

// Decoder turns a raw inbound body into the unified request.
type Decoder interface {
	Decode(body []byte) (*dto.ChatRequest, error)
}

// Encoder turns unified output back into the inbound format.
type Encoder interface {
	EncodeChat(resp *dto.ChatResponse) ([]byte, error)
	EncodeError(status int, errType, msg string) []byte
	ContentType() string
}

// Codec bundles decode + encode for one format.
type Codec interface {
	Format() Format
	Decoder
	Encoder
}

// ── OpenAI codec ─────────────────────────────────────────────────────────────

type openaiCodec struct{}

// NewOpenAI returns the OpenAI wire codec.
func NewOpenAI() Codec { return openaiCodec{} }

func (openaiCodec) Format() Format { return FormatOpenAI }

func (openaiCodec) Decode(body []byte) (*dto.ChatRequest, error) {
	var req dto.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid openai request: %w", err)
	}
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages is required")
	}
	return &req, nil
}

func (openaiCodec) EncodeChat(resp *dto.ChatResponse) ([]byte, error) {
	resp.Object = "chat.completion"
	for i := range resp.Choices {
		resp.Choices[i].Index = i
	}
	return json.Marshal(resp)
}

func (openaiCodec) EncodeError(status int, errType, msg string) []byte {
	b, _ := json.Marshal(dto.ErrorBody{Error: dto.ErrorDetail{
		Message: msg,
		Type:    errType,
	}})
	return b
}

func (openaiCodec) ContentType() string { return "application/json" }

// ── Anthropic codec ──────────────────────────────────────────────────────────

// anthropicRequest is the Anthropic /v1/messages wire format.
type anthropicRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []anthropicMsg  `json:"messages"`
	System    json.RawMessage `json:"system,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
	Temp      *float64        `json:"temperature,omitempty"`
	TopP      *float64        `json:"top_p,omitempty"`
	Stop      []string        `json:"stop_sequences,omitempty"`
}

type anthropicMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type anthropicCodec struct{}

// NewAnthropic returns the Anthropic wire codec.
func NewAnthropic() Codec { return anthropicCodec{} }

func (anthropicCodec) Format() Format { return FormatAnthropic }

func (anthropicCodec) Decode(body []byte) (*dto.ChatRequest, error) {
	var ar anthropicRequest
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("invalid anthropic request: %w", err)
	}
	if ar.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(ar.Messages) == 0 {
		return nil, fmt.Errorf("messages is required")
	}

	req := &dto.ChatRequest{
		Model:       ar.Model,
		Stream:      ar.Stream,
		Temperature: ar.Temp,
		TopP:        ar.TopP,
		Stop:        ar.Stop,
		User:        "",
	}
	if ar.MaxTokens > 0 {
		mt := ar.MaxTokens
		req.MaxTokens = &mt
	}

	// System prompt: string or content blocks.
	if len(ar.System) > 0 {
		if sys := extractText(ar.System); sys != "" {
			req.Messages = append([]dto.Message{{Role: "system", Content: sys}}, req.Messages...)
		}
	}

	for _, m := range ar.Messages {
		req.Messages = append(req.Messages, dto.Message{
			Role:    m.Role,
			Content: extractText(m.Content),
		})
	}
	return req, nil
}

// extractText handles both "string" and [{"type":"text","text":...}] shapes.
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if b.Type == "text" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(b.Text)
			}
		}
		return sb.String()
	}
	return ""
}

func (anthropicCodec) EncodeChat(resp *dto.ChatResponse) ([]byte, error) {
	type aMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type aResp struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Model        string `json:"model"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason  string `json:"stop_reason,omitempty"`
		Usage       struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	out := aResp{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
	}
	text := ""
	if len(resp.Choices) > 0 {
		text = resp.Choices[0].Message.Content
		switch resp.Choices[0].FinishReason {
		case "length":
			out.StopReason = "max_tokens"
		case "stop", "":
			out.StopReason = "end_turn"
		default:
			out.StopReason = resp.Choices[0].FinishReason
		}
	}
	if text != "" {
		out.Content = append(out.Content, struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{Type: "text", Text: text})
	} else {
		out.Content = []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{}
	}
	out.Usage.InputTokens = resp.Usage.PromptTokens
	out.Usage.OutputTokens = resp.Usage.CompletionTokens

	return json.Marshal(out)
}

func (anthropicCodec) EncodeError(status int, errType, msg string) []byte {
	type aErr struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var e aErr
	e.Type = "error"
	e.Error.Type = errType
	e.Error.Message = msg
	b, _ := json.Marshal(e)
	return b
}

func (anthropicCodec) ContentType() string { return "application/json" }

// ── Google (Gemini) codec ────────────────────────────────────────────────────

// googleRequest is the Gemini generateContent wire format.
type googleRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	SystemInstruction *struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"systemInstruction,omitempty"`
	GenerationConfig *struct {
		Temperature *float64 `json:"temperature,omitempty"`
		TopP        *float64 `json:"topP,omitempty"`
		MaxTokens   int      `json:"maxOutputTokens,omitempty"`
		StopSeqs    []string `json:"stopSequences,omitempty"`
	} `json:"generationConfig,omitempty"`
}

type googleCodec struct{}

// NewGoogle returns the Google Gemini wire codec.
func NewGoogle() Codec { return googleCodec{} }

func (googleCodec) Format() Format { return FormatGoogle }

func (googleCodec) Decode(body []byte) (*dto.ChatRequest, error) {
	var gr googleRequest
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, fmt.Errorf("invalid google request: %w", err)
	}
	if len(gr.Contents) == 0 {
		return nil, fmt.Errorf("contents is required")
	}

	req := &dto.ChatRequest{}

	// Model comes from the URL path for Gemini; set later by the handler.
	if gr.SystemInstruction != nil {
		sys := ""
		for _, p := range gr.SystemInstruction.Parts {
			sys += p.Text
		}
		if sys != "" {
			req.Messages = append(req.Messages, dto.Message{Role: "system", Content: sys})
		}
	}

	for _, c := range gr.Contents {
		role := c.Role
		if role == "model" {
			role = "assistant"
		}
		text := ""
		for _, p := range c.Parts {
			if p.Text != "" {
				if text != "" {
					text += "\n"
				}
				text += p.Text
			}
		}
		req.Messages = append(req.Messages, dto.Message{Role: role, Content: text})
	}

	if gr.GenerationConfig != nil {
		req.Temperature = gr.GenerationConfig.Temperature
		req.TopP = gr.GenerationConfig.TopP
		if gr.GenerationConfig.MaxTokens > 0 {
			mt := gr.GenerationConfig.MaxTokens
			req.MaxTokens = &mt
		}
		req.Stop = gr.GenerationConfig.StopSeqs
	}
	return req, nil
}

func (googleCodec) EncodeChat(resp *dto.ChatResponse) ([]byte, error) {
	type gPart struct {
		Text string `json:"text"`
	}
	type gContent struct {
		Role  string  `json:"role,omitempty"`
		Parts []gPart `json:"parts"`
	}
	type gResp struct {
		Candidates []struct {
			Content       gContent `json:"content"`
			FinishReason  string   `json:"finishReason"`
			Index         int      `json:"index"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	var out gResp
	text := ""
	finish := "STOP"
	if len(resp.Choices) > 0 {
		text = resp.Choices[0].Message.Content
		if resp.Choices[0].FinishReason == "length" {
			finish = "MAX_TOKENS"
		}
	}
	out.Candidates = append(out.Candidates, struct {
		Content       gContent `json:"content"`
		FinishReason  string   `json:"finishReason"`
		Index         int      `json:"index"`
	}{
		Content:      gContent{Role: "model", Parts: []gPart{{Text: text}}},
		FinishReason: finish,
		Index:        0,
	})
	out.UsageMetadata.PromptTokenCount = resp.Usage.PromptTokens
	out.UsageMetadata.CandidatesTokenCount = resp.Usage.CompletionTokens
	out.UsageMetadata.TotalTokenCount = resp.Usage.TotalTokens

	return json.Marshal(out)
}

func (googleCodec) EncodeError(status int, errType, msg string) []byte {
	type gErr struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	var e gErr
	e.Error.Code = status
	e.Error.Message = msg
	e.Error.Status = errType
	b, _ := json.Marshal(e)
	return b
}

func (googleCodec) ContentType() string { return "application/json" }

// ── Responses (OpenAI Responses API) codec ────────────────────────────────────

// responsesRequest is the OpenAI Responses API wire format:
// POST /v1/responses { model, input, instructions?, stream?, ... }
// Input can be a string or an array of role/content/content_part objects.
type responsesRequest struct {
	Model        string          `json:"model"`
	Input        json.RawMessage `json:"input"`
	Instructions string          `json:"instructions,omitempty"`
	Stream       bool            `json:"stream,omitempty"`
	Temperature  *float64        `json:"temperature,omitempty"`
	TopP         *float64        `json:"top_p,omitempty"`
	MaxTokens    *int            `json:"max_output_tokens,omitempty"`
	Stop         []string        `json:"stop,omitempty"`
}

type responsesCodec struct{}

// NewResponses returns the OpenAI Responses API wire codec.
func NewResponses() Codec { return responsesCodec{} }

func (responsesCodec) Format() Format { return FormatResponses }

func (responsesCodec) Decode(body []byte) (*dto.ChatRequest, error) {
	var rr responsesRequest
	if err := json.Unmarshal(body, &rr); err != nil {
		return nil, fmt.Errorf("invalid responses request: %w", err)
	}
	if rr.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(rr.Input) == 0 {
		return nil, fmt.Errorf("input is required")
	}

	req := &dto.ChatRequest{
		Model:       rr.Model,
		Stream:      rr.Stream,
		Temperature: rr.Temperature,
		TopP:        rr.TopP,
		Stop:        rr.Stop,
	}

	if rr.MaxTokens != nil {
		req.MaxTokens = rr.MaxTokens
	}

	// Prepend system instructions if present.
	if rr.Instructions != "" {
		req.Messages = append(req.Messages, dto.Message{Role: "system", Content: rr.Instructions})
	}

	// Parse input: string → single user message; array → role/content objects.
	inputStr := ""
	if err := json.Unmarshal(rr.Input, &inputStr); err == nil {
		// Input is a plain string.
		req.Messages = append(req.Messages, dto.Message{Role: "user", Content: inputStr})
	} else {
		// Input is an array of objects: [{role, content}, ...]
		type inputPart struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		var parts []inputPart
		if err := json.Unmarshal(rr.Input, &parts); err != nil {
			return nil, fmt.Errorf("invalid responses input format: %w", err)
		}
		for _, p := range parts {
			req.Messages = append(req.Messages, dto.Message{
				Role:    p.Role,
				Content: extractText(p.Content),
			})
		}
	}

	return req, nil
}

func (responsesCodec) EncodeChat(resp *dto.ChatResponse) ([]byte, error) {
	// Responses API output: { id, object, output: [...], usage: {...} }
	type outputBlock struct {
		Type    string `json:"type"`
		Content string `json:"content,omitempty"`
	}

	type rResp struct {
		ID      string        `json:"id"`
		Object  string        `json:"object"`
		Output  []outputBlock `json:"output"`
		Usage   struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	var out rResp
	out.ID = resp.ID
	out.Object = "response"
	out.Usage.InputTokens = resp.Usage.PromptTokens
	out.Usage.OutputTokens = resp.Usage.CompletionTokens
	out.Usage.TotalTokens = resp.Usage.TotalTokens

	text := ""
	if len(resp.Choices) > 0 {
		text = resp.Choices[0].Message.Content
	}
	if text != "" {
		out.Output = append(out.Output, outputBlock{Type: "output_text", Content: text})
	} else {
		out.Output = []outputBlock{}
	}

	return json.Marshal(out)
}

func (responsesCodec) EncodeError(status int, errType, msg string) []byte {
	b, _ := json.Marshal(dto.ErrorBody{Error: dto.ErrorDetail{
		Message: msg,
		Type:    errType,
	}})
	return b
}

func (responsesCodec) ContentType() string { return "application/json" }

// ── Registry ─────────────────────────────────────────────────────────────────

var codecs = map[Format]Codec{
	FormatOpenAI:    NewOpenAI(),
	FormatResponses: NewResponses(),
	FormatAnthropic: NewAnthropic(),
	FormatGoogle:    NewGoogle(),
}

// For returns the codec for a wire format.
func For(f Format) (Codec, error) {
	c, ok := codecs[f]
	if !ok {
		return nil, fmt.Errorf("unsupported inbound format: %s", f)
	}
	return c, nil
}

var _ = io.Discard // keep io import if future code needs it