package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// tokenPrefix marks generated API keys. The 32 hex chars give 128 bits of
// entropy; the prefix makes accidental secret-scanner hits identifiable.
const tokenPrefix = "sk-rr-"

// TokenHandler serves the self-service API-key endpoints.
type TokenHandler struct{}

// NewTokenHandler creates a TokenHandler.
func NewTokenHandler() *TokenHandler {
	return &TokenHandler{}
}

type tokenPayload struct {
	Name        *string `json:"name"`
	Group       *string `json:"group"`        // optional; defaults to the user's group
	Status      *string `json:"status"`       // active, disabled
	Models      *string `json:"models"`       // comma whitelist; empty = unrestricted
	Subnet      *string `json:"subnet"`       // comma CIDRs; empty = unrestricted
	ExpiresDays *int    `json:"expires_days"` // >0 sets expiry N days out; 0 = never
}

/* ── List ────────────────────────────────────────────────────────────────── */

// List returns the caller's API keys (never the raw key again — only a mask).
func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var tokens []model.Token
	tx := database.DB.Where("user_id = ?", u.ID)
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		tx = tx.Where("name LIKE ?", "%"+q+"%")
	}
	if err := tx.Order("id DESC").Find(&tokens).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list tokens")
		return
	}
	type tokenView struct {
		model.Token
		KeyMasked string `json:"key_masked"`
	}
	out := make([]tokenView, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, tokenView{Token: t, KeyMasked: maskTokenKey(t.Key)})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tokens": out})
}

// normalizeCSV trims each element of a comma list and drops empties.
func normalizeCSV(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// validSubnets checks every entry parses as an IP or CIDR.
func validSubnets(csv string) bool {
	for _, c := range strings.Split(csv, ",") {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Contains(c, "/") {
			if _, _, err := net.ParseCIDR(c); err != nil {
				return false
			}
			continue
		}
		if net.ParseIP(c) == nil {
			return false
		}
	}
	return true
}

func maskTokenKey(key string) string {
	if len(key) < 12 {
		return "••••"
	}
	return key[:8] + "••••" + key[len(key)-4:]
}

/* ── Create ──────────────────────────────────────────────────────────────── */

// Create issues a new API key. The full key is returned exactly once, at
// creation time — it is never retrievable afterwards.
func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	var p tokenPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	name := ""
	if p.Name != nil {
		name = strings.TrimSpace(*p.Name)
	}
	if name == "" || len(name) > 64 {
		writeAPIError(w, http.StatusBadRequest, "name must be 1-64 characters")
		return
	}
	group := u.Group
	if p.Group != nil && strings.TrimSpace(*p.Group) != "" {
		group = strings.TrimSpace(*p.Group)
	}
	if !setting.GroupExists(group) {
		writeAPIError(w, http.StatusBadRequest, "group does not exist; create it via /api/admin/groups first")
		return
	}

	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}
	key := tokenPrefix + hex.EncodeToString(raw)

	tok := model.Token{
		UserID: u.ID,
		Key:    key,
		Name:   name,
		Group:  group,
		Status: "active",
		Quota:  -1, // unlimited unless the account enforces one
	}
	if p.Models != nil {
		if len(*p.Models) > 512 {
			writeAPIError(w, http.StatusBadRequest, "models must be at most 512 characters")
			return
		}
		tok.Models = normalizeCSV(*p.Models)
	}
	if p.Subnet != nil {
		sub := normalizeCSV(*p.Subnet)
		if sub != "" && !validSubnets(sub) {
			writeAPIError(w, http.StatusBadRequest, "subnet must be IPv4/IPv6 addresses or CIDRs, comma-separated")
			return
		}
		tok.Subnet = sub
	}
	if p.ExpiresDays != nil && *p.ExpiresDays > 0 {
		tok.ExpiredAt = time.Now().AddDate(0, 0, *p.ExpiresDays)
	}
	if u.Quota != -1 {
		tok.Quota = u.Quota
	}
	if err := database.DB.Create(&tok).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create token")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"token": tok})
}

/* ── Update ──────────────────────────────────────────────────────────────── */

// Update applies a partial patch (name / status / group).
func (h *TokenHandler) Update(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	var tok model.Token
	if err := database.DB.Where("id = ? AND user_id = ?", id, u.ID).First(&tok).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "token not found")
		return
	}

	var p tokenPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	updates := map[string]interface{}{}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" || len(name) > 64 {
			writeAPIError(w, http.StatusBadRequest, "name must be 1-64 characters")
			return
		}
		updates["name"] = name
	}
	if p.Status != nil {
		if *p.Status != "active" && *p.Status != "disabled" {
			writeAPIError(w, http.StatusBadRequest, "status must be active or disabled")
			return
		}
		updates["status"] = *p.Status
	}
	if p.Group != nil && strings.TrimSpace(*p.Group) != "" {
		group := strings.TrimSpace(*p.Group)
		if !setting.GroupExists(group) {
			writeAPIError(w, http.StatusBadRequest, "group does not exist")
			return
		}
		updates["group"] = group
	}
	if p.Models != nil {
		if len(*p.Models) > 512 {
			writeAPIError(w, http.StatusBadRequest, "models must be at most 512 characters")
			return
		}
		updates["models"] = normalizeCSV(*p.Models)
	}
	if p.Subnet != nil {
		sub := normalizeCSV(*p.Subnet)
		if sub != "" && !validSubnets(sub) {
			writeAPIError(w, http.StatusBadRequest, "subnet must be IPv4/IPv6 addresses or CIDRs, comma-separated")
			return
		}
		updates["subnet"] = sub
	}
	if p.ExpiresDays != nil {
		if *p.ExpiresDays <= 0 {
			updates["expired_at"] = time.Time{}
		} else {
			updates["expired_at"] = time.Now().AddDate(0, 0, *p.ExpiresDays)
		}
	}
	if len(updates) > 0 {
		if err := database.DB.Model(&tok).Updates(updates).Error; err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to update token")
			return
		}
	}
	database.DB.First(&tok, tok.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"token": tok, "key_masked": maskTokenKey(tok.Key)})
}

/* ── Delete ──────────────────────────────────────────────────────────────── */

// Delete removes one of the caller's keys.
func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid token id")
		return
	}
	res := database.DB.Where("id = ? AND user_id = ?", id, u.ID).Delete(&model.Token{})
	if res.Error != nil || res.RowsAffected == 0 {
		writeAPIError(w, http.StatusNotFound, "token not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}
