package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/provider"
)

func TestSupports(t *testing.T) {
	ch := &model.Channel{Models: "gpt-4o, gpt-4o-mini ,claude-3"}
	cases := []struct {
		name string
		want bool
	}{
		{"gpt-4o", true},
		{"gpt-4o-mini", true}, // trimmed match
		{"claude-3", true},
		{"gpt-4", false},
		{"", false},
	}
	for _, c := range cases {
		if got := supports(ch, c.name); got != c.want {
			t.Errorf("supports(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestInGroupMultiGroupSemantics(t *testing.T) {
	cases := []struct {
		name    string
		chGroup string
		req     string
		want    bool
	}{
		{"single exact", "vip", "vip", true},
		{"single mismatch", "vip", "basic", false},
		{"multi contains", "default, vip", "vip", true},
		{"multi first", "default, vip", "default", true},
		{"multi miss", "default, vip", "basic", false},
		{"multi with spaces", "default,vip, enterprise", "enterprise", true},
		{"default matches default", "default", "default", true},
		{"default matches empty chGroup", "", "default", true},
		{"default does not match vip-only", "vip", "default", false},
		{"empty request is default", "vip, default", "", true},
	}
	for _, c := range cases {
		ch := &model.Channel{Group: c.chGroup}
		if got := inGroup(ch, c.req); got != c.want {
			t.Errorf("%s: inGroup(%q, %q) = %v, want %v", c.name, c.chGroup, c.req, got, c.want)
		}
	}
}

// fakeOpenAIUpstream returns an httptest server speaking OpenAI wire format.
func fakeOpenAIUpstream(t *testing.T, status int, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"object":"list","data":[{"id":"test-model","object":"model"}]}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"id":"x","object":"chat.completion","model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`, content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// inject is a test helper that bypasses the DB reload path by installing a
// fresh channel snapshot that stays valid (loaded = now).
func inject(t *testing.T, g *Gateway, channels []*model.Channel) {
	t.Helper()
	g.mu.Lock()
	g.channels = channels
	g.providers = make(map[uint]provider.Provider)
	g.loaded = time.Now()
	g.mu.Unlock()
}

func TestChatPrefersHigherPriority(t *testing.T) {
	ok := fakeOpenAIUpstream(t, 200, "from-ok")
	g := New(nil)
	inject(t, g, []*model.Channel{
		{ID: 1, Name: "primary", Protocol: "openai", BaseURL: ok.URL, Models: "m", Priority: 10},
		{ID: 2, Name: "backup", Protocol: "openai", BaseURL: "http://127.0.0.1:1", Models: "m", Priority: 1},
	})

	resp, ch, err := g.Chat(context.Background(), &dto.ChatRequest{Model: "m"}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if ch.Name != "primary" {
		t.Errorf("served by %q, want primary", ch.Name)
	}
	if resp.Choices[0].Message.Content != "from-ok" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
}

func TestChatFailoverFallsToBackup(t *testing.T) {
	backup := fakeOpenAIUpstream(t, 200, "from-backup")
	g := New(nil)
	inject(t, g, []*model.Channel{
		{ID: 1, Name: "primary", Protocol: "openai", BaseURL: "http://127.0.0.1:1", Models: "m", Priority: 10},
		{ID: 2, Name: "backup", Protocol: "openai", BaseURL: backup.URL, Models: "m", Priority: 1},
	})

	resp, ch, err := g.Chat(context.Background(), &dto.ChatRequest{Model: "m"}, "")
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if ch.Name != "backup" {
		t.Errorf("served by %q, want backup", ch.Name)
	}
	if resp.Choices[0].Message.Content != "from-backup" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
}

func TestChatAllChannelsFail(t *testing.T) {
	g := New(nil)
	inject(t, g, []*model.Channel{
		{ID: 1, Name: "a", Protocol: "openai", BaseURL: "http://127.0.0.1:1", Models: "m", Priority: 10},
		{ID: 2, Name: "b", Protocol: "openai", BaseURL: "http://127.0.0.1:2", Models: "m", Priority: 1},
	})

	_, _, err := g.Chat(context.Background(), &dto.ChatRequest{Model: "m"}, "")
	if err == nil {
		t.Fatal("expected error when all channels fail")
	}
	if !strings.Contains(err.Error(), "all channels failed") {
		t.Errorf("err = %v", err)
	}
}

func TestChatNoChannelServesModel(t *testing.T) {
	ok := fakeOpenAIUpstream(t, 200, "x")
	g := New(nil)
	inject(t, g, []*model.Channel{
		{ID: 1, Name: "a", Protocol: "openai", BaseURL: ok.URL, Models: "other-model", Priority: 10},
	})

	_, _, err := g.Chat(context.Background(), &dto.ChatRequest{Model: "m"}, "")
	if err == nil || !strings.Contains(err.Error(), "no active channel") {
		t.Fatalf("err = %v, want no-active-channel error", err)
	}
}

func TestModelsAggregationDedup(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"object":"list","data":[{"id":"a","object":"model"},{"id":"b","object":"model"}]}`)
	}))
	defer s1.Close()
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"object":"list","data":[{"id":"b","object":"model"},{"id":"c","object":"model"}]}`)
	}))
	defer s2.Close()

	g := New(nil)
	inject(t, g, []*model.Channel{
		{ID: 1, Name: "s1", Protocol: "openai", BaseURL: s1.URL, Models: "a,b", Priority: 5},
		{ID: 2, Name: "s2", Protocol: "openai", BaseURL: s2.URL, Models: "b,c", Priority: 5},
	})

	models := g.Models(context.Background())
	ids := map[string]bool{}
	for _, m := range models {
		ids[m.ID] = true
	}
	if len(models) != 3 || !ids["a"] || !ids["b"] || !ids["c"] {
		t.Errorf("models = %+v, want deduped a,b,c", models)
	}
}
