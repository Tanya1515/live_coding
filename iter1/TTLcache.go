package main

// Задача: Необходимо реализовать конкурентный кэш,
// который хранит значения с временем жизни TTL.
// Кэш должен быть потокобезопасным и автоматически
// удалять просроченные записи.

import (
	_ "sync"
	"time"
)

type CacheItem struct {
	value  interface{}
	expiry time.Time
}

type ConcurrentTTLCache struct {
	// Добавьте поля
}

func NewConcurrentTTLCache() *ConcurrentTTLCache {
	return &ConcurrentTTLCache{}
}

func (c *ConcurrentTTLCache) Set(key string, value interface{}, ttl time.Duration) {
	// Реализуйте
}

func (c *ConcurrentTTLCache) Get(key string) (interface{}, bool) {
	// Реализуйте
	return nil, false
}
