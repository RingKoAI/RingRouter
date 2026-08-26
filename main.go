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

	"github.com/RingKoAI/RingRouter/internal/config"
	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/gateway"
	"github.com/RingKoAI/RingRouter/internal/handler"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/provider"
	"github.com/RingKoAI/RingRouter/internal/router"
)

//go:embed web/dist
var frontendFS embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[ringrouter] starting...")

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
	if err := database.Connect(dbCfg); err != nil {
		log.Printf("[ringrouter] WARNING: database connection failed: %v", err)
		log.Println("[ringrouter] running without database - admin key auth only")
	} else {
		defer database.Close()
	}

	// Env-configured fallback provider (used when no DB channel matches).
	var envProvider provider.Provider
	if cfg.OpenAIKey != "" {
		envProvider = provider.NewOpenAI(cfg.OpenAIKey, cfg.OpenAIBaseURL)
	}

	// Gateway routes across DB channels with failover.
	gw := gateway.New()
	proxy := handler.NewProxy(gw, envProvider)

	// Auth middleware
	auth := middleware.NewAuth(cfg.AdminKey)

	// Embedded frontend
	frontend, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		log.Fatalf("[ringrouter] failed to open frontend: %v", err)
	}
	h := router.Setup(proxy, auth, frontend)

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