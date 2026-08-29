package middleware

import (
	"context"
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

		// Check admin key fallback
		if a.AdminKey != "" && key == a.AdminKey {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":{"message":"` + msg + `","type":"auth_error"}}`))
}

// Logging returns an HTTP middleware that logs requests.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[ringrouter] %s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS returns an HTTP middleware that sets CORS headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
