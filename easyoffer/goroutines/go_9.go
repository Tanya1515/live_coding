package main

import (
	"fmt"
	"time"
)

// Будет напечатано 6. 

func worker() chan int {
	ch := make(chan int)

	go func() {
		time.Sleep(3 * time.Second)
		ch <- 42
	}()

	return ch
}
func main() {
	timeStart := time.Now()

	_, _ = <-worker(), <-worker()
	fmt.Println(int(time.Since(timeStart).Seconds()))
}
