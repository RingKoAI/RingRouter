package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
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

func TestQuotaAndCreditsUseEffectiveTokenBalance(t *testing.T) {
	h := &Proxy{}
	u := &model.User{ID: 7, Quota: 1000, UsedQuota: 200}
	if limit, used := effectiveQuota(u, &model.Token{Quota: 500, UsedQuota: 125}); limit != 625 || used != 125 {
		t.Fatalf("effective token quota = (%d, %d), want (625, 125)", limit, used)
	}
	if limit, used := effectiveQuota(u, nil); limit != 1200 || used != 200 {
		t.Fatalf("effective user quota = (%d, %d), want (1200, 200)", limit, used)
	}
	ctx := middleware.WithUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), u)

	for _, tc := range []struct {
		name string
		call http.HandlerFunc
		want map[string]interface{}
	}{
		{"quota", h.QuotaLimit, map[string]interface{}{"limit": float64(1200), "used": float64(200), "remaining": float64(1000)}},
		{"credits", h.Credits, map[string]interface{}{"credits": float64(1000), "balance": float64(1000)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			tc.call(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			var got map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			for key, want := range tc.want {
				if got[key] != want {
					t.Errorf("%s = %v, want %v", key, got[key], want)
				}
			}
		})
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
