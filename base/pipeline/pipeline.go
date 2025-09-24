package main

import "fmt"

// Pipeline - паттерн, при котором:

// 1) Создается некоторая функция

// 2) В рамках этой функции создается результирующий канал

// 3) Канал возвращается

// 4) Асинхронно начинается обработка некоторых данных

// Задание - необходимо реализовать pipeline следующего вида:
// 1) Отдельная функция генерирует числа от 1 до 10
// 2) Следующая функция прибавляет 10 к каждому из чисел
// 3) Следующая функция умножает каждое из чисел на 2
// 4) В main-функции каждое из чисел печатается в канал.

func gen() chan int {
	chanGen := make(chan int, 5)
	go func() {
		defer close(chanGen)
		for i := 1; i < 11; i++ {
			chanGen <- i
		}
	}()
	return chanGen
}

func add(inputChan chan int) chan int {
	addChan := make(chan int, 5)
	go func() {
		defer close(addChan)
		for num := range inputChan {
			addChan <- num + 10
		}
	}()
	return addChan
}

func mul(inputChan chan int) chan int {
	resultChan := make(chan int, 5)
	go func() {
		defer close(resultChan)
		for num := range inputChan {
			resultChan <- num * 2
		}
	}()
	return resultChan
}

func main() {
	result := gen()

	for num := range mul(add(result)) {
		fmt.Println(num)
	}

}
