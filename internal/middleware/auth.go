package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

type contextKey string

const (
	ctxKeyUser  contextKey = "user"
	ctxKeyToken contextKey = "token"
)

// Auth validates API keys from the Authorization header.
type Auth struct {
	AdminKey string // fallback admin key from env
}

// NewAuth creates a new Auth middleware.
func NewAuth(adminKey string) *Auth {
	return &Auth{AdminKey: adminKey}
}

// Middleware returns an HTTP middleware that validates API keys.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractKey(r)
		if key == "" {
			writeAuthError(w, "missing API key")
			return
		}

		// Check admin key fallback (constant-time so the comparison leaks no
		// timing signal about the key's content or length).
		if a.AdminKey != "" && subtle.ConstantTimeCompare([]byte(key), []byte(a.AdminKey)) == 1 {
			ctx := context.WithValue(r.Context(), ctxKeyUser, &model.User{
				ID:       0,
				Username: "admin",
				Role:     "admin",
				Quota:    -1,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Look up token in database
		if database.DB == nil {
			writeAuthError(w, "database not available")
			return
		}

		var token model.Token
		if err := database.DB.Where("key = ? AND status = ?", key, "active").Preload("User").First(&token).Error; err != nil {
			writeAuthError(w, "invalid API key")
			return
		}

		// Token-level restrictions (one-api semantics).
		if !token.ExpiredAt.IsZero() && time.Now().After(token.ExpiredAt) {
			writeAuthError(w, "this API key has expired")
			return
		}
		if token.Subnet != "" && !IPInSubnets(clientIP(r), token.Subnet) {
			writeAuthError(w, "this API key is restricted to specific subnets")
			return
		}

		// Check user status
		if token.User.Status != "active" {
			writeAuthError(w, "user account disabled")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUser, &token.User)
		ctx = context.WithValue(ctx, ctxKeyToken, &token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IPInSubnets reports whether ip falls inside any of the comma-separated
// CIDR ranges (or matches a bare IP entry). Malformed entries are skipped
// rather than failing the request.
func IPInSubnets(ipStr, csv string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, cidr := range strings.Split(csv, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			// Bare IP: exact match.
			if net.ParseIP(cidr).Equal(ip) {
				return true
			}
			continue
		}
		if _, network, err := net.ParseCIDR(cidr); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func extractKey(r *http.Request) string {
	// OpenAI style: Authorization: Bearer <key>
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Anthropic native header
	if k := r.Header.Get("X-Api-Key"); k != "" {
		return k
	}
	// Google Gemini native header
	if k := r.Header.Get("X-Goog-Api-Key"); k != "" {
		return k
	}
	return ""
}

// GetUser extracts the user from context.
func GetUser(ctx context.Context) *model.User {
	if u, ok := ctx.Value(ctxKeyUser).(*model.User); ok {
		return u
	}
	return nil
}

// WithUser attaches a user to a context (used by tests and internal calls).
func WithUser(ctx context.Context, u *model.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// GetToken extracts the token from context.
func GetToken(ctx context.Context) *model.Token {
	if t, ok := ctx.Value(ctxKeyToken).(*model.Token); ok {
		return t
	}
	return nil
}

func writeAuthError(w http.ResponseWriter, msg string) {
	writeJSONString(w, http.StatusUnauthorized, "auth_error", msg)
}

// writeJSONString encodes an OpenAI-style error envelope with proper JSON
// escaping — never hand-concatenated strings, which would let a future dynamic
// message break out of the JSON string and inject markup.
func writeJSONString(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]map[string]string{
		"error": {"message": msg, "type": errType},
	})
}

// Logging returns an HTTP middleware that logs requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[ringrouter] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS returns an HTTP middleware enabling cross-origin access for the
// programmatic gateway surface (/v1/*). It deliberately never sets
// Access-Control-Allow-Credentials, so browsers will not attach management
// session cookies to cross-origin requests. The management plane (/api/*)
// stays same-origin: its SPA is embedded in the same binary and cross-site
// readers get no explicit grant.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key, X-Goog-Api-Key, Anthropic-Version")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
