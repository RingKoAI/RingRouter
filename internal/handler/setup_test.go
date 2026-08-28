package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func TestSetupCompletePersistsSiteAndPasskey(t *testing.T) {
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		setting.Reset()
		database.Close()
		database.DB = nil
	})

	h := NewSetupHandler(nil, nil)
	body := `{
		"username":"root","email":"root@example.com","password":"password123",
		"site_name":"My Gateway","usage_mode":"self",
		"passkey":{"enabled":true,"rp_id":"example.com","rp_origins":"https://example.com"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Complete(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("complete = %d: %s", rec.Code, rec.Body.String())
	}

	if got := setting.SiteName(); got != "My Gateway" {
		t.Errorf("site_name = %q, want My Gateway", got)
	}
	pk := setting.Passkey()
	if !pk.Enabled || pk.RPID != "example.com" || len(pk.Origins) != 1 || pk.Origins[0] != "https://example.com" {
		t.Errorf("passkey settings wrong: %+v", pk)
	}

	// Admin created; replay refused.
	rec2 := httptest.NewRecorder()
	h.Complete(rec2, httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(body)))
	if rec2.Code != http.StatusConflict {
		t.Errorf("replay = %d, want 409", rec2.Code)
	}
	var admins int64
	database.DB.Model(&model.User{}).Where("role = ?", "admin").Count(&admins)
	if admins != 1 {
		t.Errorf("admins = %d, want 1", admins)
	}
}

func TestSetupCompleteRejectsIncompletePasskey(t *testing.T) {
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		setting.Reset()
		database.Close()
		database.DB = nil
	})

	h := NewSetupHandler(nil, nil)
	body := `{
		"username":"root","email":"root@example.com","password":"password123",
		"usage_mode":"self",
		"passkey":{"enabled":true,"rp_id":"","rp_origins":""}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Complete(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("incomplete passkey = %d, want 400", rec.Code)
	}
	// Nothing persisted on rejection.
	if setting.Get(setting.KeyPasskeyEnabled) != "" {
		t.Error("passkey must not be persisted on rejected setup")
	}
}
