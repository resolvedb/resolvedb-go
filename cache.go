package resolvedb

import (
	"strings"
	"sync"
	"time"
)

// CacheConfig configures response caching.
type CacheConfig struct {
	Enabled    bool          // Enable caching
	MaxEntries int           // Maximum cache entries (0 = unlimited)
	DefaultTTL time.Duration // Default TTL if not specified in response
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:    true,
		MaxEntries: 1000,
		DefaultTTL: 5 * time.Minute,
	}
}

// Cache provides TTL-aware response caching.
type Cache interface {
	Get(key string) (*Response, bool)
	Set(key string, resp *Response, ttl time.Duration)
	Delete(key string)
	Clear()
}

// memoryCache is an in-memory cache implementation.
type memoryCache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	defaultTTL time.Duration
}

type cacheEntry struct {
	response  *Response
	expiresAt time.Time
}

// newMemoryCache creates a new in-memory cache.
func newMemoryCache(config CacheConfig) *memoryCache {
	return &memoryCache{
		entries:    make(map[string]*cacheEntry),
		maxEntries: config.MaxEntries,
		defaultTTL: config.DefaultTTL,
	}
}

// Get retrieves a cached response.
func (c *memoryCache) Get(key string) (*Response, bool) {
	c.mu.RLock()
	entry, ok := c.entries[normalizeKey(key)]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.Delete(key)
		return nil, false
	}

	return entry.response, true
}

// Set stores a response in the cache.
func (c *memoryCache) Set(key string, resp *Response, ttl time.Duration) {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// Use response TTL if available and shorter
	if resp.TTL > 0 && resp.TTL < ttl {
		ttl = resp.TTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: remove expired entries if at capacity
	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		c.evictExpired()
	}

	c.entries[normalizeKey(key)] = &cacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a cached response.
func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, normalizeKey(key))
	c.mu.Unlock()
}

// Clear removes all cached responses.
func (c *memoryCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*cacheEntry)
	c.mu.Unlock()
}

// evictExpired removes expired entries. Must be called with lock held.
func (c *memoryCache) evictExpired() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// normalizeKey normalizes a cache key for consistent lookups.
// Per security review: lowercase before hashing to prevent cache poisoning.
func normalizeKey(key string) string {
	return strings.ToLower(key)
}

// buildCacheKey creates a cache key from query parameters.
func buildCacheKey(operation, resource, key, namespace, version string) string {
	parts := []string{operation, resource, key, namespace, version}
	return normalizeKey(strings.Join(parts, "."))
}

// noopCache is a no-op cache implementation for when caching is disabled.
type noopCache struct{}

func (noopCache) Get(string) (*Response, bool)        { return nil, false }
func (noopCache) Set(string, *Response, time.Duration) {}
func (noopCache) Delete(string)                        {}
func (noopCache) Clear()                               {}
