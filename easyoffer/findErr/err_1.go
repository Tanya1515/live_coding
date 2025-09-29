package main

import (
	"fmt"
	"sync"

	"golang.org/x/exp/rand"
)

/*

Дана задача. Необходимо внести исправления в код,
который генерирует случайные числа от 0 до 9, выбирает среди
них повторяющиеся значения, проверяет их на уникальность и
отправляет уникальные числа в канал.

*/

func main() {
	// забыли проинициализировать мапу
	// было: var alreadyStored map[int]struct{} // nil map
	alreadyStored := make(map[int]struct{}, 0) 
	mu := sync.RWMutex{}
	capacity := 1000

	doubles := make([]int, 0, capacity)
	for i := 0; i < capacity; i++ {
		doubles = append(doubles, rand.Intn(10)) // create rand num 0..9
	}
	// 3, 4, 5, 0, 4, 9, 9, 8, 6, 6, 5, 5, 4, 4, 2, 1, 2, 3, 1 ...

	uniqueIDs := make(chan int, capacity)
	wg := sync.WaitGroup{}

	for i := 0; i < capacity; i++ {
		i := i

		wg.Add(1)
		go func() {
			defer wg.Done()
			// гонка данных, поскольку несколько горутин могут попробовать одновременно
			// прочитать данные из мапы и одновременно начать ее модифицировать и записывать
			// в канал уже не уникальное число

			// Было: if _, ok := alreadyStored[doubles[i]]; !ok {
			// 	mu.Lock()
			// 	alreadyStored[doubles[i]] = struct{}{}
			// 	mu.Unlock()
			// 	uniqueIDs <- doubles[i]
			// }
			mu.RLock()
			_, ok := alreadyStored[doubles[i]]
			mu.RUnlock()
			if !ok {
				mu.Lock()
				if _, ok := alreadyStored[doubles[i]]; !ok {
					alreadyStored[doubles[i]] = struct{}{}
					uniqueIDs <- doubles[i]
				}
				mu.Unlock()
			}
		}()
	}

	// забыли закрыть канал uniqueIDs
	// было: wg.Wait() - без специальной горутины и закрытия канала 
	go func() {
		wg.Wait()
		close(uniqueIDs)
	}()

	for val := range uniqueIDs {
		fmt.Println(val)
	}
	// нормальное поведение, будет просто напечатан адрес, например, 0xc0000bc000
	fmt.Println(uniqueIDs)
}
