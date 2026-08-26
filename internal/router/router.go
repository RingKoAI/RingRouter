package router

import (
	"net/http"

	"github.com/RingKoAI/RingRouter/internal/handler"
	"github.com/RingKoAI/RingRouter/internal/middleware"
)

// Setup creates and returns the HTTP handler with all routes configured.
func Setup(proxy *handler.Proxy, auth *middleware.Auth) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", proxy.Health)

	// API v1 routes (authenticated)
	v1 := http.NewServeMux()
	v1.HandleFunc("POST /chat/completions", proxy.ChatCompletion)
	v1.HandleFunc("GET /models", proxy.ListModels)
	mux.Handle("/v1/", auth.Middleware(http.StripPrefix("/v1", v1)))

	// Apply global middleware
	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.CORS(h)

	return h
}