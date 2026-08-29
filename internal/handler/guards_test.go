package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func setupGuardTest(t *testing.T) {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		setting.Reset()
		os.Unsetenv("CHANNEL_ALLOW_PRIVATE_ADDR")
		database.Close()
		database.DB = nil
	})
	setting.Set(setting.KeySensitiveWords, "")
}

/* ── Sensitive words ── */

func TestSensitiveWordFiltering(t *testing.T) {
	setupGuardTest(t)
	if err := setting.Set(setting.KeySensitiveWords, "forbidden, bomb"); err != nil {
		t.Fatal(err)
	}

	req := &dto.ChatRequest{Model: "m", Messages: []dto.Message{{Role: "user", Content: "hello BOMB world"}}}
	if hit := hitsSensitiveWord(req); hit != "bomb" {
		t.Errorf("hit = %q, want bomb (case-insensitive)", hit)
	}
	clean := &dto.ChatRequest{Model: "m", Messages: []dto.Message{{Role: "user", Content: "totally fine"}}}
	if hit := hitsSensitiveWord(clean); hit != "" {
		t.Errorf("clean request flagged: %q", hit)
	}

	// Disabled when unset.
	setting.Set(setting.KeySensitiveWords, "")
	if hit := hitsSensitiveWord(req); hit != "" {
		t.Errorf("filter must be off when list empty, got %q", hit)
	}
}

/* ── Token model whitelist ── */

func TestTokenModelWhitelist(t *testing.T) {
	if !tokenAllowsModel("gpt-4o, claude-3", "gpt-4o") {
		t.Error("listed model should pass")
	}
	if !tokenAllowsModel("gpt-4o,claude-3", "claude-3") {
		t.Error("second listed model (no space) should pass")
	}
	if tokenAllowsModel("gpt-4o", "gpt-4o-mini") {
		t.Error("unlisted model must be rejected")
	}
	if tokenAllowsModel("", "anything") {
		t.Error("empty whitelist means allow-all — helper must not be called with empty, but guard anyway")
	}
}

/* ── SSRF ── */

func TestChannelURLValidation(t *testing.T) {
	setupGuardTest(t)

	// Default mode: internal relays stay allowed (one-api compatibility).
	for _, u := range []string{"http://172.17.0.1:8080/v1", "http://localhost:11434"} {
		if err := validateChannelURL(u); err != nil {
			t.Errorf("default mode should allow internal relay %q: %v", u, err)
		}
	}
	if err := validateChannelURL("ftp://example.com"); err == nil {
		t.Error("non-http scheme must be rejected")
	}

	// Hardened mode: private/loopback hosts rejected.
	os.Setenv("CHANNEL_ALLOW_PRIVATE_ADDR", "false")
	if err := validateChannelURL("http://127.0.0.1:9911"); err == nil {
		t.Error("loopback must be rejected in hardened mode")
	}
	if err := validateChannelURL("http://192.168.1.10/v1"); err == nil {
		t.Error("private net must be rejected in hardened mode")
	}
	os.Unsetenv("CHANNEL_ALLOW_PRIVATE_ADDR")
}

/* ── Token subnet / expiry enforcement (middleware) ── */

func TestIPInSubnets(t *testing.T) {
	cases := []struct {
		ip   string
		csv  string
		want bool
	}{
		{"10.0.0.5", "10.0.0.0/24", true},
		{"10.0.1.5", "10.0.0.0/24", false},
		{"1.2.3.4", "1.2.3.4", true}, // bare IP exact match
		{"1.2.3.5", "1.2.3.4", false},
		{"10.1.1.1", "10.0.0.0/8, 192.168.0.0/16", true}, // csv + spaces
		{"172.16.0.1", "10.0.0.0/8", false},
		{"garbage", "10.0.0.0/8", false},
	}
	for _, c := range cases {
		if got := middleware.IPInSubnets(c.ip, c.csv); got != c.want {
			t.Errorf("ipInSubnets(%q, %q) = %v, want %v", c.ip, c.csv, got, c.want)
		}
	}
}

func TestTokenAuthRejectsSubnetAndExpiry(t *testing.T) {
	setupGuardTest(t)
	user := model.User{Username: "g", Role: "user", Status: "active"}
	database.DB.Create(&user)

	outside := model.Token{UserID: user.ID, Key: "k-out", Name: "o", Group: "default", Status: "active", Quota: -1, Subnet: "10.99.0.0/16"}
	database.DB.Create(&outside)
	expired := model.Token{UserID: user.ID, Key: "k-exp", Name: "e", Group: "default", Status: "active", Quota: -1,
		ExpiredAt: time.Now().Add(-time.Hour)}
	database.DB.Create(&expired)
	inside := model.Token{UserID: user.ID, Key: "k-in", Name: "i", Group: "default", Status: "active", Quota: -1, Subnet: "127.0.0.0/8"}
	database.DB.Create(&inside)

	auth := middleware.NewAuth("")

	call := func(key string) int {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		req.RemoteAddr = "127.0.0.1:5555"
		rec := httptest.NewRecorder()
		auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })).ServeHTTP(rec, req)
		return rec.Code
	}

	if c := call("k-out"); c != http.StatusUnauthorized {
		t.Errorf("outside subnet = %d, want 401", c)
	}
	if c := call("k-exp"); c != http.StatusUnauthorized {
		t.Errorf("expired token = %d, want 401", c)
	}
	if c := call("k-in"); c != http.StatusOK {
		t.Errorf("inside subnet = %d, want 200", c)
	}
}

/* ── tokens API validation ── */

func TestTokenPayloadSubnetValidation(t *testing.T) {
	if !validSubnets("10.0.0.0/8, 192.168.1.1") {
		t.Error("valid csv rejected")
	}
	if validSubnets("10.0.0.0/8, nonsense") {
		t.Error("invalid entry accepted")
	}
	if got := normalizeCSV(" a , b ,, c "); got != "a,b,c" {
		t.Errorf("normalizeCSV = %q", got)
	}
	_ = strings.TrimSpace
}
