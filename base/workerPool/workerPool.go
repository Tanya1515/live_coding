package main

import (
	"fmt"
	"sync"
	"time"
)

// Паттерн Worker Pool - основная идея заключается в том, что запускается
// некоторое множество горутин, которые ждут исполнения задач. Когда
// горутина выполнила некоторую задачу, горутина не завершается, а продолжает
// ждать, когда ей делегируют другую задачу. На этапе решения задачи на
// собеседовании нужно обговорить детали, на базе которых будет реализована задача:

// 1) Когда добавляется задача, а готовых воркеров либо нет, либо они все заняты,
// как должна вести себя программа: блокироваться или возвращать ошибку.
// 2) Как завершить воркер: при помощи контекста или при помощи контекста или
// при помощи shutdown - то есть дождаться, когда все воркеры завершаться.
// 3) Как быть с задачами, которые находятся в буфере?

func worker(inputChan chan int, wg *sync.WaitGroup, resultChan chan int) {
	defer wg.Done()
	for {
		select {
		case value, ok := <-inputChan:
			if !ok {
				return
			}
			resultChan <- value * 2
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	amountOfWorkers := 5
	inputChan := make(chan int, amountOfWorkers)
	resultChan := make(chan int)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer close(inputChan)
		defer wg.Done()
		for i := 0; i <= 50; i++ {
			inputChan <- i
		}
	}()

	for i := 0; i < amountOfWorkers; i++ {
		wg.Add(1)
		go worker(inputChan, &wg, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for num := range resultChan {
		fmt.Println(num)
	}
}
