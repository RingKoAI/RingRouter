package handler

import (
	"testing"
)

func TestExtractGeminiModel(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/v1beta/models/gemini-2.0-flash:generateContent", "gemini-2.0-flash"},
		{"/v1beta/models/gemini-2.0-flash:streamGenerateContent", "gemini-2.0-flash"},
		{"/v1beta/models/gemini-1.5-pro", "gemini-1.5-pro"},
		{"/v1/chat/completions", ""},
		{"/nope", ""},
	}
	for _, c := range cases {
		if got := extractGeminiModel(c.path); got != c.want {
			t.Errorf("extractGeminiModel(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestAccumulateOpenAISSE(t *testing.T) {
	raw := []byte(`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"message":{"role":"assistant","content":" world"},"finish_reason":null}]}

data: {"id":"chatcmpl-1","object":"chat.completion.chunk","choices":[{"index":0,"message":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}

data: [DONE]

`)
	got := accumulateOpenAISSE(raw)
	if got == nil {
		t.Fatal("accumulateOpenAISSE returned nil")
	}
	if got.ID != "chatcmpl-1" {
		t.Errorf("ID = %q, want chatcmpl-1", got.ID)
	}
	if len(got.Choices) != 1 || got.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content = %+v, want 'Hello world'", got.Choices)
	}
	if got.Choices[0].FinishReason != "stop" {
		t.Errorf("finish = %q, want stop", got.Choices[0].FinishReason)
	}
	if got.Usage.TotalTokens != 7 {
		t.Errorf("usage = %+v, want total 7", got.Usage)
	}
}

func TestAccumulateEmptySSE(t *testing.T) {
	got := accumulateOpenAISSE([]byte("data: [DONE]\n\n"))
	if got == nil || got.Choices[0].Message.Content != "" {
		t.Errorf("got %+v, want empty content", got)
	}
	if got.Choices[0].FinishReason != "stop" {
		t.Errorf("finish = %q, want stop (default)", got.Choices[0].FinishReason)
	}
}