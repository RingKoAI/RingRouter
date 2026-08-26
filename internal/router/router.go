package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/handler"
	"github.com/RingKoAI/RingRouter/internal/middleware"
)

// Setup creates and returns the HTTP handler with all routes configured.
func Setup(proxy *handler.Proxy, auth *middleware.Auth, frontend fs.FS) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", proxy.Health)

	// API v1 routes (authenticated)
	v1 := http.NewServeMux()
	v1.HandleFunc("POST /chat/completions", proxy.ChatCompletion)
	v1.HandleFunc("GET /models", proxy.ListModels)
	mux.Handle("/v1/", auth.Middleware(http.StripPrefix("/v1", v1)))

	// Frontend static files
	fsys := http.FS(frontend)
	fileServer := http.FileServer(fsys)
	mux.Handle("/assets/", fileServer)
	mux.Handle("/favicon.svg", fileServer)
	mux.Handle("/icons.svg", fileServer)

	// SPA fallback: serve index.html for all non-API routes
	indexHTML, err := fs.ReadFile(frontend, "index.html")
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, "frontend not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.Handle("/", spaHandler)

	// Apply global middleware
	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.CORS(h)

	return h
}

// Ensure unused import is not flagged
var _ = strings.TrimSpace