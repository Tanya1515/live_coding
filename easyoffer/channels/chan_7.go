package main

import (
	"fmt"
	"time"
)

// Deadlock

func main() {
	// небуферизованный канал
	sync := make(chan bool)

	go func() {
		time.Sleep(time.Second * 3)
		fmt.Println("get normal  signal") // напечатается "get normal  signal"
		// блокировка, пока не прочитается значение
		sync <- false
	}()

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println("ger interrupted signal") // напечатается "ger interrupted signal"
			// блокировка, пока не прочитается значение
			sync <- true
		case value := <-sync:
			fmt.Printf("finish %t", value)
			return
		}
	}
}
