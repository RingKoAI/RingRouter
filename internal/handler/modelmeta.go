package handler

import (
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// modelMetaPriceCap rejects absurd fat-fingered prices ($100k / 1M tokens).
const modelMetaPriceCap = 100000

// ModelMetaHandler serves the admin model-catalogue API. Metas are keyed by
// the exact model name exposed by channels; upsert semantics keep the admin
// flow simple (no separate create/update dance).
type ModelMetaHandler struct{}

// NewModelMetaHandler creates a ModelMetaHandler.
func NewModelMetaHandler() *ModelMetaHandler {
	return &ModelMetaHandler{}
}

type modelMetaPayload struct {
	Name          *string  `json:"name"`
	Vendor        *string  `json:"vendor"`
	Description   *string  `json:"description"`
	InputPrice    *float64 `json:"input_price"`
	OutputPrice   *float64 `json:"output_price"`
	CachePrice    *float64 `json:"cache_price"`
	ContextWindow *int64   `json:"context_window"`
	Status        *string  `json:"status"` // active, hidden
}

func validPrice(p float64) bool { return p >= 0 && p <= modelMetaPriceCap }

/* ── List ────────────────────────────────────────────────────────────────── */

// List returns all metas.
func (h *ModelMetaHandler) List(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var metas []model.ModelMeta
	if err := database.DB.Order("name ASC").Find(&metas).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list model metas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"metas": metas})
}

/* ── Upsert ──────────────────────────────────────────────────────────────── */

// Upsert creates or updates the meta for one model name.
func (h *ModelMetaHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var p modelMetaPayload
	if !decodeJSON(w, r, &p) {
		return
	}
	// The model name travels in the path; a body name is accepted only when
	// the path segment is absent (plain /api/admin/models without a segment).
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" && p.Name != nil {
		name = strings.TrimSpace(*p.Name)
	}
	if name == "" || len(name) > 128 {
		writeAPIError(w, http.StatusBadRequest, "name is required (max 128 characters)")
		return
	}

	updates := map[string]interface{}{}
	if p.Vendor != nil {
		v := strings.TrimSpace(*p.Vendor)
		if len(v) > 64 {
			writeAPIError(w, http.StatusBadRequest, "vendor must be at most 64 characters")
			return
		}
		updates["vendor"] = v
	}
	if p.Description != nil {
		d := strings.TrimSpace(*p.Description)
		if len(d) > 512 {
			writeAPIError(w, http.StatusBadRequest, "description must be at most 512 characters")
			return
		}
		updates["description"] = d
	}
	for key, val := range map[string]*float64{
		"input_price": p.InputPrice, "output_price": p.OutputPrice, "cache_price": p.CachePrice,
	} {
		if val != nil {
			if !validPrice(*val) {
				writeAPIError(w, http.StatusBadRequest, key+" must be 0-"+strconv.Itoa(modelMetaPriceCap))
				return
			}
			updates[key] = *val
		}
	}
	if p.ContextWindow != nil {
		if *p.ContextWindow < 0 || *p.ContextWindow > 100_000_000 {
			writeAPIError(w, http.StatusBadRequest, "context_window out of range")
			return
		}
		updates["context_window"] = *p.ContextWindow
	}
	if p.Status != nil {
		if *p.Status != "active" && *p.Status != "hidden" {
			writeAPIError(w, http.StatusBadRequest, "status must be active or hidden")
			return
		}
		updates["status"] = *p.Status
	}

	var meta model.ModelMeta
	err := database.DB.Where("name = ?", name).First(&meta).Error
	switch {
	case err == nil:
		if len(updates) > 0 {
			if err := database.DB.Model(&meta).Updates(updates).Error; err != nil {
				writeAPIError(w, http.StatusInternalServerError, "failed to update meta")
				return
			}
		}
	case err == gorm.ErrRecordNotFound:
		meta = model.ModelMeta{
			Name: name, Vendor: "openai",
			InputPrice: 0, OutputPrice: 0, CachePrice: 0, Status: "active",
		}
		if v, ok := updates["vendor"].(string); ok {
			meta.Vendor = v
		}
		for _, key := range []string{"description", "input_price", "output_price", "cache_price", "context_window"} {
			if v, ok := updates[key]; ok {
				switch key {
				case "description":
					meta.Description = v.(string)
				case "input_price":
					meta.InputPrice = v.(float64)
				case "output_price":
					meta.OutputPrice = v.(float64)
				case "cache_price":
					meta.CachePrice = v.(float64)
				case "context_window":
					meta.ContextWindow = v.(int64)
				}
			}
		}
		if s, ok := updates["status"].(string); ok {
			meta.Status = s
		}
		if err := database.DB.Create(&meta).Error; err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to create meta")
			return
		}
	default:
		writeAPIError(w, http.StatusInternalServerError, "failed to load meta")
		return
	}
	database.DB.Where("name = ?", name).First(&meta)
	writeJSON(w, http.StatusOK, map[string]interface{}{"meta": meta})
}

/* ── Delete ──────────────────────────────────────────────────────────────── */

// Delete removes a meta; the model keeps serving, just without catalogue info.
func (h *ModelMetaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	name := strings.TrimSpace(r.PathValue("name"))
	if name == "" || len(name) > 128 {
		writeAPIError(w, http.StatusBadRequest, "invalid model name")
		return
	}
	res := database.DB.Where("name = ?", name).Delete(&model.ModelMeta{})
	if res.Error != nil || res.RowsAffected == 0 {
		writeAPIError(w, http.StatusNotFound, "model meta not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}
