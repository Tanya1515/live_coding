package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Задача: Доработайте функцию ready() так, чтобы все бегуны
// стартовали одновременно только после того, как функция Stready() произнесет "МАРШ!".

// Htfkbpjdfnm  sync.Cond

func runnerStart() {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go ready(ctx, i, &wg)
	}

	Stready()
	cancel()
	wg.Wait()
}

func Stready() {
	// Функция должна обеспечить, чтобы все горутины started
	// одновременно после вызова этой функции
	fmt.Println("На старт...")
	time.Sleep(1 * time.Second)
	fmt.Println("Внимание...")
	time.Sleep(1 * time.Second)
	fmt.Println("МАРШ!")
}

func ready(ctx context.Context, id int, wg *sync.WaitGroup) {
	fmt.Printf("Бегун %d на стартовой позиции\n", id)
	<-ctx.Done()
	// TODO: Бегун ждет команды "МАРШ!"

	// Старт!
	fmt.Printf("Бегун %d начал бег!\n", id)
	time.Sleep(time.Duration(id+1) * time.Second) // Имитация бега
	fmt.Printf("Бегун %d финишировал!\n", id)
	wg.Done()
}

func main() {
	runnerStart()
}
