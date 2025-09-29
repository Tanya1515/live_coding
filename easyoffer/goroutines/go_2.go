package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	m := make(map[int]int)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			m[key] = key * key // будет ошибка конкуретного доступа к мапе на запись (попытки записи в мапу из нескольких горутин)
		}(i)
	}
	wg.Wait()

	for key, val := range m {
		fmt.Printf("Key; %d, Value: %d\n", key, val)
	}
}
