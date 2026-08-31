// Package safenet centralizes network-level trust decisions shared across
// layers: private-address classification (SSRF hardening) and trusted-proxy
// resolution for client-IP extraction. Both behaviors are opt-in/opt-out via
// environment so operators can match their deployment topology, while the
// defaults fail secure.
package safenet

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

/* ── Private-address classification (SSRF) ──────────────────────────────── */

// IsPrivateIP covers loopback, RFC1918, link-local (incl. cloud metadata),
// unique-local, and unspecified addresses. nil is treated as private so a
// failed resolution can never be talked into looking public.
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if v6 := ip.To16(); v6 != nil && ip.To4() == nil {
		// IPv6 unique-local fc00::/7 (IsPrivate covers it on modern Go, but
		// keep the prefix check for older toolchains).
		return v6.IsPrivate() || strings.HasPrefix(v6.String(), "fc") || strings.HasPrefix(v6.String(), "fd")
	}
	return false
}

// HostResolvesPrivate reports whether host resolves to at least one private
// address (or fails to resolve at all, treated as private for safety).
func HostResolvesPrivate(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return true
	}
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return true
		}
	}
	return false
}

/* ── Channel private-address policy ─────────────────────────────────────── */

// PrivateAddrDenied reports whether the deployment forbids outbound requests
// to private addresses (CHANNEL_ALLOW_PRIVATE_ADDR=false). Defaults to
// allowed: operators commonly point channels at an internal relay. The env
// is read per call (a cheap map lookup) so tests and runtime changes apply
// immediately.
func PrivateAddrDenied() bool {
	return strings.TrimSpace(os.Getenv("CHANNEL_ALLOW_PRIVATE_ADDR")) == "false"
}

// ValidateOutboundHost enforces the private-address policy for a host that is
// about to receive an outbound request. It performs a fresh DNS resolution so
// rebinding between write-time validation and request time is also caught by
// the per-request callers.
func ValidateOutboundHost(host string) error {
	if host == "" {
		return errRejected("host is required")
	}
	if !PrivateAddrDenied() {
		return nil
	}
	if HostResolvesPrivate(host) {
		return errRejected("private or loopback addresses are not allowed")
	}
	return nil
}

type policyError string

func (e policyError) Error() string { return string(e) }

func errRejected(msg string) error { return policyError(msg) }

// IsPolicyError reports whether err came from ValidateOutboundHost, letting
// callers surface a clean message instead of leaking resolver internals.
func IsPolicyError(err error) bool {
	_, ok := err.(policyError)
	return ok
}

/* ── Trusted proxies (client-IP resolution) ─────────────────────────────── */

var (
	proxyOnce  sync.Once
	trusted    []*net.IPNet
	trustAll   bool
	trustNone  bool
	hasTrusted bool
)

// loopbackCIDRs are always trusted: a reverse proxy on the same host is the
// most common topology and cannot be spoofed from outside.
var loopbackCIDRs = []string{"127.0.0.0/8", "::1/128"}

func loadProxies() {
	proxyOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
		switch {
		case strings.EqualFold(raw, "*"):
			trustAll = true
		case strings.EqualFold(raw, "none"):
			trustNone = true
		case raw == "":
			// Default: loopback only. Private ranges are deliberately NOT
			// trusted by default — behind NAT-style publishing (Docker port
			// maps) the apparent source is a gateway address while the
			// forwarded headers remain fully attacker-controlled.
			raw = strings.Join(loopbackCIDRs, ",")
			fallthrough
		default:
			for _, item := range strings.Split(raw, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				if !strings.Contains(item, "/") {
					if ip := net.ParseIP(item); ip != nil {
						if ip.To4() != nil {
							item += "/32"
						} else {
							item += "/128"
						}
					} else {
						continue
					}
				}
				if _, n, err := net.ParseCIDR(item); err == nil {
					trusted = append(trusted, n)
				}
			}
			hasTrusted = len(trusted) > 0
		}
	})
}

func isTrustedAddr(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP resolves the real client address for a request.
//
// Trust model: forwarded headers (X-Forwarded-For / X-Real-IP) are honored
// only when the direct TCP peer is a trusted proxy (loopback by default, or
// the CIDRs listed in TRUSTED_PROXIES). The XFF list is then walked from the
// right, skipping trusted hops; the first untrusted entry is the client.
// When the peer is untrusted, headers are ignored and RemoteAddr wins —
// closing the spoofing hole that let callers rotate fake XFF values to
// bypass rate limits and subnet allowlists.
func ClientIP(r *http.Request) string {
	loadProxies()

	remote := remoteAddrIP(r.RemoteAddr)
	if trustNone || (!trustAll && !isTrustedAddr(remote)) {
		return remote
	}

	// Peer is a trusted proxy: honor forwarded headers.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Walk right-to-left, skipping trusted proxies. Under trust-all every
		// hop is presumed a proxy, so the walk falls through to the leftmost
		// (originating) entry.
		for i := len(parts) - 1; i >= 0; i-- {
			cand := strings.TrimSpace(parts[i])
			if cand == "" || net.ParseIP(cand) == nil {
				continue
			}
			if trustAll || isTrustedAddr(cand) {
				continue
			}
			return cand
		}
		// Every hop was trusted (e.g. proxy chains on loopback): the leftmost
		// entry is then the originating client.
		if first := strings.TrimSpace(parts[0]); first != "" && net.ParseIP(first) != nil {
			return first
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" && net.ParseIP(xri) != nil {
		return xri
	}
	return remote
}

func remoteAddrIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return host
}
