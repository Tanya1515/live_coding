package main

// Задача:
// Создайте пулл горутин (Worker Pool), который выполняет задачи
// (func()) с ограничением на максимальное количество одновременно
// работающих горутин

import (
	"errors"
	"sync"
)

type WorkerPool struct {
	WG           *sync.WaitGroup
	CountChannel chan struct{}
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		return nil
	}
	var wg sync.WaitGroup
	countChannel := make(chan struct{}, maxWorkers)

	return &WorkerPool{WG: &wg, CountChannel: countChannel}
}

func (wp *WorkerPool) Submit(task func()) error {

	select {
	case wp.CountChannel <- struct{}{}:
		wp.WG.Add(1)
		go func(wg *sync.WaitGroup, countChan chan struct{}) {
			defer wg.Done()
			task()
			<-countChan
		}(wp.WG, wp.CountChannel)
	default:
		return errors.New("pool is full")
	}
	return nil
}

func (wp *WorkerPool) Stop() {
	wp.WG.Wait()
	close(wp.CountChannel)
}
