package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Предполагаемые интерфейсы и структуры из исходного контекста
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
	// Добавляем TryUpdate для атомарного обновления (conditional update)
	// err != nil если условие не выполнено (например, LockToken изменился)
	TryUpdate(uuid UUID, newItem T, expectedLockToken UUID) error
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

type LocalItem struct {
	Value     []byte
	ExpiresAt time.Time
}

// Конфигурация прокси
type Config struct {
	BackendTimeout time.Duration // Таймаут вызова бэкенда
	CacheTTL       time.Duration // Время жизни кэша
	MaxConcurrency int           // Макс. одновременных запросов к бэкенду
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
	localCache := make(map[string]LocalItem)

	cache := &CacheProxy{
		cacheId:    UUID(id.String()),
		semaphore:  semaphore,
		chanClose:  chanClose,
		mu:         sync.RWMutex{},
		backend:    backend,
		storage:    storage, // Исправлено: используем переданный storage
		localCache: localCache,
		cfg:        cfg,
		timeClean:  cfg.CacheTTL / 10, // Чистим каждые 1/10 TTL, например, 60s для 10min TTL
	}

	go cache.CleanupExpired()

	cache.RecoverStaleLocks() // Восстанавливаем при старте

	return cache
}

// checkCache: Проверяет кэш, возвращает value, hit (есть валидный value), shouldWait (ждать, если другой инстанс обрабатывает), uuid (для update)
func (cp *CacheProxy) checkCache(id string) ([]byte, bool, bool, UUID) {
	cp.mu.RLock()
	item, exists := cp.localCache[id]
	cp.mu.RUnlock()

	now := time.Now()

	if exists && !now.After(item.ExpiresAt) {
		return item.Value, true, false, ""
	}

	// Ищем в storage по Key (предполагаем, что для одного Key один item, иначе взять последний)
	findOperator := []FindOperator{{Key: "Key", Operator: "=", Value: id}}
	values := cp.storage.Find(findOperator)

	if len(values) == 0 {
		// Создаём новый item с локом
		elem := CacheItem{
			Key:           id,
			LockToken:     cp.cacheId,
			LockExpiresAt: now.Add(cp.cfg.BackendTimeout),
			InProgress:    true,
		}
		uuid := cp.storage.Store(elem)
		return nil, false, false, uuid
	}

	cacheItem := values[0] // Берём первый (или добавьте логику выбора)

	if !now.After(item.ExpiresAt) && len(item.Value) != 0 {
		// Hit: обновляем localCache
		cp.mu.Lock()
		cp.localCache[id] = LocalItem{Value: item.Value, ExpiresAt: item.ExpiresAt}
		cp.mu.Unlock()
		return item.Value, true, false, ""
	}

	// Если другой инстанс обрабатывает и лок не expired
	if cacheItem.InProgress && !now.After(cacheItem.LockExpiresAt) && cacheItem.LockToken != cp.cacheId {
		return nil, false, true, ""
	}

	// Пытаемся захватить лок атомарно (на expired или не InProgress)
	newItem := cacheItem
	newItem.LockToken = cp.cacheId
	newItem.LockExpiresAt = now.Add(cp.cfg.BackendTimeout)
	newItem.InProgress = true

	// Используем TryUpdate с expected LockToken из текущего item (для CAS-like)
	if err := cp.storage.TryUpdate(cacheItem.RecordUUID, newItem, cacheItem.LockToken); err == nil {
		return nil, false, false, cacheItem.RecordUUID
	}

	// Не удалось захватить: кто-то другой взял, ждём
	return nil, false, true, ""
}

// Основной метод получения ресурса
func (cp *CacheProxy) GetResource(id string) ([]byte, error) {
	const maxWaitAttempts = 20 // Макс. итераций ожидания (e.g., 20 * 200ms = 4s)
	var uuid UUID
	var value []byte
	var hit, shouldWait bool
	attempt := 0
	
	for {
		value, hit, shouldWait, uuid = cp.checkCache(id)
		if hit {
			return value, nil
		}
		if !shouldWait {
			// Захватили лок, fetch'им
			break
		}
		attempt++
		if attempt >= maxWaitAttempts {
			return nil, fmt.Errorf("timeout waiting for cache item for id %s", id)
		}
		time.Sleep(200 * time.Millisecond) // Backoff, можно экспоненциальный
	}

	// Fetch из backend с семафором
	cp.semaphore <- struct{}{}
	defer func() { <-cp.semaphore }()

	ctx, cancel := context.WithTimeout(context.Background(), cp.cfg.BackendTimeout)
	defer cancel()

	value, err := cp.backend.GetResource(ctx, id)

	// Обновляем item
	elem := cp.storage.Get(uuid)
	elem.InProgress = false
	elem.LockToken = "" // Освобождаем лок
	if err == nil {
		elem.Value = value
		elem.ExpiresAt = time.Now().Add(cp.cfg.CacheTTL)
	}
	cp.storage.Update(uuid, elem) // Всегда Update, т.к. uuid существует

	// Обновляем localCache
	if err == nil {
		cp.mu.Lock()
		cp.localCache[id] = LocalItem{Value: value, ExpiresAt: elem.ExpiresAt}
		cp.mu.Unlock()
	}

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
			now := time.Now()
			keysToDelete := []string{}
			cp.mu.RLock()
			for key, item := range cp.localCache {
				if now.After(item.ExpiresAt) {
					keysToDelete = append(keysToDelete, key)
				}
			}
			cp.mu.RUnlock()

			cp.mu.Lock()
			for _, key := range keysToDelete {
				if now.After(cp.localCache[key].ExpiresAt) {
					delete(cp.localCache, key)
				}
			}
			cp.mu.Unlock()
		}
	}
}

// Метод восстановления состояния при старте
func (cp *CacheProxy) RecoverStaleLocks() {
	findOperator := []FindOperator{{Key: "LockToken", Operator: "=", Value: string(cp.cacheId)}}
	values := cp.storage.Find(findOperator)

	for _, value := range values {
		if value.InProgress {
			elem := value
			elem.InProgress = false
			elem.LockToken = ""
			// Не трогаем ExpiresAt, если value есть - оно остаётся, иначе следующий запрос обновит
			cp.storage.Update(elem.RecordUUID, elem)
		}
	}
}

// Graceful shutdown
func (cp *CacheProxy) Close() {
	close(cp.chanClose)
}
