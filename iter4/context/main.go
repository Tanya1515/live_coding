package main

import (
	"context"
	"fmt"
	"time"
)

func worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Worker done:", ctx.Err())
			return
		default:
			fmt.Println("Working...")
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go worker(ctx)

	time.Sleep(3 * time.Second)
}

// Шаги исполнения программы: 
// 1) Создание context-а c timeout-ом на 2 секунды 
// 2) Запуск горутины 
// 3) Main-горутина заснет на 3 секунды 
// 4) Вывод в консоль: "Working..." и заснет на 500 Millisecond
// 5) Вывод в консоль: "Working..." и заснет на 500 Millisecond 
// 6) Вывод в консоль: "Working..." и заснет на 500 Millisecond 
// 7) Вывод в консоль: "Working..." и заснет на 500 Millisecond 
// 8) Сработает timeout для context-а: вывод в консоль -  "Worker done:" и ctx.Err() 
// 9) Горутина завершит свою работу и затем main-горутина завершит сво работу. 
