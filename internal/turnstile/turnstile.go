package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Result holds the parsed response from the Cloudflare Turnstile siteverify API.
type Result struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

var (
	mu        sync.RWMutex
	secret    string
	sitekey   string
	enabled   bool
)

// Init loads the Turnstile configuration from environment variables.
// TURNSTILE_SECRET: the secret key for server-side verification.
// TURNSTILE_SITEKEY: the public sitekey returned to the frontend.
func Init() {
	s := os.Getenv("TURNSTILE_SECRET")
	sk := os.Getenv("TURNSTILE_SITEKEY")

	mu.Lock()
	defer mu.Unlock()

	secret = strings.TrimSpace(s)
	sitekey = strings.TrimSpace(sk)
	enabled = secret != "" && sitekey != ""
}

// Enabled reports whether Turnstile is configured.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// Sitekey returns the public sitekey for the frontend widget.
func Sitekey() string {
	mu.RLock()
	defer mu.RUnlock()
	return sitekey
}

// Verify sends the token to Cloudflare's siteverify endpoint and returns the
// parsed result. remoteIP is optional (may be empty). It performs a single
// attempt with a 10-second timeout.
func Verify(ctx context.Context, token, remoteIP string) (*Result, error) {
	mu.RLock()
	s := secret
	mu.RUnlock()

	if s == "" {
		return nil, fmt.Errorf("turnstile secret not configured")
	}

	form := url.Values{
		"secret":   {s},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, siteverifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("siteverify network: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("siteverify read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("siteverify HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("siteverify parse: %w", err)
	}
	return &result, nil
}

// ValidateToken is a convenience wrapper: verifies a token and returns an
// error if the verification failed (success=false, network error, etc.).
func ValidateToken(ctx context.Context, token, remoteIP string) error {
	if token == "" {
		return fmt.Errorf("turnstile token is empty")
	}
	result, err := Verify(ctx, token, remoteIP)
	if err != nil {
		return err
	}
	if !result.Success {
		codes := strings.Join(result.ErrorCodes, ", ")
		return fmt.Errorf("turnstile verification failed: %s", codes)
	}
	return nil
}
