package main

import (
	"fmt"
	"sync"
	"time"
)

/*

Задача по Go:

Вывести числа от 1 до 10 функцией с PrintLn и задежкой
1 секунда выводом по 5 чисел в одном потоке

*/

func printNumber(n int) {
	time.Sleep(1 * time.Second)
	fmt.Println(n)
}

func main() {
	numbersPrint := make(chan int, 5)
	var wg sync.WaitGroup
	for i := 1; i < 11; i++ {
		numbersPrint <- i
		wg.Add(1)
		go func() {
			defer wg.Done()
			printNumber(i)
			<-numbersPrint
		}()
	}
	wg.Wait()
	close(numbersPrint)
}
