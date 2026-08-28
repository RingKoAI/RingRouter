// Package handler: billing.go implements per-request quota deduction.
// Cost model: quota points where $1 = 500,000 quota (the one-api
// convention). The list price comes from ModelMeta (USD per 1M tokens) and
// is multiplied by the caller's group ratio. Models without a configured
// meta are free — billing is strictly opt-in per model.
package handler

import (
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
)

// QuotaPerUSD converts USD cost into quota points.
const QuotaPerUSD = 500000

// metaCacheTTL bounds how stale model prices may be after an admin edit.
const metaCacheTTL = time.Minute

type priceEntry struct {
	input  float64 // USD / 1M tokens
	output float64
}

var (
	priceMu    sync.RWMutex
	priceCache map[string]priceEntry
	priceAt    time.Time
)

// modelPrices resolves per-model list prices with a short process cache.
func modelPrices(name string) priceEntry {
	priceMu.RLock()
	if priceCache != nil && time.Since(priceAt) < metaCacheTTL {
		e, ok := priceCache[name]
		priceMu.RUnlock()
		if !ok {
			return priceEntry{}
		}
		return e
	}
	priceMu.RUnlock()

	// Refresh the whole (small) catalogue.
	fresh := make(map[string]priceEntry)
	if database.DB != nil {
		var metas []model.ModelMeta
		database.DB.Where("status = ?", "active").Find(&metas)
		for _, m := range metas {
			fresh[m.Name] = priceEntry{input: m.InputPrice, output: m.OutputPrice}
		}
	}
	priceMu.Lock()
	priceCache, priceAt = fresh, time.Now()
	priceMu.Unlock()

	e, ok := fresh[name]
	if !ok {
		return priceEntry{}
	}
	return e
}

// InvalidatePriceCache drops cached prices after admin edits so the next
// request bills with fresh values.
func InvalidatePriceCache() {
	priceMu.Lock()
	priceCache = nil
	priceMu.Unlock()
}

// ComputeCostQuota turns token usage into quota points:
// (prompt/1M × in + completion/1M × out) × group ratio × 500k.
// Returns 0 for unpriced models.
func ComputeCostQuota(modelName string, promptTokens, completionTokens int, group string) int64 {
	p := modelPrices(modelName)
	if p.input == 0 && p.output == 0 {
		return 0
	}
	usd := float64(promptTokens)/1e6*p.input + float64(completionTokens)/1e6*p.output
	q := int64(usd * QuotaPerUSD * setting.GroupRatio(group))
	if q < 0 {
		return 0
	}
	return q
}

// QuotaAvailable reports whether the account and token still have quota.
// Unlimited (-1) never blocks; exhausted (<= 0) does.
func QuotaAvailable(u *model.User, tok *model.Token) bool {
	if u == nil {
		return false
	}
	if u.Quota != -1 && u.Quota <= 0 {
		return false
	}
	if tok != nil && tok.Quota != -1 && tok.Quota <= 0 {
		return false
	}
	return true
}

// DeductQuota atomically removes cost from the token and the user, and
// records the consumption on both used_quota counters. Deduction is
// best-effort after the fact: a race that drives quota below zero clamps it
// at zero rather than failing the already-delivered response.
func DeductQuota(u *model.User, tok *model.Token, cost int64, group string) {
	if cost <= 0 || database.DB == nil {
		return
	}

	if tok != nil {
		if tok.Quota == -1 {
			database.DB.Model(&model.Token{}).Where("id = ?", tok.ID).
				Update("used_quota", gorm.Expr("used_quota + ?", cost))
		} else {
			database.DB.Model(&model.Token{}).Where("id = ?", tok.ID).Updates(map[string]interface{}{
				"quota":      gorm.Expr("CASE WHEN quota >= ? THEN quota - ? ELSE 0 END", cost, cost),
				"used_quota": gorm.Expr("used_quota + ?", cost),
			})
		}
	}

	if u != nil {
		if u.Quota == -1 {
			database.DB.Model(&model.User{}).Where("id = ?", u.ID).
				Update("used_quota", gorm.Expr("used_quota + ?", cost))
		} else {
			database.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
				"quota":      gorm.Expr("CASE WHEN quota >= ? THEN quota - ? ELSE 0 END", cost, cost),
				"used_quota": gorm.Expr("used_quota + ?", cost),
			})
		}
	}
}
