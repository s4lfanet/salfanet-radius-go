package cache

import (
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry holds a cached value with expiration.
type CacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// MemoryCache is a thread-safe in-memory cache with TTL support.
// Acts as a Redis replacement when Redis is not available.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	stopCh  chan struct{}
}

// New creates a new MemoryCache and starts a cleanup goroutine.
func New() *MemoryCache {
	c := &MemoryCache{
		entries: make(map[string]*CacheEntry),
		stopCh:  make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// Set stores a value with a TTL. The value is JSON-marshaled.
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.mu.Lock()
	c.entries[key] = &CacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Get retrieves a value and unmarshals it into dest. Returns false if not found or expired.
func (c *MemoryCache) Get(key string, dest interface{}) bool {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return false
	}

	return json.Unmarshal(entry.data, dest) == nil
}

// Delete removes a key from the cache.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// DeletePattern removes all keys matching a prefix.
func (c *MemoryCache) DeletePattern(prefix string) {
	c.mu.Lock()
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}

// Flush removes all entries.
func (c *MemoryCache) Flush() {
	c.mu.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.mu.Unlock()
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	active := 0
	expired := 0
	for _, entry := range c.entries {
		if now.After(entry.expiresAt) {
			expired++
		} else {
			active++
		}
	}
	return map[string]interface{}{
		"total":   len(c.entries),
		"active":  active,
		"expired": expired,
	}
}

// Close stops the cleanup goroutine.
func (c *MemoryCache) Close() {
	close(c.stopCh)
}

// cleanup periodically removes expired entries.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.entries {
				if now.After(entry.expiresAt) {
					delete(c.entries, key)
				}
			}
			c.mu.Unlock()
		}
	}
}

// Global cache instance
var defaultCache *MemoryCache

// Init initializes the global cache.
func Init() {
	if defaultCache == nil {
		defaultCache = New()
	}
}

// Default returns the global cache instance.
func Default() *MemoryCache {
	if defaultCache == nil {
		Init()
	}
	return defaultCache
}
