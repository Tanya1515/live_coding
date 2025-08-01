package main

import (
	"sync/atomic"
)

// Задача: реализовать WaitGroup на базе каналов

type WaitGroup struct {
	GoroutinesCount int32
	Channel         chan struct{}
}

func NewWaitGrouop() *WaitGroup {
	waitGroupChannel := make(chan struct{}, 1)
	wg := WaitGroup{
		Channel: waitGroupChannel,
	}

	return &wg
}

func (wg *WaitGroup) Add(amount int32) {
	if atomic.AddInt32(&wg.GoroutinesCount, amount) < 0 {
		panic("Error: Invalid goroutines count")
	}
}

func (wg *WaitGroup) Wait() {

	for {
		<-wg.Channel
		if atomic.LoadInt32(&wg.GoroutinesCount) == 0 {
			return
		}
	}
}

func (wg *WaitGroup) Done() {
	atomic.AddInt32(&wg.GoroutinesCount, -1)
	if atomic.CompareAndSwapInt32(&wg.GoroutinesCount, -1, 0) {
		panic("Error: Invalid goroutines count")
	} else if atomic.LoadInt32(&wg.GoroutinesCount) == 0 {
		select {
		case wg.Channel <- struct{}{}:
		default:
		}
	}
}

// func main() {
// 	wg := NewWaitGrouop()
// 	wg.Add(3)

// 	for i := 0; i < 100; i++ {
// 		go func() {
// 			defer wg.Done()
// 			fmt.Println(i)
// 		}()
// 	}
// 	time.Sleep(1 * time.Second)
// 	wg.Add(4)
// 	fmt.Println("Hello!")
// 	for i := 0; i < 4; i++ {
// 		go func() {
// 			defer wg.Done()
// 			fmt.Println(i)
// 		}()
// 	}

// 	wg.Wait()
// }
