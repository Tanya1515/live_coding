package main

import (
	"fmt"
	"sync"
)

// Fan_in - паттерн, который позволяет смержить несколько каналов.

// Алгоритм:

// 1) Создали результирующий канал
// 2) Вернули резульирующий канал
// 3) Асинхронно начинаем обрабатывать данные

// Функция, которая принимает на вход несколько каналов и возвращает один.
// Здесь ... - variardic parameters - тип параметров, которые позволяют
// принимать 0 или более аргументов конкретного типа.

func fanIn(channels ...chan int) chan int {
	result := make(chan int)
	var wg sync.WaitGroup
	for _, chanIn := range channels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for num := range chanIn {
				result <- num
			}
		}()
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return result
}

func main() {

	chan1 := make(chan int)
	chan2 := make(chan int)
	chan3 := make(chan int)
	chan4 := make(chan int)
	chan5 := make(chan int)

	go func() {
		defer close(chan1)
		for i := 0; i < 10; i++ {
			chan1 <- i
		}
	}()

	go func() {
		defer close(chan2)
		for i := 10; i < 20; i++ {
			chan2 <- i
		}
	}()

	go func() {
		defer close(chan3)
		for i := 20; i < 30; i++ {
			chan3 <- i
		}
	}()

	go func() {
		defer close(chan4)
		for i := 30; i < 40; i++ {
			chan4 <- i
		}
	}()

	go func() {
		defer close(chan5)
		for i := 40; i <= 50; i++ {
			chan5 <- i
		}
	}()

	for num := range fanIn(chan1, chan2, chan3, chan4, chan5) {
		fmt.Println(num)
	}
}
