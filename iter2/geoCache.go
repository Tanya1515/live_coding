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

3) Оптимальный поиск по радиусу (не перебирать все точки) - разделить на квадраты + посмотреть на 
сторону  квадрату < радиуса и посмотреть на точки, которые попадают

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

	return geoCache
}

func geoHashDecode(geoHash string) GeoPoint {
	/*
	1) Каждую букву из Base32 необходимо преобразовать в последовательность из 5 битов
	2) Затем поочередном обходим массив, если четная позиция - долгота, нечетная позиция - широта. 
	3) В зависимости от значения бита - выбираем, к какому диапозону относится точка и делим его пополам.
	4) В цикле обхим биты и вычисляем примерные координаты geohash
	5) Возвращаем итоговую координату. 
	*/
	return GeoPoint{}
} 

func geoHashCode(point GeoPoint) string {
	/*
	1) определяем границы по широте и долготе: [-90, 90] и [-180, 180] соответсвенно
	2) Делим каждый из диапозонов пополам, для широты: [-90, 0] и [0, 90], например. 
	3) Если координата по широте находится в дипозоне [-90, 0] - записываем 0, 
	в диапозоне [0, 90] - записываем 1. 
	4) Аналогично для долготы. 
	5) Диапозон сужается и алгоритм повторяется, начиная с 3его пункта.
	6) Собираем результирующий массив: на четные позиции записываются биты долготы, на нечетные - биты широты. 
	7) Биты в результирующем массиве разбиваются по 5 бит и кодируются в Base32: 0123456789bcdefghjkmnpqrstuvwxyz 
	*/
	return ""
}

func (gc *GeoCacheEx) Set(point GeoPoint, item CacheItem) error {

	hashPoint := geoHashCode(point)

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
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make(map[GeoPoint]CacheItem, 10)

	gc.mu.RLock()
	geoHashMap := gc.hashMap
	gc.mu.RUnlock()

	for geoHash, pointsList := range geoHashMap {
		wg.Add(1)
		go func(){
			defer wg.Done()
			geoPoint := geoHashDecode(geoHash)
		}()
	}

	wg.Wait()
	// можно итерироаться по хэшам и проверять каждый отдельный хэш в горутине, и затем итерироватьс 
	// по точкам, если они подходят - складываем в результирующую мапу 
	return result, nil
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
