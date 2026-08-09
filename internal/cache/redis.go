package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisCache is a Redis-backed cache with TTL support.
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedis creates a new RedisCache. Returns nil if connection fails (caller should fallback to MemoryCache).
func NewRedis(addr, password string, db int, prefix string) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Str("addr", addr).Msg("cache: redis connection failed, falling back to memory")
		return nil
	}

	log.Info().Str("addr", addr).Str("prefix", prefix).Msg("cache: redis connected")
	return &RedisCache{client: client, prefix: prefix}
}

func (c *RedisCache) key(k string) string {
	if c.prefix != "" {
		return c.prefix + ":" + k
	}
	return k
}

// Set stores a value with a TTL. The value is JSON-marshaled.
func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.client.Set(ctx, c.key(key), data, ttl).Err(); err != nil {
		log.Error().Err(err).Str("key", key).Msg("cache: redis SET error")
	}
}

// Get retrieves a value and unmarshals it into dest. Returns false if not found or expired.
func (c *RedisCache) Get(key string, dest interface{}) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Error().Err(err).Str("key", key).Msg("cache: redis GET error")
		}
		return false
	}
	return json.Unmarshal(data, dest) == nil
}

// Delete removes a key from the cache.
func (c *RedisCache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c.client.Del(ctx, c.key(key))
}

// DeletePattern removes all keys matching a prefix using SCAN.
func (c *RedisCache) DeletePattern(prefix string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fullPrefix := c.key(prefix)
	iter := c.client.Scan(ctx, 0, fullPrefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			c.client.Del(ctx, keys...)
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		c.client.Del(ctx, keys...)
	}
}

// Flush removes all entries with the current prefix.
func (c *RedisCache) Flush() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if c.prefix != "" {
		c.DeletePattern("")
	} else {
		c.client.FlushDB(ctx)
	}
}

// Stats returns cache statistics.
func (c *RedisCache) Stats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info, err := c.client.DBSize(ctx).Result()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"type":  "redis",
		"total": info,
	}
}

// Close closes the Redis connection.
func (c *RedisCache) Close() {
	c.client.Close()
}

// Ping checks Redis connectivity.
func (c *RedisCache) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.client.Ping(ctx).Err()
}

// HybridCache uses Redis as primary cache with MemoryCache as fallback.
// If Redis is not available, it falls back to MemoryCache automatically.
type HybridCache struct {
	redis  *RedisCache
	memory *MemoryCache
}

// NewHybrid creates a HybridCache. If Redis connection fails, it uses MemoryCache only.
func NewHybrid(addr, password string, db int, prefix string) *HybridCache {
	rc := NewRedis(addr, password, db, prefix)
	return &HybridCache{
		redis:  rc,
		memory: New(),
	}
}

// Set stores a value with a TTL.
func (h *HybridCache) Set(key string, value interface{}, ttl time.Duration) {
	if h.redis != nil {
		h.redis.Set(key, value, ttl)
	}
	h.memory.Set(key, value, ttl)
}

// Get retrieves a value. Tries Redis first, then memory.
func (h *HybridCache) Get(key string, dest interface{}) bool {
	if h.redis != nil && h.redis.Get(key, dest) {
		return true
	}
	return h.memory.Get(key, dest)
}

// Delete removes a key from both caches.
func (h *HybridCache) Delete(key string) {
	if h.redis != nil {
		h.redis.Delete(key)
	}
	h.memory.Delete(key)
}

// DeletePattern removes all keys matching a prefix from both caches.
func (h *HybridCache) DeletePattern(prefix string) {
	if h.redis != nil {
		h.redis.DeletePattern(prefix)
	}
	h.memory.DeletePattern(prefix)
}

// Flush removes all entries from both caches.
func (h *HybridCache) Flush() {
	if h.redis != nil {
		h.redis.Flush()
	}
	h.memory.Flush()
}

// Stats returns cache statistics.
func (h *HybridCache) Stats() map[string]interface{} {
	if h.redis != nil {
		s := h.redis.Stats()
		s["fallback"] = h.memory.Stats()
		s["mode"] = "redis+memory"
		return s
	}
	return map[string]interface{}{
		"mode":  "memory-only",
		"stats": h.memory.Stats(),
	}
}

// Close closes both caches.
func (h *HybridCache) Close() {
	if h.redis != nil {
		h.redis.Close()
	}
	h.memory.Close()
}

// IsRedis returns true if Redis is available.
func (h *HybridCache) IsRedis() bool {
	return h.redis != nil
}

// RedisClient returns the underlying Redis client (or nil).
func (h *HybridCache) RedisClient() *redis.Client {
	if h.redis == nil {
		return nil
	}
	return h.redis.client
}

// RateLimiter implements a Redis-based sliding window rate limiter.
type RateLimiter struct {
	redis  *redis.Client
	prefix string
}

// NewRateLimiter creates a rate limiter. Falls back to nil if Redis is unavailable.
func NewRateLimiter(rc *redis.Client, prefix string) *RateLimiter {
	if rc == nil {
		return nil
	}
	return &RateLimiter{redis: rc, prefix: prefix}
}

// Allow checks if a request is allowed under the rate limit.
// Returns (allowed, remaining, resetAt).
func (r *RateLimiter) Allow(key string, max int, window time.Duration) (bool, int, time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fullKey := fmt.Sprintf("%s:%s", r.prefix, key)
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove old entries
	r.redis.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current entries
	count, err := r.redis.ZCard(ctx, fullKey).Result()
	if err != nil {
		return true, max, now.Add(window)
	}

	if int(count) >= max {
		// Get the oldest entry to calculate reset time
		oldest, err := r.redis.ZRange(ctx, fullKey, 0, 0).Result()
		if err == nil && len(oldest) > 0 {
			var oldestScore float64
			fmt.Sscanf(oldest[0], "%f", &oldestScore)
			resetAt := time.Unix(0, int64(oldestScore)).Add(window)
			return false, 0, resetAt
		}
		return false, 0, now.Add(window)
	}

	// Add current request
	member := fmt.Sprintf("%d", now.UnixNano())
	r.redis.ZAdd(ctx, fullKey, redis.Z{Score: float64(now.UnixNano()), Member: member})
	r.redis.Expire(ctx, fullKey, window)

	remaining := max - int(count) - 1
	return true, remaining, now.Add(window)
}
