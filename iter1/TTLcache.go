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
				// RLock
				// пройтись по всей мапе и достать кандидатов на удаление
				// RUnlock

				// Проходимся по кандидатм на удаление и под Lock-ом на запись выполняем проверку, что элемент не поменялся и чистим


				for key, value := range cacheTTL.cacheMap {
					// в RLock лучше только получение, затем еще раз проверить внутри lock-а на запись, 
					// что значение не поменялось
					cacheTTL.cacheMutex.Lock()
					if !value.expiry.After(time.Now()) {
						delete(cacheTTL.cacheMap, key)
					}
					cacheTTL.cacheMutex.Unlock()
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
	// Mutex -> RWMutex, 
	c.cacheMutex.Lock()
	value, exists := c.cacheMap[key]
	c.cacheMutex.Unlock()

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
