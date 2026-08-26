package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/gateway"
	"github.com/RingKoAI/RingRouter/internal/inbound"
	"github.com/RingKoAI/RingRouter/internal/provider"
)

const maxBodyBytes = 16 << 20 // 16 MiB request cap

// Proxy is the gateway HTTP handler. It accepts any inbound wire format,
// routes through the multi-channel gateway, and replies in the inbound format.
type Proxy struct {
	gw        *gateway.Gateway
	envOpenAI provider.Provider
}

// NewProxy creates the gateway handler. envOpenAI is the fallback provider
// configured from environment variables, used when no DB channel matches.
func NewProxy(gw *gateway.Gateway, envOpenAI provider.Provider) *Proxy {
	return &Proxy{gw: gw, envOpenAI: envOpenAI}
}

// Health reports service liveness.
func (p *Proxy) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

// ChatCompletion handles chat endpoints in any inbound format:
//
//	OpenAI:     POST /v1/chat/completions
//	Anthropic:  POST /v1/messages
//	Google:     POST /v1beta/models/{model}:generateContent
func (p *Proxy) ChatCompletion(w http.ResponseWriter, r *http.Request) {
	format := inbound.DetectFormat(r.URL.Path)
	codec, err := inbound.For(format)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", "failed to read body")
		return
	}
	req, err := codec.Decode(body)
	if err != nil {
		p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	// Gemini carries the model in the URL path, not the body.
	if format == inbound.FormatGoogle {
		req.Model = extractGeminiModel(r.URL.Path)
		if req.Model == "" {
			p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", "model is required in path")
			return
		}
	}

	if req.Stream {
		p.handleStream(w, r, codec, req)
		return
	}

	resp, err := p.routeChat(r, req)
	if err != nil {
		log.Printf("[proxy] route error: %v", err)
		p.writeErr(w, codec, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	resp.Model = req.Model
	resp.Created = time.Now().Unix()

	out, err := codec.EncodeChat(resp)
	if err != nil {
		p.writeErr(w, codec, http.StatusInternalServerError, "internal_error", "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", codec.ContentType())
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

// routeChat tries DB channels first, then falls back to the env provider.
func (p *Proxy) routeChat(r *http.Request, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	resp, _, err := p.gw.Chat(r.Context(), req)
	if err == nil {
		return resp, nil
	}
	if p.envOpenAI != nil {
		if resp, err2 := p.envOpenAI.Chat(r.Context(), req); err2 == nil {
			log.Printf("[proxy] served by env-default provider (no channel matched: %v)", err)
			return resp, nil
		} else {
			log.Printf("[proxy] env fallback failed: %v", err2)
		}
	}
	return nil, err
}

// handleStream serves streaming requests.
//
// Fast path: an OpenAI adapter streaming to an OpenAI client forwards the
// upstream SSE untouched. Cross-format streams are consumed fully server-side
// and re-emitted as a single data frame in the inbound format.
func (p *Proxy) handleStream(w http.ResponseWriter, r *http.Request, codec inbound.Codec, req *dto.ChatRequest) {
	res, _, err := p.gw.ChatStream(r.Context(), req)
	if err != nil && p.envOpenAI != nil {
		if res2, err2 := p.envOpenAI.ChatStream(r.Context(), req); err2 == nil {
			res, err = res2, nil
		}
	}
	if err != nil || res == nil {
		if err == nil {
			err = fmt.Errorf("no stream result")
		}
		p.writeErr(w, codec, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer res.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		p.writeErr(w, codec, http.StatusInternalServerError, "internal_error", "streaming not supported")
		return
	}

	// Fast path: raw SSE passthrough (OpenAI upstream ↔ OpenAI client).
	if !res.Buffered && codec.Format() == inbound.FormatOpenAI {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		buf := make([]byte, 32*1024)
		for {
			n, rerr := res.Body.Read(buf)
			if n > 0 {
				if _, werr := w.Write(buf[:n]); werr != nil {
					return
				}
				flusher.Flush()
			}
			if rerr != nil {
				return
			}
		}
	}

	// Cross-format path: consume the upstream fully, then re-emit.
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes))
	if err != nil {
		return
	}
	var unified *dto.ChatResponse
	if res.Buffered {
		var u dto.ChatResponse
		if json.Unmarshal(raw, &u) == nil {
			unified = &u
		}
	} else {
		unified = accumulateOpenAISSE(raw)
	}
	if unified == nil {
		return
	}
	unified.Model = req.Model
	unified.Created = time.Now().Unix()

	out, err := codec.EncodeChat(unified)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "data: %s\n\n", out)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// ListModels returns the aggregated model list across channels.
func (p *Proxy) ListModels(w http.ResponseWriter, r *http.Request) {
	models := p.gw.Models(r.Context())
	if len(models) == 0 && p.envOpenAI != nil {
		if m, err := p.envOpenAI.Models(r.Context()); err == nil {
			models = m
		}
	}
	writeJSON(w, http.StatusOK, dto.ModelList{Object: "list", Data: models})
}

func (p *Proxy) writeErr(w http.ResponseWriter, codec inbound.Codec, status int, errType, msg string) {
	w.Header().Set("Content-Type", codec.ContentType())
	w.WriteHeader(status)
	w.Write(codec.EncodeError(status, errType, msg))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// extractGeminiModel pulls the model out of /v1beta/models/{model}:method.
func extractGeminiModel(path string) string {
	const marker = "/models/"
	i := strings.Index(path, marker)
	if i < 0 {
		return ""
	}
	rest := path[i+len(marker):]
	if j := strings.Index(rest, ":"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// accumulateOpenAISSE folds an OpenAI SSE stream into one unified response.
func accumulateOpenAISSE(raw []byte) *dto.ChatResponse {
	var (
		text   strings.Builder
		usage  dto.Usage
		finish string
		id     string
	)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk dto.ChatResponse
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}
		if chunk.ID != "" {
			id = chunk.ID
		}
		if len(chunk.Choices) > 0 {
			text.WriteString(chunk.Choices[0].Message.Content)
			if chunk.Choices[0].FinishReason != "" {
				finish = chunk.Choices[0].FinishReason
			}
		}
		if chunk.Usage.TotalTokens > 0 {
			usage = chunk.Usage
		}
	}
	if finish == "" {
		finish = "stop"
	}
	return &dto.ChatResponse{
		ID:     id,
		Object: "chat.completion",
		Choices: []dto.Choice{{
			Index:        0,
			Message:      dto.Message{Role: "assistant", Content: text.String()},
			FinishReason: finish,
		}},
		Usage: usage,
	}
}