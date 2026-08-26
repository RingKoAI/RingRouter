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

	// Authenticated API surface — every inbound wire format.
	api := http.NewServeMux()
	// OpenAI
	api.HandleFunc("POST /chat/completions", proxy.ChatCompletion)
	api.HandleFunc("GET /models", proxy.ListModels)
	// Anthropic
	api.HandleFunc("POST /messages", proxy.ChatCompletion)
	// Google Gemini (generateContent + streamGenerateContent)
	api.HandleFunc("POST /v1beta/models/{model...}", proxy.ChatCompletion)

	mux.Handle("/v1/", auth.Middleware(http.StripPrefix("/v1", api)))

	// Frontend static files
	fileServer := http.FileServer(http.FS(frontend))
	mux.Handle("/assets/", fileServer)
	mux.Handle("/favicon.svg", fileServer)
	mux.Handle("/icons.svg", fileServer)

	// SPA fallback: index.html for everything that is not an API path.
	indexHTML, err := fs.ReadFile(frontend, "index.html")
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, "frontend not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			http.NotFound(w, r)
			return
		}
		spaHandler.ServeHTTP(w, r)
	})

	// Global middleware
	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.CORS(h)

	return h
}