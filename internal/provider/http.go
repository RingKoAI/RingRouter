package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/dto"
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
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
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
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
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

func (e *UpstreamError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("%s upstream returned %d", e.Provider, e.Status)
	}
	return fmt.Sprintf("%s upstream returned %d: %s", e.Provider, e.Status, e.Body)
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
