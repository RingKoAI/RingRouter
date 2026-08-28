package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// setupGroupTest wires an in-memory SQLite database.
func setupGroupTest(t *testing.T) *GroupHandler {
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
	return NewGroupHandler()
}

func groupRequest(t *testing.T, h *GroupHandler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Handlers read r.PathValue("id"); without a ServeMux it must be set
	// explicitly on the request.
	if seg := path[strings.LastIndexByte(path, '/')+1:]; seg != "groups" && seg != "" {
		req.SetPathValue("id", seg)
	}
	rec := httptest.NewRecorder()
	switch method {
	case http.MethodGet:
		if strings.HasSuffix(path, "/groups") {
			h.List(rec, req)
		} else {
			h.Read(rec, req)
		}
	case http.MethodPost:
		h.Create(rec, req)
	case http.MethodPut:
		h.Update(rec, req)
	case http.MethodDelete:
		h.Delete(rec, req)
	}
	return rec
}

func TestGroupCreateGeneratesAndAcceptsUUID(t *testing.T) {
	h := setupGroupTest(t)

	// Auto-generated UUID.
	rec := groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"alpha"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create alpha = %d: %s", rec.Code, rec.Body.String())
	}
	auto := rec.Body.String()
	if !strings.Contains(auto, `"uuid":"`) || strings.Contains(auto, `"uuid":""`) {
		t.Errorf("auto uuid missing: %s", auto)
	}

	// Caller-supplied UUID is honoured verbatim.
	rec = groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"beta","uuid":"ABCDEF0123456789abcdef0123456789"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create beta = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"uuid":"abcdef0123456789abcdef0123456789"`) {
		t.Errorf("custom uuid not stored lowercased: %s", rec.Body.String())
	}

	// Malformed UUID is rejected without side effects.
	rec = groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"gamma","uuid":"not-hex"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad uuid = %d, want 400", rec.Code)
	}
	var n int64
	database.DB.Model(&model.Group{}).Where("name = ?", "gamma").Count(&n)
	if n != 0 {
		t.Error("group gamma must not exist after rejected create")
	}

	// Duplicate name is rejected.
	rec = groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"alpha"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("dup name = %d, want 409", rec.Code)
	}
}

func TestGroupUpdateRenameCascadesAndMetadataClears(t *testing.T) {
	h := setupGroupTest(t)
	groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"vip"}`)
	groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"std"}`)

	database.DB.Create(&model.User{Username: "u1", Group: "vip"})
	database.DB.Create(&model.Token{Key: "k1", Group: "vip"})
	database.DB.Create(&model.Channel{Name: "c1", Protocol: "openai", Group: "std,vip"})
	database.DB.Create(&model.Channel{Name: "c2", Protocol: "openai", Group: "vip-vip"}) // must stay untouched

	var g model.Group
	if err := database.DB.Where("name = ?", "vip").First(&g).Error; err != nil {
		t.Fatalf("load vip: %v", err)
	}

	rec := groupRequest(t, h, http.MethodPut, "/api/admin/groups/"+itoa(g.ID),
		`{"name":"premium","metadata":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename = %d: %s", rec.Code, rec.Body.String())
	}

	var u model.User
	database.DB.Where("username = ?", "u1").First(&u)
	if u.Group != "premium" {
		t.Errorf("user group = %q, want premium", u.Group)
	}
	var tok model.Token
	database.DB.Where("key = ?", "k1").First(&tok)
	if tok.Group != "premium" {
		t.Errorf("token group = %q, want premium", tok.Group)
	}
	var c1 model.Channel
	database.DB.Where("name = ?", "c1").First(&c1)
	if c1.Group != "std,premium" {
		t.Errorf("multi-group channel = %q, want std,premium", c1.Group)
	}
	var c2 model.Channel
	database.DB.Where("name = ?", "c2").First(&c2)
	if c2.Group != "vip-vip" {
		t.Errorf("prefix-named channel must stay %q, got %q", "vip-vip", c2.Group)
	}
	if g2 := loadGroupByName(t, "premium"); g2.Metadata != "" {
		t.Errorf("metadata = %q, want cleared", g2.Metadata)
	}
}

func TestGroupDeleteBlockedWhileInUse(t *testing.T) {
	h := setupGroupTest(t)
	groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"vip"}`)

	database.DB.Create(&model.Token{Key: "k1", Group: "vip"})
	g := loadGroupByName(t, "vip")

	rec := groupRequest(t, h, http.MethodDelete, "/api/admin/groups/"+itoa(g.ID), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete with token = %d, want 409", rec.Code)
	}
	database.DB.Where("key = ?", "k1").Delete(&model.Token{})

	// Channel listing the group inside a multi-group value still blocks.
	database.DB.Create(&model.Channel{Name: "c1", Protocol: "openai", Group: "default,vip"})
	rec = groupRequest(t, h, http.MethodDelete, "/api/admin/groups/"+itoa(g.ID), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete with multi-group channel = %d, want 409", rec.Code)
	}

	// Similar-but-different names must not block.
	database.DB.Delete(&model.Channel{}, "name = ?", "c1")
	database.DB.Create(&model.Channel{Name: "c2", Protocol: "openai", Group: "vip-plus"})
	rec = groupRequest(t, h, http.MethodDelete, "/api/admin/groups/"+itoa(g.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete with prefix-named channel = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestGroupRatioLifecycle(t *testing.T) {
	h := setupGroupTest(t)

	// Created with an explicit discount ratio.
	rec := groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"vip","ratio":0.8}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create vip = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ratio":0.8`) {
		t.Errorf("ratio missing in response: %s", rec.Body.String())
	}
	if got := setting.GroupRatio("vip"); got != 0.8 {
		t.Errorf("GroupRatio(vip) = %v, want 0.8", got)
	}

	// Default ratio is list price.
	groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"std"}`)
	if got := setting.GroupRatio("std"); got != 1 {
		t.Errorf("GroupRatio(std) = %v, want 1", got)
	}

	// Unknown group fails open to list price, never zero.
	if got := setting.GroupRatio("nope"); got != 1 {
		t.Errorf("GroupRatio(unknown) = %v, want 1", got)
	}

	// Out-of-range ratios are rejected on create and update.
	rec = groupRequest(t, h, http.MethodPost, "/api/admin/groups", `{"name":"bad","ratio":0}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("zero ratio create = %d, want 400", rec.Code)
	}
	g := loadGroupByName(t, "vip")
	rec = groupRequest(t, h, http.MethodPut, "/api/admin/groups/"+itoa(g.ID), `{"ratio":500}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("huge ratio update = %d, want 400", rec.Code)
	}

	// Update applies a new ratio.
	rec = groupRequest(t, h, http.MethodPut, "/api/admin/groups/"+itoa(g.ID), `{"ratio":1.5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ratio update = %d: %s", rec.Code, rec.Body.String())
	}
	if got := setting.GroupRatio("vip"); got != 1.5 {
		t.Errorf("GroupRatio(vip) after update = %v, want 1.5", got)
	}
}

func loadGroupByName(t *testing.T, name string) model.Group {
	t.Helper()
	var g model.Group
	if err := database.DB.Where("name = ?", name).First(&g).Error; err != nil {
		t.Fatalf("load group %s: %v", name, err)
	}
	return g
}

func itoa(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}
