package main

import (
	"fmt"
	"strings"
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

var Base32Geohash = map[rune]string{
	'0': "00000",
	'1': "00001",
	'2': "00010",
	'3': "00011",
	'4': "00100",
	'5': "00101",
	'6': "00110",
	'7': "00111",
	'8': "01000",
	'9': "01001",
	'b': "01010",
	'c': "01011",
	'd': "01100",
	'e': "01101",
	'f': "01110",
	'g': "01111",
	'h': "10000",
	'j': "10001",
	'k': "10010",
	'm': "10011",
	'n': "10100",
	'p': "10101",
	'q': "10110",
	'r': "10111",
	's': "11000",
	't': "11001",
	'u': "11010",
	'v': "11011",
	'w': "11100",
	'x': "11101",
	'y': "11110",
	'z': "11111",
}

var GeohashBase32 = map[string]rune{
	"00000": '0',
	"00001": '1',
	"00010": '2',
	"00011": '3',
	"00100": '4',
	"00101": '5',
	"00110": '6',
	"00111": '7',
	"01000": '8',
	"01001": '9',
	"01010": 'b',
	"01011": 'c',
	"01100": 'd',
	"01101": 'e',
	"01110": 'f',
	"01111": 'g',
	"10000": 'h',
	"10001": 'j',
	"10010": 'k',
	"10011": 'm',
	"10100": 'n',
	"10101": 'p',
	"10110": 'q',
	"10111": 'r',
	"11000": 's',
	"11001": 't',
	"11010": 'u',
	"11011": 'v',
	"11100": 'w',
	"11101": 'x',
	"11110": 'y',
	"11111": 'z',
}

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

/*
	1) Каждую букву из Base32 необходимо преобразовать в последовательность из 5 битов
	2) Затем поочередном обходим массив, если четная позиция - долгота, нечетная позиция - широта.
	3) В зависимости от значения бита - выбираем, к какому диапозону относится точка и делим его пополам.
	4) В цикле обхим биты и вычисляем примерные координаты geohash
	5) Возвращаем итоговую координату.
*/

func geoHashDecode(geoHash string) GeoPoint {

	geoHashBit := make([]rune, 0)
	var latitudeUpperBounder, latitudeLowerBounder, longtitudeUpperBounder, longtitudeLowerBounder float64
	latitudeUpperBounder = 90
	latitudeLowerBounder = -90

	longtitudeUpperBounder = 180
	longtitudeLowerBounder = -180

	for _, let := range geoHash {
		bitLet, _ := Base32Geohash[let]
		for _, bit := range bitLet {
			geoHashBit = append(geoHashBit, bit)
		}
	}
	for key, value := range geoHashBit {
		if key%2 == 0 {
			if string(value) == "1" {
				longtitudeLowerBounder = (longtitudeLowerBounder + longtitudeUpperBounder) / 2
			} else {
				longtitudeUpperBounder = (longtitudeUpperBounder + longtitudeLowerBounder) / 2
			}
		} else {
			if string(value) == "1" {
				latitudeLowerBounder = (latitudeLowerBounder + latitudeUpperBounder) / 2
			} else {
				latitudeUpperBounder = (latitudeUpperBounder + latitudeLowerBounder) / 2
			}
		}
	}

	point := GeoPoint{Lat: ((latitudeUpperBounder-latitudeLowerBounder)/2 + latitudeLowerBounder), Lng: ((longtitudeUpperBounder-longtitudeLowerBounder)/2 + longtitudeLowerBounder)}

	return point
}

/*
 1. определяем границы по широте и долготе: [-90, 90] и [-180, 180] соответсвенно
 2. Делим каждый из диапозонов пополам, для широты: [-90, 0] и [0, 90], например.
 3. Если координата по широте находится в дипозоне [-90, 0] - записываем 0,
    в диапозоне [0, 90] - записываем 1.
 4. Аналогично для долготы.
 5. Диапозон сужается и алгоритм повторяется, начиная с 3его пункта.
 6. Собираем результирующий массив: на четные позиции записываются биты долготы, на нечетные - биты широты.
 7. Биты в результирующем массиве разбиваются по 5 бит и кодируются в Base32: 0123456789bcdefghjkmnpqrstuvwxyz
*/

// Лучше рещать через битовые сдвиги

// Пример кода:

/*

	mid = (latLower + latUpper) / 2
    if point.Lat >= mid {
        bits = (bits << 1) | 1
        latLower = mid
    } else {
        bits <<= 1
        latUpper = mid
    }

	Преобразовывать можно следующим образом:

	То есть здесь происходит сдвиг вправо и побитовое
	умножение, чтобы оставить последние 5 битов.

	idx := (bits >> (bitCount - 5)) & 0x1F
    result.WriteByte(GeohashBase32[idx])

	Здесь: bits >> (bitCount - 5) - сдвиг числа bits вправо на (bitCount - 5)
	0x1F - шестнадцатеричное число, равное 31 в десятичной - 00011111 в двоичной.
	При умножении на это число остается только последние 5 битов результата сдвига.

*/

func geoHashCode(point GeoPoint) string {

	var latitudeUpperBounder, latitudeLowerBounder, longtitudeUpperBounder, longtitudeLowerBounder float64
	latitudeUpperBounder = 90
	latitudeLowerBounder = -90

	longtitudeUpperBounder = 180
	longtitudeLowerBounder = -180

	bitResult := make([]string, 0)

	var result strings.Builder
	var res strings.Builder

	precision := 30

	for precision > 0 {

		precision--
		if point.Lng >= (longtitudeUpperBounder+longtitudeLowerBounder)/2 {
			bitResult = append(bitResult, "1")
			longtitudeLowerBounder = (longtitudeLowerBounder + longtitudeUpperBounder) / 2
		} else {
			bitResult = append(bitResult, "0")
			longtitudeUpperBounder = (longtitudeUpperBounder + longtitudeLowerBounder) / 2
		}

		precision--
		if point.Lat >= (latitudeLowerBounder+latitudeUpperBounder)/2 {
			bitResult = append(bitResult, "1")
			latitudeLowerBounder = (latitudeLowerBounder + latitudeUpperBounder) / 2
		} else {
			bitResult = append(bitResult, "0")
			latitudeUpperBounder = (latitudeUpperBounder + latitudeLowerBounder) / 2
		}

	}

	fmt.Println(bitResult)

	for key, let := range bitResult {
		if key%5 == 0 {
			letRes := GeohashBase32[res.String()]
			result.WriteRune(letRes)
			res.Reset()
		}
		res.WriteString(let)
	}

	letRes := GeohashBase32[res.String()]
	result.WriteRune(letRes)
	res.Reset()

	return result.String()
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

// Алгоритм следующий:

// 1. Вычисляем bounding box - прямоугольник, который охватывает все точки, которые входят в окружность
// 2. Вычисляем, какие geohash попадают в этот прямоугольник
// 3. Проходимся по конкретным точкам geohash из GeoCache и проверяем, принадлежат они окружности или нет.

func (gc *GeoCacheEx) GetInRadius(center GeoPoint, radius float64) (map[GeoPoint]CacheItem, error) {
	// var wg sync.WaitGroup
	// var mu sync.Mutex
	// result := make(map[GeoPoint]CacheItem, 10)

	// gc.mu.RLock()
	// geoHashMap := gc.hashMap
	// gc.mu.RUnlock()

	// for geoHash, pointsList := range geoHashMap {
	// 	wg.Add(1)
	// 	go func(){
	// 		defer wg.Done()
	// 		geoPoint := geoHashDecode(geoHash)
	// 	}()
	// }

	// wg.Wait()

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

// func main() {
// 	ex := geoHashCode(GeoPoint{Lat: 48.8588443, Lng: 2.2943506})
// 	fmt.Println(geoHashCode(GeoPoint{Lat: 48.8588443, Lng: 2.2943506}))
// 	fmt.Println(geoHashDecode(ex))
// }
