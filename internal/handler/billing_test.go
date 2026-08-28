package handler

import (
	"testing"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

func setupBillingTest(t *testing.T) {
	t.Helper()
	if database.DB != nil {
		database.Close()
		database.DB = nil
	}
	if err := database.Connect(database.Config{Type: "sqlite", Path: "file::memory:?cache=shared"}); err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(func() {
		setting.Reset()
		database.Close()
		database.DB = nil
		InvalidatePriceCache()
	})
	if !setting.GroupExists("default") {
		if _, err := setting.CreateGroup("default", "", "", 1); err != nil {
			t.Fatalf("default group: %v", err)
		}
	}
}

func priceModel(t *testing.T, name string, in, out float64) {
	t.Helper()
	if err := database.DB.Create(&model.ModelMeta{
		Name: name, Vendor: "test", InputPrice: in, OutputPrice: out, Status: "active",
	}).Error; err != nil {
		t.Fatalf("seed meta %s: %v", name, err)
	}
	InvalidatePriceCache()
}

func TestComputeCostQuota(t *testing.T) {
	setupBillingTest(t)
	priceModel(t, "m1", 1.0, 2.0) // $1/$2 per 1M

	// 1M prompt + 1M completion at ratio 1 = $3 = 1.5M quota points.
	if got := ComputeCostQuota("m1", 1_000_000, 1_000_000, "default"); got != 1_500_000 {
		t.Errorf("cost = %d, want 1500000", got)
	}
	// Half scale: 500k + 500k = $1.5 = 750k points.
	if got := ComputeCostQuota("m1", 500_000, 500_000, "default"); got != 750_000 {
		t.Errorf("cost = %d, want 750000", got)
	}

	// Group ratio folds in: 0.5 discount halves the cost.
	if _, err := setting.CreateGroup("half", "", "", 0.5); err != nil {
		t.Fatalf("half group: %v", err)
	}
	if got := ComputeCostQuota("m1", 1_000_000, 1_000_000, "half"); got != 750_000 {
		t.Errorf("discounted cost = %d, want 750000", got)
	}

	// Unpriced models are free (billing is opt-in per model).
	if got := ComputeCostQuota("unknown-model", 1_000_000, 1_000_000, "default"); got != 0 {
		t.Errorf("unpriced cost = %d, want 0", got)
	}
	// Zero usage bills nothing.
	if got := ComputeCostQuota("m1", 0, 0, "default"); got != 0 {
		t.Errorf("zero usage cost = %d, want 0", got)
	}
}

func TestQuotaAvailable(t *testing.T) {
	cases := []struct {
		name  string
		user  int64
		token *model.Token
		want  bool
	}{
		{"unlimited both", -1, nil, true},
		{"positive balance", 100, nil, true},
		{"exhausted user", 0, nil, false},
		{"negative user", -5, nil, false},
		{"token unlimited", 100, &model.Token{Quota: -1}, true},
		{"token exhausted", 100, &model.Token{Quota: 0}, false},
		{"nil user", 0, nil, false},
	}
	for _, c := range cases {
		var u *model.User
		if c.name != "nil user" {
			u = &model.User{Quota: c.user}
		}
		if got := QuotaAvailable(u, c.token); got != c.want {
			t.Errorf("%s: QuotaAvailable = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDeductQuotaAtomicAndClamped(t *testing.T) {
	setupBillingTest(t)

	user := model.User{Username: "bill", Quota: 100, UsedQuota: 0}
	database.DB.Create(&user)
	tok := model.Token{UserID: user.ID, Key: "k", Name: "k", Quota: 30, UsedQuota: 0}
	database.DB.Create(&tok)

	// Regular deduction lands on both levels.
	DeductQuota(&user, &tok, 20, "default")
	database.DB.First(&user, user.ID)
	database.DB.First(&tok, tok.ID)
	if user.Quota != 80 || user.UsedQuota != 20 {
		t.Errorf("user after deduct = %d/%d, want 80/20", user.Quota, user.UsedQuota)
	}
	if tok.Quota != 10 || tok.UsedQuota != 20 {
		t.Errorf("token after deduct = %d/%d, want 10/20", tok.Quota, tok.UsedQuota)
	}

	// Overdraft clamps at zero instead of going negative; used_quota still counts.
	DeductQuota(&user, &tok, 999, "default")
	database.DB.First(&user, user.ID)
	database.DB.First(&tok, tok.ID)
	if user.Quota != 0 {
		t.Errorf("user quota = %d, want clamped 0", user.Quota)
	}
	if tok.Quota != 0 || tok.UsedQuota != 1019 {
		t.Errorf("token = %d/%d, want 0/1019", tok.Quota, tok.UsedQuota)
	}

	// Unlimited accounts never deduct but still count usage.
	inf := model.User{Username: "inf", Quota: -1}
	database.DB.Create(&inf)
	infTok := model.Token{UserID: inf.ID, Key: "inf", Name: "inf", Quota: -1}
	database.DB.Create(&infTok)
	DeductQuota(&inf, &infTok, 50, "default")
	database.DB.First(&inf, inf.ID)
	database.DB.First(&infTok, infTok.ID)
	if inf.Quota != -1 || inf.UsedQuota != 50 {
		t.Errorf("unlimited user = %d/%d, want -1/50", inf.Quota, inf.UsedQuota)
	}
	if infTok.Quota != -1 || infTok.UsedQuota != 50 {
		t.Errorf("unlimited token = %d/%d, want -1/50", infTok.Quota, infTok.UsedQuota)
	}

	// Zero cost is a no-op.
	DeductQuota(&inf, &infTok, 0, "default")
	database.DB.First(&inf, inf.ID)
	if inf.UsedQuota != 50 {
		t.Errorf("used after zero-cost = %d, want 50", inf.UsedQuota)
	}
}
