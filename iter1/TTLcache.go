package main

// Задача: Необходимо реализовать конкурентный кэш,
// который хранит значения с временем жизни TTL.
// Кэш должен быть потокобезопасным и автоматически
// удалять просроченные записи.

import (
	"sync"
	_ "sync"
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
	}
	go func(cacheTTL *ConcurrentTTLCache) {
		ticker := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ticker.C:
				ElementsToRemove := make(map[string]struct{}, len(cacheTTL.cacheMap))
				cacheTTL.cacheMutex.RLock()
				for key, value := range cacheTTL.cacheMap {
					if !value.expiry.After(time.Now()) {
						ElementsToRemove[key] = struct{}{}
					}
				}
				cacheTTL.cacheMutex.RUnlock()

				for key := range ElementsToRemove {
					cacheTTL.cacheMutex.Lock()
					value, _ := cacheTTL.cacheMap[key]
					if !value.expiry.After(time.Now()) {
						delete(cacheTTL.cacheMap, key)
					}
					cacheTTL.cacheMutex.Unlock()
				}

				for key, value := range cacheTTL.cacheMap {
					cacheTTL.cacheMutex.Lock()
					if !value.expiry.After(time.Now()) {
						delete(cacheTTL.cacheMap, key)
					}
					cacheTTL.cacheMutex.Unlock()
				}
			case <-chanStop:
				ticker.Stop()
				return
			}
		}
	}(cacheTTL)

	return cacheTTL
}

func (c *ConcurrentTTLCache) Set(key string, value interface{}, ttl time.Duration) {

	timeToExpire := time.Now().Add(ttl)
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cacheMap[key] = CacheItem{value: value, expiry: timeToExpire}
	return
}

func (c *ConcurrentTTLCache) Get(key string) (interface{}, bool) {
	c.cacheMutex.RLock()
	value, exists := c.cacheMap[key]
	c.cacheMutex.RUnlock()

	if !exists {
		return nil, false
	}

	if !value.expiry.After(time.Now()) {
		return nil, false
	}

	return value.value, true

}

func (c *ConcurrentTTLCache) Stop() {
	close(c.chanStop)
}
