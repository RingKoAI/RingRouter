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

// Plan limits.
const (
	maxPlanDurationDays = 3650 // ~10 years; 0 = perpetual
)

// PlanHandler serves the admin subscription-plan API.
type PlanHandler struct{}

// NewPlanHandler creates a PlanHandler.
func NewPlanHandler() *PlanHandler {
	return &PlanHandler{}
}

type planPayload struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	PriceCents   *int64  `json:"price_cents"`
	Quota        *int64  `json:"quota"` // -1 = unlimited
	Group        *string `json:"group"`
	DurationDays *int    `json:"duration_days"` // 0 = perpetual
	Status       *string `json:"status"`        // active, disabled
}

/* ── List ────────────────────────────────────────────────────────────────── */

// List returns all plans (plans are few; no pagination needed).
func (h *PlanHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var plans []model.Plan
	if err := database.DB.Order("price_cents ASC, id ASC").Find(&plans).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list plans")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"plans": plans})
}

/* ── Create ──────────────────────────────────────────────────────────────── */

// Create persists a new plan.
func (h *PlanHandler) Create(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var p planPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	if p.Name == nil || len(strings.TrimSpace(*p.Name)) > 64 {
		writeAPIError(w, http.StatusBadRequest, "name is required (max 64 characters)")
		return
	}
	name := strings.TrimSpace(*p.Name)
	quota := int64(0)
	if p.Quota != nil {
		if *p.Quota < -1 {
			writeAPIError(w, http.StatusBadRequest, "quota must be >= 0, or -1 for unlimited")
			return
		}
		quota = *p.Quota
	}
	group := "default"
	if p.Group != nil && strings.TrimSpace(*p.Group) != "" {
		group = strings.TrimSpace(*p.Group)
	}
	if !setting.GroupExists(group) {
		writeAPIError(w, http.StatusBadRequest, "group does not exist; create it via /api/admin/groups first")
		return
	}
	duration := 30
	if p.DurationDays != nil {
		if *p.DurationDays < 0 || *p.DurationDays > maxPlanDurationDays {
			writeAPIError(w, http.StatusBadRequest, "duration_days must be 0-3650 (0 = perpetual)")
			return
		}
		duration = *p.DurationDays
	}
	price := int64(0)
	if p.PriceCents != nil && *p.PriceCents >= 0 {
		price = *p.PriceCents
	}
	desc := ""
	if p.Description != nil {
		if len(*p.Description) > 256 {
			writeAPIError(w, http.StatusBadRequest, "description must be at most 256 characters")
			return
		}
		desc = *p.Description
	}
	status := "active"
	if p.Status != nil && *p.Status == "disabled" {
		status = "disabled"
	}

	// Pre-check the name (the DB unique index remains the final guard; the
	// check gives a portable 409 on every backend).
	var n int64
	database.DB.Model(&model.Plan{}).Where("name = ?", name).Count(&n)
	if n > 0 {
		writeAPIError(w, http.StatusConflict, "a plan with this name already exists")
		return
	}

	plan := model.Plan{
		Name: name, Description: desc, PriceCents: price,
		Quota: quota, Group: group, DurationDays: duration, Status: status,
	}
	if err := database.DB.Create(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeAPIError(w, http.StatusConflict, "a plan with this name already exists")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to create plan")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"plan": plan})
}

/* ── Update ──────────────────────────────────────────────────────────────── */

// Update applies a partial patch. Existing subscriptions keep their snapshot,
// so edits only affect future grants.
func (h *PlanHandler) Update(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid plan id")
		return
	}
	var plan model.Plan
	if err := database.DB.First(&plan, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "plan not found")
		return
	}

	var p planPayload
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
		var dup int64
		database.DB.Model(&model.Plan{}).Where("name = ? AND id <> ?", name, plan.ID).Count(&dup)
		if dup > 0 {
			writeAPIError(w, http.StatusConflict, "a plan with this name already exists")
			return
		}
		updates["name"] = name
	}
	if p.Description != nil {
		if len(*p.Description) > 256 {
			writeAPIError(w, http.StatusBadRequest, "description must be at most 256 characters")
			return
		}
		updates["description"] = *p.Description
	}
	if p.PriceCents != nil {
		if *p.PriceCents < 0 {
			writeAPIError(w, http.StatusBadRequest, "price_cents must be >= 0")
			return
		}
		updates["price_cents"] = *p.PriceCents
	}
	if p.Quota != nil {
		if *p.Quota < -1 {
			writeAPIError(w, http.StatusBadRequest, "quota must be >= 0, or -1 for unlimited")
			return
		}
		updates["quota"] = *p.Quota
	}
	if p.Group != nil && strings.TrimSpace(*p.Group) != "" {
		group := strings.TrimSpace(*p.Group)
		if !setting.GroupExists(group) {
			writeAPIError(w, http.StatusBadRequest, "group does not exist; create it via /api/admin/groups first")
			return
		}
		updates["group"] = group
	}
	if p.DurationDays != nil {
		if *p.DurationDays < 0 || *p.DurationDays > maxPlanDurationDays {
			writeAPIError(w, http.StatusBadRequest, "duration_days must be 0-3650 (0 = perpetual)")
			return
		}
		updates["duration_days"] = *p.DurationDays
	}
	if p.Status != nil {
		if *p.Status != "active" && *p.Status != "disabled" {
			writeAPIError(w, http.StatusBadRequest, "status must be active or disabled")
			return
		}
		updates["status"] = *p.Status
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&plan).Updates(updates).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				writeAPIError(w, http.StatusConflict, "a plan with this name already exists")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "failed to update plan")
			return
		}
	}
	database.DB.First(&plan, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"plan": plan})
}

/* ── Delete ──────────────────────────────────────────────────────────────── */

// Delete removes a plan. Active subscriptions survive on their snapshot, so
// grants already made keep working.
func (h *PlanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid plan id")
		return
	}
	if err := database.DB.Delete(&model.Plan{}, id).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}
