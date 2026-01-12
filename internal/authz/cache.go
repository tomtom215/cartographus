// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package authz

import (
	"sync"
	"time"
)

// enforcementCache caches authorization decisions.
type enforcementCache struct {
	ttl      time.Duration
	mu       sync.RWMutex
	items    map[string]*cacheItem
	stopChan chan struct{}
	stopOnce sync.Once
}

type cacheItem struct {
	allowed   bool
	expiresAt time.Time
}

// newEnforcementCache creates a new cache.
func newEnforcementCache(ttl time.Duration) *enforcementCache {
	// Ensure TTL is at least 1 second
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c := &enforcementCache{
		ttl:      ttl,
		items:    make(map[string]*cacheItem),
		stopChan: make(chan struct{}),
	}
	go c.cleanup()
	return c
}

// key generates a cache key.
func (c *enforcementCache) key(subject, object, action string) string {
	return subject + ":" + object + ":" + action
}

// get retrieves a cached decision.
func (c *enforcementCache) get(subject, object, action string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[c.key(subject, object, action)]
	if !ok {
		return false, false
	}

	if time.Now().After(item.expiresAt) {
		return false, false
	}

	return item.allowed, true
}

// set stores a decision in the cache.
func (c *enforcementCache) set(subject, object, action string, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[c.key(subject, object, action)] = &cacheItem{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// invalidateUser removes all cached decisions for a user.
func (c *enforcementCache) invalidateUser(subject string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := subject + ":"
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

// clear removes all cached decisions.
func (c *enforcementCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// cleanup periodically removes expired items.
func (c *enforcementCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, item := range c.items {
				if now.After(item.expiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		}
	}
}

// stop stops the cleanup goroutine.
// It is safe to call multiple times (idempotent).
func (c *enforcementCache) stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
}
