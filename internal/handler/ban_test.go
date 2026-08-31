package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// setupBanTest provisions one active user with an API token and an open
// management session — both must stop working the moment the account is
// disabled.
func setupBanTest(t *testing.T) (*AdminHandler, *model.User, string, string) {
	t.Helper()
	setupGuardTest(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	u := &model.User{
		Username: "victim", Email: "victim@example.com", DisplayName: "victim",
		Password: string(hash), Role: "user", Group: "default", Quota: 100, Status: "active",
	}
	if err := database.DB.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	tok := &model.Token{UserID: u.ID, Key: "sk-rr-bantest0001", Name: "t", Status: "active", Quota: -1}
	if err := database.DB.Create(tok).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	sess := &model.Session{
		ID:        middleware.SessionDigest("raw-session-token"),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := database.DB.Create(sess).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	return NewAdminHandler(), u, tok.Key, "raw-session-token"
}

// TestDisableUserBlocksTokensAndSessions pins the enforcement chain: an
// admin disabling an account takes effect on the very next request for both
// gateway API keys and management sessions — no restart, no TTL grace.
func TestDisableUserBlocksTokensAndSessions(t *testing.T) {
	adminH, u, apiKey, sessionToken := setupBanTest(t)

	gatewayAuth := middleware.NewAuth("")
	sessionAuth := middleware.NewSessionAuth("")
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	callViaAPIKey := func() int {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		rec := httptest.NewRecorder()
		gatewayAuth.Middleware(okHandler).ServeHTTP(rec, req)
		return rec.Code
	}
	callViaSession := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: sessionToken})
		rec := httptest.NewRecorder()
		sessionAuth.Middleware(okHandler).ServeHTTP(rec, req)
		return rec.Code
	}

	if got := callViaAPIKey(); got != http.StatusOK {
		t.Fatalf("precondition: active token auth = %d, want 200", got)
	}
	if got := callViaSession(); got != http.StatusOK {
		t.Fatalf("precondition: active session auth = %d, want 200", got)
	}

	// Admin disables the account through the real handler.
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+itoa(u.ID)+"/status",
		strings.NewReader(`{"status":"disabled"}`))
	req.SetPathValue("id", itoa(u.ID))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUser(req.Context(), &model.User{ID: 99, Role: "admin"}))
	rec := httptest.NewRecorder()
	adminH.UpdateStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable user = %d: %s", rec.Code, rec.Body.String())
	}

	if got := callViaAPIKey(); got != http.StatusUnauthorized {
		t.Errorf("disabled user token auth = %d, want 401", got)
	}
	if got := callViaSession(); got != http.StatusUnauthorized {
		t.Errorf("disabled user session auth = %d, want 401", got)
	}

	// The disabled user's sessions are also removed from the store.
	var remaining int64
	database.DB.Model(&model.Session{}).Where("user_id = ?", u.ID).Count(&remaining)
	if remaining != 0 {
		t.Errorf("disabled user still has %d sessions, want 0", remaining)
	}
}
