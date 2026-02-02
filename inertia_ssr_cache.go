package goinertia

import (
	"sync"
	"time"
)

type ssrCacheEntry struct {
	value     SsrDTO
	expiresAt time.Time
}

type ssrCache struct {
	mu         sync.RWMutex
	ttl        time.Duration
	maxEntries int
	items      map[string]ssrCacheEntry
}

func newSSRCache(ttl time.Duration, maxEntries int) *ssrCache {
	if ttl <= 0 || maxEntries <= 0 {
		return nil
	}

	return &ssrCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		items:      make(map[string]ssrCacheEntry),
	}
}

func (c *ssrCache) Get(key string) (*SsrDTO, bool) {
	if c == nil {
		return nil, false
	}

	now := time.Now()
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return cloneSSR(&entry.value), true
}

func (c *ssrCache) Set(key string, value *SsrDTO) {
	if c == nil || value == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxEntries {
		c.evictOneLocked()
	}

	c.items[key] = ssrCacheEntry{
		value:     *cloneSSR(value),
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *ssrCache) evictOneLocked() {
	for key, entry := range c.items {
		if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
			delete(c.items, key)
			return
		}
	}

	for key := range c.items {
		delete(c.items, key)
		return
	}
}

func cloneSSR(src *SsrDTO) *SsrDTO {
	if src == nil {
		return nil
	}

	dst := &SsrDTO{
		Body: src.Body,
	}

	if len(src.Head) > 0 {
		dst.Head = make([]string, len(src.Head))
		copy(dst.Head, src.Head)
	}

	return dst
}
