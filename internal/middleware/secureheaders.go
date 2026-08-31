// Package middleware: secureheaders.go applies baseline HTTP security headers
// to every response. The SPA is a static, self-contained bundle, so the CSP
// can be locked to same-origin scripts/styles with no inline script access.
package middleware

import (
	"net/http"
	"strings"
)

// contentSecurityPolicy locks the embedded SPA to same-origin resources.
// Inline style attributes (used by a few components for dynamic sizing) are
// permitted via style-src; scripts are strictly same-origin.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; " +
	"connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; " +
	"object-src 'none'"

// SecureHeaders sets defensive response headers on every outbound response:
//
//	X-Content-Type-Options: nosniff      — no MIME sniffing
//	X-Frame-Options: DENY                — clickjacking (plus frame-ancestors)
//	Referrer-Policy                     — strip referrer cross-origin
//	Content-Security-Policy             — lock the SPA to same-origin
//	Strict-Transport-Security           — only on TLS / forwarded-HTTPS
//	Cross-Origin-Opener-Policy: same-origin — isolation from other origins
//	Cross-Origin-Resource-Policy: same-origin — no cross-origin resource reads
//
// API JSON responses are unaffected functionally; the headers are blanket-set
// because they cost nothing and protect the HTML surface uniformly.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		if h.Get("Content-Security-Policy") == "" {
			h.Set("Content-Security-Policy", contentSecurityPolicy)
		}
		// HSTS only when the connection is actually encrypted — either
		// terminated here (r.TLS) or by a trusted upstream proxy that told us
		// the scheme. An X-Forwarded-Proto spoof can only *add* the Secure
		// requirement, which fails safe.
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
