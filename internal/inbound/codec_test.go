package inbound

import (
	"encoding/json"
	"testing"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

func chatRespForTest(content, finish string, pt, ct int) *dto.ChatResponse {
	return &dto.ChatResponse{
		ID:     "id-1",
		Object: "chat.completion",
		Model:  "m",
		Choices: []dto.Choice{{
			Message:      dto.Message{Role: "assistant", Content: content},
			FinishReason: finish,
		}},
		Usage: dto.Usage{PromptTokens: pt, CompletionTokens: ct, TotalTokens: pt + ct},
	}
}

// ── OpenAI codec ─────────────────────────────────────────────────────────────

func TestOpenAIDecodeValid(t *testing.T) {
	c := NewOpenAI()
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":false}`
	req, err := c.Decode([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Model != "gpt-4o" || len(req.Messages) != 1 || req.Messages[0].Content != "hi" {
		t.Errorf("req = %+v", req)
	}
}

func TestOpenAIDecodeMissingModel(t *testing.T) {
	c := NewOpenAI()
	if _, err := c.Decode([]byte(`{"messages":[]}`)); err == nil {
		t.Error("expected error for missing model")
	}
}

func TestOpenAIDecodeMissingMessages(t *testing.T) {
	c := NewOpenAI()
	if _, err := c.Decode([]byte(`{"model":"gpt-4o"}`)); err == nil {
		t.Error("expected error for missing messages")
	}
}

// ── Anthropic codec ──────────────────────────────────────────────────────────

func TestAnthropicDecodeStringSystem(t *testing.T) {
	c := NewAnthropic()
	body := `{
		"model": "claude-sonnet-4-5",
		"max_tokens": 100,
		"system": "You are helpful.",
		"messages": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "content": "hi"},
			{"role": "user", "content": "bye"}
		]
	}`
	req, err := c.Decode([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Model != "claude-sonnet-4-5" {
		t.Errorf("model = %q", req.Model)
	}
	// system prompt must be prepended as first message
	if len(req.Messages) != 4 || req.Messages[0].Role != "system" || req.Messages[0].Content != "You are helpful." {
		t.Errorf("messages = %+v", req.Messages)
	}
	if req.Messages[1].Content != "hello" || req.Messages[3].Content != "bye" {
		t.Errorf("messages = %+v", req.Messages)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 100 {
		t.Errorf("max_tokens = %v", req.MaxTokens)
	}
}

func TestAnthropicDecodeBlockContent(t *testing.T) {
	c := NewAnthropic()
	body := `{"model":"claude-sonnet-4-5","max_tokens":10,"messages":[{"role":"user","content":[{"type":"text","text":"part1"},{"type":"text","text":"part2"}]}]}`
	req, err := c.Decode([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Messages[0].Content != "part1\npart2" {
		t.Errorf("content = %q", req.Messages[0].Content)
	}
}

func TestAnthropicEncodeChat(t *testing.T) {
	c := NewAnthropic()
	resp := chatRespForTest("Hello!", "length", 3, 5)
	out, err := c.EncodeChat(resp)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if m["type"] != "message" || m["role"] != "assistant" {
		t.Errorf("m = %v", m)
	}
	if m["stop_reason"] != "max_tokens" {
		t.Errorf("stop_reason = %v (openai 'length' must map to anthropic 'max_tokens')", m["stop_reason"])
	}
	usage := m["usage"].(map[string]interface{})
	if usage["input_tokens"].(float64) != 3 || usage["output_tokens"].(float64) != 5 {
		t.Errorf("usage = %v", usage)
	}
}

func TestAnthropicEncodeError(t *testing.T) {
	c := NewAnthropic()
	out := c.EncodeError(401, "authentication_error", "bad key")
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if m["type"] != "error" {
		t.Errorf("m = %v", m)
	}
}

// ── Google codec ─────────────────────────────────────────────────────────────

func TestGoogleDecode(t *testing.T) {
	c := NewGoogle()
	body := `{
		"systemInstruction": {"parts": [{"text": "be brief"}]},
		"contents": [
			{"role": "user", "parts": [{"text": "hello"}]},
			{"role": "model", "parts": [{"text": "hi"}]},
			{"role": "user", "parts": [{"text": "and now?"}]}
		],
		"generationConfig": {"temperature": 0.5, "maxOutputTokens": 77}
	}`
	req, err := c.Decode([]byte(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(req.Messages) != 4 {
		t.Fatalf("messages = %+v", req.Messages)
	}
	if req.Messages[0].Role != "system" || req.Messages[0].Content != "be brief" {
		t.Errorf("system = %+v", req.Messages[0])
	}
	// "model" role must map to "assistant"
	if req.Messages[2].Role != "assistant" {
		t.Errorf("role = %q, want assistant", req.Messages[2].Role)
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Errorf("temperature = %v", req.Temperature)
	}
	if req.MaxTokens == nil || *req.MaxTokens != 77 {
		t.Errorf("max_tokens = %v", req.MaxTokens)
	}
}

func TestGoogleEncodeChat(t *testing.T) {
	c := NewGoogle()
	resp := chatRespForTest("Hi there", "stop", 2, 3)
	out, err := c.EncodeChat(resp)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	cands := m["candidates"].([]interface{})[0].(map[string]interface{})
	content := cands["content"].(map[string]interface{})
	if content["role"] != "model" {
		t.Errorf("role = %v", content["role"])
	}
	if cands["finishReason"] != "STOP" {
		t.Errorf("finishReason = %v", cands["finishReason"])
	}
	um := m["usageMetadata"].(map[string]interface{})
	if um["promptTokenCount"].(float64) != 2 {
		t.Errorf("usageMetadata = %v", um)
	}
}

// ── Format detection ─────────────────────────────────────────────────────────

func TestDetectFormat(t *testing.T) {
	cases := []struct {
		path string
		want Format
	}{
		{"/v1/chat/completions", FormatOpenAI},
		{"/v1/messages", FormatAnthropic},
		{"/v1beta/models/gemini-2.0-flash:generateContent", FormatGoogle},
		{"/v1/models", FormatOpenAI},
	}
	for _, c := range cases {
		if got := DetectFormat(c.path); got != c.want {
			t.Errorf("DetectFormat(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestForUnsupported(t *testing.T) {
	if _, err := For(Format("nope")); err == nil {
		t.Error("expected error for unsupported format")
	}
}