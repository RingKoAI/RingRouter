package handler

import (
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// maxUserList is a hard cap for the user listing endpoint.
const (
	maxUserList     = 500
	maxUserPageSize = 100
	defaultUserPage = 20
)

// AdminHandler serves admin-only user management endpoints.
type AdminHandler struct{}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type updateGroupRequest struct {
	Group string `json:"group"`
}

type updateQuotaRequest struct {
	Quota *int64 `json:"quota"` // -1 = unlimited
}

type adminResetPasswordRequest struct {
	Password string `json:"password"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`  // admin | user (default user)
	Group    string `json:"group"` // optional; defaults to "default"
	Quota    *int64 `json:"quota"` // optional; default 0
}

// Create provisions an account directly from the console. Validation mirrors
// self-registration (username shape, email shape, bcrypt password bounds,
// case-insensitive uniqueness) with admin-only extras: role and quota.
//
// Authorization is enforced in three layers: the router mounts this handler
// behind session auth + RequireAdmin; the handler itself re-verifies the
// actor's role (defense in depth against future route mis-wiring); and every
// input field is whitelist-validated before touching the database.
func (h *AdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	if actor := middleware.GetUser(r.Context()); actor == nil || actor.Role != "admin" ||
		(actor.Status != "" && actor.Status != "active") {
		writeAPIError(w, http.StatusForbidden, "admin privileges required")
		return
	}

	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = normalizeEmail(req.Email)
	req.Group = strings.TrimSpace(req.Group)

	if !usernameRe.MatchString(req.Username) {
		writeAPIError(w, http.StatusBadRequest, "username must be 3-32 chars of letters, digits, - or _")
		return
	}
	if req.Email == "" {
		writeAPIError(w, http.StatusBadRequest, "a valid email address is required")
		return
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		writeAPIError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}
	role := "user"
	if req.Role != "" {
		if req.Role != "admin" && req.Role != "user" {
			writeAPIError(w, http.StatusBadRequest, "role must be admin or user")
			return
		}
		role = req.Role
	}
	if req.Group == "" {
		req.Group = "default"
	}
	if !setting.GroupExists(req.Group) {
		writeAPIError(w, http.StatusBadRequest, "group does not exist")
		return
	}
	quota := int64(0)
	if req.Quota != nil {
		if *req.Quota < -1 {
			writeAPIError(w, http.StatusBadRequest, "quota must be >= 0, or -1 for unlimited")
			return
		}
		quota = *req.Quota
	}

	// Case-insensitive uniqueness, same as self-registration.
	if err := database.DB.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)",
		req.Username, req.Email).First(&model.User{}).Error; err == nil {
		writeAPIError(w, http.StatusConflict, "username or email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	user := model.User{
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.Username,
		Password:    string(hash),
		Role:        role,
		Group:       req.Group,
		Quota:       quota,
		Status:      "active",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"user": user})
}

// parseUserPage reads ?q=&page=&page_size= with sane bounds.
func parseUserPage(r *http.Request) (q string, page, pageSize int) {
	q = strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = defaultUserPage
	}
	if pageSize > maxUserPageSize {
		pageSize = maxUserPageSize
	}
	return q, page, pageSize
}

// ListUsers returns users with optional keyword search and pagination.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	q, page, pageSize := parseUserPage(r)

	tx := database.DB.Model(&model.User{})
	if q != "" {
		like := likePattern(q)
		tx = tx.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?", like, like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to count users")
		return
	}

	var users []model.User
	if err := tx.Order("id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&users).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"pagination": map[string]int64{
			"page":      int64(page),
			"page_size": int64(pageSize),
			"total":     total,
		},
	})
}

// UpdateRole promotes or demotes a user. The last remaining admin cannot be
// demoted, so an instance can never end up without an administrator.
func (h *AdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateRoleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		writeAPIError(w, http.StatusBadRequest, "role must be admin or user")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}

	if target.Role == req.Role {
		writeJSON(w, http.StatusOK, map[string]interface{}{"user": target})
		return
	}

	// Demotion guard: at least one other admin must remain.
	if target.Role == "admin" {
		var others int64
		database.DB.Model(&model.User{}).
			Where("role = ? AND id <> ?", "admin", target.ID).
			Where("status = ?", "active").
			Count(&others)
		if others == 0 {
			writeAPIError(w, http.StatusConflict, "cannot demote the last active admin")
			return
		}
	}

	if err := database.DB.Model(&target).Update("role", req.Role).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update role")
		return
	}
	database.DB.First(&target, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": target})
}

// UpdateGroup moves a user into a different routing group. Tokens created
// by this user inherit the new group, so changes take effect on the next
// request without an explicit key rotation.
func (h *AdminHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateGroupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	group := strings.TrimSpace(req.Group)
	if group == "" {
		group = "default"
	}
	if len(group) > 64 {
		writeAPIError(w, http.StatusBadRequest, "group must be 1-64 characters")
		return
	}
	if !setting.GroupExists(group) {
		writeAPIError(w, http.StatusBadRequest, "group does not exist; create it via /api/admin/groups first")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}

	if err := database.DB.Model(&target).Update("group", group).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update group")
		return
	}
	// Propagate to existing tokens so they continue to route.
	if err := database.DB.Model(&model.Token{}).Where("user_id = ?", target.ID).Update("group", group).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to propagate group to tokens")
		return
	}
	database.DB.First(&target, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": target})
}

// UpdateStatus enables or disables an account. Disabling takes effect on the
// next request (existing sessions are rejected once the user is disabled).
func (h *AdminHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req updateStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		writeAPIError(w, http.StatusBadRequest, "status must be active or disabled")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}

	// Self-lockout guard.
	actor := middleware.GetUser(r.Context())
	if actor != nil && actor.ID == target.ID && req.Status == "disabled" {
		writeAPIError(w, http.StatusConflict, "cannot disable your own account")
		return
	}

	// Lockout guard: the last active admin cannot be disabled.
	if target.Role == "admin" && req.Status == "disabled" {
		var others int64
		database.DB.Model(&model.User{}).
			Where("role = ? AND id <> ?", "admin", target.ID).
			Where("status = ?", "active").
			Count(&others)
		if others == 0 {
			writeAPIError(w, http.StatusConflict, "cannot disable the last active admin")
			return
		}
	}

	if err := database.DB.Model(&target).Update("status", req.Status).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	// Disabling an account invalidates its sessions immediately.
	if req.Status == "disabled" {
		database.DB.Where("user_id = ?", target.ID).Delete(&model.Session{})
	}

	database.DB.First(&target, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": target})
}

// UpdateQuota sets a user's remaining quota. -1 means unlimited.
func (h *AdminHandler) UpdateQuota(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req updateQuotaRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Quota == nil || *req.Quota < -1 {
		writeAPIError(w, http.StatusBadRequest, "quota must be >= 0, or -1 for unlimited")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}
	if err := database.DB.Model(&target).Update("quota", *req.Quota).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update quota")
		return
	}
	database.DB.First(&target, id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": target})
}

// ResetPassword sets a new password on behalf of the user and revokes every
// existing session, forcing a fresh sign-in with the new credential.
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req adminResetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Password) < minPasswordLen || len(req.Password) > maxPasswordLen {
		writeAPIError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	if err := database.DB.Model(&target).Update("password", string(hash)).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	database.DB.Where("user_id = ?", target.ID).Delete(&model.Session{})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

// Delete removes a user together with their sessions and API tokens. The
// actor cannot delete themselves and the last active admin is protected.
func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var target model.User
	if err := database.DB.First(&target, id).Error; err != nil {
		writeAPIError(w, http.StatusNotFound, "user not found")
		return
	}

	// Self-deletion guard.
	actor := middleware.GetUser(r.Context())
	if actor != nil && actor.ID == target.ID {
		writeAPIError(w, http.StatusConflict, "cannot delete your own account")
		return
	}

	// The bootstrap admin session (user id 0) may delete anyone except the
	// guard above; a DB admin cannot be removed while they are the last
	// active administrator.
	if target.Role == "admin" {
		var others int64
		database.DB.Model(&model.User{}).
			Where("role = ? AND id <> ?", "admin", target.ID).
			Where("status = ?", "active").
			Count(&others)
		if others == 0 {
			writeAPIError(w, http.StatusConflict, "cannot delete the last active admin")
			return
		}
	}

	if err := database.DB.Delete(&model.User{}, target.ID).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	// Cascade: revoke sessions and remove their API tokens.
	database.DB.Where("user_id = ?", target.ID).Delete(&model.Session{})
	database.DB.Where("user_id = ?", target.ID).Delete(&model.Token{})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}
