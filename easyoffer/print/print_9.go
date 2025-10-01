package main

import "fmt"

type Count int

func (c Count) Increment() {
	c++
}

func main() {
	var count Count
	count.Increment() // копируем count, и увеличиваем его, но потом 
	fmt.Print(count) // 0, поскольку модифицировалась копия
}
