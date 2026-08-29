package handler

import (
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/service"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// SecretSealer abstracts seal/open for secrets stored at rest.
type SecretSealer interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(payload string) (string, error)
}

// smtpConfigured reports whether the persisted SMTP configuration is complete
// enough to deliver mail. It is the single source of truth shared by the
// status endpoints and the auth code path, so the frontend's "email service
// not configured" state can never drift from the backend's actual capability.
func smtpConfigured(cfg setting.SMTPConfig) bool {
	return cfg.Host != "" && cfg.Port != 0 && cfg.From != ""
}

// SetupHandler serves the first-run installation wizard API.
type SetupHandler struct {
	sealer SecretSealer
	mailer *service.Mailer
}

// NewSetupHandler creates a SetupHandler. The sealer encrypts SMTP passwords
// before they touch the database.
func NewSetupHandler(sealer SecretSealer, mailer *service.Mailer) *SetupHandler {
	return &SetupHandler{sealer: sealer, mailer: mailer}
}

/* ── Requests ─────────────────────────────────────────────────────────────── */

type smtpPayload struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type passkeySetupPayload struct {
	Enabled bool   `json:"enabled"`
	RPID    string `json:"rp_id"`
	Origins string `json:"rp_origins"` // csv
}

type completeRequest struct {
	Username       string               `json:"username"`
	Email          string               `json:"email"`
	Password       string               `json:"password"`
	SiteName       string               `json:"site_name"` // optional, defaults kept when empty
	UsageMode      string               `json:"usage_mode"`
	SMTP           *smtpPayload         `json:"smtp,omitempty"`    // nil/absent = skipped
	Passkey        *passkeySetupPayload `json:"passkey,omitempty"` // nil/absent = left disabled
	TurnstileToken string               `json:"cf_turnstile_response"`
}

type testSMTPRequest struct {
	SMTP smtpPayload `json:"smtp"`
	To   string      `json:"to"`
}

/* ── Status ──────────────────────────────────────────────────────────────── */

// Status reports whether the first-run wizard must run.
func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	smtpCfg := setting.SMTP(nil)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needed":          h.needed(),
		"usage_mode":      string(setting.CurrentUsageMode()),
		"smtp_configured": smtpConfigured(smtpCfg),
	})
}

// needed is true while the instance has no administrator account.
func (h *SetupHandler) needed() bool {
	if database.DB == nil {
		return false // cannot bootstrap without a database
	}
	var admins int64
	database.DB.Model(&model.User{}).Where("role = ?", "admin").Count(&admins)
	return admins == 0
}

/* ── SMTP test ───────────────────────────────────────────────────────────── */

// TestSMTP sends a probe message using the (unsaved) configuration submitted
// by the wizard so the operator can verify credentials before committing.
func (h *SetupHandler) TestSMTP(w http.ResponseWriter, r *http.Request) {
	var req testSMTPRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	cfg := req.SMTP
	if cfg.Host == "" || cfg.Port <= 0 || cfg.Port > 65535 || cfg.From == "" || req.To == "" {
		writeAPIError(w, http.StatusBadRequest, "host, port, from and to are required")
		return
	}

	if err := h.mailer.SendWith(setting.SMTPConfig{
		Host: cfg.Host, Port: cfg.Port,
		Username: cfg.Username, Password: cfg.Password, From: cfg.From,
	}, req.To, "RingRouter SMTP test",
		"This is a test message from your RingRouter setup wizard.\n\nIf you received it, SMTP is working."); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

/* ── Complete ────────────────────────────────────────────────────────────── */

// Complete finishes installation: creates the administrator account and
// persists usage mode + optional SMTP settings. It is refused once any admin
// exists, so the endpoint cannot be replayed.
func (h *SetupHandler) Complete(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	if !h.needed() {
		writeAPIError(w, http.StatusConflict, "setup already completed")
		return
	}

	var req completeRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := verifyTurnstile(r.Context(), req.TurnstileToken, extractIP(r)); err != nil {
		writeAPIError(w, http.StatusForbidden, "turnstile verification failed")
		return
	}

	// Validate admin credentials.
	if !usernameRe.MatchString(req.Username) {
		writeAPIError(w, http.StatusBadRequest, "username must be 3-32 chars of letters, digits, - or _")
		return
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		writeAPIError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}
	req.Email = normalizeEmail(req.Email)
	if req.Email == "" {
		writeAPIError(w, http.StatusBadRequest, "a valid email address is required for the administrator")
		return
	}

	// Validate usage mode.
	mode := setting.UsageMode(req.UsageMode)
	if !setting.IsValidUsageMode(mode) {
		writeAPIError(w, http.StatusBadRequest, "usage_mode must be one of external, self, demo")
		return
	}

	// Site name (optional; kept as-is when empty).
	siteName := strings.TrimSpace(req.SiteName)
	if siteName != "" {
		if len(siteName) > 64 {
			writeAPIError(w, http.StatusBadRequest, "site_name must be at most 64 characters")
			return
		}
	}

	// Passkey block: rp fields required when enabling.
	if req.Passkey != nil && req.Passkey.Enabled {
		rpID := strings.TrimSpace(req.Passkey.RPID)
		origins := strings.TrimSpace(req.Passkey.Origins)
		if rpID == "" || len(rpID) > 253 {
			writeAPIError(w, http.StatusBadRequest, "passkey rp_id must be 1-253 characters")
			return
		}
		if origins == "" || len(origins) > 1024 {
			writeAPIError(w, http.StatusBadRequest, "passkey rp_origins must be 1-1024 characters")
			return
		}
	}

	// Validate SMTP block when supplied.
	if req.SMTP != nil && (req.SMTP.Host == "" || req.SMTP.Port <= 0 || req.SMTP.Port > 65535 || req.SMTP.From == "") {
		writeAPIError(w, http.StatusBadRequest, "smtp requires host, port and from")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	admin := model.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.Username,
		Password:    string(hash),
		Role:        "admin",
		Quota:       -1, // unlimited
	}
	if err := database.DB.Create(&admin).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create administrator")
		return
	}

	// Persist usage mode.
	if err := setting.Set(setting.KeyUsageMode, string(mode)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to persist usage mode")
		return
	}

	// Persist site name when supplied.
	if siteName != "" {
		if err := setting.Set(setting.KeySiteName, siteName); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to persist site_name")
			return
		}
	}

	// Persist passkey settings when supplied.
	if req.Passkey != nil {
		val := ""
		if req.Passkey.Enabled {
			val = "true"
		}
		opts := map[string]string{setting.KeyPasskeyEnabled: val}
		if rpID := strings.TrimSpace(req.Passkey.RPID); rpID != "" {
			opts[setting.KeyPasskeyRPID] = rpID
		}
		if origins := strings.TrimSpace(req.Passkey.Origins); origins != "" {
			opts[setting.KeyPasskeyOrigins] = origins
		}
		for k, v := range opts {
			if err := setting.Set(k, v); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to persist passkey settings")
				return
			}
		}
	}

	// Persist SMTP settings when provided.
	if req.SMTP != nil {
		sealed := ""
		if req.SMTP.Password != "" {
			sealed, err = h.sealer.Encrypt(req.SMTP.Password)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to seal SMTP password")
				return
			}
		}
		opts := map[string]string{
			setting.KeySMTPEnabled:  "true",
			setting.KeySMTPHost:     req.SMTP.Host,
			setting.KeySMTPPort:     strconv.Itoa(req.SMTP.Port),
			setting.KeySMTPUsername: req.SMTP.Username,
			setting.KeySMTPPassword: sealed,
			setting.KeySMTPFrom:     req.SMTP.From,
		}
		for k, v := range opts {
			if err := setting.Set(k, v); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to persist SMTP settings")
				return
			}
		}
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":         true,
		"usage_mode": string(mode),
	})
}
