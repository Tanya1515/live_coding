package main

import "fmt"

func main() {
	ch := make(chan int, 1)
	ch <- 10
	close(ch)
	val, e := <-ch

	fmt.Println(val, e) // 10 true
}
