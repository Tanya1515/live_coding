package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var counter atomic.Int32

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	// недерменированный результат из-за data race и отсутствия синхронизации горутин
	fmt.Println(counter.Load())
}
