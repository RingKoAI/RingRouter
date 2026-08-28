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

// GroupHandler serves the admin group management API. Each group is a
// routing partition: tokens pick a group at creation, and only channels
// sharing the same group are eligible.
type GroupHandler struct{}

// NewGroupHandler creates a GroupHandler.
func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

/* ── Requests ─────────────────────────────────────────────────────────────── */

type groupPayload struct {
	Name     *string  `json:"name"`
	UUID     *string  `json:"uuid"`     // optional, server-generated when empty
	Metadata *string  `json:"metadata"` // nil = leave unchanged; "" = clear
	Ratio    *float64 `json:"ratio"`    // billing multiplier, default 1.0
}

/* ── List ────────────────────────────────────────────────────────────────── */

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var groups []model.Group
	if err := database.DB.Order("is_default DESC, name ASC").Find(&groups).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"groups": groups})
}

/* ── Read ────────────────────────────────────────────────────────────────── */

func (h *GroupHandler) Read(w http.ResponseWriter, r *http.Request) {
	g, ok := lookupGroup(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"group": g})
}

/* ── Create ──────────────────────────────────────────────────────────────── */

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var p groupPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	if p.Name == nil {
		writeAPIError(w, http.StatusBadRequest, "name is required")
		return
	}
	name := strings.TrimSpace(*p.Name)
	if name == "" || len(name) > 64 {
		writeAPIError(w, http.StatusBadRequest, "name must be 1-64 characters")
		return
	}
	metadata := ""
	if p.Metadata != nil {
		if len(*p.Metadata) > 1024 {
			writeAPIError(w, http.StatusBadRequest, "metadata must be at most 1024 characters")
			return
		}
		metadata = *p.Metadata
	}
	// A caller-supplied UUID must be well-formed up front so a bad value
	// cannot leave a half-created group behind.
	uuid := ""
	if p.UUID != nil {
		uuid = strings.ToLower(strings.TrimSpace(*p.UUID))
		if uuid != "" && !setting.ValidGroupUUID(uuid) {
			writeAPIError(w, http.StatusBadRequest, "uuid must be 32 hex characters")
			return
		}
	}
	// Billing multiplier: default 1.0 (list price); reject zero/negative
	// values that would silently make usage free.
	ratio := 1.0
	if p.Ratio != nil {
		if !setting.ValidGroupRatio(*p.Ratio) {
			writeAPIError(w, http.StatusBadRequest, "ratio must be between 0.01 and 100")
			return
		}
		ratio = *p.Ratio
	}
	if setting.GroupExists(name) {
		writeAPIError(w, http.StatusConflict, "a group with this name already exists")
		return
	}

	g, err := setting.CreateGroup(name, uuid, metadata, ratio)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeAPIError(w, http.StatusConflict, "a group with this name or uuid already exists")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to create group")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"group": g})
}

/* ── Update ──────────────────────────────────────────────────────────────── */

func (h *GroupHandler) Update(w http.ResponseWriter, r *http.Request) {
	g, ok := lookupGroup(w, r)
	if !ok {
		return
	}
	// Capture before any DB call: GORM's Updates writes the new values
	// back into the struct, which would otherwise mask the rename we
	// need to cascade.
	oldName := g.Name

	var p groupPayload
	if !decodeJSON(w, r, &p) {
		return
	}

	updates := map[string]interface{}{}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" || len(n) > 64 {
			writeAPIError(w, http.StatusBadRequest, "name must be 1-64 characters")
			return
		}
		updates["name"] = n
	}
	if p.Metadata != nil {
		if len(*p.Metadata) > 1024 {
			writeAPIError(w, http.StatusBadRequest, "metadata must be at most 1024 characters")
			return
		}
		updates["metadata"] = *p.Metadata // explicit empty string clears it
	}
	if p.UUID != nil {
		u := strings.ToLower(strings.TrimSpace(*p.UUID))
		if u == "" || !setting.ValidGroupUUID(u) {
			writeAPIError(w, http.StatusBadRequest, "uuid must be 32 hex characters")
			return
		}
		updates["uuid"] = u
	}
	if p.Ratio != nil {
		if !setting.ValidGroupRatio(*p.Ratio) {
			writeAPIError(w, http.StatusBadRequest, "ratio must be between 0.01 and 100")
			return
		}
		updates["ratio"] = *p.Ratio
	}

	if len(updates) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"group": g})
		return
	}

	// The default group cannot be renamed; doing so would orphan every
	// user/channel that already references "default".
	if _, hasName := updates["name"]; hasName && g.IsDefault {
		writeAPIError(w, http.StatusConflict, "cannot rename the default group")
		return
	}

	if err := database.DB.Model(g).Updates(updates).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeAPIError(w, http.StatusConflict, "a group with this name or uuid already exists")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to update group")
		return
	}

	// Cascade the rename to existing users, tokens, and channels. The
	// column is a SQL reserved word, so it is quoted per dialect (MySQL
	// backticks vs ANSI double quotes) — PostgreSQL rejects backticks.
	if newName, ok := updates["name"].(string); ok && newName != oldName {
		col := database.QuoteIdentifier("group")
		for _, table := range []string{"users", "tokens"} {
			if err := database.DB.Exec(
				`UPDATE `+database.QuoteIdentifier(table)+` SET `+col+` = ? WHERE `+col+` = ?`,
				newName, oldName,
			).Error; err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to cascade group rename")
				return
			}
		}
		// Channels may carry comma-separated multi-group lists, so an
		// exact-match update would miss "default,old". Rewrite them in
		// memory instead of REPLACE(), which would also mangle names that
		// merely start with the old one (e.g. "old-vip").
		var channels []model.Channel
		if err := database.DB.Where(col+` LIKE ?`, "%"+oldName+"%").Find(&channels).Error; err == nil {
			for _, ch := range channels {
				groups := strings.Split(ch.Group, ",")
				changed := false
				for i, g := range groups {
					if strings.TrimSpace(g) == oldName {
						groups[i] = newName
						changed = true
					}
				}
				if changed {
					database.DB.Model(&model.Channel{}).Where("id = ?", ch.ID).
						Update("group", strings.Join(groups, ","))
				}
			}
		}
	}

	database.DB.First(g, g.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"group": g})
}

/* ── Delete ──────────────────────────────────────────────────────────────── */

func (h *GroupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	g, ok := lookupGroup(w, r)
	if !ok {
		return
	}
	if g.IsDefault {
		writeAPIError(w, http.StatusConflict, "cannot delete the default group")
		return
	}
	// Reject deletion when anything still references the group, to keep
	// routing deterministic. Admin can move users/channels first.
	col := database.QuoteIdentifier("group")
	var count int64
	database.DB.Raw(`SELECT COUNT(*) FROM users WHERE `+col+` = ?`, g.Name).Scan(&count)
	if count > 0 {
		writeAPIError(w, http.StatusConflict, "group still in use by users")
		return
	}
	database.DB.Raw(`SELECT COUNT(*) FROM tokens WHERE `+col+` = ?`, g.Name).Scan(&count)
	if count > 0 {
		writeAPIError(w, http.StatusConflict, "group still in use by tokens")
		return
	}
	// Channels may list the group inside a comma-separated multi-group
	// value, so match on substring and confirm precisely in memory.
	database.DB.Raw(`SELECT COUNT(*) FROM channels WHERE `+col+` LIKE ?`, "%"+g.Name+"%").Scan(&count)
	if count > 0 {
		var channels []model.Channel
		database.DB.Where(col + ` LIKE ?`, "%"+g.Name+"%").Find(&channels)
		for _, ch := range channels {
			for _, cg := range strings.Split(ch.Group, ",") {
				if strings.TrimSpace(cg) == g.Name {
					writeAPIError(w, http.StatusConflict, "group still in use by channels")
					return
				}
			}
		}
	}
	if err := database.DB.Delete(&model.Group{}, g.ID).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to delete group")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

/* ── Helpers ────────────────────────────────────────────────────────────── */

func lookupGroup(w http.ResponseWriter, r *http.Request) (*model.Group, bool) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return nil, false
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid group id")
		return nil, false
	}
	var g model.Group
	if err := database.DB.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeAPIError(w, http.StatusNotFound, "group not found")
			return nil, false
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to read group")
		return nil, false
	}
	return &g, true
}
