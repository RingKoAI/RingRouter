package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// SessionCookieName is the management-plane session cookie.
const SessionCookieName = "ringrouter_session"

// sessionLifetime is how long a management session stays valid.
const sessionLifetime = 7 * 24 * time.Hour

// SessionAuth authenticates management-plane requests (/api/*) via either
// the session cookie or the bootstrap admin key (Bearer). This is a separate
// track from the gateway API-key auth used on /v1/*.
type SessionAuth struct {
	AdminKey string
}

// NewSessionAuth creates a SessionAuth middleware.
func NewSessionAuth(adminKey string) *SessionAuth {
	return &SessionAuth{AdminKey: adminKey}
}

// Middleware returns an HTTP middleware validating management sessions.
func (s *SessionAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track 1: session cookie.
		if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
			digest := SessionDigest(c.Value)
			var sess model.Session
			if err := database.DB.Where("id = ?", digest).First(&sess).Error; err == nil {
				if time.Now().After(sess.ExpiresAt) {
					database.DB.Delete(&sess)
				} else {
					var user model.User
					if sess.UserID == 0 {
						// Bootstrap admin-key session: virtual admin user.
						user = model.User{ID: 0, Username: "admin", Role: "admin", Quota: -1, Status: "active"}
					} else if err := database.DB.First(&user, sess.UserID).Error; err != nil {
						writeAuthError(w, "session user no longer exists")
						return
					} else if user.Status != "active" {
						writeAuthError(w, "user account disabled")
						return
					}
					ctx := context.WithValue(r.Context(), ctxKeyUser, &user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// Track 2: bootstrap admin key via Authorization header.
		key := extractKey(r)
		if s.AdminKey != "" && key != "" &&
			subtle.ConstantTimeCompare([]byte(key), []byte(s.AdminKey)) == 1 {
			ctx := context.WithValue(r.Context(), ctxKeyUser, &model.User{
				ID: 0, Username: "admin", Role: "admin", Quota: -1,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		writeAuthError(w, "not authenticated")
	})
}

// SessionDigest hashes a raw session token for storage/lookup.
func SessionDigest(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// OptionalSession attaches the user when a valid session cookie is present
// but lets anonymous requests through. Used by endpoints whose visibility
// is runtime-configurable (e.g. the model plaza).
func (s *SessionAuth) OptionalSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" && database.DB != nil {
			var sess model.Session
			if err := database.DB.Where("id = ?", SessionDigest(c.Value)).First(&sess).Error; err == nil {
				if time.Now().After(sess.ExpiresAt) {
					database.DB.Delete(&sess)
				} else if sess.UserID == 0 {
					ctx := context.WithValue(r.Context(), ctxKeyUser, &model.User{
						ID: 0, Username: "admin", Role: "admin", Quota: -1, Status: "active",
					})
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				} else {
					var user model.User
					if err := database.DB.First(&user, sess.UserID).Error; err == nil && user.Status == "active" {
						ctx := context.WithValue(r.Context(), ctxKeyUser, &user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin rejects non-admin users. Must run after SessionAuth.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := GetUser(r.Context())
		if u == nil || u.Role != "admin" {
			writeAuthError(w, "admin privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
