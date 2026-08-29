package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// maxChannels is a hard cap for the list endpoint.
const maxChannels = 500

// ChannelHandler serves the admin channel management API.
type ChannelHandler struct {
	sealer     SecretSealer
	invalidate func() // invoked after mutations; drops the gateway channel cache
}

// NewChannelHandler creates a ChannelHandler. The sealer encrypts channel
// API keys before they touch the database; invalidate is called after every
// write so the gateway re-reads the channel snapshot immediately.
func NewChannelHandler(sealer SecretSealer, invalidate func()) *ChannelHandler {
	if invalidate == nil {
		invalidate = func() {}
	}
	return &ChannelHandler{sealer: sealer, invalidate: invalidate}
}

/* ── Requests ─────────────────────────────────────────────────────────────── */

type channelPayload struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	Models       string `json:"models"`
	ModelMapping string `json:"model_mapping"`
	Status       string `json:"status"`
	Priority     *int   `json:"priority"`
	Weight       *int   `json:"weight"`
	Group        string `json:"group"`
}

// channelResponse is the wire-format channel: never exposes the raw API key.
type channelResponse struct {
	model.Channel
	APIKeyMasked string `json:"api_key_masked"`
}

func toChannelResponse(c model.Channel) channelResponse {
	return channelResponse{
		Channel:      c,
		APIKeyMasked: maskAPIKey(c.APIKey),
	}
}

func maskAPIKey(sealed string) string {
	if sealed == "" {
		return ""
	}
	// We never expose the raw ciphertext either; just show a presence hint.
	return "••••••" + "sealed"
}

/* ── Validation ──────────────────────────────────────────────────────────── */

var allowedChannelTypes = map[string]bool{
	"openai":    true,
	"anthropic": true,
	"google":    true,
}

var allowedChannelStatus = map[string]bool{
	"active":   true,
	"disabled": true,
}

func validateChannelPayload(p *channelPayload, requireAll bool) error {
	if p == nil {
		return errors.New("missing body")
	}
	name := strings.TrimSpace(p.Name)
	if requireAll && name == "" {
		return errors.New("name is required")
	}
	if name != "" && len(name) > 64 {
		return errors.New("name must be 1-64 characters")
	}
	if t := strings.TrimSpace(p.Type); t != "" {
		if !allowedChannelTypes[strings.ToLower(t)] {
			return errors.New("type must be one of: openai, anthropic, google")
		}
	} else if requireAll {
		return errors.New("type is required")
	}
	base := strings.TrimSpace(p.BaseURL)
	if requireAll && base == "" {
		return errors.New("base_url is required")
	}
	if base != "" && len(base) > 256 {
		return errors.New("base_url must be at most 256 characters")
	}
	if base != "" {
		if err := validateChannelURL(base); err != nil {
			return errors.New("base_url rejected: " + err.Error())
		}
	}
	if requireAll && strings.TrimSpace(p.APIKey) == "" {
		return errors.New("api_key is required when creating a channel")
	}
	if requireAll && strings.TrimSpace(p.Models) == "" {
		return errors.New("models is required (comma-separated list)")
	}
	if p.Status != "" && !allowedChannelStatus[strings.ToLower(p.Status)] {
		return errors.New("status must be active or disabled")
	}
	if p.Priority != nil && (*p.Priority < -1000000 || *p.Priority > 1000000) {
		return errors.New("priority out of range")
	}
	if p.Weight != nil && (*p.Weight < 0 || *p.Weight > 1000) {
		return errors.New("weight must be 0-1000")
	}
	if len(strings.TrimSpace(p.Group)) > 256 {
		return errors.New("group must be at most 256 characters")
	}
	// Group may list multiple groups comma-separated (one-api semantics);
	// every referenced group must already exist.
	for _, g := range strings.Split(p.Group, ",") {
		if g = strings.TrimSpace(g); g != "" && !setting.GroupExists(g) {
			return errors.New("group does not exist; create it via /api/admin/groups first")
		}
	}
	return nil
}

/* ── List ────────────────────────────────────────────────────────────────── */

// List returns channels filtered by an optional group.
func (h *ChannelHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	group := strings.TrimSpace(r.URL.Query().Get("group"))

	tx := database.DB.Model(&model.Channel{}).Order("priority DESC, id ASC")
	if group == "" && q == "" {
		var rows []model.Channel
		if err := tx.Limit(maxChannels).Find(&rows).Error; err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list channels")
			return
		}
		resp := make([]channelResponse, 0, len(rows))
		for _, c := range rows {
			resp = append(resp, toChannelResponse(c))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"channels": resp})
		return
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name LIKE ? OR models LIKE ? OR protocol LIKE ?", like, like, like)
	}
	var rows []model.Channel
	if err := tx.Limit(maxChannels).Find(&rows).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}
	// Group membership is comma-separated on the channel, so an exact SQL
	// match would miss multi-group channels — filter in memory.
	resp := make([]channelResponse, 0, len(rows))
	for _, c := range rows {
		if group != "" && !channelInGroup(&c, group) {
			continue
		}
		resp = append(resp, toChannelResponse(c))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"channels": resp})
}

// channelInGroup mirrors the gateway's routing semantics for admin filters.
func channelInGroup(c *model.Channel, group string) bool {
	for _, g := range strings.Split(c.Group, ",") {
		if strings.TrimSpace(g) == group {
			return true
		}
	}
	return false
}

// Groups returns the registered group names so the management UI can
// populate the channel form even when no channel uses the group yet.
func (h *ChannelHandler) Groups(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var groups []model.Group
	if err := database.DB.Order("is_default DESC, name ASC").Find(&groups).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"groups": names})
}

/* ── Create ──────────────────────────────────────────────────────────────── */

// Create persists a new channel after encrypting the API key.
func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var p channelPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	if err := validateChannelPayload(&p, true); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	sealed, err := h.sealer.Encrypt(strings.TrimSpace(p.APIKey))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to seal api_key")
		return
	}

	ch := model.Channel{
		Name:         strings.TrimSpace(p.Name),
		Protocol:     strings.ToLower(strings.TrimSpace(p.Type)),
		BaseURL:      strings.TrimSpace(p.BaseURL),
		APIKey:       sealed,
		Models:       strings.TrimSpace(p.Models),
		ModelMapping: strings.TrimSpace(p.ModelMapping),
		Status:       defaultIfEmpty(strings.ToLower(strings.TrimSpace(p.Status)), "active"),
		Group:        normalizeGroupList(p.Group),
		Priority:     derefOr(p.Priority, 0),
		Weight:       derefOr(p.Weight, 0),
	}
	if err := database.DB.Create(&ch).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create channel")
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusCreated, map[string]interface{}{"channel": toChannelResponse(ch)})
}

/* ── Read ────────────────────────────────────────────────────────────────── */

// Read returns one channel by id.
func (h *ChannelHandler) Read(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var ch model.Channel
	if err := database.DB.First(&ch, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeAPIError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to read channel")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"channel": toChannelResponse(ch)})
}

/* ── Update ──────────────────────────────────────────────────────────────── */

// Update applies a partial patch. api_key left empty keeps the existing one.
func (h *ChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var p channelPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	if err := validateChannelPayload(&p, false); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	var ch model.Channel
	if err := database.DB.First(&ch, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeAPIError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to load channel")
		return
	}

	updates := map[string]interface{}{}
	if name := strings.TrimSpace(p.Name); name != "" {
		updates["name"] = name
	}
	if t := strings.ToLower(strings.TrimSpace(p.Type)); t != "" {
		updates["protocol"] = t
	}
	if b := strings.TrimSpace(p.BaseURL); b != "" {
		if err := validateChannelURL(b); err != nil {
			writeAPIError(w, http.StatusBadRequest, "base_url rejected: "+err.Error())
			return
		}
		updates["base_url"] = b
	}
	if m := strings.TrimSpace(p.Models); m != "" {
		updates["models"] = m
	}
	if p.Status != "" {
		updates["status"] = strings.ToLower(strings.TrimSpace(p.Status))
	}
	if p.Priority != nil {
		updates["priority"] = *p.Priority
	}
	if p.Weight != nil {
		updates["weight"] = *p.Weight
	}
	if raw := strings.TrimSpace(p.Group); raw != "" {
		updates["group"] = normalizeGroupList(raw)
	}
	if strings.TrimSpace(p.APIKey) != "" {
		sealed, err := h.sealer.Encrypt(strings.TrimSpace(p.APIKey))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to seal api_key")
			return
		}
		updates["api_key"] = sealed
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&ch).Updates(updates).Error; err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to update channel")
			return
		}
	}
	h.invalidate()
	database.DB.First(&ch, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"channel": toChannelResponse(ch)})
}

/* ── Delete ──────────────────────────────────────────────────────────────── */

func (h *ChannelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := database.DB.Delete(&model.Channel{}, id).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to delete channel")
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

/* ── Helpers ────────────────────────────────────────────────────────────── */

func parseID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid channel id")
		return 0, false
	}
	return uint(id), true
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// normalizeGroupList trims and re-joins a comma-separated group list;
// empty input falls back to the default group.
func normalizeGroupList(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, g := range parts {
		if g = strings.TrimSpace(g); g != "" {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return "default"
	}
	return strings.Join(out, ",")
}

func derefOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
