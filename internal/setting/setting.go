// Package setting provides instance-level options persisted in the options
// table, with an in-process cache for hot-path reads.
package setting

import (
	"strconv"
	"strings"
	"sync"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/model"
)

// UsageMode controls how the instance is operated.
type UsageMode string

const (
	// UsageModeExternal serves external users with billing enabled.
	UsageModeExternal UsageMode = "external"
	// UsageModeSelf is personal use; unbilled models pass through.
	UsageModeSelf UsageMode = "self"
	// UsageModeDemo is a public demo with heavy restrictions.
	UsageModeDemo UsageMode = "demo"
)

// ValidUsageModes is the whitelist accepted from the setup wizard / API.
var ValidUsageModes = []UsageMode{UsageModeExternal, UsageModeSelf, UsageModeDemo}

// IsValidUsageMode reports whether m is an accepted mode value.
func IsValidUsageMode(m UsageMode) bool {
	for _, v := range ValidUsageModes {
		if v == m {
			return true
		}
	}
	return false
}

// Option keys persisted in the options table.
const (
	KeySiteName     = "site_name"
	KeyAnnouncement = "announcement"
	KeyUsageMode    = "usage_mode"
	KeySMTPEnabled  = "smtp_enabled"
	KeySMTPHost     = "smtp_host"
	KeySMTPPort     = "smtp_port"
	KeySMTPUsername = "smtp_username"
	KeySMTPPassword = "smtp_password" // AES-GCM sealed
	KeySMTPFrom     = "smtp_from"

	// Passkey (WebAuthn) settings.
	KeyPasskeyEnabled = "passkey_enabled"
	KeyPasskeyRPID    = "passkey_rp_id"     // e.g. example.com
	KeyPasskeyOrigins = "passkey_rp_origins" // csv, e.g. https://example.com
)

// DefaultSiteName is used when the site_name option is unset.
const DefaultSiteName = "RingRouter"

// SiteName returns the configured instance name.
func SiteName() string {
	if n := strings.TrimSpace(Get(KeySiteName)); n != "" {
		return n
	}
	return DefaultSiteName
}

// Announcement returns the operator-configured notice shown on the dashboard
// header. Empty string hides the announcement button.
func Announcement() string {
	return Get(KeyAnnouncement)
}

// SMTPConfig is a resolved mail transport configuration.
type SMTPConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string // decrypted
	From     string
}

var (
	mu    sync.RWMutex
	cache = map[string]string{}
)

// LoadFromDB populates the option cache. Called once at startup after the
// database connection is established.
func LoadFromDB() error {
	if database.DB == nil {
		return nil
	}
	var opts []model.Option
	if err := database.DB.Find(&opts).Error; err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	cache = make(map[string]string, len(opts))
	for _, o := range opts {
		cache[o.Key] = o.Value
	}
	return nil
}

// Get returns a raw option value.
func Get(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	return cache[key]
}

// Reset clears the in-process option cache (used by tests for isolation).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	cache = make(map[string]string)
}

// Set persists an option and updates the cache. Empty value deletes the key.
func Set(key, value string) error {
	if database.DB == nil {
		return nil
	}
	if value == "" {
		delete(cache, key)
		return database.DB.Where("key = ?", key).Delete(&model.Option{}).Error
	}
	if err := database.DB.Save(&model.Option{Key: key, Value: value}).Error; err != nil {
		return err
	}
	mu.Lock()
	cache[key] = value
	mu.Unlock()
	return nil
}

// CurrentUsageMode returns the configured mode (default external).
func CurrentUsageMode() UsageMode {
	if m := UsageMode(Get(KeyUsageMode)); IsValidUsageMode(m) {
		return m
	}
	return UsageModeExternal
}

// PasskeyConfig is the resolved WebAuthn relying-party configuration.
type PasskeyConfig struct {
	Enabled bool
	RPID    string
	Origins []string
}

// Defaults target local development; production deployments must configure
// the real domain via the admin settings API.
const (
	DefaultPasskeyRPID    = "localhost"
	DefaultPasskeyOrigins = "http://localhost:5173,http://localhost:3000"
)

// Passkey returns the resolved WebAuthn configuration.
func Passkey() PasskeyConfig {
	rpID := strings.TrimSpace(Get(KeyPasskeyRPID))
	if rpID == "" {
		rpID = DefaultPasskeyRPID
	}
	raw := strings.TrimSpace(Get(KeyPasskeyOrigins))
	if raw == "" {
		raw = DefaultPasskeyOrigins
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	return PasskeyConfig{
		Enabled: Get(KeyPasskeyEnabled) == "true",
		RPID:    rpID,
		Origins: origins,
	}
}

// SMTP returns the resolved mail configuration. Pass an Encryptor to decrypt
// the stored password; a nil encryptor yields the sealed value, which callers
// treating it as opaque (e.g. status checks) can use.
func SMTP(decrypt func(string) (string, error)) SMTPConfig {
	port, _ := strconv.Atoi(Get(KeySMTPPort))
	cfg := SMTPConfig{
		Enabled:  Get(KeySMTPEnabled) == "true",
		Host:     Get(KeySMTPHost),
		Port:     port,
		Username: Get(KeySMTPUsername),
		From:     Get(KeySMTPFrom),
	}
	sealed := Get(KeySMTPPassword)
	if decrypt != nil && sealed != "" {
		if plain, err := decrypt(sealed); err == nil {
			cfg.Password = plain
		}
	} else if decrypt == nil {
		cfg.Password = sealed
	}
	return cfg
}
