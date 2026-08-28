package handler

import (
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// logPageSize bounds pagination.
const (
	defaultLogPage = 20
	maxLogPageSize = 100
)

// LogHandler serves usage-log queries.
type LogHandler struct{}

// NewLogHandler creates a LogHandler.
func NewLogHandler() *LogHandler {
	return &LogHandler{}
}

// Mine returns the caller's logs, newest first, with pagination and optional
// model/status filters.
func (h *LogHandler) Mine(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	u := middleware.GetUser(r.Context())
	if u == nil || u.ID == 0 {
		writeAPIError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	h.query(w, r, func(tx *gorm.DB) *gorm.DB { return tx.Where("user_id = ?", u.ID) })
}

// All returns every log (admin only), with user/model/status filters.
func (h *LogHandler) All(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	h.query(w, r, func(tx *gorm.DB) *gorm.DB {
		if uid := r.URL.Query().Get("user_id"); uid != "" {
			if id, err := strconv.ParseUint(uid, 10, 64); err == nil && id > 0 {
				return tx.Where("user_id = ?", id)
			}
		}
		return tx
	})
}

// query applies shared filters (model, status), pagination, and shaping.
func (h *LogHandler) query(w http.ResponseWriter, r *http.Request, scope func(*gorm.DB) *gorm.DB) {
	page, pageSize := parseLogPage(r)
	modelName := strings.TrimSpace(r.URL.Query().Get("model"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	tx := scope(database.DB.Model(&model.Log{}))
	if modelName != "" {
		tx = tx.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	if status == "success" || status == "failed" {
		tx = tx.Where("status = ?", status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to count logs")
		return
	}
	var logs []model.Log
	if err := tx.Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&logs).Error; err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list logs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs": logs,
		"pagination": map[string]int64{
			"page":      int64(page),
			"page_size": int64(pageSize),
			"total":     total,
		},
	})
}

func parseLogPage(r *http.Request) (page, pageSize int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = defaultLogPage
	}
	if pageSize > maxLogPageSize {
		pageSize = maxLogPageSize
	}
	return page, pageSize
}
