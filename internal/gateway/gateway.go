// Package gateway implements multi-channel routing: it selects upstream
// channels that serve the requested model, tries them in priority order with
// failover, and aggregates model listings across channels.
package gateway

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RingKoAI/RingRouter/internal/database"
	"github.com/RingKoAI/RingRouter/internal/dto"
	"github.com/RingKoAI/RingRouter/internal/model"
	"github.com/RingKoAI/RingRouter/internal/provider"
)

// Gateway routes unified requests across channels.
type Gateway struct {
	mu        sync.RWMutex
	channels  []*model.Channel
	providers map[uint]provider.Provider
	loaded    time.Time
}

const channelCacheTTL = 30 * time.Second

// New creates a Gateway.
func New() *Gateway {
	return &Gateway{providers: make(map[uint]provider.Provider)}
}

// reload refreshes the channel snapshot from the database. On failure the
// previous snapshot stays in place (graceful degradation).
func (g *Gateway) reload(ctx context.Context) {
	if database.DB == nil {
		return
	}
	if time.Since(g.loaded) < channelCacheTTL {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if time.Since(g.loaded) < channelCacheTTL { // re-check under lock
		return
	}
	var channels []model.Channel
	if err := database.DB.Where("status = ?", "active").Find(&channels).Error; err != nil {
		log.Printf("[gateway] reload channels failed: %v", err)
		return
	}
	snapshot := make([]*model.Channel, len(channels))
	for i := range channels {
		snapshot[i] = &channels[i]
	}
	// Higher priority first; stable within the same tier.
	sort.SliceStable(snapshot, func(i, j int) bool {
		return snapshot[i].Priority > snapshot[j].Priority
	})
	g.channels = snapshot
	g.loaded = time.Now()
}

// supports reports whether a channel serves the model.
func supports(ch *model.Channel, modelName string) bool {
	for _, m := range strings.Split(ch.Models, ",") {
		if strings.TrimSpace(m) == modelName {
			return true
		}
	}
	return false
}

// Chat routes a unified request with failover across matching channels.
// It returns the response and the channel that served it.
func (g *Gateway) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, *model.Channel, error) {
	g.reload(ctx)

	g.mu.RLock()
	candidates := make([]*model.Channel, 0, len(g.channels))
	for _, ch := range g.channels {
		if supports(ch, req.Model) {
			candidates = append(candidates, ch)
		}
	}
	g.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no active channel serves model %q", req.Model)
	}

	var lastErr error
	for _, ch := range candidates {
		p, err := g.providerFor(ch)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := p.Chat(ctx, req)
		if err != nil {
			log.Printf("[gateway] channel %q (id=%d) failed: %v — trying next", ch.Name, ch.ID, err)
			lastErr = err
			continue
		}
		return resp, ch, nil
	}
	return nil, nil, fmt.Errorf("all channels failed for model %q: %w", req.Model, lastErr)
}

// ChatStream routes a streaming request with failover. When a matching
// OpenAI-compatible channel serves the request, its raw SSE stream is
// forwarded; adapters without stream translation return a buffered result.
func (g *Gateway) ChatStream(ctx context.Context, req *dto.ChatRequest) (*provider.StreamResult, *model.Channel, error) {
	g.reload(ctx)

	g.mu.RLock()
	candidates := make([]*model.Channel, 0, len(g.channels))
	for _, ch := range g.channels {
		if supports(ch, req.Model) {
			candidates = append(candidates, ch)
		}
	}
	g.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no active channel serves model %q", req.Model)
	}

	var lastErr error
	for _, ch := range candidates {
		p, err := g.providerFor(ch)
		if err != nil {
			lastErr = err
			continue
		}
		res, err := p.ChatStream(ctx, req)
		if err != nil {
			log.Printf("[gateway] channel %q (id=%d) stream failed: %v — trying next", ch.Name, ch.ID, err)
			lastErr = err
			continue
		}
		return res, ch, nil
	}
	return nil, nil, fmt.Errorf("all channels failed for model %q: %w", req.Model, lastErr)
}

// providerFor returns a cached Provider for a channel.
func (g *Gateway) providerFor(ch *model.Channel) (provider.Provider, error) {
	g.mu.RLock()
	p, ok := g.providers[ch.ID]
	g.mu.RUnlock()
	if ok {
		return p, nil
	}
	p, err := provider.NewFromChannel(ch)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	g.providers[ch.ID] = p
	g.mu.Unlock()
	return p, nil
}

// Models aggregates the model list across all active channels, deduped.
func (g *Gateway) Models(ctx context.Context) []dto.Model {
	g.reload(ctx)

	g.mu.RLock()
	channels := append([]*model.Channel(nil), g.channels...)
	g.mu.RUnlock()

	seen := make(map[string]bool)
	var out []dto.Model
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(ch *model.Channel) {
			defer wg.Done()
			p, err := g.providerFor(ch)
			if err != nil {
				log.Printf("[gateway] channel %q adapter error: %v", ch.Name, err)
				return
			}
			models, err := p.Models(ctx)
			if err != nil {
				log.Printf("[gateway] channel %q models failed: %v", ch.Name, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, m := range models {
				if !seen[m.ID] {
					seen[m.ID] = true
					out = append(out, m)
				}
			}
		}(ch)
	}
	wg.Wait()
	return out
}

// ChatEnv handles the request using an env-configured default provider when
// no database channels serve the model.
func (g *Gateway) ChatEnv(ctx context.Context, req *dto.ChatRequest, p provider.Provider) (*dto.ChatResponse, error) {
	return p.Chat(ctx, req)
}