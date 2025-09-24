package main

// Задача: Необходимо реализовать конкурентный кэш,
// который хранит значения с временем жизни TTL.
// Кэш должен быть потокобезопасным и автоматически
// удалять просроченные записи.

import (
	"sync"
	"time"
)

type CacheItem struct {
	value  interface{}
	expiry time.Time
}

type ConcurrentTTLCache struct {
	cacheMap map[string]CacheItem
	// RWMutex
	cacheMutex *sync.RWMutex
	chanStop   chan struct{}
}

func NewConcurrentTTLCache() *ConcurrentTTLCache {
	cache := make(map[string]CacheItem, 10)
	chanStop := make(chan struct{})

	cacheTTL := &ConcurrentTTLCache{
		cacheMutex: &sync.RWMutex{},
		cacheMap:   cache,
		chanStop:   chanStop,
	}
	go cacheTTL.clearCache()

	return cacheTTL
}

func (c *ConcurrentTTLCache) clearCache() {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			ElementsToRemove := make(map[string]struct{}, len(c.cacheMap))
			c.cacheMutex.RLock()
			for key, value := range c.cacheMap {
				if !value.expiry.After(time.Now()) {
					ElementsToRemove[key] = struct{}{}
				}
			}
			c.cacheMutex.RUnlock()

			for key := range ElementsToRemove {
				c.cacheMutex.Lock()
				value, _ := c.cacheMap[key]
				if !value.expiry.After(time.Now()) {
					delete(c.cacheMap, key)
				}
				c.cacheMutex.Unlock()
			}

		case <-c.chanStop:
			ticker.Stop()
			return
		}
	}
}

func (c *ConcurrentTTLCache) Set(key string, value interface{}, ttl time.Duration) {

	timeToExpire := time.Now().Add(ttl)
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cacheMap[key] = CacheItem{value: value, expiry: timeToExpire}
}

func (c *ConcurrentTTLCache) Get(key string) (interface{}, bool) {
	c.cacheMutex.RLock()
	value, exists := c.cacheMap[key]
	c.cacheMutex.RUnlock()

	if !exists {
		return nil, false
	}

	if !value.expiry.After(time.Now()) {
		c.cacheMutex.Lock()
		delete(c.cacheMap, key)
		c.cacheMutex.Unlock()
		return nil, false
	}

	return value.value, true

}

func (c *ConcurrentTTLCache) Stop() {
	close(c.chanStop)
}
