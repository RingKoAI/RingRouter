package handler

import (
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// SubscriptionHandler serves the admin subscription API and the user-facing
// self-service endpoint.
type SubscriptionHandler struct{}

// NewSubscriptionHandler creates a SubscriptionHandler.
func NewSubscriptionHandler() *SubscriptionHandler {
	return &SubscriptionHandler{}
}

type grantRequest struct {
	UserID uint `json:"user_id"`
	PlanID uint `json:"plan_id"`
}

// expireDue flips overdue active subscriptions to "expired". It runs lazily on
// every read path — no background worker needed for correctness.
func expireDue() {
	if database.DB == nil {
		return
	}
	database.DB.Model(&model.Subscription{}).
		Where("status = ? AND expires_at <> ? AND expires_at < ?", "active", time.Time{}, time.Now()).
		Update("status", "expired")
}

/* ── Admin: grant ───────────────────────────────────────────────────────── */

// Grant subscribes a user to a plan. The plan's quota and routing group are
// snapshotted onto the subscription and applied immediately: the user's
// routing group switches to the plan's group, and the granted quota is added
// to their balance (-1 switches the account to unlimited).
func (h *SubscriptionHandler) Grant(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var req grantRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.UserID == 0 || req.PlanID == 0 {
		writeAPIError(w, http.StatusBadRequest, "user_id and plan_id are required")
		return
	}

	var user model.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}
	var plan model.Plan
	if err := database.DB.First(&plan, req.PlanID).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "plan not found")
		return
	}
	if plan.Status != "active" {
		writeAPIError(w, http.StatusConflict, "plan is disabled")
		return
	}

	sub := model.Subscription{
		UserID:   user.ID,
		PlanID:   plan.ID,
		PlanName: plan.Name,
		Group:    plan.Group,
		Quota:    plan.Quota,
	}
	if plan.DurationDays > 0 {
		sub.ExpiresAt = time.Now().AddDate(0, 0, plan.DurationDays)
	}
	if err := database.DB.Create(&sub).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to grant subscription")
		return
	}

	// Apply immediately: routing group switches, quota is granted (additive,
	// like a top-up; -1 flips the account to unlimited).
	userUpdates := map[string]interface{}{"group": plan.Group}
	if plan.Quota == -1 {
		userUpdates["quota"] = -1
	} else if plan.Quota > 0 && user.Quota != -1 {
		userUpdates["quota"] = user.Quota + plan.Quota
	}
	if err := database.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(userUpdates).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to apply subscription to user")
		return
	}
	// Tokens follow the user's new group so routing takes effect at once.
	database.DB.Model(&model.Token{}).Where("user_id = ?", user.ID).Update("group", plan.Group)

	writeJSON(w, http.StatusCreated, map[string]interface{}{"subscription": sub})
}

/* ── Admin: list ────────────────────────────────────────────────────────── */

// List returns subscriptions, optionally filtered by ?user_id=, newest first.
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	expireDue()

	tx := database.DB.Model(&model.Subscription{})
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 64); err == nil && id > 0 {
			tx = tx.Where("user_id = ?", id)
		}
	}
	var subs []model.Subscription
	if err := tx.Order("id DESC").Limit(maxUserList).Find(&subs).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"subscriptions": subs})
}

/* ── Admin: cancel ──────────────────────────────────────────────────────── */

// Cancel marks a subscription cancelled. Granted quota stays spent; the
// routing group is intentionally left alone — reverting could clobber a
// manual group change made after the grant.
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid subscription id")
		return
	}
	res := database.DB.Model(&model.Subscription{}).
		Where("id = ? AND status = ?", id, "active").
		Update("status", "cancelled")
	if res.Error != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}
	if res.RowsAffected == 0 {
		writeAPIError(w, http.StatusConflict, "subscription not active")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

/* ── Self-service ───────────────────────────────────────────────────────── */

// Mine returns the caller's subscription history (session-authenticated).
func (h *SubscriptionHandler) Mine(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	expireDue()

	var subs []model.Subscription
	if err := database.DB.Where("user_id = ?", u.ID).Order("id DESC").Limit(100).
		Find(&subs).Error; err != nil && err != gorm.ErrRecordNotFound {
		writeAPIError(w, http.StatusInternalServerError, "failed to load subscriptions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"subscriptions": subs})
}
