package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func setupPasskeyTest(t *testing.T) *PasskeyHandler {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	setting.ResetGroups() // fresh DB instance: drop the group lookup snapshot
	t.Cleanup(func() {
		database.Close()
		database.DB = nil
		setting.Set(setting.KeyPasskeyEnabled, "")
	})
	return NewPasskeyHandler(NewAuthHandler("test-admin-key", &fakeMailer{}))
}

func pkReq(t *testing.T, h *PasskeyHandler, method, path, body string, actor *model.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	cleanPath := path
	if i := strings.IndexByte(cleanPath, '?'); i >= 0 {
		cleanPath = cleanPath[:i]
	}
	if segs := strings.Split(strings.Trim(cleanPath, "/"), "/"); len(segs) >= 3 {
		req.SetPathValue("id", segs[len(segs)-1])
	}
	if actor != nil {
		req = req.WithContext(middleware.WithUser(req.Context(), actor))
	}
	rec := httptest.NewRecorder()
	switch {
	case strings.HasSuffix(cleanPath, "register/begin"):
		h.RegisterBegin(rec, req)
	case strings.HasSuffix(cleanPath, "register/finish"):
		h.RegisterFinish(rec, req)
	case strings.HasSuffix(cleanPath, "login/begin"):
		h.LoginBegin(rec, req)
	case strings.HasSuffix(cleanPath, "login/finish"):
		h.LoginFinish(rec, req)
	case strings.HasSuffix(cleanPath, "passkeys"):
		h.List(rec, req)
	case strings.HasPrefix(cleanPath, "/auth/passkeys"):
		h.Delete(rec, req)
	}
	return rec
}

func enablePasskey(t *testing.T) {
	t.Helper()
	if err := setting.Set(setting.KeyPasskeyEnabled, "true"); err != nil {
		t.Fatalf("enable passkey: %v", err)
	}
}

func TestPasskeyDisabledAnswers503(t *testing.T) {
	h := setupPasskeyTest(t)
	user := mkUser(t, "alice", "user", "active")

	for _, tc := range []struct{ path, method string }{
		{"/auth/passkey/register/begin", http.MethodPost},
		{"/auth/passkey/login/begin", http.MethodPost},
	} {
		rec := pkReq(t, h, tc.method, tc.path, "", &user)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s = %d, want 503 when disabled", tc.path, rec.Code)
		}
	}
}

func TestPasskeyRegisterRequiresSessionAndValidChallenge(t *testing.T) {
	h := setupPasskeyTest(t)
	enablePasskey(t)
	user := mkUser(t, "alice", "user", "active")

	// No session.
	rec := pkReq(t, h, http.MethodPost, "/auth/passkey/register/begin", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("begin without session = %d, want 401", rec.Code)
	}

	// Valid begin returns a challenge and options.
	rec = pkReq(t, h, http.MethodPost, "/auth/passkey/register/begin", "", &user)
	if rec.Code != http.StatusOK {
		t.Fatalf("begin = %d: %s", rec.Code, rec.Body.String())
	}
	challenge := rec.Body.String()
	if !strings.Contains(challenge, `"challenge":"`) || !strings.Contains(challenge, `"publicKey"`) {
		t.Errorf("begin response malformed: %s", challenge)
	}

	// Unknown challenge rejected on finish.
	rec = pkReq(t, h, http.MethodPost, "/auth/passkey/register/finish?challenge=nope", `{}`, &user)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("finish with bad challenge = %d, want 400", rec.Code)
	}
}

func TestPasskeyLoginBeginNoEnumeration(t *testing.T) {
	h := setupPasskeyTest(t)
	enablePasskey(t)
	mkUser(t, "alice", "user", "active")

	// Unknown user: 200 with a dead challenge (no account enumeration).
	rec := pkReq(t, h, http.MethodPost, "/auth/passkey/login/begin", `{"username":"ghost"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ghost login begin = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"options":null`) {
		t.Errorf("ghost must return null options: %s", rec.Body.String())
	}

	// Known user without any passkey: explicit 400 (username was already
	// supplied, so the message leaks nothing beyond it).
	rec = pkReq(t, h, http.MethodPost, "/auth/passkey/login/begin", `{"username":"alice"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("passkeyless login begin = %d, want 400", rec.Code)
	}
}

func TestPasskeyChallengeSingleUseAndExpiry(t *testing.T) {
	h := setupPasskeyTest(t)
	enablePasskey(t)
	user := mkUser(t, "alice", "user", "active")

	rec := pkReq(t, h, http.MethodPost, "/auth/passkey/register/begin", "", &user)
	key := extractChallenge(t, rec.Body.String())

	// First take consumes it; second take misses.
	if _, ok := h.challenges.take(key); !ok {
		t.Fatal("challenge should be retrievable once")
	}
	if _, ok := h.challenges.take(key); ok {
		t.Error("challenge must be single-use")
	}

	// Expired entries are refused.
	h.challenges.put(key, &challengeEntry{userID: user.ID})
	h.challenges.mu.Lock()
	h.challenges.entries[key].expires = time.Now().Add(-time.Minute)
	h.challenges.mu.Unlock()
	if _, ok := h.challenges.take(key); ok {
		t.Error("expired challenge must be refused")
	}
}

func TestPasskeyListAndDeleteOwnOnly(t *testing.T) {
	h := setupPasskeyTest(t)
	alice := mkUser(t, "alice", "user", "active")
	bob := mkUser(t, "bob", "user", "active")

	database.DB.Create(&model.Passkey{UserID: alice.ID, Name: "Alice key", CredentialID: []byte("c1")})
	database.DB.Create(&model.Passkey{UserID: bob.ID, Name: "Bob key", CredentialID: []byte("c2")})

	// Bob sees only his own.
	rec := pkReq(t, h, http.MethodGet, "/auth/passkeys", "", &bob)
	if strings.Contains(rec.Body.String(), "Alice key") {
		t.Errorf("cross-user leak: %s", rec.Body.String())
	}

	// Bob cannot delete Alice's key: find its id first.
	var aliceKey model.Passkey
	database.DB.Where("user_id = ?", alice.ID).First(&aliceKey)
	rec = pkReq(t, h, http.MethodDelete, "/auth/passkeys/"+strconv.Itoa(int(aliceKey.ID)), "", &bob)
	if rec.Code != http.StatusNotFound {
		t.Errorf("delete other's key = %d, want 404", rec.Code)
	}
	// Alice deletes her own.
	rec = pkReq(t, h, http.MethodDelete, "/auth/passkeys/"+strconv.Itoa(int(aliceKey.ID)), "", &alice)
	if rec.Code != http.StatusOK {
		t.Errorf("delete own key = %d, want 200", rec.Code)
	}
}

func extractChallenge(t *testing.T, body string) string {
	t.Helper()
	i := strings.Index(body, `"challenge":"`)
	if i < 0 {
		t.Fatalf("no challenge in %s", body)
	}
	rest := body[i+len(`"challenge":"`):]
	return rest[:strings.IndexByte(rest, '"')]
}
