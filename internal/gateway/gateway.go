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

	"github.com/RingKoAI/RingRouter/internal/cache"
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
	redis     *cache.Client
}

// Snapshot cadence: each instance re-reads the shared Redis snapshot at most
// once per memoryTTL, and the shared copy outlives many instance refreshes so
// it is usually warm for every instance. Correctness after channel mutations
// is guaranteed by InvalidateChannels deleting the shared key, never by TTL.
const (
	channelCacheTTL  = 30 * time.Second // in-memory snapshot freshness
	redisSnapshotTTL = 5 * time.Minute  // shared snapshot lifetime
)

// channelSnapshotKey is the Redis key holding the serialized channel list.
const channelSnapshotKey = "gateway:channels"

// New creates a Gateway. redis is optional; when nil, the in-memory channel
// snapshot cache is used exactly as before.
func New(redis *cache.Client) *Gateway {
	return &Gateway{providers: make(map[uint]provider.Provider), redis: redis}
}

// snapshotChannel is the wire form used for the shared Redis snapshot. It
// shadows model.Channel's hidden APIKey so the sealed key (never plaintext)
// survives the round trip — providers need it to unseal credentials. The
// HTTP layer keeps using model.Channel directly and never leaks the key.
type snapshotChannel struct {
	model.Channel
	APIKey string `json:"api_key"` // AES-GCM sealed form
}

// reload refreshes the channel snapshot. The in-memory copy serves all hot
// requests (no network hop, one-api style); when it goes stale, a shared
// Redis snapshot is consulted first for cross-instance consistency, and the
// database remains the source of truth on cache miss. All reads/writes of
// g.channels happen under g.mu.
func (g *Gateway) reload(ctx context.Context) {
	if database.DB == nil {
		return
	}

	g.mu.RLock()
	fresh := time.Since(g.loaded) < channelCacheTTL
	g.mu.RUnlock()
	if fresh {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if time.Since(g.loaded) < channelCacheTTL { // re-check under lock
		return
	}

	// Shared Redis snapshot: fast and consistent across instances.
	if g.redis != nil {
		var shared []snapshotChannel
		if g.redis.Get(ctx, channelSnapshotKey, &shared) && len(shared) >= 0 {
			g.channels = reviveSnapshot(shared)
			g.loaded = time.Now()
			return
		}
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

	// Publish for other instances.
	if g.redis != nil {
		wire := make([]snapshotChannel, len(snapshot))
		for i, ch := range snapshot {
			wire[i] = snapshotChannel{Channel: *ch, APIKey: ch.APIKey}
		}
		g.redis.Set(ctx, channelSnapshotKey, wire, redisSnapshotTTL)
	}
}

// reviveSnapshot converts the wire form back into channel pointers.
func reviveSnapshot(wire []snapshotChannel) []*model.Channel {
	out := make([]*model.Channel, len(wire))
	for i := range wire {
		ch := wire[i].Channel
		ch.APIKey = wire[i].APIKey
		out[i] = &ch
	}
	return out
}

// InvalidateChannels drops the cached channel snapshot and provider pool so
// the next request re-reads the database with fresh credentials. Called by
// the admin API after channel mutations.
func (g *Gateway) InvalidateChannels(ctx context.Context) {
	g.mu.Lock()
	g.loaded = time.Time{}
	g.providers = make(map[uint]provider.Provider)
	g.mu.Unlock()
	if g.redis != nil {
		g.redis.Del(ctx, channelSnapshotKey)
	}
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

// inGroup reports whether the channel belongs to the requested routing
// group. A channel may belong to multiple groups (comma-separated, one-api
// semantics); the default group matches any channel that did not opt out.
func inGroup(ch *model.Channel, group string) bool {
	group = strings.TrimSpace(group)
	if group == "" || group == "default" {
		for _, g := range strings.Split(ch.Group, ",") {
			if g = strings.TrimSpace(g); g == "" || g == "default" {
				return true
			}
		}
		return false
	}
	for _, g := range strings.Split(ch.Group, ",") {
		if strings.TrimSpace(g) == group {
			return true
		}
	}
	return false
}

// Chat routes a unified request with failover across matching channels.
// It returns the response and the channel that served it. When group is
// non-empty (and not "default") it scopes the candidate set to channels
// whose group field matches.
func (g *Gateway) Chat(ctx context.Context, req *dto.ChatRequest, group string) (*dto.ChatResponse, *model.Channel, error) {
	g.reload(ctx)

	g.mu.RLock()
	candidates := make([]*model.Channel, 0, len(g.channels))
	for _, ch := range g.channels {
		if supports(ch, req.Model) && inGroup(ch, group) {
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
func (g *Gateway) ChatStream(ctx context.Context, req *dto.ChatRequest, group string) (*provider.StreamResult, *model.Channel, error) {
	g.reload(ctx)

	g.mu.RLock()
	candidates := make([]*model.Channel, 0, len(g.channels))
	for _, ch := range g.channels {
		if supports(ch, req.Model) && inGroup(ch, group) {
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
