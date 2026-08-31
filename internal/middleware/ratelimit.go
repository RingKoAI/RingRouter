// Package middleware: ratelimit.go implements per-IP sliding-window rate
// limiting (one-api semantics): critical auth endpoints get a tight budget,
// the gateway API and the management plane each get their own. Backed by an
// in-process window store; deploy behind a trusted reverse proxy so the
// forwarded client IP is honest.
package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/RingKoAI/RingRouter/internal/safenet"
)

/* ── Sliding-window store ────────────────────────────────────────────────── */

type slidingWindow struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newSlidingWindow() *slidingWindow {
	return &slidingWindow{hits: make(map[string][]time.Time)}
}

// allow records a hit and reports whether it fits the budget. Expired
// entries are pruned on touch; stale keys are swept opportunistically so the
// map cannot grow without bound under address churn.
func (s *slidingWindow) allow(key string, max int, window time.Duration) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	arr := s.hits[key]
	keep := arr[:0]
	for _, t := range arr {
		if now.Sub(t) < window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= max {
		s.hits[key] = keep
		return false
	}
	s.hits[key] = append(keep, now)

	if len(s.hits) > 4096 {
		for k, v := range s.hits {
			if len(v) == 0 || now.Sub(v[len(v)-1]) > window {
				delete(s.hits, k)
			}
		}
	}
	return true
}

/* ── Middleware ──────────────────────────────────────────────────────────── */

var limiter = newSlidingWindow()

// Budget tiers (one-api defaults), overridable via env; 0 disables a tier.
const (
	DefaultAPILimit      = 480
	DefaultWebLimit      = 240
	DefaultCriticalLimit = 20
	apiWindow            = 3 * time.Minute
	webWindow            = 3 * time.Minute
	criticalWindow       = 20 * time.Minute
)

func envLimit(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}

// RateLimitAPI guards the gateway surface (/v1/*).
func RateLimitAPI() func(http.Handler) http.Handler {
	return rateLimit("api", envLimit("RATE_LIMIT_API", DefaultAPILimit), apiWindow)
}

// RateLimitWeb guards the management plane (/api/*).
func RateLimitWeb() func(http.Handler) http.Handler {
	return rateLimit("web", envLimit("RATE_LIMIT_WEB", DefaultWebLimit), webWindow)
}

// RateLimitCritical guards expensive/auth-sensitive endpoints
// (login, register, codes, passkey ceremonies).
func RateLimitCritical() func(http.Handler) http.Handler {
	return rateLimit("crit", envLimit("RATE_LIMIT_CRITICAL", DefaultCriticalLimit), criticalWindow)
}

func rateLimit(mark string, max int, window time.Duration) func(http.Handler) http.Handler {
	if max <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow(mark+"|"+clientIP(r), max, window) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"message":"rate limit exceeded, slow down","type":"rate_limit_error"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP resolves the caller address through the shared trusted-proxy logic
// (see internal/safenet): forwarded headers are honored only when the TCP
// peer is a trusted proxy, so limits cannot be bypassed by rotating a forged
// X-Forwarded-For.
func clientIP(r *http.Request) string {
	return safenet.ClientIP(r)
}
