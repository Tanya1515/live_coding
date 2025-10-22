package main

import (
	"fmt"
	"sync"
)

// Fan_out - паттерн, который разделяет один канал на несколько.

// Алгоритм:
// 1) Создали результирующие каналы
// 2) Вернули массив результирующих каналов
// 3) Асинхронно начинаем писать данные в каналы

func fanOut(inputCh <-chan int, n int) []chan int {
	channels := make([]chan int, 0, n)
	for j := 0; j < n; j++ {
		channel := make(chan int)
		channels = append(channels, channel)
	}
	var i int
	go func() {
		defer func() {
			for _, channel := range channels {
				close(channel)
			}
		}()
		for num := range inputCh {
			channels[i] <- num
			i = (i + 1) % n
		}
	}()
	return channels
}

func main() {
	var wg sync.WaitGroup
	inputChan := make(chan int)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(inputChan)
		for i := 0; i < 50; i++ {
			inputChan <- i
		}
	}()
	channels := fanOut(inputChan, 5)

	for ind, channel := range channels {
		wg.Add(1)

		go func(ind int) {
			defer wg.Done()
			for num := range channel {
				fmt.Printf("Data from channel %d: %d\n", ind, num)
			}
		}(ind)
	}

	wg.Wait()
}
