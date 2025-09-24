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

	"github.com/google/uuid"
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

type Item struct {
	uuid      UUID
	ExpiresAt time.Time
}

type CacheProxy struct {
	stopChan  chan struct{}
	back      BackendClient
	config    Config
	storage   Storage[CacheItem]
	mu        *sync.RWMutex
	uuidCache map[string]Item
	sem       chan struct{}
}

func New(
	backend BackendClient,
	storage Storage[CacheItem],
	config Config,
) *CacheProxy {
	stopChan := make(chan struct{})
	itemCache := make(map[string]Item, 10)
	sem := make(chan struct{}, config.MaxConcurrency)

	cache := &CacheProxy{
		back:      backend,
		stopChan:  stopChan,
		mu:        &sync.RWMutex{},
		uuidCache: itemCache,
		sem:       sem,
		storage:   storage,
	}
	cache.RecoverStaleLocks()
	go cache.CleanupExpired()
	return cache
}

func (cp *CacheProxy) saveElem(id string, data []byte, inProgress bool, uuidToken UUID) {
	var cacheItem CacheItem
	startTime := time.Now()
	cacheItem.InProgress = inProgress
	if inProgress {
		cacheItem.Key = id
		cacheItem.LockExpiresAt = startTime.Add(100 * time.Second)
		cacheItem.LockToken = UUID(uuid.New().String())
		if uuidToken == "" {
			uuidToken = cp.storage.Store(cacheItem)
		} else {
			cp.storage.Update(uuidToken, cacheItem)
		}
		cp.uuidCache[id] = Item{uuid: uuidToken}
		return
	}
	if len(data) != 0 {
		cacheItem.Value = data
		cacheItem.LockToken = ""
		cacheItem.LockExpiresAt = time.Time{}
		cacheItem.ExpiresAt = startTime.Add(cp.config.CacheTTL)
	}
	item := cp.uuidCache[id]
	cp.uuidCache[id] = Item{uuid: item.uuid, ExpiresAt: cacheItem.ExpiresAt}
	cp.storage.Update(item.uuid, cacheItem)
	return
}
func (cp *CacheProxy) checkElem(item CacheItem) bool {
	return time.Now().After(item.LockExpiresAt)
}

// Основной метод получения ресурса
func (cp *CacheProxy) GetResource(id string) ([]byte, error) {
	data := make([]byte, 0)
	sliceOperator := make([]FindOperator, 0)
	findOperator := FindOperator{
		Key:      "Key",
		Operator: "=",
		Value:    id,
	}
	sliceOperator = append(sliceOperator, findOperator)
	// находим cacheItem, если он существует
	cp.mu.Lock()
	cacheItems := cp.storage.Find(sliceOperator)
	if len(cacheItems) == 0 {
		cp.saveElem(id, data, true, "")
		cp.mu.Unlock()
	} else {
		cp.mu.Unlock()
		for {
			// проверяем, появился ли элемент в cache? 
			cp.mu.Lock()
			items := cp.storage.Find(sliceOperator)
			var item CacheItem 
			if len(items) != 0 {
				item = items[0]
				// если появился - проверяем, что элемент обработан и есть значение, которое актуально
				if !item.InProgress && len(item.Value) != 0 && time.Now().Before(item.ExpiresAt) {
					cp.mu.Unlock()
					return item.Value, nil
				}
				// если значения нет или оно неактуально, тогда этот конкретный инстанс будет теперь обслуживать 
				// этот элемент. 
				if !item.InProgress {
					if _, exists := cp.uuidCache[id]; exists && (item.LockToken == "" || time.Now().After(item.LockExpiresAt)) {
						cp.saveElem(id, data, true, item.RecordUUID)
					}
					cp.mu.Unlock()
					break
				}
				cp.mu.Unlock()
			}
			time.Sleep(3 * time.Second)
		}
	}
	cp.sem <- struct{}{}
	data, err := cp.back.GetResource(id)
	<-cp.sem
	cp.mu.Lock()
	cp.saveElem(id, data, false, "")
	cp.mu.Unlock()
	return data, err
}

// Метод для очистки просроченных записей (вызывается периодически)
func (cp *CacheProxy) CleanupExpired() {
	for {
		select {
		case <-cp.stopChan:
			return
		default:
			time.Sleep(500 * time.Second)
			copyCache := make(map[string]struct{}, len(cp.uuidCache))
			cp.mu.RLock()
			for key, value := range cp.uuidCache {
				if !time.Now().After(value.ExpiresAt){
					copyCache[key] = struct{}{}
				}
			}
			cp.mu.RUnlock()
			for key := range copyCache {
				cp.mu.Lock()
				if !time.Now().After(cp.uuidCache[key].ExpiresAt) {
					delete(cp.uuidCache, key)
				}
				cp.mu.Unlock()
			}
		}
	}
}

// Метод восстановления состояния при старте
func (cp *CacheProxy) RecoverStaleLocks() {
	sliceOperator := make([]FindOperator, 0)
	findOperator := FindOperator{
		Key:      "InProgress",
		Operator: "=",
		Value:    "true",
	}
	sliceOperator = append(sliceOperator, findOperator)
	cacheItems := cp.storage.Find(sliceOperator)

	for _, item := range cacheItems {
		if time.Now().After(item.LockExpiresAt) {
			item.InProgress = false
			item.LockToken = ""
			item.LockExpiresAt = time.Time{}
			cp.storage.Update(item.RecordUUID, item)
		}
	}
}
