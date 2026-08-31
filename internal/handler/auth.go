package handler

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/safenet"
	"github.com/RingKoAI/RingRouter/internal/setting"
	"github.com/RingKoAI/RingRouter/internal/turnstile"
)

// Auth limits.
const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt hard limit
	sessionTTL     = 7 * 24 * time.Hour
)

// Email verification-code limits.
const (
	codeDigits         = 6
	codeTTL            = 5 * time.Minute
	codeMaxAttempts    = 5
	codeResendCooldown = 60 * time.Second
)

var (
	usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
	emailRe    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	codeRe     = regexp.MustCompile(`^\d{6}$`)
)

// mailSender delivers transactional email. service.Mailer implements it; tests
// may inject a fake to avoid a live SMTP dependency.
type mailSender interface {
	Send(to, subject, body string) error
}

// AuthHandler serves the management-plane authentication API.
type AuthHandler struct {
	adminKey string
	mailer   mailSender
}

// NewAuthHandler creates an AuthHandler. adminKey is the bootstrap key from
// the environment; mailer delivers verification codes (nil disables the
// code flow). The announcement shown to operators is read live from the
// options table so it hot-reloads after a settings update.
func NewAuthHandler(adminKey string, mailer mailSender) *AuthHandler {
	return &AuthHandler{adminKey: adminKey, mailer: mailer}
}

/* ── Requests ─────────────────────────────────────────────────────────────── */

type registerRequest struct {
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	TurnstileToken string `json:"cf_turnstile_response"`
}

type loginRequest struct {
	Account        string `json:"account"` // username or email
	Password       string `json:"password"`
	TurnstileToken string `json:"cf_turnstile_response"`
}

type adminKeyRequest struct {
	Key            string `json:"key"`
	TurnstileToken string `json:"cf_turnstile_response"`
}

type sendCodeRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

/* ── Handlers ─────────────────────────────────────────────────────────────── */

// verifyTurnstile validates the cf_turnstile_response token when Turnstile is
// enabled. Returns nil if Turnstile is not configured (skip validation).
func verifyTurnstile(ctx context.Context, token, remoteIP string) error {
	if !turnstile.Enabled() {
		return nil
	}
	return turnstile.ValidateToken(ctx, token, remoteIP)
}

// Register creates a new member account. Available only after the first-run
// setup wizard has created an administrator.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	if h.setupPending() {
		writeAPIError(w, http.StatusForbidden, "instance setup is not completed yet")
		return
	}

	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := verifyTurnstile(r.Context(), req.TurnstileToken, extractIP(r)); err != nil {
		writeAPIError(w, http.StatusForbidden, "turnstile verification failed")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = normalizeEmail(req.Email)

	if !usernameRe.MatchString(req.Username) {
		writeAPIError(w, http.StatusBadRequest, "username must be 3-32 chars of letters, digits, - or _")
		return
	}
	if req.Email == "" || len(req.Email) > 256 || !emailRe.MatchString(req.Email) {
		writeAPIError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		writeAPIError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	// All self-registered accounts are regular members; the built-in admin
	// is bootstrapped at startup and promotes others from the console.
	user := model.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.Username,
		Password:    string(hash),
		Role:        "user",
		Group:       "default",
		Quota:       0,
	}

	// Usernames and emails are matched case-insensitively: "Admin" and
	// "admin" are the same identity (emails are additionally normalized to
	// lower case at rest). The regex still governs the display shape.
	if err := database.DB.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)",
		req.Username, req.Email).First(&model.User{}).Error; err == nil {
		writeAPIError(w, http.StatusConflict, "username or email already registered")
		return
	}
	if err := database.DB.Create(&user).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	h.startSession(w, r, &user)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"user": user})
}

// Login authenticates with username/email + password.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := verifyTurnstile(r.Context(), req.TurnstileToken, extractIP(r)); err != nil {
		writeAPIError(w, http.StatusForbidden, "turnstile verification failed")
		return
	}

	account := strings.TrimSpace(req.Account)
	if account == "" || req.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "account and password are required")
		return
	}

	var user model.User
	// Case-insensitive account lookup: usernames are unique ignoring case,
	// emails are stored lower-cased.
	err := database.DB.Where("LOWER(username) = LOWER(?) OR email = ?",
		account, strings.ToLower(account)).First(&user).Error
	if err != nil {
		// Constant-ish work to avoid user-enumeration timing signal.
		bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5B0G1S3kAxKq0mMNrXYvJrOQzCkS"), []byte(req.Password))
		writeAPIError(w, http.StatusUnauthorized, "invalid account or password")
		return
	}
	if user.Status != "active" {
		writeAPIError(w, http.StatusForbidden, "user account disabled")
		return
	}
	if user.Password == "" {
		writeAPIError(w, http.StatusConflict, "account has no password set")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid account or password")
		return
	}

	h.startSession(w, r, &user)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

// AdminKey exchanges the bootstrap admin key for a management session.
func (h *AuthHandler) AdminKey(w http.ResponseWriter, r *http.Request) {
	if h.adminKey == "" {
		writeAPIError(w, http.StatusNotFound, "admin key not configured")
		return
	}

	var req adminKeyRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := verifyTurnstile(r.Context(), req.TurnstileToken, extractIP(r)); err != nil {
		writeAPIError(w, http.StatusForbidden, "turnstile verification failed")
		return
	}

	if req.Key == "" || subtle.ConstantTimeCompare([]byte(req.Key), []byte(h.adminKey)) != 1 {
		writeAPIError(w, http.StatusUnauthorized, "invalid admin key")
		return
	}

	// Bootstrap sessions use UserID 0 (virtual admin, no DB record).
	h.startSession(w, r, &model.User{ID: 0, Username: "admin", Role: "admin", Quota: -1, Status: "active"})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{"id": 0, "username": "admin", "role": "admin"},
	})
}

// Logout revokes the current session.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(middleware.SessionCookieName); err == nil && c.Value != "" {
		if database.DB != nil {
			database.DB.Delete(&model.Session{}, "id = ?", middleware.SessionDigest(c.Value))
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// Me returns the authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u := middleware.GetUser(r.Context())
	if u == nil {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": u})
}

// SendCode issues a 6-digit password-reset code by email. It fails closed
// (503) when no complete SMTP configuration exists — the same condition the
// frontend uses to show the "contact administrator" screen, so the two sides
// stay aligned. Unregistered addresses return success without sending, which
// prevents account enumeration.
func (h *AuthHandler) SendCode(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	if h.mailer == nil || !smtpConfigured(setting.SMTP(nil)) {
		writeAPIError(w, http.StatusServiceUnavailable, "email service not configured")
		return
	}

	var req sendCodeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	// Cooldown: at most one code per address per window.
	var last model.Code
	database.DB.Where("email = ?", email).Order("created_at DESC").Limit(1).Find(&last)
	if last.ID != 0 && time.Since(last.CreatedAt) < codeResendCooldown {
		writeAPIError(w, http.StatusTooManyRequests, "please wait before requesting another code")
		return
	}

	// Unregistered addresses complete silently.
	var count int64
	database.DB.Model(&model.User{}).Where("email = ?", email).Count(&count)
	if count == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
		return
	}

	code, err := generateNumericCode(codeDigits)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}

	// A fresh code invalidates any outstanding one for the same address.
	if err := database.DB.Where("email = ?", email).Delete(&model.Code{}).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to store code")
		return
	}
	rec := model.Code{
		Email:     email,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(codeTTL),
	}
	if err := database.DB.Create(&rec).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to store code")
		return
	}

	// Deliver after persist; roll the record back on failure so a dead relay
	// cannot accumulate unusable codes.
	name := setting.SiteName()
	subject := fmt.Sprintf("[%s] Password reset code", name)
	body := fmt.Sprintf("Your %s password reset code is: %s\n\nIt expires in %d minutes. If you did not request this, you can safely ignore this email.",
		name, code, int(codeTTL.Minutes()))
	if err := h.mailer.Send(email, subject, body); err != nil {
		database.DB.Delete(&rec)
		writeAPIError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// ResetPassword verifies a sent code and sets a new password. Success also
// revokes every existing session for the account, so a stolen session cookie
// cannot outlive the reset.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var req resetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if !codeRe.MatchString(req.Code) {
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		writeAPIError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}

	var rec model.Code
	database.DB.Where("email = ? AND used = ?", email, false).
		Order("created_at DESC").Limit(1).Find(&rec)
	if rec.ID == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if time.Now().After(rec.ExpiresAt) {
		database.DB.Delete(&rec)
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if rec.Attempts >= codeMaxAttempts {
		database.DB.Delete(&rec)
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(rec.CodeHash), []byte(req.Code)) != nil {
		database.DB.Model(&rec).Update("attempts", gorm.Expr("attempts + 1"))
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}

	// Code accepted: wipe it and any siblings so it cannot be replayed.
	if err := database.DB.Where("email = ?", email).Delete(&model.Code{}).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	var user model.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	if user.Status != "active" {
		writeAPIError(w, http.StatusForbidden, "user account disabled")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if err := database.DB.Model(&user).Update("password", string(hash)).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	// Revoke all sessions: sessions must be re-established with the new password.
	database.DB.Where("user_id = ?", user.ID).Delete(&model.Session{})

	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// generateNumericCode produces a uniform decimal code of the requested length
// using rejection sampling, so every digit is equally likely.
func generateNumericCode(digits int) (string, error) {
	if digits <= 0 || digits > 16 {
		return "", fmt.Errorf("code length out of range: %d", digits)
	}
	out := make([]byte, 0, digits)
	buf := make([]byte, 64)
	for len(out) < digits {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, v := range buf {
			if v < 250 { // avoid modulo bias
				out = append(out, '0'+v%10)
				if len(out) == digits {
					break
				}
			}
		}
	}
	return string(out), nil
}

// Announcement returns the operator-configured notice (may be empty).
// It is read live from the options table; updates via the admin settings API
// become effective on the next request without a restart.
func (h *AuthHandler) Announcement(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"announcement": setting.Announcement()})
}

/* ── Helpers ──────────────────────────────────────────────────────────────── */

// setupPending reports whether the instance still lacks an administrator.
func (h *AuthHandler) setupPending() bool {
	var admins int64
	database.DB.Model(&model.User{}).Where("role = ?", "admin").Count(&admins)
	return admins == 0
}

// normalizeEmail trims/lower-cases an address and validates its shape.
// Returns "" when invalid.
func normalizeEmail(raw string) string {
	e := strings.ToLower(strings.TrimSpace(raw))
	if e == "" || len(e) > 256 || !emailRe.MatchString(e) {
		return ""
	}
	return e
}

// extractIP resolves the client address through the shared trusted-proxy
// logic (internal/safenet): forwarded headers are honored only when the TCP
// peer is a trusted proxy (loopback by default; TRUSTED_PROXIES to extend).
func extractIP(r *http.Request) string {
	return safenet.ClientIP(r)
}

// startSession creates a DB-backed session and sets the cookie.
func (h *AuthHandler) startSession(w http.ResponseWriter, r *http.Request, u *model.User) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	token := hex.EncodeToString(raw)

	sess := model.Session{
		ID:        middleware.SessionDigest(token),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if database.DB != nil {
		if err := database.DB.Create(&sess).Error; err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to persist session")
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsHTTPS(r),
	})
}

// requestIsHTTPS reports whether the connection reached the server over TLS,
// either terminated locally or by a proxy that forwarded the scheme. A forged
// X-Forwarded-Proto can only force the Secure attribute on, which fails safe.
func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// decodeJSON decodes a JSON body, rejecting oversized payloads, malformed
// documents, and trailing non-whitespace content after the first value (which
// would otherwise let smuggled second documents slip through unnoticed).
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB cap for auth payloads
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON body: unexpected trailing data")
		return false
	}
	return true
}

// writeAPIError emits the management-API error envelope.
func writeAPIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
