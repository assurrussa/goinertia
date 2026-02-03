package goinertia

import (
	"container/list"
	"sync"
	"time"
)

type ssrCacheEntry struct {
	value     SsrDTO
	expiresAt time.Time
	element   *list.Element
}

type ssrCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	items      map[string]*ssrCacheEntry
	order      *list.List
}

func newSSRCache(ttl time.Duration, maxEntries int) *ssrCache {
	if ttl <= 0 || maxEntries <= 0 {
		return nil
	}

	return &ssrCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		items:      make(map[string]*ssrCacheEntry, maxEntries),
		order:      list.New(),
	}
}

func (c *ssrCache) Get(key string) (*SsrDTO, bool) {
	if c == nil {
		return nil, false
	}

	now := time.Now()
	c.mu.Lock()
	entry, ok := c.items[key]
	if !ok {
		c.mu.Unlock()
		return nil, false
	}

	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		c.removeEntryLocked(key, entry)
		c.mu.Unlock()
		return nil, false
	}

	c.order.MoveToFront(entry.element)
	value := cloneSSR(&entry.value)
	c.mu.Unlock()
	return value, true
}

func (c *ssrCache) Set(key string, value *SsrDTO) {
	if c == nil || value == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.items[key]; ok {
		entry.value = *cloneSSR(value)
		entry.expiresAt = time.Now().Add(c.ttl)
		c.order.MoveToFront(entry.element)
		return
	}

	if len(c.items) >= c.maxEntries {
		c.evictOneLocked()
	}

	elem := c.order.PushFront(key)
	c.items[key] = &ssrCacheEntry{
		value:     *cloneSSR(value),
		expiresAt: time.Now().Add(c.ttl),
		element:   elem,
	}
}

func (c *ssrCache) evictOneLocked() {
	tm := time.Now()
	for elem := c.order.Back(); elem != nil; elem = elem.Prev() {
		key, ok := elem.Value.(string)
		if !ok {
			c.order.Remove(elem)
			continue
		}
		entry, ok := c.items[key]
		if !ok {
			c.order.Remove(elem)
			continue
		}
		if !entry.expiresAt.IsZero() && tm.After(entry.expiresAt) {
			c.removeEntryLocked(key, entry)
			return
		}
	}

	elem := c.order.Back()
	if elem == nil {
		return
	}
	key, ok := elem.Value.(string)
	if !ok {
		c.order.Remove(elem)
		return
	}
	entry, ok := c.items[key]
	if !ok {
		c.order.Remove(elem)
		return
	}
	c.removeEntryLocked(key, entry)
}

func (c *ssrCache) removeEntryLocked(key string, entry *ssrCacheEntry) {
	if entry != nil && entry.element != nil {
		c.order.Remove(entry.element)
	}
	delete(c.items, key)
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
