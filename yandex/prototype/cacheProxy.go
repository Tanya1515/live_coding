package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Задача: Реализация кэширующего прокси для медленного сервиса

// Контекст:
// Есть внешний HTTP-сервис, обрабатывающий запросы медленно (500мс-2с). Наш сервис должен кэшировать ответы, уменьшая нагрузку на бэкенд. Работаем в cloud-native среде с несколькими инстансами.

// Требования:
// Кэшировать ответы на GET /resource/{id} (ресурсы неизменны)
// Ограничивать число одновременных запросов к бэкенду - worker pool
// Поддерживать TTL записей в кэше (10 минут)
// Гарантировать обработку запросов при падении инстансов
// Избегать дублирования запросов к бэкенду при конкурентном доступе

// Интерфейсы и шаблон:
type BackendClient interface {
	GetResource(ctx context.Context, id string) ([]byte, error) // 500мс-2с
}

type FindOperator struct {
	Key, Operator, Value string
}

type UUID string

type Storage[T any] interface {
	Store(T) UUID
	Get(UUID) T
	Find([]FindOperator) []T
	Update(UUID, T)
}

type CacheItem struct {
	RecordUUID    UUID
	Key           string
	Value         []byte
	ExpiresAt     time.Time
	InProgress    bool // Флаг обработки
	LockToken     UUID // Токен блокировки, который помечает, к какому инстансу принадлежат
	LockExpiresAt time.Time
}

// Конфигурация прокси
type Config struct {
	BackendTimeout time.Duration // Таймаут вызова бэкенда
	CacheTTL       time.Duration // Время жизни кэша
	MaxConcurrency int           // Макс. одновременных запросов к бэкенду
}

type LocalItem struct {
	Value     []byte
	ExpiresAt time.Time
}

type CacheProxy struct {
	cacheId    UUID
	localCache map[string]LocalItem
	backend    BackendClient
	mu         sync.RWMutex
	cfg        Config
	storage    Storage[CacheItem]
	semaphore  chan struct{}
	chanClose  chan struct{}
	timeClean  time.Duration
}

func New(cfg Config, storage Storage[CacheItem], backend BackendClient) *CacheProxy {
	id := uuid.New()
	semaphore := make(chan struct{}, cfg.MaxConcurrency)
	chanClose := make(chan struct{})
	localCache := make(map[string]LocalItem, 0)

	var cacheStorage Storage[CacheItem]

	cache := &CacheProxy{
		cacheId:    UUID(id.String()),
		semaphore:  semaphore,
		chanClose:  chanClose,
		mu:         sync.RWMutex{},
		backend:    backend,
		storage:    cacheStorage,
		localCache: localCache,
		cfg:        cfg,
		timeClean:  1000*time.Second,
	}

	cache.RecoverStaleLocks()

	go cache.CleanupExpired()

	return cache
}

func (cp *CacheProxy) checkCache(id string) ([]byte, bool, UUID) {
	cp.mu.RLock()
	item, exists := cp.localCache[id]
	cp.mu.RUnlock()

	timeNow := time.Now()

	if exists {
		if !timeNow.After(item.ExpiresAt) {
			return item.Value, true, ""
		}
	}

	findOperator := make([]FindOperator, 0)
	find := FindOperator{
		Key:      "Key",
		Operator: "=",
		Value:    id,
	}
	findOperator = append(findOperator, find)
	values := cp.storage.Find(findOperator)

	for _, value := range values {
		if !timeNow.After(value.ExpiresAt) && len(value.Value) != 0 {
			return value.Value, true, ""
		}
	}

	for _, value := range values {
		if value.InProgress && timeNow.After(value.LockExpiresAt) {
			elem := value
			elem.LockToken = cp.cacheId
			elem.LockExpiresAt = time.Now().Add(cp.cfg.BackendTimeout)
			elem.InProgress = true
			cp.storage.Update(elem.RecordUUID, elem)
			return nil, false, elem.RecordUUID
		}
		if 
	}

	elem := CacheItem{
		Key:                id,
		LockToken:          cp.cacheId,
		LockExpiresAt: time.Now().Add(cp.cfg.BackendTimeout),
		InProgress:         true,
	}

	uuid := cp.storage.Store(elem)

	return nil, false, uuid

}

// Основной метод получения ресурса
func (cp *CacheProxy) GetResource(id string) ([]byte, error) {

	var value []byte
	var uuid UUID
	var exists bool
	for {
		value, exists, uuid = cp.checkCache(id)
		if exists && value != nil {
			return value, nil
		}
		if !exists {
			break
		}
		time.Sleep(10 * time.Second)
	}

	cp.semaphore <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), cp.cfg.BackendTimeout)
	defer cancel()
	value, err := cp.backend.GetResource(ctx, id)
	<-cp.semaphore

	elem := cp.storage.Get(uuid)
	elem.InProgress = false
	elem.LockToken = ""
	if err == nil {
		elem.Value = value
		elem.ExpiresAt = time.Now().Add(cp.cfg.CacheTTL)
	}
	cp.storage.Store(elem)

	localElem := LocalItem{
		Value:     value,
		ExpiresAt: elem.ExpiresAt,
	}

	cp.mu.Lock()
	cp.localCache[id] = localElem
	cp.mu.Unlock()

	return value, err

}

func (cp *CacheProxy) CleanupExpired() {
	ticker := time.NewTicker(cp.timeClean)
	for {
		select {
		case <-cp.chanClose:
			ticker.Stop()
			return
		case <-ticker.C:
			cacheCopy := make([]string, 0)
			timeNow := time.Now()
			cp.mu.RLock()
			for key, value := range cp.localCache {
				if timeNow.After(value.ExpiresAt) {
					cacheCopy = append(cacheCopy, key)
				}
			}
			cp.mu.RUnlock()
			timeNow = time.Now()
			for _, key := range cacheCopy {
				cp.mu.Lock()
				if timeNow.After(cp.localCache[key].ExpiresAt) {
					delete(cp.localCache, key)
				}
				cp.mu.Unlock()
			}
		}
	}
}

// Метод восстановления состояния при старте
func (cp *CacheProxy) RecoverStaleLocks() {
	findOperator := make([]FindOperator, 0)
	find := FindOperator{
		Key:      "LockToken",
		Operator: "=",
		Value:    string(cp.cacheId),
	}
	findOperator = append(findOperator, find)
	values := cp.storage.Find(findOperator)

	for _, value := range values {
		if value.InProgress {
			elem := value
			elem.InProgress = false
			elem.LockToken = ""
			elem.LockExpiresAt = time.Now().Add(cp.cfg.CacheTTL)
			cp.storage.Update(elem.RecordUUID, elem)
		}
	}

}
