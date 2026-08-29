package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/handler"
	"github.com/RingKoAI/RingRouter/internal/middleware"
)

// Setup creates and returns the HTTP handler with all routes configured.
func Setup(proxy *handler.Proxy, auth *middleware.Auth, sess *middleware.SessionAuth, authH *handler.AuthHandler, adminH *handler.AdminHandler, groupH *handler.GroupHandler, setupH *handler.SetupHandler, settingsH *handler.SettingsHandler, channelH *handler.ChannelHandler, statusH *handler.StatusHandler, planH *handler.PlanHandler, subH *handler.SubscriptionHandler, tokenH *handler.TokenHandler, logH *handler.LogHandler, metaH *handler.ModelMetaHandler, passkeyH *handler.PasskeyHandler, frontend fs.FS) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", proxy.Health)

	// Authenticated API surface — every inbound wire format.
	api := http.NewServeMux()
	// OpenAI
	api.HandleFunc("POST /chat/completions", proxy.ChatCompletion)
	api.HandleFunc("GET /models", proxy.ListModels)
	// OpenAI Responses API
	api.HandleFunc("POST /responses", proxy.ChatCompletion)
	// Anthropic
	api.HandleFunc("POST /messages", proxy.ChatCompletion)
	// Google Gemini (generateContent + streamGenerateContent)
	api.HandleFunc("POST /v1beta/models/{model...}", proxy.ChatCompletion)

	mux.Handle("/v1/", middleware.RateLimitAPI()(auth.Middleware(http.StripPrefix("/v1", api))))

	// Management plane (/api/*).
	mgmt := http.NewServeMux()
	// Public: no session required.
	mgmt.HandleFunc("GET /status", statusH.Status)
	mgmt.Handle("GET /plaza", sess.OptionalSession(http.HandlerFunc(statusH.Plaza)))
	mgmt.HandleFunc("GET /setup/status", setupH.Status)
	mgmt.HandleFunc("POST /setup/test-smtp", setupH.TestSMTP)
	mgmt.HandleFunc("POST /setup/complete", setupH.Complete)
	critical := middleware.RateLimitCritical()
	mgmt.Handle("POST /auth/register", critical(http.HandlerFunc(authH.Register)))
	mgmt.Handle("POST /auth/login", critical(http.HandlerFunc(authH.Login)))
	mgmt.Handle("POST /auth/admin-key", critical(http.HandlerFunc(authH.AdminKey)))
	mgmt.HandleFunc("POST /auth/logout", authH.Logout)
	mgmt.Handle("POST /auth/code", critical(http.HandlerFunc(authH.SendCode)))
	mgmt.Handle("POST /auth/reset-password", critical(http.HandlerFunc(authH.ResetPassword)))
	mgmt.Handle("POST /auth/passkey/login/begin", critical(http.HandlerFunc(passkeyH.LoginBegin)))
	mgmt.Handle("POST /auth/passkey/login/finish", critical(http.HandlerFunc(passkeyH.LoginFinish)))
	mgmt.HandleFunc("GET /announcement", authH.Announcement)
	// Session required.
	mgmt.Handle("GET /auth/me", sess.Middleware(http.HandlerFunc(authH.Me)))
	mgmt.Handle("GET /subscriptions/me", sess.Middleware(http.HandlerFunc(subH.Mine)))
	mgmt.Handle("GET /tokens", sess.Middleware(http.HandlerFunc(tokenH.List)))
	mgmt.Handle("POST /tokens", sess.Middleware(http.HandlerFunc(tokenH.Create)))
	mgmt.Handle("PUT /tokens/{id}", sess.Middleware(http.HandlerFunc(tokenH.Update)))
	mgmt.Handle("DELETE /tokens/{id}", sess.Middleware(http.HandlerFunc(tokenH.Delete)))
	mgmt.Handle("GET /logs", sess.Middleware(http.HandlerFunc(logH.Mine)))
	mgmt.Handle("POST /auth/passkey/register/begin", sess.Middleware(http.HandlerFunc(passkeyH.RegisterBegin)))
	mgmt.Handle("POST /auth/passkey/register/finish", sess.Middleware(http.HandlerFunc(passkeyH.RegisterFinish)))
	mgmt.Handle("GET /auth/passkeys", sess.Middleware(http.HandlerFunc(passkeyH.List)))
	mgmt.Handle("DELETE /auth/passkeys/{id}", sess.Middleware(http.HandlerFunc(passkeyH.Delete)))
	mux.Handle("/api/", middleware.RateLimitWeb()(http.StripPrefix("/api", mgmt)))

	// Admin-only management plane (/api/admin/*).
	admin := http.NewServeMux()
	admin.HandleFunc("GET /users", adminH.ListUsers)
	admin.HandleFunc("DELETE /users/{id}", adminH.Delete)
	admin.HandleFunc("PUT /users/{id}/role", adminH.UpdateRole)
	admin.HandleFunc("PUT /users/{id}/status", adminH.UpdateStatus)
	admin.HandleFunc("PUT /users/{id}/group", adminH.UpdateGroup)
	admin.HandleFunc("PUT /users/{id}/quota", adminH.UpdateQuota)
	admin.HandleFunc("PUT /users/{id}/password", adminH.ResetPassword)
	admin.HandleFunc("GET /settings", settingsH.Get)
	admin.HandleFunc("PUT /settings", settingsH.Update)
	admin.HandleFunc("GET /groups", groupH.List)
	admin.HandleFunc("POST /groups", groupH.Create)
	admin.HandleFunc("GET /groups/{id}", groupH.Read)
	admin.HandleFunc("PUT /groups/{id}", groupH.Update)
	admin.HandleFunc("DELETE /groups/{id}", groupH.Delete)
	admin.HandleFunc("GET /channels", channelH.List)
	admin.HandleFunc("POST /channels", channelH.Create)
	admin.HandleFunc("GET /channels/groups", channelH.Groups)
	admin.HandleFunc("GET /channels/{id}", channelH.Read)
	admin.HandleFunc("PUT /channels/{id}", channelH.Update)
	admin.HandleFunc("DELETE /channels/{id}", channelH.Delete)
	admin.HandleFunc("GET /plans", planH.List)
	admin.HandleFunc("POST /plans", planH.Create)
	admin.HandleFunc("PUT /plans/{id}", planH.Update)
	admin.HandleFunc("DELETE /plans/{id}", planH.Delete)
	admin.HandleFunc("GET /subscriptions", subH.List)
	admin.HandleFunc("POST /subscriptions", subH.Grant)
	admin.HandleFunc("DELETE /subscriptions/{id}", subH.Cancel)
	admin.HandleFunc("GET /logs", logH.All)
	admin.HandleFunc("GET /models", metaH.List)
	admin.HandleFunc("PUT /models/{name}", metaH.Upsert)
	admin.HandleFunc("DELETE /models/{name}", metaH.Delete)
	admin.HandleFunc("GET /system", statusH.System)
	mux.Handle("/api/admin/", http.StripPrefix("/api/admin",
		sess.Middleware(middleware.RequireAdmin(admin))))

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
