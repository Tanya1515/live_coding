package main

import "fmt"

// Если передавать в функцию add указатель, 
// то будет напечатан весь модифицированный слайс 1 2 3 4

func main() {
	a := make([]int, 0, 3)
	a = append(a, 1) // 1, len = 1, cap = 3
	a = append(a, 2) // 1 2, len = 2, cap = 3
	add(a)
	fmt.Printf("%v", a) // 1 2
}

func add(a []int) {
	a = append(a, 3) // 1 2 3, измениттся длина
	a = append(a, 4) // выделится новый участок памяти, поскольку мы превзошла capacity
}
