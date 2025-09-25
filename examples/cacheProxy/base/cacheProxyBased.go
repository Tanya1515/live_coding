package cacheproxy

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

import (
	"sync"
	"time"
)

// Медленный бэкенд-сервис

type BackendClient interface {
	GetResource(id string) ([]byte, error) // 500мс-2с
}

type FindOperator struct {
	Key, Operator, Value string
}

type UUID string

// Персистентное хранилище
type Storage[T any] interface {
	Store(T) UUID
	Get(UUID) T
	Find([]FindOperator) []T
	Update(UUID, T)
}

// Элемент кэша
type CacheItem struct {
	Key        string
	Value      []byte
	ExpiresAt  time.Time
	InProgress bool // Флаг обработки
	// разобраться с LockToken
	LockToken UUID // Токен блокировки
}

// Конфигурация прокси
type Config struct {
	BackendTimeout time.Duration // Таймаут вызова бэкенда
	CacheTTL       time.Duration // Время жизни кэша
	MaxConcurrency int           // Макс. одновременных запросов к бэкенду
}

type Item struct {
	uuid          UUID
	timeToExppire time.Time
}

type CacheProxy struct {
	backend  BackendClient
	storage  Storage[CacheItem]
	config   Config
	idToUUID map[string]Item
	wg       *sync.WaitGroup
	stopChan chan struct{}
	mu       *sync.RWMutex
	semChan  chan struct{}
	// сделать мапу, в которую бы записывался таймер,
	// по которому следует завершать ожидание
	// Далее - поля для реализации
}

func New(
	backend BackendClient,
	storage Storage[CacheItem],
	config Config,
) *CacheProxy {
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	idToUUID := make(map[string]Item, 10)
	semChan := make(chan struct{}, config.MaxConcurrency)

	wg.Add(1)

	cacheProxy := &CacheProxy{
		backend:  backend,
		storage:  storage,
		config:   config,
		wg:       &wg,
		stopChan: stopChan,
		idToUUID: idToUUID,
		mu:       &sync.RWMutex{},
		semChan:  semChan,
	}
	go cacheProxy.CleanupExpired()
	return cacheProxy
}

func (cp *CacheProxy) saveCacheItem(id string, result []byte, inProgress bool) {
	if inProgress {
		cacheItem := CacheItem{
			Key:        id,
			InProgress: true,
		}
		cp.mu.Lock()
		uuid := cp.storage.Store(cacheItem)
		cp.idToUUID[id] = Item{
			uuid: uuid,
		}
		cp.mu.Unlock()
		return
	}
	timeNow := time.Now()
	timeToExpire := timeNow.Add(cp.config.CacheTTL)
	cacheItem := CacheItem{
		Key:        id,
		Value:      result,
		ExpiresAt:  timeToExpire,
		InProgress: false,
	}
	cp.mu.Lock()
	uuid := cp.storage.Store(cacheItem)
	cp.idToUUID[id] = Item{
		uuid:          uuid,
		timeToExppire: timeToExpire,
	}
	cp.mu.Unlock()
}

// Основной метод получения ресурса
func (cp *CacheProxy) GetResource(id string) ([]byte, error) {
	cp.mu.RLock()
	item, exists := cp.idToUUID[id]

	if exists {
		cacheItem := cp.storage.Get(item.uuid)
		cp.mu.RUnlock()
		// подумать, а что делать если элемент протух по времени и удалился из кэша
		for {
			time.Sleep(10 * time.Second)
			timeNow := time.Now()
			cp.mu.RLock()
			cacheItem = cp.storage.Get(item.uuid)
			if !cacheItem.InProgress {
				if timeNow.After(cacheItem.ExpiresAt) && (len(cacheItem.Value) != 0) {
					cp.mu.RUnlock()
					return cacheItem.Value, nil
				}
				cacheItem.InProgress = true
				cp.storage.Update(item.uuid, cacheItem)
				cp.mu.RUnlock()
				break
			}
			cp.mu.RUnlock()
		}
	} else {
		cp.mu.RUnlock()
		cp.saveCacheItem(id, nil, true)
	}
	cp.semChan <- struct{}{}
	result, err := cp.backend.GetResource(id)
	<-cp.semChan
	// изменить: не надо создавать заново элемент
	cp.saveCacheItem(id, result, false)
	return result, err
}

// Метод для очистки просроченных записей (вызывается периодически)
func (cp *CacheProxy) CleanupExpired() {
	defer cp.wg.Done()
	var timeNow time.Time
	for {
		select {
		case <-cp.stopChan:
			return
		default:
			copy := make(map[string]Item, 10)
			cp.mu.RLock()
			for key, value := range cp.idToUUID {
				copy[key] = value
			}
			cp.mu.RUnlock()
			for id, item := range copy {
				timeNow = time.Now()
				if timeNow.After(item.timeToExppire) {
					delete(copy, id)
				}
			}

			for id := range copy {
				cp.mu.Lock()
				delete(cp.idToUUID, id)
				cp.mu.Unlock()
			}

			time.Sleep(3 * time.Second)
		}
	}
}

// Метод восстановления состояния при старте
func (cp *CacheProxy) RecoverStaleLocks() {
	// TODO: освобождать зависшие блокировки
}
