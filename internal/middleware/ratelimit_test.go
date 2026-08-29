package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimitAllowsUnderBudgetAndBlocksOver(t *testing.T) {
	h := rateLimit("test-a", 3, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	hit := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < 3; i++ {
		if c := hit(); c != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, c)
		}
	}
	if c := hit(); c != http.StatusTooManyRequests {
		t.Errorf("4th request = %d, want 429", c)
	}
}

func TestRateLimitSeparatesIPsAndMarks(t *testing.T) {
	hA := rateLimit("mark-a", 1, time.Minute)(http.HandlerFunc(okHandler))
	hB := rateLimit("mark-b", 1, time.Minute)(http.HandlerFunc(okHandler))

	mk := func(ip string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":9999"
		return req
	}

	rec := httptest.NewRecorder()
	hA.ServeHTTP(rec, mk("10.0.0.1"))
	if rec.Code != http.StatusOK {
		t.Fatal("first hit should pass")
	}
	// Same IP, same mark: blocked.
	rec = httptest.NewRecorder()
	hA.ServeHTTP(rec, mk("10.0.0.1"))
	if rec.Code != http.StatusTooManyRequests {
		t.Error("same ip+mark must be blocked at limit")
	}
	// Same IP, different mark: independent budget.
	rec = httptest.NewRecorder()
	hB.ServeHTTP(rec, mk("10.0.0.1"))
	if rec.Code != http.StatusOK {
		t.Error("different mark must have its own budget")
	}
	// Different IP, same mark: independent budget.
	rec = httptest.NewRecorder()
	hA.ServeHTTP(rec, mk("10.0.0.2"))
	if rec.Code != http.StatusOK {
		t.Error("different ip must have its own budget")
	}
}

func TestRateLimitWindowSlides(t *testing.T) {
	s := newSlidingWindow()
	if !s.allow("k", 1, 30*time.Millisecond) {
		t.Fatal("first should pass")
	}
	if s.allow("k", 1, 30*time.Millisecond) {
		t.Fatal("second inside window must be refused")
	}
	time.Sleep(35 * time.Millisecond)
	if !s.allow("k", 1, 30*time.Millisecond) {
		t.Fatal("after window expiry it must pass again")
	}
}

func TestRateLimitZeroDisables(t *testing.T) {
	h := rateLimit("off", 0, time.Minute)(http.HandlerFunc(okHandler))
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter blocked at %d", i)
		}
	}
}

func TestRateLimitErrorShape(t *testing.T) {
	h := rateLimit("shape", 1, time.Minute)(http.HandlerFunc(okHandler))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.9.9.9:1"
	h.ServeHTTP(httptest.NewRecorder(), req) // consume budget
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatal("expected 429")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "rate_limit_error") || !strings.Contains(body, "slow down") {
		t.Errorf("error body not OpenAI-shaped: %s", body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header missing")
	}
}

func okHandler(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
