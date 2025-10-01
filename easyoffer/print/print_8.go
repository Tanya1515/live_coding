package main

import (
	"fmt"
	"time"
)

/* 

1) Сначала будет блокировка на первом <-worker() - 3 Second

2) Затем после того, как будет запись в канал - разблокируется первая переменная

3) Сначала будет блокировка на первом <-worker() - еще 3 Second

4) Далее напечатается 6 Second

*/

func main() {
	timeStart := time.Now()
	_, _ = <-worker(), <-worker()
	fmt.Println(int(time.Since(timeStart).Seconds()))

}

func worker() chan int {
	ch := make(chan int)
	go func() {
		time.Sleep(3 * time.Second)
		ch <- 1
	}()
	return ch

}
