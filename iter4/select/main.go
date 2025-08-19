package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)

	go func() {
		time.Sleep(1 * time.Second)

		ch <- 42

		close(ch)
	}()

	for {
		select {
		case val, ok := <-ch:
			if !ok {
				fmt.Println("Channel closed")

				return
			}

			fmt.Println(val)

		default:
			fmt.Println("No value yet")

			time.Sleep(200 * time.Millisecond)
		}
	}
}
