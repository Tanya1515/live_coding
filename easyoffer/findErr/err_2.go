package main

import (
	"fmt"
	"sync"
)

func main() {
	cnt := 100
	var wg sync.WaitGroup // не было синхронизации при помощи wait group
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println(i)
		}()
	}
	wg.Wait()
}
