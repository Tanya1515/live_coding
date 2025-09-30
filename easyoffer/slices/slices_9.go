package main

import (
	"fmt"
)

// append изменяет исходный массив, если есть достаточная емкость

func main() {
	list := []int{1, 2, 3, 4}
	fmt.Println(cap(list))
	handle(list[:1])
	fmt.Println("after", list) // 1 5 3 4
}

func handle(list []int) {
	list = append(list, 5)
	fmt.Println("append", list) // 1 5
}
