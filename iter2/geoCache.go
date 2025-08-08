package main

import (
	"sync"
	"time"
)

/*

Полное условие: Реализуйте кеш для хранения объектов с привязкой к географическим координатам.

Требования:

1) Автоматическое партиционирование по geohash с precision=6 - данные автоматически разделяются
по различным партициям (разделам) на основе из geohash-значений с длинной в 6 символов.
Каждой партиции соответсвует определенная географическая область, определенная geohash-кодом.
Для каждой записи с географическими координатами (широта долгота) вычисляется geohash, длина
которого составляет 6 символов. Записи с одинаковыми geohash-кодами группируются в одну партицию.
Каждая партиция может хранится или обрабатываться отдельно.

2) Потокобезопасный доступ к разным партициям

3) Оптимальный поиск по радиусу (не перебирать все точки)

4) TTL-based инвалидация

Тесты:

 - Точность поиска в радиусе

 -  Конкурентный доступ

 - Распределение по партициям

*/

type GeoPoint struct {
	Lat float64 // [-90, 90]
	Lng float64 // [-180, 180]
}

type CacheItem struct {
	Value    interface{}
	Expires  time.Time
	Metadata map[string]string
}

type GeoCache interface {
	// Добавляет или обновляет точку
	Set(point GeoPoint, item CacheItem) error

	// Ищет точки в радиусе (в метрах)
	GetInRadius(center GeoPoint, radius float64) (map[GeoPoint]CacheItem, error)

	// Удаляет просроченные записи
	Cleanup(now time.Time) int
}

type GeoCacheEx struct {
	hashMap  map[string][]GeoPoint
	geoMap   map[GeoPoint]CacheItem
	mu       *sync.RWMutex
	stopChan chan struct{}
}

func NewGeoCahche() *GeoCacheEx {
	geoMap := make(map[GeoPoint]CacheItem, 10)
	hashMap := make(map[string][]GeoPoint)

	geoCache := &GeoCacheEx{
		hashMap: hashMap,
		geoMap:  geoMap,
		mu:      &sync.RWMutex{},
	}

	go geoCache.Cleanup(time.Now())

	return geoCache
}

func geoHashCode(point GeoPoint) string {
	// как вычислять geoHash??
	return ""
}

func (gc *GeoCacheEx) Set(point GeoPoint, item CacheItem) error {
	// 1) Выислить geohash

	hashPoint := geoHashCode(point)

	// 2) Добавить в мапу по hash-у под мьютексом

	gc.mu.Lock()
	if _, exists := gc.hashMap[hashPoint]; !exists {
		gc.hashMap[hashPoint] = make([]GeoPoint, 0)
		gc.hashMap[hashPoint] = append(gc.hashMap[hashPoint], point)
	}
	gc.geoMap[point] = item
	gc.mu.Unlock()

	return nil
}

func (gc *GeoCacheEx) GetInRadius(center GeoPoint, radius float64) (map[GeoPoint]CacheItem, error) {
	result := make(map[GeoPoint]CacheItem, 10)

	// 1) Итерируемся по GeoPoint, прибавляя каждый раз радиус. (Непонятно, какая дельта используется)
	// 2) На каждой итерации под RLock-ом читаем данные из кэша и сохраняем в возвращаемую мапу
	return nil, nil
}

func (gc *GeoCacheEx) Cleanup(now time.Time) int {
	var wg sync.WaitGroup

	removedElements := 0

	gc.mu.RLock()
	geoCache := gc.hashMap
	gc.mu.RUnlock()

	for _, geoArray := range geoCache {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, geo := range geoArray {
				gc.mu.Lock()
				if cacheIt, exists := gc.geoMap[geo]; exists {
					if !now.After(cacheIt.Expires) {
						delete(gc.geoMap, geo)
						removedElements++
					}
				}
				gc.mu.Unlock()
			}
		}()
	}
	wg.Wait()

	return removedElements

}
