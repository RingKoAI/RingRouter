package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RingKoAI/RingRouter/internal/cache"
	"github.com/RingKoAI/RingRouter/internal/config"
	"github.com/RingKoAI/RingRouter/internal/crypto"
	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/gateway"
	"github.com/RingKoAI/RingRouter/internal/handler"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/provider"
	"github.com/RingKoAI/RingRouter/internal/router"
	"github.com/RingKoAI/RingRouter/internal/service"
	"github.com/RingKoAI/RingRouter/internal/setting"
	"github.com/RingKoAI/RingRouter/internal/turnstile"
)

//go:embed web/dist
var frontendFS embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[ringrouter] starting...")

	// Initialize Turnstile (reads TURNSTILE_SECRET / TURNSTILE_SITEKEY from env).
	turnstile.Init()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[ringrouter] failed to load config: %v", err)
	}

	// Initialize database
	dbCfg := database.Config{
		Type: cfg.DBType,
		Path: cfg.DBPath,
		DSN:  cfg.DBDSN,
	}
	// Containers can restart out of order; retry briefly before degrading
	// to admin-key-only mode so a fast app never races a slower database.
	var dbErr error
	for attempt := 1; attempt <= 15; attempt++ {
		if dbErr = database.Connect(dbCfg); dbErr == nil {
			break
		}
		if attempt == 1 {
			log.Printf("[ringrouter] database not ready, retrying: %v", dbErr)
		}
		time.Sleep(2 * time.Second)
	}
	if dbErr != nil {
		log.Printf("[ringrouter] WARNING: database connection failed after retries: %v", dbErr)
		log.Println("[ringrouter] running without database - admin key auth only")
	} else {
		defer database.Close()
		if err := setting.LoadFromDB(); err != nil {
			log.Printf("[ringrouter] WARNING: failed to load settings: %v", err)
		}
		setting.EnsureDefaultGroup()
		// Hot reload: pick up option edits made on any instance (SMTP,
		// passkey, announcement, ...) without a restart.
		settingCtx, stopSettings := context.WithCancel(context.Background())
		defer stopSettings()
		setting.StartAutoRefresh(settingCtx)
	}

	// Env-configured fallback provider (used when no DB channel matches).
	var envProvider provider.Provider
	if cfg.OpenAIKey != "" {
		envProvider = provider.NewOpenAI(cfg.OpenAIKey, cfg.OpenAIBaseURL)
	}

	// Gateway routes across DB channels with failover. Redis is optional:
	// when enabled it serves a shared channel snapshot across instances and
	// degrades to the DB on any failure.
	redis := cache.New(cfg.RedisEnabled, cache.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if cfg.RedisEnabled {
		if redis.Ping(context.Background()) {
			log.Printf("[ringrouter] redis connected: %s (db %d)", cfg.RedisAddr, cfg.RedisDB)
		} else {
			log.Printf("[ringrouter] WARNING: redis at %s unreachable — falling back to in-memory cache", cfg.RedisAddr)
		}
	}
	defer redis.Close()

	gw := gateway.New(redis)
	proxy := handler.NewProxy(gw, envProvider)

	// Auth middleware
	auth := middleware.NewAuth(cfg.AdminKey)
	sess := middleware.NewSessionAuth(cfg.AdminKey)
	adminH := handler.NewAdminHandler()
	groupH := handler.NewGroupHandler()

	// Secret sealer for options stored at rest (SMTP passwords, channel keys).
	sealer, err := crypto.NewEncryptor(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("[ringrouter] failed to init encryptor: %v", err)
	}
	service.SetDecryptor(sealer.Decrypt)
	provider.SetDecryptor(sealer.Decrypt)

	mailer := service.NewMailer()
	authH := handler.NewAuthHandler(cfg.AdminKey, mailer)
	setupH := handler.NewSetupHandler(sealer, mailer)
	settingsH := handler.NewSettingsHandler(sealer)
	channelH := handler.NewChannelHandler(sealer, func() {
		gw.InvalidateChannels(context.Background())
	})
	statusH := handler.NewStatusHandler(cfg.RedisEnabled)
	planH := handler.NewPlanHandler()
	subH := handler.NewSubscriptionHandler()
	tokenH := handler.NewTokenHandler()
	logH := handler.NewLogHandler()
	metaH := handler.NewModelMetaHandler()
	passkeyH := handler.NewPasskeyHandler(authH)

	// Seed the announcement from the env on first boot; later updates go
	// through /api/admin/settings and hot-reload on the next read.
	if cfg.Announcement != "" && setting.Get(setting.KeyAnnouncement) == "" {
		if err := setting.Set(setting.KeyAnnouncement, cfg.Announcement); err != nil {
			log.Printf("[ringrouter] WARNING: failed to seed announcement: %v", err)
		}
	}

	// Embedded frontend
	frontend, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		log.Fatalf("[ringrouter] failed to open frontend: %v", err)
	}
	h := router.Setup(proxy, auth, sess, authH, adminH, groupH, setupH, settingsH, channelH, statusH, planH, subH, tokenH, logH, metaH, passkeyH, frontend)

	// HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // long timeout for streaming
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[ringrouter] listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[ringrouter] listen error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[ringrouter] shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[ringrouter] shutdown error: %v", err)
	}
	log.Println("[ringrouter] exited")
}
