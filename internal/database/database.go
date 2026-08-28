package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/RingKoAI/RingRouter/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance.
var DB *gorm.DB

// quoteIdentifier quotes a SQL identifier for the active dialect: MySQL uses
// backticks, PostgreSQL/SQLite use double quotes. Needed for reserved-word
// columns (e.g. `group`) in raw statements. Returns the input unchanged when
// no database is connected (tests that bypass Connect).
var quoteIdentifier = func(name string) string { return name }

// QuoteIdentifier quotes a column/table name for the active dialect.
func QuoteIdentifier(name string) string {
	return quoteIdentifier(name)
}

// Config holds database connection configuration.
type Config struct {
	Type string // sqlite, mysql, postgres
	Path string // for sqlite: file path
	DSN  string // for mysql/postgres: connection string
}

// Connect initializes the database connection and runs auto-migration.
func Connect(cfg Config) error {
	var dialector gorm.Dialector
	var dbName string

	switch cfg.Type {
	case "sqlite":
		dir := filepath.Dir(cfg.Path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}
		}
		dialector = sqlite.Open(cfg.Path)
		dbName = cfg.Path
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
		dbName = "mysql"
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
		dbName = "postgres"
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	logLevel := logger.Warn
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("connect %s: %w", dbName, err)
	}

	// Connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if cfg.Type == "sqlite" {
		// SQLite pragmas
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA busy_timeout=5000")
	}

	DB = db

	// Wire dialect-aware identifier quoting for raw statements.
	switch db.Dialector.Name() {
	case "mysql":
		quoteIdentifier = func(name string) string { return "`" + name + "`" }
	default:
		quoteIdentifier = func(name string) string { return `"` + name + `"` }
	}

	// Auto-migrate
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("auto-migrate: %w", err)
	}

	// Drop sessions that expired while the server was down.
	if err := DB.Where("expires_at < ?", time.Now()).Delete(&model.Session{}).Error; err != nil {
		log.Printf("[ringrouter] WARNING: session cleanup failed: %v", err)
	}

	log.Printf("[ringrouter] database connected: %s", cfg.Type)
	return nil
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Channel{},
		&model.Log{},
		&model.Session{},
		&model.Option{},
		&model.Group{},
		&model.Code{},
		&model.Plan{},
		&model.Subscription{},
		&model.Passkey{},
		&model.ModelMeta{},
	)
}

// Close closes the database connection.
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
