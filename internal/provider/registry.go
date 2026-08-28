package provider

import (
	"fmt"

	"github.com/RingKoAI/RingRouter/internal/model"
)

// decryptFunc is wired by main.go so the registry can decode AES-GCM
// sealed channel API keys before constructing a Provider.
var decryptFunc func(string) (string, error)

// SetDecryptor wires the function used to unseal channel API keys.
func SetDecryptor(f func(string) (string, error)) {
	decryptFunc = f
}

// unseal returns the plain API key, falling back to the raw value when no
// decryptor is configured (development with a plaintext database).
func unseal(sealed string) string {
	if decryptFunc == nil || sealed == "" {
		return sealed
	}
	if plain, err := decryptFunc(sealed); err == nil {
		return plain
	}
	// Wrong key / corrupt ciphertext: surface the original so the request
	// fails with an obvious upstream auth error rather than silently empty.
	return sealed
}

// NewFromChannel creates a Provider for the given channel configuration.
// Each channel.Protocol maps to a dedicated adapter; all adapters share a
// unified Provider interface so the gateway is protocol-agnostic.
func NewFromChannel(ch *model.Channel) (Provider, error) {
	if ch == nil {
		return nil, fmt.Errorf("channel is nil")
	}
	if ch.BaseURL == "" {
		return nil, fmt.Errorf("channel %q has empty base_url", ch.Name)
	}

	key := unseal(ch.APIKey)
	switch ch.Protocol {
	case "anthropic":
		return NewAnthropic(key, ch.BaseURL), nil
	case "google":
		return NewGoogle(key, ch.BaseURL), nil
	case "openai-responses":
		return NewOpenAIResponses(key, ch.BaseURL), nil
	case "openai", "":
		return NewOpenAI(key, ch.BaseURL), nil
	default:
		// Unknown protocols: assume OpenAI-compatible (most vendors are).
		return NewOpenAI(key, ch.BaseURL), nil
	}
}
