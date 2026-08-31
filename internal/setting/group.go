package setting

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// groupUUIDLen is the byte length of a generated group UUID (16 bytes → 32 hex chars).
const groupUUIDLen = 16

// ValidGroupUUID reports whether s is a well-formed group UUID: exactly 32
// lowercase/uppercase hex characters (canonical form is lowercase).
func ValidGroupUUID(s string) bool {
	if len(s) != groupUUIDLen*2 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// newGroupUUID returns a 32-char hex token. Uniqueness is enforced at the
// database level (unique index on groups.uuid).
func newGroupUUID() (string, error) {
	buf := make([]byte, groupUUIDLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// EnsureDefaultGroup creates the "default" group on first run so the
// existing user/channel group strings remain valid. Idempotent.
func EnsureDefaultGroup() {
	if database.DB == nil {
		return
	}
	var g model.Group
	err := database.DB.Where("name = ?", "default").First(&g).Error
	if err == nil {
		return
	}
	uuid, err := newGroupUUID()
	if err != nil {
		return
	}
	_ = database.DB.Create(&model.Group{
		Name:      "default",
		UUID:      uuid,
		Metadata:  "Default routing group. All new users and ungrouped channels fall back here.",
		Ratio:     1,
		IsDefault: true,
	}).Error
	InvalidateGroups() // first-boot seeding must be visible immediately
}

// Group ratio bounds: strictly positive (a zero ratio would silently make
// everything free) and capped to reject fat-fingered values.
const (
	MinGroupRatio = 0.01
	MaxGroupRatio = 100
)

// ValidGroupRatio reports whether r is an acceptable billing multiplier.
func ValidGroupRatio(r float64) bool {
	return r >= MinGroupRatio && r <= MaxGroupRatio
}

/* ── Group lookup cache ───────────────────────────────────────────────────── */

// groupCacheTTL bounds staleness for cross-instance convergence: writes on
// the local instance invalidate immediately (see InvalidateGroups), other
// instances converge within the TTL — the same contract as the channel
// snapshot in the gateway.
const groupCacheTTL = 15 * time.Second

type groupEntry struct {
	ratio  float64
	exists bool
}

var (
	groupMu    sync.RWMutex
	groupCache = map[string]groupEntry{}
	groupAt    time.Time
)

// InvalidateGroups drops the group lookup cache. The admin group handlers
// call it after every mutation so changes take effect on the next request
// instead of waiting out the TTL.
func InvalidateGroups() {
	groupMu.Lock()
	groupCache = map[string]groupEntry{}
	groupAt = time.Time{}
	groupMu.Unlock()
}

// ResetGroups clears the group cache unconditionally (used by tests when
// the underlying database instance is swapped out).
func ResetGroups() {
	groupMu.Lock()
	groupCache = nil
	groupAt = time.Time{}
	groupMu.Unlock()
}

// loadGroups returns the (possibly cached) name→entry view of the groups
// table, refreshing it when stale.
func loadGroups() map[string]groupEntry {
	groupMu.RLock()
	if groupCache != nil && time.Since(groupAt) < groupCacheTTL {
		defer groupMu.RUnlock()
		return groupCache
	}
	groupMu.RUnlock()

	fresh := map[string]groupEntry{}
	if database.DB != nil {
		var groups []model.Group
		if err := database.DB.Find(&groups).Error; err == nil {
			for _, g := range groups {
				ratio := 1.0
				if ValidGroupRatio(g.Ratio) {
					ratio = g.Ratio
				}
				fresh[g.Name] = groupEntry{ratio: ratio, exists: true}
			}
		}
	}

	groupMu.Lock()
	groupCache, groupAt = fresh, time.Now()
	groupMu.Unlock()
	return fresh
}

// GroupRatio returns the billing multiplier for a group (default 1.0 when
// the group or its ratio is unset — fail open to list price, never to zero).
func GroupRatio(name string) float64 {
	if database.DB == nil || name == "" {
		return 1
	}
	if e, ok := loadGroups()[name]; ok {
		return e.ratio
	}
	return 1
}

// GroupExists reports whether a group with the given name exists.
func GroupExists(name string) bool {
	if database.DB == nil || name == "" {
		return false
	}
	e, ok := loadGroups()[name]
	return ok && e.exists
}

// CreateGroup persists a new group. An empty uuid generates one (retrying on
// the astronomically unlikely collision); a caller-supplied uuid is used
// verbatim after validation. ratio is the billing multiplier (1 = list
// price). Returns gorm.ErrDuplicatedKey on name or uuid collision without
// leaving partial state behind.
func CreateGroup(name, uuid, metadata string, ratio float64) (*model.Group, error) {
	if database.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	var g *model.Group
	var err error
	if uuid != "" {
		g, err = createGroupRow(name, uuid, metadata, ratio)
	} else {
		for attempt := 0; attempt < 3; attempt++ {
			u, uerr := newGroupUUID()
			if uerr != nil {
				return nil, uerr
			}
			if g, err = createGroupRow(name, u, metadata, ratio); err == nil || err != gorm.ErrDuplicatedKey {
				break
			}
		}
	}
	if err != nil {
		return nil, err
	}
	// Keep the lookup cache coherent no matter which layer wrote the row.
	InvalidateGroups()
	return g, nil
}

func createGroupRow(name, uuid, metadata string, ratio float64) (*model.Group, error) {
	g := &model.Group{Name: name, UUID: uuid, Metadata: metadata, Ratio: ratio}
	if err := database.DB.Create(g).Error; err != nil {
		return nil, err
	}
	return g, nil
}
