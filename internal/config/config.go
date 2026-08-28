// Package config loads application configuration from environment variables
// (OneAPI style). Static bootstrap values — server port, database DSN, Redis,
// admin key — come from env; operator-tunable settings (site name, SMTP,
// usage mode, announcement) live in the options table and are served by
// internal/setting with hot reload. Provider channels are managed entirely
// through the database admin API.
package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the fully resolved runtime configuration.
type Config struct {
	Port       int
	GinMode    string
	LogLevel   string

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
		GinMode:       getEnv("GIN_MODE", "release"),
		LogLevel:      getEnv("LOG_LEVEL", DefaultLogLevel),
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

	// Derive a random secret when none was supplied, so out-of-the-box
	// deployments still seal secrets and sign sessions uniquely.
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "ringrouter-" + randomSuffix()
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

func randomSuffix() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
