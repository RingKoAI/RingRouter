package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/RingKoAI/RingRouter/internal/provider"
)

// Proxy handles LLM API proxy requests.
type Proxy struct {
	openai provider.Provider
}

// NewProxy creates a new Proxy handler.
func NewProxy(openaiKey, openaiBaseURL string) *Proxy {
	return &Proxy{
		openai: provider.NewOpenAI(openaiKey, openaiBaseURL),
	}
}

// Health returns a simple health check response.
func (p *Proxy) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

// ChatCompletion proxies chat completion requests to the upstream provider.
func (p *Proxy) ChatCompletion(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Validate model
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err == nil {
		if req.Model == "" {
			writeError(w, http.StatusBadRequest, "model is required")
			return
		}
	}

	// Select provider (for MVP, use OpenAI)
	// TODO: channel selection based on model name and priority
	prov := p.openai

	// Collect headers to forward (except auth)
	fwdHeaders := make(map[string]string)
	for k, v := range r.Header {
		kl := strings.ToLower(k)
		if kl == "authorization" || kl == "content-type" || kl == "content-length" || kl == "host" {
			continue
		}
		if len(v) > 0 {
			fwdHeaders[k] = v[0]
		}
	}

	resp, err := prov.ChatCompletion(r.Context(), strings.NewReader(string(body)), fwdHeaders)
	if err != nil {
		log.Printf("[ringrouter] proxy error: %v", err)
		writeError(w, http.StatusBadGateway, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream or copy response body
	if resp.IsStream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "streaming not supported")
			return
		}
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return
				}
				flusher.Flush()
			}
			if readErr != nil {
				break
			}
		}
	} else {
		io.Copy(w, resp.Body)
	}
}

// ListModels proxies model list requests to the upstream provider.
func (p *Proxy) ListModels(w http.ResponseWriter, r *http.Request) {
	resp, err := p.openai.ListModels(r.Context(), nil)
	if err != nil {
		log.Printf("[ringrouter] list models error: %v", err)
		writeError(w, http.StatusBadGateway, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    "ringrouter_error",
		},
	})
}