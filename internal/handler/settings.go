package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// SecretSealer is defined in setup.go (package-level interface, shared by all
// handlers that need to seal/open secrets at rest).

// SettingsHandler serves the admin settings API. Updates take effect on the
// next request because every consumer reads through the setting package's
// in-process cache.
type SettingsHandler struct {
	sealer SecretSealer
}

// NewSettingsHandler creates a SettingsHandler. The sealer is required for
// any SMTP password writes; reads are decoupled and may return sealed values
// when a decrypter is unavailable.
func NewSettingsHandler(sealer SecretSealer) *SettingsHandler {
	return &SettingsHandler{sealer: sealer}
}

/* ── Requests ─────────────────────────────────────────────────────────────── */

// smtpPayload is shared with setup.go (same package, one definition).

type passkeyPayload struct {
	Enabled *bool   `json:"enabled"`
	RPID    *string `json:"rp_id"`      // e.g. example.com
	Origins *string `json:"rp_origins"` // csv, e.g. https://example.com
}

type settingsPatch struct {
	SiteName       *string         `json:"site_name,omitempty"`
	Announcement   *string         `json:"announcement,omitempty"`
	UsageMode      *string         `json:"usage_mode,omitempty"`
	SMTP           *smtpPayload    `json:"smtp,omitempty"`    // nil/omitted = leave SMTP alone
	Passkey        *passkeyPayload `json:"passkey,omitempty"` // nil/omitted = leave passkey alone
	PlazaPublic    *bool           `json:"plaza_public,omitempty"`
	SensitiveWords *string         `json:"sensitive_words,omitempty"` // csv; empty clears
}

/* ── Get ──────────────────────────────────────────────────────────────────── */

// Get returns the current site, announcement, usage mode, and SMTP config.
// SMTP password is never echoed; the response exposes has_password instead.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	smtp := setting.SMTP(h.sealer.Decrypt)
	pk := setting.Passkey()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"site_name":    setting.SiteName(),
		"announcement": setting.Announcement(),
		"usage_mode":   string(setting.CurrentUsageMode()),
		"smtp": map[string]interface{}{
			"host":         smtp.Host,
			"port":         smtp.Port,
			"username":     smtp.Username,
			"from":         smtp.From,
			"has_password": smtp.Password != "",
			"enabled":      smtp.Enabled,
		},
		"passkey": map[string]interface{}{
			"enabled":    pk.Enabled,
			"rp_id":      pk.RPID,
			"rp_origins": strings.Join(pk.Origins, ","),
		},
		"plaza_public":    setting.PlazaPublic(),
		"sensitive_words": setting.Get(setting.KeySensitiveWords),
	})
}

/* ── Update ──────────────────────────────────────────────────────────────── */

// Update applies a partial patch. Each non-nil field is validated then
// persisted via setting.Set, which updates both the database and the
// in-process cache so handlers and the mailer see the new value on their
// next read.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	var req settingsPatch
	if !decodeJSON(w, r, &req) {
		return
	}

	applied := map[string]interface{}{}

	// site_name: trim, bounded length.
	if req.SiteName != nil {
		name := strings.TrimSpace(*req.SiteName)
		if len(name) == 0 || len(name) > 64 {
			writeAPIError(w, http.StatusBadRequest, "site_name must be 1-64 characters")
			return
		}
		if err := setting.Set(setting.KeySiteName, name); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save site_name")
			return
		}
		applied["site_name"] = name
	}

	// announcement: free-form, empty string clears the notice.
	if req.Announcement != nil {
		body := strings.TrimSpace(*req.Announcement)
		if len(body) > 1024 {
			writeAPIError(w, http.StatusBadRequest, "announcement must be at most 1024 characters")
			return
		}
		if err := setting.Set(setting.KeyAnnouncement, body); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save announcement")
			return
		}
		applied["announcement"] = body
	}

	// usage_mode: validate against whitelist.
	if req.UsageMode != nil {
		mode := setting.UsageMode(*req.UsageMode)
		if !setting.IsValidUsageMode(mode) {
			writeAPIError(w, http.StatusBadRequest, "usage_mode must be one of external, self, demo")
			return
		}
		if err := setting.Set(setting.KeyUsageMode, string(mode)); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save usage mode")
			return
		}
		applied["usage_mode"] = string(mode)
	}

	// SMTP block: only when supplied.
	if req.SMTP != nil {
		if req.SMTP.Host == "" || req.SMTP.Port <= 0 || req.SMTP.Port > 65535 || req.SMTP.From == "" {
			writeAPIError(w, http.StatusBadRequest, "smtp requires host, port and from")
			return
		}
		// Seal the password (if any) and write each field.
		sealed := ""
		if req.SMTP.Password != "" {
			var err error
			sealed, err = h.sealer.Encrypt(req.SMTP.Password)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to seal SMTP password")
				return
			}
		}
		// Read the prior sealed password so an empty payload keeps the existing one.
		prevSealed := setting.Get(setting.KeySMTPPassword)
		pwSealed := prevSealed
		if req.SMTP.Password != "" {
			pwSealed = sealed
		}
		opts := map[string]string{
			setting.KeySMTPEnabled:  "true",
			setting.KeySMTPHost:     req.SMTP.Host,
			setting.KeySMTPPort:     strconv.Itoa(req.SMTP.Port),
			setting.KeySMTPUsername: req.SMTP.Username,
			setting.KeySMTPPassword: pwSealed,
			setting.KeySMTPFrom:     req.SMTP.From,
		}
		for k, v := range opts {
			if err := setting.Set(k, v); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to save SMTP settings")
				return
			}
		}
		applied["smtp"] = map[string]interface{}{"host": req.SMTP.Host, "port": req.SMTP.Port, "from": req.SMTP.From}
	}

	// Passkey block: only when supplied.
	if req.Passkey != nil {
		if req.Passkey.RPID != nil {
			rpID := strings.TrimSpace(*req.Passkey.RPID)
			if rpID == "" || len(rpID) > 253 {
				writeAPIError(w, http.StatusBadRequest, "passkey rp_id must be 1-253 characters")
				return
			}
			if err := setting.Set(setting.KeyPasskeyRPID, rpID); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to save passkey settings")
				return
			}
		}
		if req.Passkey.Origins != nil {
			origins := strings.TrimSpace(*req.Passkey.Origins)
			if origins == "" || len(origins) > 1024 {
				writeAPIError(w, http.StatusBadRequest, "passkey rp_origins must be 1-1024 characters")
				return
			}
			if err := setting.Set(setting.KeyPasskeyOrigins, origins); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to save passkey settings")
				return
			}
		}
		if req.Passkey.Enabled != nil {
			v := ""
			if *req.Passkey.Enabled {
				v = "true"
			}
			if err := setting.Set(setting.KeyPasskeyEnabled, v); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to save passkey settings")
				return
			}
		}
		applied["passkey"] = true
	}

	// Sensitive-word blocklist (applies to gateway request text).
	if req.SensitiveWords != nil {
		v := strings.TrimSpace(*req.SensitiveWords)
		if len(v) > 2048 {
			writeAPIError(w, http.StatusBadRequest, "sensitive_words must be at most 2048 characters")
			return
		}
		if err := setting.Set(setting.KeySensitiveWords, v); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save sensitive_words")
			return
		}
		applied["sensitive_words"] = v
	}

	// Plaza visibility toggle.
	if req.PlazaPublic != nil {
		v := ""
		if !*req.PlazaPublic {
			v = "false"
		}
		if err := setting.Set(setting.KeyPlazaPublic, v); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to save plaza visibility")
			return
		}
		applied["plaza_public"] = *req.PlazaPublic
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"applied": applied})
}
