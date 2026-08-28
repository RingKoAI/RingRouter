package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// fakeMailer records deliveries instead of touching a live SMTP relay.
type fakeMailer struct {
	mu    sync.Mutex
	sends []mail
}

type mail struct {
	to, subject, body string
}

func (f *fakeMailer) Send(to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends = append(f.sends, mail{to: to, subject: subject, body: body})
	return nil
}

func (f *fakeMailer) deliveries() []mail {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]mail, len(f.sends))
	copy(out, f.sends)
	return out
}

// setupCodeTestDB wires an in-memory SQLite database and a complete SMTP
// configuration, returning a ready AuthHandler backed by a fake mailer.
func setupCodeTestDB(t *testing.T) *AuthHandler {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		database.DB = nil
	})

	opts := map[string]string{
		setting.KeySiteName:    "RingRouterTest",
		setting.KeySMTPEnabled: "true",
		setting.KeySMTPHost:    "smtp.example.com",
		setting.KeySMTPPort:    "587",
		setting.KeySMTPFrom:    "no-reply@example.com",
	}
	for k, v := range opts {
		if err := setting.Set(k, v); err != nil {
			t.Fatalf("set option %s: %v", k, err)
		}
	}

	return NewAuthHandler("test-admin-key", &fakeMailer{})
}

func createTestUser(t *testing.T, username, email, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	u := model.User{
		Username: username, Email: email,
		DisplayName: username, Password: string(hash),
		Role: "user", Group: "default", Quota: 0, Status: "active",
	}
	if err := database.DB.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func postJSON(t *testing.T, h http.HandlerFunc, path string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	out := map[string]interface{}{}
	if len(rec.Body.Bytes()) > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal response %q: %v", rec.Body.String(), err)
		}
	}
	return rec, out
}

var codeInBody = regexp.MustCompile(`code is: (\d{6})`)

func deliveredCode(t *testing.T, fm *fakeMailer) string {
	t.Helper()
	d := fm.deliveries()
	if len(d) == 0 {
		t.Fatal("no email delivered")
	}
	m := codeInBody.FindStringSubmatch(d[len(d)-1].body)
	if m == nil {
		t.Fatalf("no code found in email body: %q", d[len(d)-1].body)
	}
	return m[1]
}

func TestSendCodeWithoutSMTPReturns503(t *testing.T) {
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		database.DB = nil
	})

	h := NewAuthHandler("test-admin-key", &fakeMailer{})
	rec, _ := postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "nobody@example.com"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestSendCodeForUnknownEmailSucceedsSilently(t *testing.T) {
	h := setupCodeTestDB(t)
	rec, _ := postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "ghost@example.com"})
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (no enumeration)", rec.Code)
	}
	fm := h.mailer.(*fakeMailer)
	if n := len(fm.deliveries()); n != 0 {
		t.Errorf("delivered %d emails for unknown address, want 0", n)
	}
}

func TestSendCodeInvalidEmailReturns400(t *testing.T) {
	h := setupCodeTestDB(t)
	rec, _ := postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "not-an-email"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestSendCodeDeliversAndCooldown(t *testing.T) {
	h := setupCodeTestDB(t)
	createTestUser(t, "alice", "alice@example.com", "password123")

	rec, _ := postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "alice@example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("first code status = %d, want 200", rec.Code)
	}
	fm := h.mailer.(*fakeMailer)
	if got := deliveredCode(t, fm); len(got) != 6 {
		t.Errorf("delivered code %q, want 6 digits", got)
	}
	if d := fm.deliveries(); d[0].to != "alice@example.com" {
		t.Errorf("delivered to %q, want alice@example.com", d[0].to)
	}

	// Cooldown window: a second request within 60s must be refused.
	rec, _ = postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "alice@example.com"})
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second code status = %d, want 429", rec.Code)
	}
}

func TestResetPasswordFullFlow(t *testing.T) {
	h := setupCodeTestDB(t)
	createTestUser(t, "bob", "bob@example.com", "old-password-123")

	postJSON(t, h.SendCode, "/api/auth/code", map[string]string{"email": "bob@example.com"})
	code := deliveredCode(t, h.mailer.(*fakeMailer))

	// Wrong code is rejected and burns one attempt.
	rec, _ := postJSON(t, h.ResetPassword, "/api/auth/reset-password", map[string]string{
		"email": "bob@example.com", "code": "000000", "password": "new-password-123",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("wrong code status = %d, want 400", rec.Code)
	}

	// Correct code resets the password.
	rec, _ = postJSON(t, h.ResetPassword, "/api/auth/reset-password", map[string]string{
		"email": "bob@example.com", "code": code, "password": "new-password-123",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("reset status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}

	// Replaying the same code must fail (single use).
	rec, _ = postJSON(t, h.ResetPassword, "/api/auth/reset-password", map[string]string{
		"email": "bob@example.com", "code": code, "password": "another-password-123",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("replay status = %d, want 400", rec.Code)
	}

	// The new password must authenticate.
	var user model.User
	if err := database.DB.Where("email = ?", "bob@example.com").First(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("new-password-123")) != nil {
		t.Error("new password does not match after reset")
	}

	loginRec, _ := postJSON(t, h.Login, "/api/auth/login", map[string]string{
		"account": "bob@example.com", "password": "new-password-123",
	})
	if loginRec.Code != http.StatusOK {
		t.Errorf("login with new password = %d, want 200", loginRec.Code)
	}
}

func TestResetPasswordExpiresAttempts(t *testing.T) {
	h := setupCodeTestDB(t)
	createTestUser(t, "carol", "carol@example.com", "password123")

	// Burn all attempts with wrong codes; the record is deleted and further
	// requests are rejected with the same generic error.
	for i := 0; i < codeMaxAttempts+1; i++ {
		rec, _ := postJSON(t, h.ResetPassword, "/api/auth/reset-password", map[string]string{
			"email": "carol@example.com", "code": "111111", "password": "password123",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d status = %d, want 400", i, rec.Code)
		}
	}
}

func TestGenerateNumericCode(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generateNumericCode(6)
		if err != nil {
			t.Fatalf("generateNumericCode: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("code %q has length %d, want 6", code, len(code))
		}
		if _, err := strconv.Atoi(code); err != nil {
			t.Errorf("code %q is not numeric", code)
		}
	}
	if _, err := generateNumericCode(0); err == nil {
		t.Error("generateNumericCode(0) should error")
	}
	if _, err := generateNumericCode(17); err == nil {
		t.Error("generateNumericCode(17) should error")
	}
}

func TestSMTPConfigured(t *testing.T) {
	cases := []struct {
		name string
		cfg  setting.SMTPConfig
		want bool
	}{
		{"complete", setting.SMTPConfig{Host: "smtp.example.com", Port: 587, From: "a@b.com"}, true},
		{"missing host", setting.SMTPConfig{Port: 587, From: "a@b.com"}, false},
		{"missing port", setting.SMTPConfig{Host: "smtp.example.com", From: "a@b.com"}, false},
		{"missing from", setting.SMTPConfig{Host: "smtp.example.com", Port: 587}, false},
		{"empty", setting.SMTPConfig{}, false},
	}
	for _, c := range cases {
		if got := smtpConfigured(c.cfg); got != c.want {
			t.Errorf("%s: smtpConfigured = %v, want %v", c.name, got, c.want)
		}
	}
}
