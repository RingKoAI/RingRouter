// Package handler: ssrf.go guards outbound channel endpoints against
// server-side request forgery. Operators may keep internal-relay setups
// (e.g. pointing at 172.17.0.1 on the docker host, a common one-api pattern),
// so private addresses stay allowed by default; setting
// CHANNEL_ALLOW_PRIVATE_ADDR=false rejects them. Address classification and
// the runtime policy live in internal/safenet so the provider layer can
// enforce the same policy on every outbound dial (including redirects).
package handler

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/safenet"
)

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
	if u.Hostname() == "" {
		return fmt.Errorf("host is required")
	}
	if err := safenet.ValidateOutboundHost(u.Hostname()); err != nil {
		return fmt.Errorf("private or loopback addresses are not allowed for channels")
	}
	return nil
}
