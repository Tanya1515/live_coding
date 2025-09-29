package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

/*

runtime.GOMAXPROCS(1) означает:
1 логический поток ОС для выполнения горутин

НЕ ограничение количества горутин!

Go все равно создаст 3 горутины:

1) main goroutine

2) Первая анонимная горутина (с sleep)

3) Вторая анонимная горутина

Что происходит по шагам:

1) main goroutine запускается первой

2) Запускается горутина 1 → сразу уходит в sleep на 2 секунды

3) Запускается горутина 2 → печатает "2" и завершается

4) main goroutine вызывает wg.Wait() и блокируется

5) Через 2 секунды горутина 1 просыпается:

5.1) Печатает "1"

5.2) Вызывает wg.Done()

6) main goroutine разблокируется и печатает "3"

*/

func main() {
	runtime.GOMAXPROCS(1)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		time.Sleep(time.Second * 2)
		fmt.Println("1")
		wg.Done()
	}()

	go func() {
		fmt.Println("2")
	}()

	wg.Wait()

	fmt.Println("3")
}

// 2 1 3
