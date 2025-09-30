package main

import "fmt"

func main() {
	ch := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		ch <- i
	}

	close(ch)

	for true {
		fmt.Println(<-ch) // 1 2 3 4 5 0 0 0 0 0 0 0 ...
	}
}
