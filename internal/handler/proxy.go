package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/gateway"
	"github.com/RingKoAI/RingRouter/internal/inbound"
	"github.com/RingKoAI/RingRouter/internal/middleware"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/provider"
	"github.com/RingKoAI/RingRouter/internal/setting"
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

	// Gemini carries the model in the URL path, not the body. The name is
	// validated segment-by-segment before it is used for upstream URL
	// construction, so path traversal / query injection dies at the door.
	if format == inbound.FormatGoogle {
		req.Model = extractGeminiModel(r.URL.Path)
		if req.Model == "" || !validModelPath(req.Model) {
			p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", "model is required in path")
			return
		}
	}

	tok := middleware.GetToken(r.Context())
	if !QuotaAvailable(middleware.GetUser(r.Context()), tok) {
		p.writeErr(w, codec, http.StatusTooManyRequests, "insufficient_quota", "account or API key quota exhausted")
		return
	}

	// Token model whitelist (one-api semantics): empty = unrestricted.
	if tok != nil && tok.Models != "" && !tokenAllowsModel(tok.Models, req.Model) {
		p.writeErr(w, codec, http.StatusForbidden, "permission_error", "this API key is not allowed to use model "+req.Model)
		return
	}

	// Operator-configured text blocklist.
	if hit := hitsSensitiveWord(req); hit != "" {
		p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", "request rejected by content policy: "+hit)
		return
	}

	if req.Stream {
		p.handleStream(w, r, codec, req)
		return
	}

	start := time.Now()
	resp, ch, err := p.routeChat(r, req)
	if err != nil {
		log.Printf("[proxy] route error: %v", err)
		p.recordLog(r, req.Model, ch, "failed", err.Error(), time.Since(start))
		p.writeErr(w, codec, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	resp.Model = req.Model
	resp.Created = time.Now().Unix()
	p.recordUsage(r, req.Model, ch, resp, time.Since(start))

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
func (p *Proxy) routeChat(r *http.Request, req *dto.ChatRequest) (*dto.ChatResponse, *model.Channel, error) {
	resp, ch, err := p.gw.Chat(r.Context(), req, p.routeGroup(r))
	if err == nil {
		return resp, ch, nil
	}
	if p.envOpenAI != nil {
		if resp, err2 := p.envOpenAI.Chat(r.Context(), req); err2 == nil {
			log.Printf("[proxy] served by env-default provider (no channel matched: %v)", err)
			return resp, nil, nil
		} else {
			log.Printf("[proxy] env fallback failed: %v", err2)
		}
	}
	return nil, nil, err
}

// recordUsage bills the request and writes a success log entry (async).
// Cost is computed from token usage × model price × group ratio and deducted
// from both the token and the user (unlimited accounts skip deduction).
func (p *Proxy) recordUsage(r *http.Request, modelName string, ch *model.Channel, resp *dto.ChatResponse, elapsed time.Duration) {
	var channelID uint
	if ch != nil {
		channelID = ch.ID
	}
	var prompt, completion int
	if resp != nil {
		prompt = resp.Usage.PromptTokens
		completion = resp.Usage.CompletionTokens
	}
	cost := ComputeCostQuota(modelName, prompt, completion, p.routeGroup(r))
	if cost > 0 {
		DeductQuota(middleware.GetUser(r.Context()), middleware.GetToken(r.Context()), cost, p.routeGroup(r))
	}
	p.recordLogFull(r, modelName, channelID, prompt, completion, "success", "", elapsed, cost)
}

// recordLog writes a failure log entry (async).
func (p *Proxy) recordLog(r *http.Request, modelName string, ch *model.Channel, status, errMsg string, elapsed time.Duration) {
	var channelID uint
	if ch != nil {
		channelID = ch.ID
	}
	p.recordLogFull(r, modelName, channelID, 0, 0, status, errMsg, elapsed, 0)
}

// recordLogFull persists a request log asynchronously; a logging failure must
// never fail the proxy response.
func (p *Proxy) recordLogFull(r *http.Request, modelName string, channelID uint, prompt, completion int, status, errMsg string, elapsed time.Duration, cost int64) {
	if database.DB == nil {
		return
	}
	u := middleware.GetUser(r.Context())
	tok := middleware.GetToken(r.Context())
	entry := model.Log{
		ModelName:    modelName,
		ChannelID:    channelID,
		PromptTokens: prompt,
		CompTokens:   completion,
		Quota:        cost,
		Status:       status,
		ErrorMsg:     truncate(errMsg, 512),
		IP:           extractIP(r),
		CreatedAt:    time.Now(),
	}
	if u != nil {
		entry.UserID = u.ID
	}
	if tok != nil {
		entry.TokenID = tok.ID
	}
	entry.ElapsedMs = elapsed.Milliseconds()
	go func() {
		if err := database.DB.Create(&entry).Error; err != nil {
			log.Printf("[proxy] log write failed: %v", err)
		}
	}()
}

// billStream extracts usage from a forwarded OpenAI SSE stream and bills it.
func (p *Proxy) billStream(r *http.Request, modelName string, ch *model.Channel, sniff []byte, start time.Time) {
	if u := accumulateOpenAISSE(sniff); u != nil {
		p.recordUsage(r, modelName, ch, u, time.Since(start))
		return
	}
	var chID uint
	if ch != nil {
		chID = ch.ID
	}
	p.recordLogFull(r, modelName, chID, 0, 0, "success", "", time.Since(start), 0)
}

// tokenAllowsModel matches against the comma-separated whitelist (trimmed).
func tokenAllowsModel(list, model string) bool {
	for _, m := range strings.Split(list, ",") {
		if strings.TrimSpace(m) == model {
			return true
		}
	}
	return false
}

// hitsSensitiveWord scans the request text (prompt + system) against the
// operator blocklist. Returns the first hit, or "" when clean/disabled.
func hitsSensitiveWord(req *dto.ChatRequest) string {
	words := setting.SensitiveWords()
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(req.Model)
	for _, m := range req.Messages {
		b.WriteByte('\n')
		b.WriteString(m.Content)
	}
	text := strings.ToLower(b.String())
	for _, w := range words {
		if strings.Contains(text, w) {
			return w
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// handleStream serves streaming requests.
//
// Fast path: an OpenAI adapter streaming to an OpenAI client forwards the
// upstream SSE untouched. Cross-format streams are consumed fully server-side
// and re-emitted as a single data frame in the inbound format.
func (p *Proxy) handleStream(w http.ResponseWriter, r *http.Request, codec inbound.Codec, req *dto.ChatRequest) {
	tok := middleware.GetToken(r.Context())
	if !QuotaAvailable(middleware.GetUser(r.Context()), tok) {
		p.writeErr(w, codec, http.StatusTooManyRequests, "insufficient_quota", "account or API key quota exhausted")
		return
	}
	if tok != nil && tok.Models != "" && !tokenAllowsModel(tok.Models, req.Model) {
		p.writeErr(w, codec, http.StatusForbidden, "permission_error", "this API key is not allowed to use model "+req.Model)
		return
	}
	if hit := hitsSensitiveWord(req); hit != "" {
		p.writeErr(w, codec, http.StatusBadRequest, "invalid_request_error", "request rejected by content policy: "+hit)
		return
	}
	start := time.Now()
	res, ch, err := p.gw.ChatStream(r.Context(), req, p.routeGroup(r))
	if err != nil && p.envOpenAI != nil {
		if res2, err2 := p.envOpenAI.ChatStream(r.Context(), req); err2 == nil {
			res, err = res2, nil
		}
	}
	if err != nil || res == nil {
		if err == nil {
			err = fmt.Errorf("no stream result")
		}
		var chID uint
		if ch != nil {
			chID = ch.ID
		}
		p.recordLogFull(r, req.Model, chID, 0, 0, "failed", err.Error(), time.Since(start), 0)
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
		var sniff []byte
		for {
			n, rerr := res.Body.Read(buf)
			if n > 0 {
				sniff = append(sniff, buf[:n]...)
				if len(sniff) > maxBodyBytes {
					sniff = sniff[len(sniff)-maxBodyBytes:] // keep the tail (usage rides at the end)
				}
				if _, werr := w.Write(buf[:n]); werr != nil {
					p.billStream(r, req.Model, ch, sniff, start)
					return
				}
				flusher.Flush()
			}
			if rerr != nil {
				p.billStream(r, req.Model, ch, sniff, start)
				return
			}
		}
	}

	// Cross-format path: consume the upstream fully, then re-emit.
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes))
	if err != nil {
		p.recordLogFull(r, req.Model, chIDOf(ch), 0, 0, "failed", "upstream stream read: "+err.Error(), time.Since(start), 0)
		p.writeErr(w, codec, http.StatusBadGateway, "upstream_error", "upstream stream ended prematurely")
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
		p.recordLogFull(r, req.Model, chIDOf(ch), 0, 0, "failed", "upstream stream could not be decoded", time.Since(start), 0)
		p.writeErr(w, codec, http.StatusBadGateway, "upstream_error", "upstream stream could not be decoded")
		return
	}
	unified.Model = req.Model
	unified.Created = time.Now().Unix()
	p.recordUsage(r, req.Model, ch, unified, time.Since(start))

	out, err := codec.EncodeChat(unified)
	if err != nil {
		p.writeErr(w, codec, http.StatusInternalServerError, "internal_error", "failed to encode response")
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
	w.Write(codec.EncodeError(status, errType, sanitizeUpstreamMessage(msg)))
}

// chIDOf returns the channel id (0 when the request was served by the env
// fallback provider).
func chIDOf(ch *model.Channel) uint {
	if ch != nil {
		return ch.ID
	}
	return 0
}

// upsteamMsgSanitizer keeps reflected upstream errors single-line and bounded
// before they reach clients and log rows.
var upsteamMsgSanitizer = strings.NewReplacer("\r", " ", "\n", " ")

// sanitizeUpstreamMessage flattens CR/LF and truncates overly long upstream
// payloads (which may embed vendor HTML) before they are reflected.
func sanitizeUpstreamMessage(msg string) string {
	msg = upsteamMsgSanitizer.Replace(msg)
	if len(msg) > 512 {
		msg = msg[:512]
	}
	return msg
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// routeGroup reads the routing group from the authenticated token in the
// request context. Empty / "default" selects the default channel pool. The
// group is bound to the API key, so clients cannot escalate by setting a
// header.
func (p *Proxy) routeGroup(r *http.Request) string {
	tok := middleware.GetToken(r.Context())
	if tok != nil {
		if g := strings.TrimSpace(tok.Group); g != "" {
			return g
		}
	}
	return "default"
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

// modelPathSegmentRe is the whitelist for each dot-separated/ slash-separated
// segment of a model identifier that flows into upstream URL paths.
var modelPathSegmentRe = regexp.MustCompile(`^[A-Za-z0-9._~-]{1,128}$`)

// validModelPath reports whether a client-supplied model name is safe to
// interpolate into an upstream URL path (no traversal, query, or fragment).
func validModelPath(model string) bool {
	if len(model) > 512 {
		return false
	}
	for _, seg := range strings.Split(model, "/") {
		if seg == "" || seg == "." || seg == ".." || !modelPathSegmentRe.MatchString(seg) {
			return false
		}
	}
	return true
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
