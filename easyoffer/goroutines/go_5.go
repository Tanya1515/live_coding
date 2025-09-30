package main

import "fmt"

func main() {
	num := make(chan int)

	values := []int{1, 2, 3}
	for _, v := range values {
		go func() {
			num <- v
		}()
	}

	for _ = range values {
		fmt.Println(<-num) // 1 2 3 в рандомном порядке
	}
}
