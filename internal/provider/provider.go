// Package provider adapts the unified request/response types to upstream
// LLM vendors. Each adapter converts dto.ChatRequest into the vendor wire
// format and converts the vendor response back into dto.ChatResponse.
package provider

import (
	"context"
	"io"

	"github.com/RingKoAI/RingRouter/internal/dto"
)

// StreamResult carries a streaming upstream response. Body is the untouched
// upstream stream (SSE for streaming, full JSON for buffered); it can be
// forwarded directly to clients whose inbound format matches the upstream.
type StreamResult struct {
	Body        io.ReadCloser
	ContentType string
	// Buffered is true when the body is a complete (non-stream) response.
	Buffered bool
}

// Provider is one upstream vendor adapter.
type Provider interface {
	// Name returns the adapter identifier (e.g. "openai", "anthropic").
	Name() string

	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req *dto.ChatRequest) (*StreamResult, error)

	// Models lists models available on this upstream.
	Models(ctx context.Context) ([]dto.Model, error)
}
