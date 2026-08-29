// Package handler: ssrf.go guards outbound channel endpoints against
// server-side request forgery. Operators may keep internal-relay setups
// (e.g. pointing at 172.17.0.1 on the docker host, a common one-api pattern),
// so private addresses stay allowed by default; setting
// CHANNEL_ALLOW_PRIVATE_ADDR=false rejects them.
package handler

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// privateAddrDenied reports whether the deployment forbids private addresses.
func privateAddrDenied() bool {
	return strings.TrimSpace(os.Getenv("CHANNEL_ALLOW_PRIVATE_ADDR")) == "false"
}

// isPrivateIP covers loopback, RFC1918, link-local (incl. cloud metadata),
// unique-local, and unspecified addresses.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() ||
		(ip.To16() != nil && ip.To4() == nil && (ip.IsLinkLocalUnicast() || strings.HasPrefix(ip.String(), "fc") || strings.HasPrefix(ip.String(), "fd")))
}

// validateChannelURL enforces scheme + host sanity for a channel base URL.
// When private addresses are denied the host must also resolve publicly.
func validateChannelURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("host is required")
	}
	if !privateAddrDenied() {
		return nil
	}
	// Resolve and reject any private address (SSRF hardening mode).
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q", host)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("private or loopback addresses are not allowed for channels")
		}
	}
	return nil
}
