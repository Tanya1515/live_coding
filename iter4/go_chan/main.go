package main

import (
	"fmt"
	"sync"
)

func main() {
	ch := make(chan int, 1)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		ch <- 42
		close(ch)
	}()

	go func() {
		defer wg.Done()
		val := <-ch
		fmt.Println(val)
	}()

	wg.Wait()
}
