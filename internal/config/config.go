package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for RingRouter.
type Config struct {
	Port       int
	GinMode    string
	DBType     string
	DBPath     string
	DBDSN      string
	JWTSecret  string
	AdminKey   string
	LogLevel   string
	RedisEnabled bool
	RedisAddr  string
	RedisPassword string
	RedisDB    int

	// Default upstream provider
	DefaultProvider string
	OpenAIKey       string
	OpenAIBaseURL   string
}

// Defaults
const (
	DefaultPort    = 3000
	DefaultDBType  = "sqlite"
	DefaultDBPath  = "data/ringrouter.db"
	DefaultLogLevel = "info"
)

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	// Try loading .env file, ignore error if not found
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnvInt("PORT", DefaultPort),
		GinMode:      getEnv("GIN_MODE", "release"),
		DBType:       getEnv("DB_TYPE", DefaultDBType),
		DBPath:       getEnv("DB_PATH", DefaultDBPath),
		DBDSN:        getEnv("DB_DSN", ""),
		JWTSecret:    getEnv("JWT_SECRET", "ringrouter-default-secret"),
		AdminKey:     getEnv("ADMIN_KEY", ""),
		LogLevel:     getEnv("LOG_LEVEL", DefaultLogLevel),
		RedisEnabled: getEnvBool("REDIS_ENABLED", false),
		RedisAddr:    getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:      getEnvInt("REDIS_DB", 0),
		DefaultProvider: getEnv("DEFAULT_PROVIDER", "openai"),
		OpenAIKey:    getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
	}

	// Validate
	if cfg.JWTSecret == "ringrouter-default-secret" {
		cfg.JWTSecret = "ringrouter-" + randomSuffix()
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		switch v {
		case "1", "true", "TRUE", "yes", "YES":
			return true
		case "0", "false", "FALSE", "no", "NO":
			return false
		}
	}
	return defaultVal
}

func randomSuffix() string {
	// Simple random suffix for development
	h, _ := os.Hostname()
	return h
}