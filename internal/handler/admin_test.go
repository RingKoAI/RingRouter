package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func setupAdminTest(t *testing.T) *AdminHandler {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		database.DB = nil
	})
	return NewAdminHandler()
}

func adminReq(t *testing.T, h *AdminHandler, method, path, body string, actor *model.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Paths are /users, /users/{id}, or /users/{id}/{action}: the id is the
	// second-to-last segment when an action suffix is present.
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) >= 3 && segs[len(segs)-3] == "users" {
		req.SetPathValue("id", segs[len(segs)-2])
	} else if len(segs) >= 2 && segs[len(segs)-2] == "users" {
		req.SetPathValue("id", segs[len(segs)-1])
	}
	if actor != nil {
		ctx := middleware.WithUser(req.Context(), actor)
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	switch method {
	case http.MethodGet:
		h.ListUsers(rec, req)
	case http.MethodDelete:
		h.Delete(rec, req)
	case http.MethodPut:
		switch {
		case strings.HasSuffix(path, "/role"):
			h.UpdateRole(rec, req)
		case strings.HasSuffix(path, "/status"):
			h.UpdateStatus(rec, req)
		case strings.HasSuffix(path, "/group"):
			h.UpdateGroup(rec, req)
		case strings.HasSuffix(path, "/quota"):
			h.UpdateQuota(rec, req)
		case strings.HasSuffix(path, "/password"):
			h.ResetPassword(rec, req)
		}
	}
	return rec
}

func mkUser(t *testing.T, username, role, status string) model.User {
	t.Helper()
	u := model.User{Username: username, Role: role, Status: status, Group: "default"}
	if err := database.DB.Create(&u).Error; err != nil {
		t.Fatalf("create %s: %v", username, err)
	}
	return u
}

func TestUserListSearchAndPagination(t *testing.T) {
	h := setupAdminTest(t)
	for _, n := range []string{"alice", "bob", "charlie"} {
		mkUser(t, n, "user", "active")
	}

	rec := adminReq(t, h, http.MethodGet, "/api/admin/users?q=ali", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "alice") {
		t.Errorf("search q=ali = %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "bob") {
		t.Error("search must not return bob")
	}

	rec = adminReq(t, h, http.MethodGet, "/api/admin/users?page=2&page_size=2", "", nil)
	body := rec.Body.String()
	if !strings.Contains(body, `"total":3`) {
		t.Errorf("pagination total missing: %s", body)
	}
	// page 2 of size 2 over 3 users = 1 user (charlie); bob must be on page 1.
	if strings.Contains(body, "bob") {
		t.Errorf("page 2 must not contain bob: %s", body)
	}
}

func TestUserDeleteGuardsAndCascade(t *testing.T) {
	h := setupAdminTest(t)
	admin := mkUser(t, "root", "admin", "active")
	victim := mkUser(t, "alice", "user", "active")

	database.DB.Create(&model.Session{ID: "s1", UserID: victim.ID})
	database.DB.Create(&model.Token{Key: "tok1", UserID: victim.ID})

	// Self-deletion refused.
	rec := adminReq(t, h, http.MethodDelete, "/api/admin/users/"+strconv.Itoa(int(admin.ID)), "", &admin)
	if rec.Code != http.StatusConflict {
		t.Errorf("self delete = %d, want 409", rec.Code)
	}

	// Deleting another user cascades sessions and tokens.
	rec = adminReq(t, h, http.MethodDelete, "/api/admin/users/"+strconv.Itoa(int(victim.ID)), "", &admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete = %d: %s", rec.Code, rec.Body.String())
	}
	var n int64
	database.DB.Model(&model.Session{}).Where("user_id = ?", victim.ID).Count(&n)
	if n != 0 {
		t.Error("sessions must be cascaded")
	}
	database.DB.Model(&model.Token{}).Where("user_id = ?", victim.ID).Count(&n)
	if n != 0 {
		t.Error("tokens must be cascaded")
	}

	// The last active admin cannot be deleted.
	rec = adminReq(t, h, http.MethodDelete, "/api/admin/users/"+strconv.Itoa(int(admin.ID)), "", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("last admin delete = %d, want 409", rec.Code)
	}
}

func TestUserQuotaAndPassword(t *testing.T) {
	h := setupAdminTest(t)
	u := mkUser(t, "alice", "user", "active")

	// Invalid quota values are rejected.
	rec := adminReq(t, h, http.MethodPut, "/api/admin/users/"+strconv.Itoa(int(u.ID))+"/quota", `{"quota":-2}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("quota -2 = %d, want 400", rec.Code)
	}
	rec = adminReq(t, h, http.MethodPut, "/api/admin/users/"+strconv.Itoa(int(u.ID))+"/quota", `{"quota":-1}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("quota -1 = %d: %s", rec.Code, rec.Body.String())
	}
	var got model.User
	database.DB.First(&got, u.ID)
	if got.Quota != -1 {
		t.Errorf("quota = %d, want -1", got.Quota)
	}

	// Password reset stores a bcrypt hash and revokes sessions.
	database.DB.Create(&model.Session{ID: "s2", UserID: u.ID})
	rec = adminReq(t, h, http.MethodPut, "/api/admin/users/"+strconv.Itoa(int(u.ID))+"/password", `{"password":"new-password-1"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset pw = %d: %s", rec.Code, rec.Body.String())
	}
	database.DB.First(&got, u.ID)
	if bcrypt.CompareHashAndPassword([]byte(got.Password), []byte("new-password-1")) != nil {
		t.Error("new password hash mismatch")
	}
	var n int64
	database.DB.Model(&model.Session{}).Where("user_id = ?", u.ID).Count(&n)
	if n != 0 {
		t.Error("sessions must be revoked after password reset")
	}

	// Short password rejected.
	rec = adminReq(t, h, http.MethodPut, "/api/admin/users/"+strconv.Itoa(int(u.ID))+"/password", `{"password":"short"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("short pw = %d, want 400", rec.Code)
	}
}

func TestUserUpdateGroupRequiresExistingGroup(t *testing.T) {
	h := setupAdminTest(t)
	settingEnsureDefault(t)
	u := mkUser(t, "alice", "user", "active")

	rec := adminReq(t, h, http.MethodPut, "/api/admin/users/"+strconv.Itoa(int(u.ID))+"/group", `{"group":"ghost"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unknown group = %d, want 400", rec.Code)
	}
}

func settingEnsureDefault(t *testing.T) {
	t.Helper()
	if !setting.GroupExists("default") {
		if _, err := setting.CreateGroup("default", "", "", 1); err != nil {
			t.Fatalf("ensure default group: %v", err)
		}
	}
}
