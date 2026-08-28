package setting

import (
	"crypto/rand"
	"encoding/hex"

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

// GroupRatio returns the billing multiplier for a group (default 1.0 when
// the group or its ratio is unset — fail open to list price, never to zero).
func GroupRatio(name string) float64 {
	if database.DB == nil || name == "" {
		return 1
	}
	var g model.Group
	if err := database.DB.Select("ratio").Where("name = ?", name).First(&g).Error; err != nil {
		return 1
	}
	if !ValidGroupRatio(g.Ratio) {
		return 1
	}
	return g.Ratio
}

// GroupExists reports whether a group with the given name exists.
func GroupExists(name string) bool {
	if database.DB == nil || name == "" {
		return false
	}
	var count int64
	database.DB.Model(&model.Group{}).Where("name = ?", name).Count(&count)
	return count > 0
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
	if uuid != "" {
		g := &model.Group{Name: name, UUID: uuid, Metadata: metadata, Ratio: ratio}
		if err := database.DB.Create(g).Error; err != nil {
			return nil, err
		}
		return g, nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		u, err := newGroupUUID()
		if err != nil {
			return nil, err
		}
		g := &model.Group{Name: name, UUID: u, Metadata: metadata, Ratio: ratio}
		if err := database.DB.Create(g).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				continue
			}
			return nil, err
		}
		return g, nil
	}
	return nil, gorm.ErrDuplicatedKey
}
