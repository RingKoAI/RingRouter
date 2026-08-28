package handler

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
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
	})
}

// Plaza aggregates the public model catalogue: every model served by an
// active channel, the protocols offering it, and the group multipliers that
// apply. No authentication — this is marketing/browsing surface. Channel
// names and keys are never included.
func (h *StatusHandler) Plaza(w http.ResponseWriter, r *http.Request) {
	if database.DB == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "database not available")
		return
	}
	var channels []model.Channel
	database.DB.Where("status = ?", "active").Find(&channels)
	var groups []model.Group
	database.DB.Find(&groups)

	type plazaGroup struct {
		Name  string  `json:"name"`
		Ratio float64 `json:"ratio"`
	}
	type plazaEntry struct {
		Model     string       `json:"model"`
		Protocols []string     `json:"protocols"`
		Channels  int          `json:"channels"`
		Groups    []plazaGroup `json:"groups"`
	}
	groupInfo := func(name string) plazaGroup {
		for _, g := range groups {
			if g.Name == name && setting.ValidGroupRatio(g.Ratio) {
				return plazaGroup{Name: g.Name, Ratio: g.Ratio}
			}
		}
		return plazaGroup{Name: name, Ratio: 1}
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
				e = &plazaEntry{Model: m}
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
					e.Groups = append(e.Groups, groupInfo(g))
				}
			}
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
