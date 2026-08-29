// Package handler: passkey.go implements WebAuthn (passkey) registration and
// passwordless login on top of github.com/go-webauthn/webauthn.
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// challengeTTL bounds the registration/login ceremony window.
const challengeTTL = 2 * time.Minute

/* ── Challenge store ─────────────────────────────────────────────────────── */

// challengeEntry is one pending WebAuthn ceremony.
type challengeEntry struct {
	session webauthn.SessionData
	userID  uint // 0 until discovered at finish (discoverable login)
	expires time.Time
}

// challengeStore holds pending ceremonies in memory. The TTL is short and a
// lost entry merely forces the user to retry, so eviction-on-read is enough.
type challengeStore struct {
	mu      sync.Mutex
	entries map[string]*challengeEntry
}

func newChallengeStore() *challengeStore {
	return &challengeStore{entries: make(map[string]*challengeEntry)}
}

func (s *challengeStore) put(key string, e *challengeEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Opportunistic sweep while holding the lock.
	now := time.Now()
	for k, v := range s.entries {
		if now.After(v.expires) {
			delete(s.entries, k)
		}
	}
	e.expires = now.Add(challengeTTL)
	s.entries[key] = e
}

func (s *challengeStore) take(key string) (*challengeEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if ok {
		delete(s.entries, key) // single use
	}
	return e, ok && time.Now().Before(e.expires)
}

/* ── WebAuthn user adapter ───────────────────────────────────────────────── */

// webauthnUser adapts a DB user + their stored passkeys to the library's
// interface. The WebAuthnID is the decimal user id — stable and unique per
// user, and round-trips through authenticator user handles.
type webauthnUser struct {
	user  *model.User
	creds []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte { return []byte(strconv.FormatUint(uint64(u.user.ID), 10)) }
func (u *webauthnUser) WebAuthnName() string {
	if u.user.Email != "" {
		return u.user.Email
	}
	return u.user.Username
}
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.user.DisplayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

// loadCredentials converts stored passkeys into library credentials.
func loadCredentials(userID uint) []webauthn.Credential {
	var keys []model.Passkey
	database.DB.Where("user_id = ?", userID).Find(&keys)
	creds := make([]webauthn.Credential, 0, len(keys))
	for _, k := range keys {
		var transports []protocol.AuthenticatorTransport
		for _, t := range strings.Split(k.Transport, ",") {
			if t = strings.TrimSpace(t); t != "" {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
		}
		creds = append(creds, webauthn.Credential{
			ID:              k.CredentialID,
			PublicKey:       k.PublicKey,
			AttestationType: k.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: k.BackupEligible,
				BackupState:    k.BackupState,
			},
		})
	}
	return creds
}

/* ── Handler ─────────────────────────────────────────────────────────────── */

// PasskeyHandler serves the WebAuthn registration and login ceremonies.
type PasskeyHandler struct {
	auth       *AuthHandler // reuse startSession on successful login
	challenges *challengeStore
}

// NewPasskeyHandler creates a PasskeyHandler.
func NewPasskeyHandler(auth *AuthHandler) *PasskeyHandler {
	return &PasskeyHandler{auth: auth, challenges: newChallengeStore()}
}

// webAuthn builds a client from the current settings; nil when disabled or
// misconfigured (the endpoints then answer 503).
func (h *PasskeyHandler) webAuthn() *webauthn.WebAuthn {
	cfg := setting.Passkey()
	if !cfg.Enabled {
		return nil
	}
	w, err := webauthn.New(&webauthn.Config{
		RPID:                  cfg.RPID,
		RPDisplayName:         setting.SiteName(),
		RPOrigins:             cfg.Origins,
		AttestationPreference: protocol.PreferNoAttestation,
	})
	if err != nil {
		return nil
	}
	return w
}

/* ── Registration (session required) ─────────────────────────────────────── */

// RegisterBegin starts credential enrollment for the signed-in user.
func (h *PasskeyHandler) RegisterBegin(w http.ResponseWriter, r *http.Request) {
	wa := h.webAuthn()
	if wa == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "passkey is not enabled")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	wu := &webauthnUser{user: u, creds: loadCredentials(u.ID)}
	options, session, err := wa.BeginRegistration(wu)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "failed to begin registration: "+err.Error())
		return
	}

	key := randomToken(16)
	h.challenges.put(key, &challengeEntry{session: *session, userID: u.ID})
	writeJSON(w, http.StatusOK, map[string]interface{}{"challenge": key, "options": options})
}

// RegisterFinish verifies the attestation and stores the credential. The
// challenge key and the optional credential name travel as query parameters
// so the request body stays a pristine WebAuthn protocol payload.
func (h *PasskeyHandler) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	wa := h.webAuthn()
	if wa == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "passkey is not enabled")
		return
	}
	actor := middleware.GetUser(r.Context())
	if actor == nil || actor.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	entry, ok := h.challenges.take(r.URL.Query().Get("challenge"))
	if !ok || entry.userID != actor.ID {
		writeAPIError(w, http.StatusBadRequest, "challenge expired or invalid")
		return
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid registration payload: "+err.Error())
		return
	}

	var user model.User
	if err := database.DB.First(&user, actor.ID).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}
	wu := &webauthnUser{user: &user, creds: loadCredentials(user.ID)}

	cred, err := wa.CreateCredential(wu, entry.session, parsed)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "registration failed: "+err.Error())
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = "Passkey " + time.Now().Format("01-02 15:04")
	}
	if len(name) > 64 {
		name = name[:64]
	}
	record := model.Passkey{
		UserID:          user.ID,
		Name:            name,
		CredentialID:    cred.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       transportsCSV(cred.Transport),
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
	}
	if err := database.DB.Create(&record).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to store credential")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"passkey": map[string]interface{}{
			"id": record.ID, "name": record.Name, "created_at": record.CreatedAt,
		},
	})
}

/* ── Login (public) ──────────────────────────────────────────────────────── */

type passkeyLoginBeginRequest struct {
	Username string `json:"username"` // empty = discoverable credential flow
}

// LoginBegin issues an authentication challenge. Without a username the
// ceremony relies on discoverable credentials (passkeys stored on the
// device); with one, it targets that account's credentials.
func (h *PasskeyHandler) LoginBegin(w http.ResponseWriter, r *http.Request) {
	wa := h.webAuthn()
	if wa == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "passkey is not enabled")
		return
	}

	var req passkeyLoginBeginRequest
	if r.ContentLength > 0 && !decodeJSON(w, r, &req) {
		return
	}

	key := randomToken(16)
	username := strings.TrimSpace(req.Username)

	if username != "" {
		var user model.User
		if err := database.DB.Where("username = ? OR email = ?", username, strings.ToLower(username)).
			First(&user).Error; err != nil || user.Status != "active" {
			// Uniform response: no account enumeration through this path.
			h.challenges.put(key, &challengeEntry{userID: 0}) // dead challenge
			writeJSON(w, http.StatusOK, map[string]interface{}{"challenge": key, "options": nil})
			return
		}
		wu := &webauthnUser{user: &user, creds: loadCredentials(user.ID)}
		options, session, err := wa.BeginLogin(wu)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "no registered passkey for this account")
			return
		}
		h.challenges.put(key, &challengeEntry{session: *session, userID: user.ID})
		writeJSON(w, http.StatusOK, map[string]interface{}{"challenge": key, "options": options})
		return
	}

	// Discoverable: the authenticator announces the user at finish time.
	options, session, err := wa.BeginDiscoverableLogin()
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "discoverable login unavailable")
		return
	}
	h.challenges.put(key, &challengeEntry{session: *session, userID: 0})
	writeJSON(w, http.StatusOK, map[string]interface{}{"challenge": key, "options": options})
}

// LoginFinish verifies the assertion, resolves the account (from the stored
// challenge, or the response user handle for discoverable flows), and opens a
// management session. The challenge key travels as ?challenge= so the body
// remains a pristine protocol payload.
func (h *PasskeyHandler) LoginFinish(w http.ResponseWriter, r *http.Request) {
	wa := h.webAuthn()
	if wa == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "passkey is not enabled")
		return
	}

	entry, ok := h.challenges.take(r.URL.Query().Get("challenge"))
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "challenge expired or invalid")
		return
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid assertion payload: "+err.Error())
		return
	}

	// Resolve the account.
	var userID uint
	if entry.userID != 0 {
		userID = entry.userID
	} else {
		handle := parsed.Response.UserHandle
		if len(handle) == 0 {
			writeAPIError(w, http.StatusBadRequest, "missing user handle")
			return
		}
		id, err := strconv.ParseUint(string(handle), 10, 64)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid user handle")
			return
		}
		userID = uint(id)
	}

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil || user.Status != "active" {
		writeAPIError(w, http.StatusUnauthorized, "account not available")
		return
	}
	wu := &webauthnUser{user: &user, creds: loadCredentials(user.ID)}

	cred, err := wa.ValidateLogin(wu, entry.session, parsed)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "passkey verification failed")
		return
	}

	// Persist the new sign count and last-used timestamp.
	database.DB.Model(&model.Passkey{}).
		Where("user_id = ? AND credential_id = ?", user.ID, cred.ID).
		Updates(map[string]interface{}{
			"sign_count":   cred.Authenticator.SignCount,
			"backup_state": cred.Flags.BackupState,
			"last_used_at": time.Now(),
		})

	h.auth.startSession(w, &user)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

/* ── Management ──────────────────────────────────────────────────────────── */

// List returns the caller's registered passkeys.
func (h *PasskeyHandler) List(w http.ResponseWriter, r *http.Request) {
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var keys []model.Passkey
	database.DB.Where("user_id = ?", u.ID).Order("id DESC").Find(&keys)
	writeJSON(w, http.StatusOK, map[string]interface{}{"passkeys": keys})
}

// Delete removes one of the caller's passkeys by id.
func (h *PasskeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid passkey id")
		return
	}
	res := database.DB.Where("id = ? AND user_id = ?", id, u.ID).Delete(&model.Passkey{})
	if res.Error != nil || res.RowsAffected == 0 {
		writeAPIError(w, http.StatusNotFound, "passkey not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

/* ── Helpers ─────────────────────────────────────────────────────────────── */

func transportsCSV(ts []protocol.AuthenticatorTransport) string {
	parts := make([]string, 0, len(ts))
	for _, t := range ts {
		parts = append(parts, string(t))
	}
	return strings.Join(parts, ",")
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}
