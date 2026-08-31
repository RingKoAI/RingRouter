package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func setupPlanTest(t *testing.T) (*PlanHandler, *SubscriptionHandler) {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	setting.ResetGroups() // fresh DB instance: drop the group lookup snapshot
	t.Cleanup(func() {
		database.Close()
		database.DB = nil
	})
	if !setting.GroupExists("default") {
		if _, err := setting.CreateGroup("default", "", "", 1); err != nil {
			t.Fatalf("default group: %v", err)
		}
	}
	if _, err := setting.CreateGroup("vip", "", "", 0.8); err != nil {
		t.Fatalf("vip group: %v", err)
	}
	return NewPlanHandler(), NewSubscriptionHandler()
}

func planReq(t *testing.T, ph *PlanHandler, sh *SubscriptionHandler, method, path, body string, actor *model.User) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i := range segs {
		if segs[i] == "plans" || segs[i] == "subscriptions" {
			if i+1 < len(segs) && segs[i+1] != "" {
				key := "id"
				if segs[i] == "plans" {
					key = "id"
				}
				req.SetPathValue(key, segs[i+1])
			}
		}
	}
	if actor != nil {
		req = req.WithContext(middleware.WithUser(req.Context(), actor))
	}
	rec := httptest.NewRecorder()
	switch {
	case strings.HasPrefix(path, "/plans"):
		switch method {
		case http.MethodGet:
			ph.List(rec, req)
		case http.MethodPost:
			ph.Create(rec, req)
		case http.MethodPut:
			ph.Update(rec, req)
		case http.MethodDelete:
			ph.Delete(rec, req)
		}
	case strings.HasPrefix(path, "/subscriptions"):
		if strings.HasSuffix(path, "/me") {
			sh.Mine(rec, req)
			return rec
		}
		switch method {
		case http.MethodGet:
			sh.List(rec, req)
		case http.MethodPost:
			sh.Grant(rec, req)
		case http.MethodDelete:
			sh.Cancel(rec, req)
		}
	}
	return rec
}

func TestPlanCRUDValidation(t *testing.T) {
	ph, _ := setupPlanTest(t)

	// Unknown group rejected.
	rec := planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"p1","group":"ghost"}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ghost group = %d, want 400", rec.Code)
	}
	// Bad quota rejected.
	rec = planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"p1","quota":-5}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("quota -5 = %d, want 400", rec.Code)
	}
	// Good create.
	rec = planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"pro","quota":100000,"group":"vip","duration_days":30,"price_cents":999}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pro = %d: %s", rec.Code, rec.Body.String())
	}
	var plan model.Plan
	database.DB.Where("name = ?", "pro").First(&plan)
	if plan.Group != "vip" || plan.Quota != 100000 || plan.DurationDays != 30 {
		t.Errorf("plan stored wrong: %+v", plan)
	}
	// Duplicate name rejected.
	rec = planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"pro"}`, nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("dup plan = %d, want 409", rec.Code)
	}
	// Update patch.
	rec = planReq(t, ph, nil, http.MethodPut, "/plans/"+strconv.Itoa(int(plan.ID)), `{"status":"disabled"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable plan = %d", rec.Code)
	}
	database.DB.First(&plan, plan.ID)
	if plan.Status != "disabled" {
		t.Error("plan not disabled")
	}
}

func TestSubscriptionGrantAppliesSnapshot(t *testing.T) {
	ph, sh := setupPlanTest(t)

	user := mkUser(t, "alice", "user", "active")
	rec := planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"pro","quota":50000,"group":"vip","duration_days":30}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create plan: %s", rec.Body.String())
	}
	var plan model.Plan
	database.DB.Where("name = ?", "pro").First(&plan)

	// Disabled plans cannot be granted.
	planReq(t, ph, nil, http.MethodPut, "/plans/"+strconv.Itoa(int(plan.ID)), `{"status":"disabled"}`, nil)
	rec = planReq(t, ph, sh, http.MethodPost, "/subscriptions",
		`{"user_id":`+strconv.Itoa(int(user.ID))+`,"plan_id":`+strconv.Itoa(int(plan.ID))+`}`, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("grant disabled plan = %d, want 409", rec.Code)
	}
	planReq(t, ph, nil, http.MethodPut, "/plans/"+strconv.Itoa(int(plan.ID)), `{"status":"active"}`, nil)

	// Grant applies group + additive quota immediately.
	rec = planReq(t, ph, sh, http.MethodPost, "/subscriptions",
		`{"user_id":`+strconv.Itoa(int(user.ID))+`,"plan_id":`+strconv.Itoa(int(plan.ID))+`}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grant = %d: %s", rec.Code, rec.Body.String())
	}
	var got model.User
	database.DB.First(&got, user.ID)
	if got.Group != "vip" {
		t.Errorf("user group = %q, want vip", got.Group)
	}
	if got.Quota != 50000 {
		t.Errorf("user quota = %d, want 50000", got.Quota)
	}

	// Snapshot survives later plan edits.
	planReq(t, ph, nil, http.MethodPut, "/plans/"+strconv.Itoa(int(plan.ID)), `{"name":"pro-max","quota":999999,"group":"default"}`, nil)
	var sub model.Subscription
	database.DB.Where("user_id = ?", user.ID).First(&sub)
	if sub.PlanName != "pro" || sub.Group != "vip" || sub.Quota != 50000 {
		t.Errorf("snapshot drifted after plan edit: %+v", sub)
	}
	if sub.ExpiresAt.Before(time.Now().AddDate(0, 0, 29)) {
		t.Errorf("expiry too early: %v", sub.ExpiresAt)
	}
}

func TestSubscriptionLazyExpiryAndCancel(t *testing.T) {
	ph, sh := setupPlanTest(t)
	user := mkUser(t, "bob", "user", "active")
	planReq(t, ph, nil, http.MethodPost, "/plans", `{"name":"day1","quota":100,"group":"vip","duration_days":1}`, nil)
	var plan model.Plan
	database.DB.Where("name = ?", "day1").First(&plan)
	rec := planReq(t, ph, sh, http.MethodPost, "/subscriptions",
		`{"user_id":`+strconv.Itoa(int(user.ID))+`,"plan_id":`+strconv.Itoa(int(plan.ID))+`}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grant: %s", rec.Body.String())
	}

	// Force the subscription into the past, then read: lazily expired.
	database.DB.Model(&model.Subscription{}).Where("user_id = ?", user.ID).
		Update("expires_at", time.Now().Add(-time.Minute))
	rec = planReq(t, ph, sh, http.MethodGet, "/subscriptions?user_id="+strconv.Itoa(int(user.ID)), "", nil)
	if !strings.Contains(rec.Body.String(), `"status":"expired"`) {
		t.Errorf("lazy expiry failed: %s", rec.Body.String())
	}

	// Cancelling an already-expired subscription is refused.
	rec = planReq(t, ph, sh, http.MethodPost, "/plans", `{"name":"keep","quota":10,"group":"vip","duration_days":30}`, nil)
	_ = rec
	database.DB.Where("name = ?", "keep").First(&plan)
	rec = planReq(t, ph, sh, http.MethodPost, "/subscriptions",
		`{"user_id":`+strconv.Itoa(int(user.ID))+`,"plan_id":`+strconv.Itoa(int(plan.ID))+`}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grant keep: %s", rec.Body.String())
	}
	var keep model.Subscription
	database.DB.Where("plan_id = ? AND status = ?", plan.ID, "active").First(&keep)
	rec = planReq(t, ph, sh, http.MethodDelete, "/subscriptions/"+strconv.Itoa(int(keep.ID)), "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel = %d: %s", rec.Code, rec.Body.String())
	}
	rec = planReq(t, ph, sh, http.MethodDelete, "/subscriptions/"+strconv.Itoa(int(keep.ID)), "", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("double cancel = %d, want 409", rec.Code)
	}

	// /me returns own subscriptions only.
	other := mkUser(t, "carol", "user", "active")
	rec = planReq(t, ph, sh, http.MethodGet, "/subscriptions/me", "", &other)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "bob") {
		t.Errorf("me leaked other users: %s", rec.Body.String())
	}
}
