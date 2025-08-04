package main

import (
	"sync/atomic"
)

// Задача: реализовать WaitGroup на базе каналов
// Спросить, есть ли требования 

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

func (wg *WaitGroup) Add(amount uint32) {
	atomic.AddInt32(&wg.GoroutinesCount, int32(amount))
}

func (wg *WaitGroup) Wait() {

	for {
		<-wg.Channel
		if atomic.LoadInt32(&wg.GoroutinesCount) == 0 {
			return
		}
	}
}

func (wg *WaitGroup) Done() error {
	count := atomic.AddInt32(&wg.GoroutinesCount, -1)
	// так горутина будет работать с фиксированным count, которое меняться не будет в рамках одной горутины
	if count == -1 {
		panic("Invalid goroutines count")
	} else if count == 0 {
		select {
		case wg.Channel <- struct{}{}:
		default:
		}
	}
	return nil
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
