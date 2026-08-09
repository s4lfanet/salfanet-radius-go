package cache

import (
	"sync"
	"time"
)

// GenieacsCache provides in-memory caching for GenieACS device data.
// This avoids hitting GenieACS NBI on every page load, reducing CPU/RAM usage.
// Cache is refreshed by a cronjob every 5 minutes and on manual refresh.
type GenieacsCache struct {
	mu        sync.RWMutex
	devices   []map[string]interface{}
	deviceMap map[string]map[string]interface{} // keyed by _id
	updatedAt time.Time
	ttl       time.Duration
}

var (
	instance *GenieacsCache
	once     sync.Once
)

// GetGenieacsCache returns the singleton cache instance.
func GetGenieacsCache() *GenieacsCache {
	once.Do(func() {
		instance = &GenieacsCache{
			deviceMap: make(map[string]map[string]interface{}),
			ttl:       5 * time.Minute,
		}
	})
	return instance
}

// SetDevices replaces the entire device cache.
func (c *GenieacsCache) SetDevices(devices []map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.devices = devices
	c.deviceMap = make(map[string]map[string]interface{}, len(devices))
	for _, d := range devices {
		if id, ok := d["_id"].(string); ok {
			c.deviceMap[id] = d
		}
	}
	c.updatedAt = time.Now()
}

// SetDevice updates or inserts a single device in the cache.
// If device is nil, the device is removed from the cache (invalidation).
func (c *GenieacsCache) SetDevice(deviceID string, device map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.deviceMap == nil {
		c.deviceMap = make(map[string]map[string]interface{})
	}
	if device == nil {
		delete(c.deviceMap, deviceID)
	} else {
		c.deviceMap[deviceID] = device
	}
	// Rebuild slice
	c.devices = make([]map[string]interface{}, 0, len(c.deviceMap))
	for _, d := range c.deviceMap {
		c.devices = append(c.devices, d)
	}
	c.updatedAt = time.Now()
}

// GetDevices returns all cached devices. Returns nil if cache is stale.
func (c *GenieacsCache) GetDevices() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.isStaleLocked() {
		return nil
	}
	return c.devices
}

// GetDevice returns a single cached device by ID. Returns nil if not found or stale.
func (c *GenieacsCache) GetDevice(deviceID string) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.isStaleLocked() {
		return nil
	}
	return c.deviceMap[deviceID]
}

// IsFresh returns true if the cache has data and is not expired.
func (c *GenieacsCache) IsFresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.isStaleLocked()
}

// UpdatedAt returns the last cache update time.
func (c *GenieacsCache) UpdatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updatedAt
}

// Invalidate forces the cache to be considered stale on next read.
func (c *GenieacsCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updatedAt = time.Time{}
}

func (c *GenieacsCache) isStaleLocked() bool {
	if len(c.devices) == 0 {
		return true
	}
	return time.Since(c.updatedAt) > c.ttl
}
