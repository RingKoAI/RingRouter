package handler

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/setting"
	"github.com/RingKoAI/RingRouter/internal/turnstile"
)

// version is the build version reported by /api/status.
const version = "0.1.0"

// processStart marks server boot time for uptime reporting.
var processStart = time.Now()

// StatusHandler serves the public instance-status endpoint.
type StatusHandler struct {
	redisEnabled bool
}

// NewStatusHandler creates a StatusHandler. redisEnabled only reports the
// deployment's cache topology; the status endpoint itself never depends on
// Redis being reachable.
func NewStatusHandler(redisEnabled bool) *StatusHandler {
	return &StatusHandler{redisEnabled: redisEnabled}
}

// Status exposes public instance metadata consumed by the frontend: site
// name, usage mode, version, Turnstile sitekey, and service availability.
// It requires no authentication so the login and home pages can render
// branded content before a session exists.
func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	smtpCfg := setting.SMTP(nil)
	pk := setting.Passkey()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"site_name":         setting.SiteName(),
		"usage_mode":        string(setting.CurrentUsageMode()),
		"version":           version,
		"smtp_configured":   smtpConfigured(smtpCfg),
		"passkey_enabled":   pk.Enabled,
		"turnstile_enabled": turnstile.Enabled(),
		"turnstile_sitekey": turnstile.Sitekey(),
		"plaza_public":      setting.PlazaPublic(),
	})
}

// Plaza aggregates the public model catalogue: every model served by an
// active channel, enriched with catalogue metadata (vendor, description,
// list prices, context window), per-group effective prices (list × group
// ratio), and measured latency/throughput from recent successful requests.
// No authentication — this is marketing/browsing surface. Channel names and
// keys are never included.
func (h *StatusHandler) Plaza(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	// Operator may restrict the plaza to signed-in users.
	if !setting.PlazaPublic() && middleware.GetUser(r.Context()) == nil {
		writeAPIError(w, http.StatusUnauthorized, "sign in to browse the model plaza")
		return
	}
	var channels []model.Channel
	database.DB.Where("status = ?", "active").Find(&channels)
	var groups []model.Group
	database.DB.Find(&groups)
	var metas []model.ModelMeta
	database.DB.Where("status = ?", "active").Find(&metas)

	type plazaGroup struct {
		Name        string  `json:"name"`
		Ratio       float64 `json:"ratio"`
		InputPrice  float64 `json:"input_price"` // $/1M tokens, effective
		OutputPrice float64 `json:"output_price"`
		CachePrice  float64 `json:"cache_price"`
	}
	type plazaStats struct {
		AvgMs   float64 `json:"avg_ms"`  // mean request latency
		Tps     float64 `json:"tps"`     // mean completion tokens/sec
		Samples int64   `json:"samples"` // successful request count
	}
	type plazaEntry struct {
		Model         string       `json:"model"`
		Vendor        string       `json:"vendor"`
		Description   string       `json:"description"`
		Protocols     []string     `json:"protocols"`
		Channels      int          `json:"channels"`
		Groups        []plazaGroup `json:"groups"`
		ContextWindow int64        `json:"context_window"`
		InputPrice    float64      `json:"input_price"` // list prices
		OutputPrice   float64      `json:"output_price"`
		CachePrice    float64      `json:"cache_price"`
		Stats         plazaStats   `json:"stats"`
	}

	groupRatio := func(name string) (string, float64) {
		for _, g := range groups {
			if g.Name == name {
				ratio := 1.0
				if setting.ValidGroupRatio(g.Ratio) {
					ratio = g.Ratio
				}
				return g.Name, ratio
			}
		}
		return name, 1
	}

	// Measured performance from recent successful logs (last 7 days).
	type statsRow struct {
		Model string
		AvgMs float64
		Tps   float64
		Count int64
	}
	var statsRows []statsRow
	database.DB.Raw(`
		SELECT model_name AS model,
		       AVG(elapsed_ms) AS avg_ms,
		       AVG(CASE WHEN elapsed_ms > 0 AND completion_tokens > 0
		                THEN completion_tokens * 1000.0 / elapsed_ms END) AS tps,
		       COUNT(*) AS count
		FROM logs
		WHERE status = 'success' AND created_at > ?
		GROUP BY model_name`, time.Now().AddDate(0, 0, -7)).Scan(&statsRows)
	stats := make(map[string]statsRow, len(statsRows))
	for _, row := range statsRows {
		stats[row.Model] = row
	}

	catalog := make(map[string]*plazaEntry)
	seenGroups := make(map[string]map[string]bool)
	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			e, ok := catalog[m]
			if !ok {
				e = &plazaEntry{Model: m, Vendor: "other"}
				catalog[m] = e
				seenGroups[m] = make(map[string]bool)
			}
			e.Channels++
			if !contains(e.Protocols, ch.Protocol) {
				e.Protocols = append(e.Protocols, ch.Protocol)
			}
			for _, g := range strings.Split(ch.Group, ",") {
				if g = strings.TrimSpace(g); g != "" && !seenGroups[m][g] {
					seenGroups[m][g] = true
					name, ratio := groupRatio(g)
					e.Groups = append(e.Groups, plazaGroup{
						Name: name, Ratio: ratio,
						InputPrice:  e.InputPrice * ratio,
						OutputPrice: e.OutputPrice * ratio,
						CachePrice:  e.CachePrice * ratio,
					})
				}
			}
		}
	}

	// Merge catalogue metadata and recompute effective group prices.
	for _, meta := range metas {
		e, ok := catalog[meta.Name]
		if !ok {
			continue // metadata for a model no channel serves: skip
		}
		e.Vendor = meta.Vendor
		e.Description = meta.Description
		e.ContextWindow = meta.ContextWindow
		e.InputPrice = meta.InputPrice
		e.OutputPrice = meta.OutputPrice
		e.CachePrice = meta.CachePrice
		for i := range e.Groups {
			e.Groups[i].InputPrice = meta.InputPrice * e.Groups[i].Ratio
			e.Groups[i].OutputPrice = meta.OutputPrice * e.Groups[i].Ratio
			e.Groups[i].CachePrice = meta.CachePrice * e.Groups[i].Ratio
		}
	}
	for m, e := range catalog {
		if row, ok := stats[m]; ok {
			e.Stats = plazaStats{AvgMs: row.AvgMs, Tps: row.Tps, Samples: row.Count}
		}
	}

	out := make([]plazaEntry, 0, len(catalog))
	for _, e := range catalog {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model < out[j].Model })
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"models":     out,
		"site_name":  setting.SiteName(),
		"usage_mode": string(setting.CurrentUsageMode()),
	})
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// System reports runtime information for the admin dashboard: non-sensitive
// operational metadata only (versions, memory, uptime, storage topology).
func (h *StatusHandler) System(w http.ResponseWriter, r *http.Request) {
	db := "none"
	if database.DB != nil {
		db = database.DB.Dialector.Name()
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"go_version":     runtime.Version(),
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"num_goroutine":  runtime.NumGoroutine(),
		"mem_alloc_mb":   m.Alloc / 1024 / 1024,
		"uptime_seconds": int64(time.Since(processStart).Seconds()),
		"db_type":        db,
		"redis_enabled":  h.redisEnabled,
	})
}
