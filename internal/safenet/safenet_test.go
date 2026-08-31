package safenet

import (
	"net"
	"net/http"
	"sync"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true}, // cloud metadata
		{"0.0.0.0", true},
		{"fd00::1", true},
		{"fc00::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2606:4700::1111", false},
	}
	for _, c := range cases {
		if got := IsPrivateIP(net.ParseIP(c.ip)); got != c.want {
			t.Errorf("IsPrivateIP(%s) = %v, want %v", c.ip, got, c.want)
		}
	}
	if !IsPrivateIP(nil) {
		t.Error("nil IP must be treated as private (fail closed)")
	}
}

func TestClientIPUntrustedPeerIgnoresHeaders(t *testing.T) {
	resetProxies(t, "")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "203.0.113.7:44321",
	}
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	r.Header.Set("X-Real-IP", "9.9.9.9")
	if got := ClientIP(r); got != "203.0.113.7" {
		t.Errorf("untrusted peer: ClientIP = %q, want RemoteAddr 203.0.113.7", got)
	}
}

func TestClientIPLoopbackPeerHonorsForwarded(t *testing.T) {
	resetProxies(t, "")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "127.0.0.1:8080",
	}
	r.Header.Set("X-Forwarded-For", "198.51.100.22")
	if got := ClientIP(r); got != "198.51.100.22" {
		t.Errorf("loopback peer: ClientIP = %q, want 198.51.100.22", got)
	}
}

func TestClientIPWalksForwardedChain(t *testing.T) {
	resetProxies(t, "")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "127.0.0.1:8080",
	}
	// Rightmost entry added by the trusted proxy itself is skipped; the next
	// untrusted hop is the real client.
	r.Header.Set("X-Forwarded-For", "198.51.100.22, 127.0.0.1")
	if got := ClientIP(r); got != "198.51.100.22" {
		t.Errorf("chain walk: ClientIP = %q, want 198.51.100.22", got)
	}
}

func TestClientIPGarbageForwardedFallsBack(t *testing.T) {
	resetProxies(t, "")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "127.0.0.1:8080",
	}
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	if got := ClientIP(r); got != "127.0.0.1" {
		t.Errorf("garbage XFF: ClientIP = %q, want RemoteAddr 127.0.0.1", got)
	}
}

func TestClientIPTrustAllUsesLeftmost(t *testing.T) {
	resetProxies(t, "*")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "203.0.113.7:44321",
	}
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if got := ClientIP(r); got != "1.2.3.4" {
		t.Errorf("trust-all: ClientIP = %q, want leftmost 1.2.3.4", got)
	}
}

func TestClientIPExplicitProxyCIDR(t *testing.T) {
	resetProxies(t, "172.16.0.0/12")
	r := &http.Request{
		Header:     http.Header{},
		RemoteAddr: "172.20.0.5:33123",
	}
	r.Header.Set("X-Forwarded-For", "198.51.100.99")
	if got := ClientIP(r); got != "198.51.100.99" {
		t.Errorf("cidr trust: ClientIP = %q, want 198.51.100.99", got)
	}

	// A peer outside the configured range must not have its headers honored.
	r.RemoteAddr = "203.0.113.8:9999"
	if got := ClientIP(r); got != "203.0.113.8" {
		t.Errorf("outside cidr: ClientIP = %q, want 203.0.113.8", got)
	}
}

/* ── helpers ─────────────────────────────────────────────────────────────── */

// resetProxies re-initializes the lazy proxy-trust state for an explicit
// TRUSTED_PROXIES value (empty string = default loopback-only).
func resetProxies(t *testing.T, env string) {
	t.Helper()
	t.Setenv("TRUSTED_PROXIES", env)
	proxyOnce = sync.Once{}
	trusted = nil
	trustAll = false
	trustNone = false
	hasTrusted = false
}
