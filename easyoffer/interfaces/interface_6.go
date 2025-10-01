package main

import "fmt"

/*

Будет напечатано: x is not nil

Поскольку в x лежит указатель на тип a

*/

func main() {
	var a *int
	var x interface{}

	x = a

	if x == nil {
		fmt.Println("x is nil")
		return
	}
	fmt.Println("x is not nil")
}
