package main

import "fmt"

/* 

Порядок выполнения при panic:
1) Выполняются все defer-функции (в LIFO порядке)
2) Если паника не была recovered, программа завершается

*/

func main() {
	defer fmt.Println("world")
	fmt.Println("hello")
	panic("error")
}

// Будет напечатано:
// "hello"
// "world"
// отработает panic
