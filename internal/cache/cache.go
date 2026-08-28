// Package cache provides an optional Redis-backed cache for hot-path data
// (channel snapshots, options). When Redis is disabled or unreachable, every
// call degrades gracefully to the in-memory fallback passed by the caller —
// the database remains the source of truth and correctness never depends on
// the cache.
package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Options configures the Redis client.
type Options struct {
	Addr     string
	Password string
	DB       int
}

// Client is a thin Redis wrapper with a JSON Get/Set surface and graceful
// degradation. A nil *Client is valid and always misses.
type Client struct {
	rdb *redis.Client
}

// keyPrefix scopes all application keys to avoid collisions in shared Redis.
const keyPrefix = "ringrouter:"

// New creates a Client. If enabled is false the client stays nil (disabled)
// and every operation falls back to the caller's in-memory value.
func New(enabled bool, opts Options) *Client {
	if !enabled {
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})
	return &Client{rdb: rdb}
}

// Ping verifies connectivity. Returns false on any error so callers can log
// a warning and continue with the memory fallback.
func (c *Client) Ping(ctx context.Context) bool {
	if c == nil || c.rdb == nil {
		return false
	}
	return c.rdb.Ping(ctx).Err() == nil
}

// Close releases the underlying connection pool.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// Get decodes a cached JSON value into v. Returns ok=false on miss or error.
func (c *Client) Get(ctx context.Context, key string, v interface{}) bool {
	if c == nil || c.rdb == nil {
		return false
	}
	raw, err := c.rdb.Get(ctx, keyPrefix+key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

// Set encodes v as JSON and stores it with a TTL. Failures are logged but
// never propagated — a failed cache write must not fail the caller.
func (c *Client) Set(ctx context.Context, key string, v interface{}, ttl time.Duration) {
	if c == nil || c.rdb == nil {
		log.Printf("[cache] set %q skipped: client disabled", key)
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		log.Printf("[cache] set %q marshal failed: %v", key, err)
		return
	}
	if err := c.rdb.Set(ctx, keyPrefix+key, raw, ttl).Err(); err != nil {
		log.Printf("[cache] set %q failed: %v", key, err)
	}
}

// Del removes a key.
func (c *Client) Del(ctx context.Context, key string) {
	if c == nil || c.rdb == nil {
		return
	}
	_ = c.rdb.Del(ctx, keyPrefix+key).Err()
}
