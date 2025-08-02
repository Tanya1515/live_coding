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
	cacheMutex *sync.Mutex
	chanStop chan struct{}
}

func NewConcurrentTTLCache() *ConcurrentTTLCache {
	var mutex sync.Mutex
	cache := make(map[string]CacheItem, 10)
	chanStop := make(chan struct{})

	cacheTTL := &ConcurrentTTLCache{
		cacheMutex: &mutex,
		cacheMap: cache,
	}
	go func(cacheTTL *ConcurrentTTLCache) {
		ticker := time.NewTicker(2 *time.Second)
		for {
			select {
			case <- ticker.C: 
				for key, value := range cacheTTL.cacheMap {
					if !value.expiry.After(time.Now()) {
						cacheTTL.cacheMutex.Lock()
						delete(cacheTTL.cacheMap, key)
						cacheTTL.cacheMutex.Unlock()
					}
				}
			case <- chanStop: 
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
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// очистка?
	if value, exists := c.cacheMap[key]; exists {
		if !value.expiry.After(time.Now()) {
			delete(c.cacheMap, key)
		} else {
			return value.value, true
		}
	}
	return nil, false
}

func (c *ConcurrentTTLCache) Stop() {
	close(c.chanStop)
}
