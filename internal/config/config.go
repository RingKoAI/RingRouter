// Package config loads application configuration from environment variables
// (OneAPI style). Static bootstrap values — server port, database DSN, Redis,
// admin key — come from env; operator-tunable settings (site name, SMTP,
// usage mode, announcement) live in the options table and are served by
// internal/setting with hot reload. Provider channels are managed entirely
// through the database admin API.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// secretFileName is where an auto-generated instance secret is persisted so
// sealed data survives restarts without forcing operators to configure one.
const secretFileName = ".instance_secret"

// legacyDefaultSecret is the pre-hardening derived default. It is checked on
// migration so existing default deployments keep their sealed channel keys
// decryptable; a fresh warning nudges the operator toward rotation.
func legacyDefaultSecret() string {
	h, err := os.Hostname()
	if err != nil || strings.TrimSpace(h) == "" {
		return "ringrouter-unknown"
	}
	return "ringrouter-" + h
}

// Config holds the fully resolved runtime configuration.
type Config struct {
	Port int

	DBType string // postgres | mysql | sqlite
	DBPath string // sqlite only
	DBDSN  string // postgres/mysql DSN

	RedisEnabled  bool
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret string
	AdminKey  string

	// Convenience env fallback for deployments without any DB channel.
	OpenAIKey     string
	OpenAIBaseURL string

	Announcement string
}

// Defaults.
const (
	DefaultPort      = 3000
	DefaultDBType    = "postgres"
	DefaultDBPath    = "data/ringrouter.db"
	DefaultLogLevel  = "info"
	DefaultRedisAddr = "127.0.0.1:6379"
	DefaultBaseURL   = "https://api.openai.com/v1"
)

// Load reads configuration from environment variables, falling back to
// built-in defaults so the instance boots with zero configuration.
func Load() (*Config, error) {
	_ = godotenv.Load() // best-effort local .env; container env wins anyway

	redisEnabled, redisAddr, redisPassword, redisDB := redisFromEnv()
	cfg := &Config{
		Port:          getEnvInt("PORT", DefaultPort),
		DBType:        getEnv("DB_TYPE", DefaultDBType),
		DBPath:        getEnv("DB_PATH", DefaultDBPath),
		DBDSN:         getEnv("DB_DSN", ""),
		RedisEnabled:  redisEnabled,
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
		JWTSecret:     getEnv("JWT_SECRET", ""),
		AdminKey:      getEnv("ADMIN_KEY", ""),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", DefaultBaseURL),
		Announcement:  getEnv("ANNOUNCEMENT", ""),
	}

	// Derive the instance secret when none was supplied. The secret seals
	// channel keys and SMTP passwords at rest, so it must be unguessable:
	// generate 256 bits from crypto/rand and persist them next to the data
	// files. Deployments upgrading from the old hostname-derived default are
	// migrated onto that same value to keep existing ciphertexts openable.
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = resolveInstanceSecret(filepath.Dir(cfg.DBPath))
	}

	return cfg, nil
}

// redisFromEnv resolves the Redis configuration. REDIS_CONN_STRING (one-api
// style, redis://[user[:password]@]host[:port][/db]) takes precedence — its
// mere presence enables Redis; otherwise the discrete REDIS_ENABLED /
// REDIS_ADDR / REDIS_PASSWORD / REDIS_DB variables apply, with Redis left
// disabled when neither is set.
func redisFromEnv() (enabled bool, addr, password string, db int) {
	if cs := os.Getenv("REDIS_CONN_STRING"); cs != "" {
		u, err := url.Parse(cs)
		if err != nil {
			return false, "", "", 0
		}
		host := u.Hostname()
		port := u.Port()
		if port == "" {
			port = "6379"
		}
		if u.User != nil {
			if p, ok := u.User.Password(); ok && p != "" {
				password = p
			} else if name := u.User.Username(); name != "" {
				password = name // tolerate redis://pass@host without a colon
			}
		}
		if s := strings.TrimPrefix(u.Path, "/"); s != "" {
			db, _ = strconv.Atoi(s)
		}
		return true, net.JoinHostPort(host, port), password, db
	}
	if !getEnvBool("REDIS_ENABLED", false) {
		return false, "", "", 0
	}
	return true, getEnv("REDIS_ADDR", DefaultRedisAddr), getEnv("REDIS_PASSWORD", ""), getEnvInt("REDIS_DB", 0)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "1", "true", "TRUE", "yes", "YES":
			return true
		case "0", "false", "FALSE", "no", "NO":
			return false
		}
	}
	return def
}

// resolveInstanceSecret loads or creates the persisted instance secret.
//
// Order of attempts:
//  1. data/.instance_secret exists → use it (stable across restarts);
//  2. legacy hostname-derived default sealed data may exist → persist that
//     value so pre-upgrade ciphertexts stay decryptable, with a loud warning;
//  3. otherwise generate fresh entropy, persist with 0600, and use it;
//  4. when the location is not writable → ephemeral random secret (works,
//     but sealed values cannot be reopened after restart; warned loudly).
func resolveInstanceSecret(dir string) string {
	legacy := legacyDefaultSecret()
	path := filepath.Join(dir, secretFileName)

	if raw, err := os.ReadFile(path); err == nil {
		if s := strings.TrimSpace(string(raw)); s != "" {
			return s
		}
	}

	// Migration: adopt the legacy derived secret when the data directory
	// already holds database files (i.e. this is an upgrade, not a fresh
	// install) so existing sealed channel keys remain decryptable.
	if legacySecretUseful(dir) {
		if err := persistSecret(path, legacy); err == nil {
			os.Stderr.WriteString("[ringrouter] WARNING: adopted the legacy hostname-derived JWT secret so existing sealed keys stay decryptable.\n" +
				"[ringrouter] Set JWT_SECRET explicitly and re-save channel keys / SMTP password to rotate onto strong entropy.\n")
			return legacy
		}
	}

	secret, err := randomHex(32)
	if err != nil {
		os.Stderr.WriteString("[ringrouter] WARNING: system entropy unavailable; using an ephemeral secret. Set JWT_SECRET explicitly.\n")
		return "ringrouter-ephemeral-" + secretFallback()
	}
	if err := persistSecret(path, secret); err != nil {
		os.Stderr.WriteString("[ringrouter] WARNING: could not persist the auto-generated instance secret (" + err.Error() + ").\n" +
			"[ringrouter] Sealed values will NOT survive a restart. Set JWT_SECRET or make " + dir + " writable.\n")
	} else {
		os.Stderr.WriteString("[ringrouter] generated instance secret at " + path + " (set JWT_SECRET to override)\n")
	}
	return secret
}

// legacySecretUseful reports whether the legacy default plausibly sealed data
// before: the data directory exists and contains database artifacts.
func legacySecretUseful(dir string) bool {
	if dir == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(name, ".db") || name == "ringrouter.db" || strings.HasPrefix(name, "ringrouter.db") {
			return true
		}
	}
	return false
}

func persistSecret(path, secret string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	// Write-then-rename keeps a partially written secret from ever being read.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(secret+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// secretFallback is a last-resort source of variability when crypto/rand is
// unavailable (effectively never on supported platforms). Deployments hitting
// this path are warned and should set JWT_SECRET explicitly.
func secretFallback() string {
	return strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
