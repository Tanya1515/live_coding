package main

import (
	"fmt"
)

func main() {
	a := 0
	defer fmt.Println(a) // 0
	defer func() {
		a++
		fmt.Println(a) // 1
	}()
}

// Общий вывод: 1 0
