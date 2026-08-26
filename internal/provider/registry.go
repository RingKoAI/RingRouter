package provider

import (
	"fmt"

	"github.com/RingKoAI/RingRouter/internal/model"
)

// NewFromChannel creates a Provider for the given channel configuration.
//
// Channel types map to native adapters when the vendor wire format differs
// from OpenAI; everything else is treated as OpenAI-compatible.
func NewFromChannel(ch *model.Channel) (Provider, error) {
	if ch == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	if ch.BaseURL == "" {
		return nil, fmt.Errorf("channel %q has empty base_url", ch.Name)
	}

	switch ch.Type {
	case "anthropic", "claude":
		return NewAnthropic(ch.APIKey, ch.BaseURL), nil
	case "google", "gemini":
		return NewGoogle(ch.APIKey, ch.BaseURL), nil
	case "openai", "":
		return NewOpenAI(ch.APIKey, ch.BaseURL), nil
	default:
		// Unknown types: assume OpenAI-compatible (most vendors are).
		return NewOpenAI(ch.APIKey, ch.BaseURL), nil
	}
}