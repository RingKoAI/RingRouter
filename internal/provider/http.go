package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/safenet"
)

// httpProvider is the shared HTTP plumbing for all vendor adapters.
type httpProvider struct {
	name       string
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func newHTTPProvider(name, apiKey, baseURL string) httpProvider {
	return httpProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: newUpstreamClient(),
	}
}

// maxRedirects bounds how far an upstream may bounce a request.
const maxRedirects = 3

const maxProviderResponseBytes = 16 << 20

// newUpstreamClient builds the shared outbound client with a redirect policy
// that re-applies the SSRF private-address policy on every hop, so a public
// channel URL cannot 302 the gateway into an internal endpoint when
// CHANNEL_ALLOW_PRIVATE_ADDR=false.
func newUpstreamClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = safeDialContext
	return &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%s: too many redirects", "upstream")
			}
			if err := safenet.ValidateOutboundHost(req.URL.Hostname()); err != nil {
				return fmt.Errorf("upstream redirect to a private address is not allowed")
			}
			return nil
		},
	}
}

// safeDialContext resolves and validates every address immediately before the
// socket connection. This closes the DNS-rebinding gap between URL validation
// and net/http's later resolver call. TLS still receives the original host
// name, while the TCP socket is opened against the validated IP.
func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	if !safenet.PrivateAddrDenied() {
		return dialer.DialContext(ctx, network, address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream address")
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("upstream host resolution failed")
	}
	var lastErr error
	for _, ip := range ips {
		if safenet.IsPrivateIP(ip) {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("all resolved upstream addresses are private")
}

// do sends a request and returns the raw response body. The caller owns
// closing resp.Body; on non-2xx it returns an error carrying the status.
func (p httpProvider) do(ctx context.Context, method, url string, body []byte, hdr map[string]string) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", p.name, err)
	}
	if hdr != nil {
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", p.name, err)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		return nil, &UpstreamError{Provider: p.name, Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
	}
	return resp, nil
}

// UpstreamError reports a non-2xx upstream response.
type UpstreamError struct {
	Provider string
	Status   int
	Body     string
}

// errorBodySanitizer strips control characters that upstream bodies may carry;
// the message flows into client responses and log rows as JSON, so keeping it
// single-line avoids log forging and message injection.
var errorBodySanitizer = strings.NewReplacer("\r", " ", "\n", " ")

func (e *UpstreamError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s upstream returned %d", e.Provider, e.Status)
	}
	body := e.Body
	if len(body) > 512 {
		body = body[:512]
	}
	return fmt.Sprintf("%s upstream returned %d: %s", e.Provider, e.Status, errorBodySanitizer.Replace(body))
}

// bufferedResult serializes a unified response for clients that requested a
// stream from a provider that cannot stream through the gateway yet.
func bufferedResult(name string, resp *dto.ChatResponse) (*StreamResult, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal buffered response: %w", name, err)
	}
	return &StreamResult{
		Body:        io.NopCloser(bytes.NewReader(b)),
		ContentType: "application/json",
		Buffered:    true,
	}, nil
}
